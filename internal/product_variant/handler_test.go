package productvariant

import (
	"context"
	"encoding/json"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/product"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeService struct {
	findAllRes  []models.ProductVariant
	findAllErr  error
	findByIDRes models.ProductVariant
	findByIDErr error

	createRes models.ProductVariant
	createErr error

	updateRes models.ProductVariant
	updateErr error

	deleteErr error
}

func (f *fakeService) Create(ctx context.Context, productID uuid.UUID, req CreateProductVariantRequest) (models.ProductVariant, error) {
	return f.createRes, f.createErr
}

func (f *fakeService) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeService) FindByID(ctx context.Context, productID uuid.UUID, id uuid.UUID) (models.ProductVariant, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeService) Update(ctx context.Context, productID uuid.UUID, id uuid.UUID, req UpdateProductVariantRequest) (models.ProductVariant, error) {
	return f.updateRes, f.updateErr
}

func (f *fakeService) Delete(ctx context.Context, productID uuid.UUID, id uuid.UUID) error {
	return f.deleteErr
}

type variantEnvelope struct {
	Variant ProductVariantResponse `json:"variant"`
}

type variantsEnvelope struct {
	Variants []ProductVariantResponse `json:"variants"`
}

func newTestRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(svc))
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestHandlerFindAll(t *testing.T) {
	svc := &fakeService{findAllRes: []models.ProductVariant{
		{ID: uuid.New(), SKU: "LAP-001", Name: "16GB", Price: decimal.NewFromInt(15000000), IsActive: true},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/variants", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got variantsEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Variants) != 1 {
		t.Fatalf("variants len = %d, want 1", len(got.Variants))
	}

	if got.Variants[0].SKU != "LAP-001" {
		t.Fatalf("sku = %q, want LAP-001", got.Variants[0].SKU)
	}
}

func TestHandlerFindAllError(t *testing.T) {
	svc := &fakeService{findAllErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/variants", "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerFindAllInvalidProductID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, "/api/v1/products/not-a-uuid/variants", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerFindByID(t *testing.T) {
	productID := uuid.New()
	id := uuid.New()
	svc := &fakeService{findByIDRes: models.ProductVariant{ID: id, ProductID: productID, SKU: "LAP-001"}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+productID.String()+"/variants/"+id.String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got variantEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Variant.ID != id {
		t.Fatalf("id = %v, want %v", got.Variant.ID, id)
	}
}

func TestHandlerFindByIDInvalidVariantID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/variants/not-a-uuid", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerFindByIDNotFound(t *testing.T) {
	svc := &fakeService{findByIDErr: ErrorProductVariantNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String()+"/variants/"+uuid.New().String(), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreate(t *testing.T) {
	productID := uuid.New()
	svc := &fakeService{createRes: models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		SKU:       "LAP-001",
		Name:      "16GB",
		Price:     decimal.NewFromInt(15000000),
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+productID.String()+"/variants",
		`{"sku":"LAP-001","name":"16GB","price":15000000}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got variantEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Variant.SKU != "LAP-001" {
		t.Fatalf("sku = %q, want LAP-001", got.Variant.SKU)
	}
}

func TestHandlerCreateInvalidBody(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/variants", `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateInvalidProductID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products/not-a-uuid/variants",
		`{"sku":"LAP-001","name":"16GB","price":15000000}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateSKUConflict(t *testing.T) {
	svc := &fakeService{createErr: ErrorVariantSKUAlreadyExists}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/variants",
		`{"sku":"LAP-001","name":"16GB","price":15000000}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandlerCreateProductNotFound(t *testing.T) {
	svc := &fakeService{createErr: product.ErrorProductNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/variants",
		`{"sku":"LAP-001","name":"16GB","price":15000000}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreateError(t *testing.T) {
	svc := &fakeService{createErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products/"+uuid.New().String()+"/variants",
		`{"sku":"LAP-001","name":"16GB","price":15000000}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerUpdate(t *testing.T) {
	productID := uuid.New()
	id := uuid.New()
	svc := &fakeService{updateRes: models.ProductVariant{ID: id, ProductID: productID, SKU: "LAP-002"}}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/products/"+productID.String()+"/variants/"+id.String(),
		`{"sku":"LAP-002","name":"32GB","price":20000000}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got variantEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Variant.ID != id {
		t.Fatalf("id = %v, want %v", got.Variant.ID, id)
	}
}

func TestHandlerUpdateInvalidVariantID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPut, "/api/v1/products/"+uuid.New().String()+"/variants/not-a-uuid",
		`{"sku":"LAP-002","name":"32GB","price":20000000}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUpdateInvalidBody(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPut, "/api/v1/products/"+uuid.New().String()+"/variants/"+uuid.New().String(), `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUpdateNotFound(t *testing.T) {
	svc := &fakeService{updateErr: ErrorProductVariantNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/products/"+uuid.New().String()+"/variants/"+uuid.New().String(),
		`{"sku":"LAP-002","name":"32GB","price":20000000}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerUpdateSKUConflict(t *testing.T) {
	svc := &fakeService{updateErr: ErrorVariantSKUAlreadyExists}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/products/"+uuid.New().String()+"/variants/"+uuid.New().String(),
		`{"sku":"LAP-002","name":"32GB","price":20000000}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandlerDelete(t *testing.T) {
	productID := uuid.New()

	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodDelete, "/api/v1/products/"+productID.String()+"/variants/"+uuid.New().String(), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandlerDeleteNotFound(t *testing.T) {
	svc := &fakeService{deleteErr: ErrorProductVariantNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodDelete, "/api/v1/products/"+uuid.New().String()+"/variants/"+uuid.New().String(), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
