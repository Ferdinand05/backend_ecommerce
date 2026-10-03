package productimage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"ferdinand/ecommerce/internal/product"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeService struct {
	uploadRes ProductImageResponse
	uploadErr error

	findByIDRes ProductImageResponse
	findByIDErr error

	findAllRes []ProductImageResponse
	findAllErr error

	deleteErr error
}

func (f *fakeService) Upload(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, file io.Reader, contentType string, alt *string, sortOrder int) (ProductImageResponse, error) {
	return f.uploadRes, f.uploadErr
}

func (f *fakeService) FindByID(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, imageID uuid.UUID) (ProductImageResponse, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeService) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]ProductImageResponse, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeService) FindAllByVariantID(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) ([]ProductImageResponse, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeService) Delete(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, imageID uuid.UUID) error {
	return f.deleteErr
}

type imageEnvelope struct {
	Image ProductImageResponse `json:"image"`
}

type imagesEnvelope struct {
	Images []ProductImageResponse `json:"images"`
}

func newTestRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(svc))
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func multipartBody(t *testing.T, content []byte, fields map[string]string) (string, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormFile("file", "upload.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return body.String(), writer.FormDataContentType()
}

var pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

func TestHandlerUploadProductImage(t *testing.T) {
	productID := uuid.New()
	svc := &fakeService{uploadRes: ProductImageResponse{ID: uuid.New(), ProductID: productID, URL: "https://cdn.test/products/x.png"}}

	body, contentType := multipartBody(t, pngMagic, map[string]string{"alt": "front", "sort_order": "1"})
	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+productID.String()+"/images", contentType, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got imageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Image.URL != "https://cdn.test/products/x.png" {
		t.Fatalf("url = %q, want cdn url", got.Image.URL)
	}
}

func TestHandlerUploadVariantImage(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	svc := &fakeService{uploadRes: ProductImageResponse{ID: uuid.New(), ProductID: productID, URL: "https://cdn.test/products/x.webp"}}

	body, contentType := multipartBody(t, pngMagic, nil)
	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+productID.String()+"/variants/"+variantID.String()+"/images", contentType, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandlerUploadMissingFile(t *testing.T) {
	body := "alt=front"
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/images", "application/x-www-form-urlencoded", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUploadUnsupportedType(t *testing.T) {
	svc := &fakeService{uploadErr: ErrorUnsupportedImageType}

	body, contentType := multipartBody(t, []byte("just text, not an image"), nil)
	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/images", contentType, body)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandlerUploadTooLarge(t *testing.T) {
	huge := make([]byte, maxImageSize+1)

	body, contentType := multipartBody(t, huge, nil)
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/images", contentType, body)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandlerUploadInvalidProductID(t *testing.T) {
	body, contentType := multipartBody(t, pngMagic, nil)
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products/not-a-uuid/images", contentType, body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUploadInvalidVariantID(t *testing.T) {
	body, contentType := multipartBody(t, pngMagic, nil)
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/variants/not-a-uuid/images", contentType, body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUploadProductNotFound(t *testing.T) {
	svc := &fakeService{uploadErr: product.ErrorProductNotFound}

	body, contentType := multipartBody(t, pngMagic, nil)
	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/images", contentType, body)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerUploadError(t *testing.T) {
	svc := &fakeService{uploadErr: errors.New("boom")}

	body, contentType := multipartBody(t, pngMagic, nil)
	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/images", contentType, body)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerFindAllByProductID(t *testing.T) {
	svc := &fakeService{findAllRes: []ProductImageResponse{
		{ID: uuid.New(), URL: "https://cdn.test/a.png"},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/images", "", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got imagesEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Images) != 1 {
		t.Fatalf("images len = %d, want 1", len(got.Images))
	}
}

func TestHandlerFindAllByVariantID(t *testing.T) {
	svc := &fakeService{findAllRes: []ProductImageResponse{
		{ID: uuid.New(), URL: "https://cdn.test/v.png"},
	}}

	path := "/api/v1/products/" + uuid.New().String() + "/variants/" + uuid.New().String() + "/images"
	w := doRequest(t, newTestRouter(svc), http.MethodGet, path, "", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerFindProductImageByID(t *testing.T) {
	imageID := uuid.New()
	svc := &fakeService{findByIDRes: ProductImageResponse{ID: imageID, URL: "https://cdn.test/x.png"}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/images/"+imageID.String(), "", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got imageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Image.ID != imageID {
		t.Fatalf("id = %v, want %v", got.Image.ID, imageID)
	}
}

func TestHandlerFindProductImageByIDInvalidUUID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/images/not-a-uuid", "", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerFindProductImageByIDNotFound(t *testing.T) {
	svc := &fakeService{findByIDErr: ErrorProductImageNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/images/"+uuid.New().String(), "", "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerDeleteProductImage(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodDelete, "/api/v1/products/"+uuid.New().String()+"/images/"+uuid.New().String(), "", "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandlerDeleteProductImageNotFound(t *testing.T) {
	svc := &fakeService{deleteErr: ErrorProductImageNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodDelete, "/api/v1/products/"+uuid.New().String()+"/images/"+uuid.New().String(), "", "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerDeleteVariantImage(t *testing.T) {
	path := "/api/v1/products/" + uuid.New().String() + "/variants/" + uuid.New().String() + "/images/" + uuid.New().String()
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodDelete, path, "", "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
