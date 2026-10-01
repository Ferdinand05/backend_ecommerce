package category

import (
	"context"
	"encoding/json"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeService struct {
	findAllRes  []models.Category
	findAllErr  error
	findByIDRes models.Category
	findByIDErr error

	findBySlugRes models.Category
	findBySlugErr error

	createRes models.Category
	createErr error

	updateRes models.Category
	updateErr error

	deleteErr error
}

func (f *fakeService) FindAll(ctx context.Context) ([]models.Category, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeService) FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeService) FindBySlug(ctx context.Context, slug string) (models.Category, error) {
	return f.findBySlugRes, f.findBySlugErr
}

func (f *fakeService) Create(ctx context.Context, category CreateCategoryRequest) (models.Category, error) {
	return f.createRes, f.createErr
}

func (f *fakeService) Update(ctx context.Context, ID uuid.UUID, category UpdateCategoryRequest) (models.Category, error) {
	return f.updateRes, f.updateErr
}

func (f *fakeService) Delete(ctx context.Context, ID uuid.UUID) error {
	return f.deleteErr
}

type categoryEnvelope struct {
	Category CategoryResponse `json:"category"`
}

type categoriesEnvelope struct {
	Categories []CategoryResponse `json:"categories"`
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
	desc := "gadgets"
	svc := &fakeService{findAllRes: []models.Category{
		{ID: uuid.New(), Name: "electronics", Slug: "electronics", Description: &desc},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/categories", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got categoriesEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Categories) != 1 {
		t.Fatalf("categories len = %d, want 1", len(got.Categories))
	}

	if got.Categories[0].Slug != "electronics" {
		t.Fatalf("slug = %q, want electronics", got.Categories[0].Slug)
	}
}

func TestHandlerFindAllError(t *testing.T) {
	svc := &fakeService{findAllErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/categories", "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerFindByID(t *testing.T) {
	id := uuid.New()
	svc := &fakeService{findByIDRes: models.Category{ID: id, Name: "electronics", Slug: "electronics"}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/categories/"+id.String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got categoryEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Category.ID != id {
		t.Fatalf("id = %v, want %v", got.Category.ID, id)
	}
}

func TestHandlerFindByIDInvalidUUID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, "/api/v1/categories/not-a-uuid", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerFindByIDNotFound(t *testing.T) {
	svc := &fakeService{findByIDErr: ErrorCategoryNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/categories/"+uuid.New().String(), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerFindBySlug(t *testing.T) {
	svc := &fakeService{findBySlugRes: models.Category{Name: "electronics", Slug: "electronics"}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/categories/slug/electronics", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got categoryEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Category.Slug != "electronics" {
		t.Fatalf("slug = %q, want electronics", got.Category.Slug)
	}
}

func TestHandlerFindBySlugNotFound(t *testing.T) {
	svc := &fakeService{findBySlugErr: ErrorCategoryNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/categories/slug/missing", "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreate(t *testing.T) {
	svc := &fakeService{createRes: models.Category{ID: uuid.New(), Name: "Electronics", Slug: "electronics"}}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/categories", `{"name":"Electronics"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got categoryEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Category.Slug != "electronics" {
		t.Fatalf("slug = %q, want electronics", got.Category.Slug)
	}
}

func TestHandlerCreateInvalidBody(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, "/api/v1/categories", `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateError(t *testing.T) {
	svc := &fakeService{createErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/categories", `{"name":"Electronics"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerUpdate(t *testing.T) {
	id := uuid.New()
	svc := &fakeService{updateRes: models.Category{ID: id, Name: "Electronics", Slug: "electronics"}}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/categories/"+id.String(), `{"name":"Electronics"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got categoryEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Category.ID != id {
		t.Fatalf("id = %v, want %v", got.Category.ID, id)
	}
}

func TestHandlerUpdateNotFound(t *testing.T) {
	svc := &fakeService{updateErr: ErrorCategoryNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/categories/"+uuid.New().String(), `{"name":"Electronics"}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerDelete(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodDelete, "/api/v1/categories/"+uuid.New().String(), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandlerDeleteNotFound(t *testing.T) {
	svc := &fakeService{deleteErr: ErrorCategoryNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodDelete, "/api/v1/categories/"+uuid.New().String(), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
