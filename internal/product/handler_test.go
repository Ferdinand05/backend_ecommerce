package product

import (
	"context"
	"encoding/json"
	"errors"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeService struct {
	findAllRes  []models.Product
	findAllErr  error
	findByIDRes models.Product
	findByIDErr error

	findBySlugRes models.Product
	findBySlugErr error

	createRes models.Product
	createErr error

	updateRes models.Product
	updateErr error

	deleteErr error
}

func (f *fakeService) FindAll(ctx context.Context) ([]models.Product, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeService) FindByID(ctx context.Context, ID uuid.UUID) (models.Product, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeService) FindBySlug(ctx context.Context, slug string) (models.Product, error) {
	return f.findBySlugRes, f.findBySlugErr
}

func (f *fakeService) Create(ctx context.Context, product CreateProductRequest) (models.Product, error) {
	return f.createRes, f.createErr
}

func (f *fakeService) Update(ctx context.Context, ID uuid.UUID, product UpdateProductRequest) (models.Product, error) {
	return f.updateRes, f.updateErr
}

func (f *fakeService) Delete(ctx context.Context, ID uuid.UUID) error {
	return f.deleteErr
}

type productEnvelope struct {
	Product ProductDetailResponse `json:"product"`
}

type productsEnvelope struct {
	Products []ProductResponse `json:"products"`
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
	desc := "portable computer"
	svc := &fakeService{findAllRes: []models.Product{
		{ID: uuid.New(), Name: "laptop", Slug: "laptop", Description: &desc, IsActive: true,
			Category: models.Category{Name: "electronics", Slug: "electronics"}},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got productsEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Products) != 1 {
		t.Fatalf("products len = %d, want 1", len(got.Products))
	}

	if got.Products[0].Slug != "laptop" {
		t.Fatalf("slug = %q, want laptop", got.Products[0].Slug)
	}

	if got.Products[0].Category == nil || got.Products[0].Category.Slug != "electronics" {
		t.Fatalf("category = %v, want electronics", got.Products[0].Category)
	}
}

func TestHandlerFindAllError(t *testing.T) {
	svc := &fakeService{findAllErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products", "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerFindByID(t *testing.T) {
	id := uuid.New()
	svc := &fakeService{findByIDRes: models.Product{
		ID:       id,
		Name:     "laptop",
		Slug:     "laptop",
		Category: models.Category{Name: "electronics", Slug: "electronics"},
		Variants: []models.ProductVariant{
			{ID: uuid.New(), SKU: "LAP-001", Name: "16GB", Price: decimal.NewFromInt(15000000)},
		},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+id.String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got productEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Product.ID != id {
		t.Fatalf("id = %v, want %v", got.Product.ID, id)
	}

	if got.Product.Category == nil || got.Product.Category.Slug != "electronics" {
		t.Fatalf("category = %v, want electronics", got.Product.Category)
	}

	if len(got.Product.Variants) != 1 || got.Product.Variants[0].SKU != "LAP-001" {
		t.Fatalf("variants = %v, want 1 variant with SKU LAP-001", got.Product.Variants)
	}
}

func TestHandlerFindByIDInvalidUUID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, "/api/v1/products/not-a-uuid", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerFindByIDNotFound(t *testing.T) {
	svc := &fakeService{findByIDErr: ErrorProductNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+uuid.New().String(), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerFindBySlug(t *testing.T) {
	svc := &fakeService{findBySlugRes: models.Product{
		Name:     "laptop",
		Slug:     "laptop",
		Category: models.Category{Name: "electronics", Slug: "electronics"},
		Variants: []models.ProductVariant{
			{ID: uuid.New(), SKU: "LAP-001", Name: "16GB", Price: decimal.NewFromInt(15000000)},
		},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/slug/laptop", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got productEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Product.Slug != "laptop" {
		t.Fatalf("slug = %q, want laptop", got.Product.Slug)
	}

	if got.Product.Category == nil || got.Product.Category.Slug != "electronics" {
		t.Fatalf("category = %v, want electronics", got.Product.Category)
	}

	if len(got.Product.Variants) != 1 || got.Product.Variants[0].SKU != "LAP-001" {
		t.Fatalf("variants = %v, want 1 variant with SKU LAP-001", got.Product.Variants)
	}
}

func TestHandlerFindBySlugNotFound(t *testing.T) {
	svc := &fakeService{findBySlugErr: ErrorProductNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/slug/missing", "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreate(t *testing.T) {
	categoryID := uuid.New()
	svc := &fakeService{createRes: models.Product{ID: uuid.New(), Name: "Laptop", Slug: "laptop"}}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products",
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got productEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Product.Slug != "laptop" {
		t.Fatalf("slug = %q, want laptop", got.Product.Slug)
	}
}

func TestHandlerCreateInvalidBody(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/products", `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateSlugConflict(t *testing.T) {
	categoryID := uuid.New()
	svc := &fakeService{createErr: ErrorProductSlugAlreadyExists}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products",
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandlerCreateCategoryNotFound(t *testing.T) {
	categoryID := uuid.New()
	svc := &fakeService{createErr: category.ErrorCategoryNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products",
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreateError(t *testing.T) {
	categoryID := uuid.New()
	svc := &fakeService{createErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/products",
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerUpdate(t *testing.T) {
	id := uuid.New()
	categoryID := uuid.New()
	svc := &fakeService{updateRes: models.Product{ID: id, Name: "Laptop", Slug: "laptop"}}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/products/"+id.String(),
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got productEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Product.ID != id {
		t.Fatalf("id = %v, want %v", got.Product.ID, id)
	}
}

func TestHandlerUpdateInvalidUUID(t *testing.T) {
	categoryID := uuid.New()

	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPut, "/api/v1/products/not-a-uuid",
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUpdateInvalidBody(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPut, "/api/v1/products/"+uuid.New().String(), `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUpdateNotFound(t *testing.T) {
	categoryID := uuid.New()
	svc := &fakeService{updateErr: ErrorProductNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/products/"+uuid.New().String(),
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerUpdateSlugConflict(t *testing.T) {
	categoryID := uuid.New()
	svc := &fakeService{updateErr: ErrorProductSlugAlreadyExists}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/products/"+uuid.New().String(),
		`{"category_id":"`+categoryID.String()+`","name":"Laptop"}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandlerDelete(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodDelete, "/api/v1/products/"+uuid.New().String(), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandlerDeleteNotFound(t *testing.T) {
	svc := &fakeService{deleteErr: ErrorProductNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodDelete, "/api/v1/products/"+uuid.New().String(), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
