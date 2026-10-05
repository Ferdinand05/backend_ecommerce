package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	productvariant "ferdinand/ecommerce/internal/product_variant"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeService struct {
	getInventoryRes InventoryResponse
	getInventoryErr error

	listRes []StockMovementResponse
	listErr error

	getMovementRes StockMovementResponse
	getMovementErr error

	createRes StockMovementResponse
	createErr error
}

func (f *fakeService) GetInventory(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) (InventoryResponse, error) {
	return f.getInventoryRes, f.getInventoryErr
}

func (f *fakeService) ListMovements(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) ([]StockMovementResponse, error) {
	return f.listRes, f.listErr
}

func (f *fakeService) GetMovement(ctx context.Context, productID uuid.UUID, variantID uuid.UUID, movementID uuid.UUID) (StockMovementResponse, error) {
	return f.getMovementRes, f.getMovementErr
}

func (f *fakeService) CreateMovement(ctx context.Context, productID uuid.UUID, variantID uuid.UUID, req CreateStockMovementRequest) (StockMovementResponse, error) {
	return f.createRes, f.createErr
}

type inventoryEnvelope struct {
	Inventory InventoryResponse `json:"inventory"`
}

type movementsEnvelope struct {
	Movements []StockMovementResponse `json:"movements"`
}

type movementEnvelope struct {
	Movement StockMovementResponse `json:"movement"`
}

func newTestRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(svc))
	return r
}

func inventoryPath(productID, variantID uuid.UUID, suffix string) string {
	return "/api/v1/products/" + productID.String() + "/variants/" + variantID.String() + "/inventory" + suffix
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestHandlerGetInventory(t *testing.T) {
	variantID := uuid.New()
	svc := &fakeService{getInventoryRes: InventoryResponse{ID: uuid.New(), ProductVariantID: variantID, Quantity: 7}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), variantID, ""), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got inventoryEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Inventory.Quantity != 7 {
		t.Fatalf("quantity = %d, want 7", got.Inventory.Quantity)
	}
}

func TestHandlerGetInventoryInvalidUUID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, "/api/v1/products/not-a-uuid/variants/"+uuid.New().String()+"/inventory", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerGetInventoryNotFound(t *testing.T) {
	svc := &fakeService{getInventoryErr: ErrorInventoryNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), ""), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerGetInventoryVariantNotFound(t *testing.T) {
	svc := &fakeService{getInventoryErr: productvariant.ErrorProductVariantNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), ""), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerGetInventoryError(t *testing.T) {
	svc := &fakeService{getInventoryErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), ""), "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerListMovements(t *testing.T) {
	svc := &fakeService{listRes: []StockMovementResponse{
		{ID: uuid.New(), Type: "RESTOCK", Quantity: 5},
		{ID: uuid.New(), Type: "SALE", Quantity: -2},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), "/movements"), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got movementsEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Movements) != 2 {
		t.Fatalf("movements len = %d, want 2", len(got.Movements))
	}
}

func TestHandlerGetMovement(t *testing.T) {
	movementID := uuid.New()
	svc := &fakeService{getMovementRes: StockMovementResponse{ID: movementID, Type: "RESTOCK", Quantity: 5}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), "/movements/"+movementID.String()), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got movementEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Movement.ID != movementID {
		t.Fatalf("movement id = %v, want %v", got.Movement.ID, movementID)
	}
}

func TestHandlerGetMovementInvalidUUID(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), "/movements/not-a-uuid"), "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerGetMovementNotFound(t *testing.T) {
	svc := &fakeService{getMovementErr: ErrorStockMovementNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, inventoryPath(uuid.New(), uuid.New(), "/movements/"+uuid.New().String()), "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreateMovement(t *testing.T) {
	svc := &fakeService{createRes: StockMovementResponse{ID: uuid.New(), Type: "RESTOCK", Quantity: 5}}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"),
		`{"type":"RESTOCK","quantity":5}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got movementEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Movement.Type != "RESTOCK" {
		t.Fatalf("type = %q, want RESTOCK", got.Movement.Type)
	}
}

func TestHandlerCreateMovementInvalidBody(t *testing.T) {
	w := doRequest(t, newTestRouter(&fakeService{}), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"), `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateMovementInvalidType(t *testing.T) {
	svc := &fakeService{createErr: ErrorInvalidMovementType}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"),
		`{"type":"UNKNOWN","quantity":1}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateMovementInvalidQuantity(t *testing.T) {
	svc := &fakeService{createErr: ErrorInvalidMovementQuantity}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"),
		`{"type":"RESTOCK","quantity":1}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateMovementInsufficientStock(t *testing.T) {
	svc := &fakeService{createErr: ErrorInsufficientStock}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"),
		`{"type":"SALE","quantity":99}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandlerCreateMovementVariantNotFound(t *testing.T) {
	svc := &fakeService{createErr: productvariant.ErrorProductVariantNotFound}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"),
		`{"type":"RESTOCK","quantity":1}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCreateMovementError(t *testing.T) {
	svc := &fakeService{createErr: errors.New("boom")}

	w := doRequest(t, newTestRouter(svc), http.MethodPost, inventoryPath(uuid.New(), uuid.New(), "/movements"),
		`{"type":"RESTOCK","quantity":1}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
