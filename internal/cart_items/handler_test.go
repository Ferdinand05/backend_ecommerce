package cartitems

import (
	"context"
	"encoding/json"
	"errors"
	"ferdinand/ecommerce/internal/middleware"
	"ferdinand/ecommerce/internal/product_variant"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userjwt "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeService struct {
	getCartRes CartResponse
	getCartErr error

	addItemRes CartItemResponse
	addItemErr error

	increaseRes CartItemResponse
	increaseErr error

	decreaseRes     CartItemResponse
	decreaseRemoved bool
	decreaseErr     error

	removeErr error
}

func (f *fakeService) GetCart(ctx context.Context, userID uuid.UUID) (CartResponse, error) {
	return f.getCartRes, f.getCartErr
}

func (f *fakeService) AddItem(ctx context.Context, userID uuid.UUID, req AddCartItemRequest) (CartItemResponse, error) {
	return f.addItemRes, f.addItemErr
}

func (f *fakeService) IncreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (CartItemResponse, error) {
	return f.increaseRes, f.increaseErr
}

func (f *fakeService) DecreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (CartItemResponse, bool, error) {
	return f.decreaseRes, f.decreaseRemoved, f.decreaseErr
}

func (f *fakeService) RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {
	return f.removeErr
}

type itemEnvelope struct {
	Item CartItemResponse `json:"item"`
}

type cartEnvelope struct {
	Cart CartResponse `json:"cart"`
}

const testSecret = "test-secret"

func newCartRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	jwt := userjwt.NewJWTService(testSecret)
	cart := r.Group("/api/v1/cart", middleware.AuthMiddleware(jwt))
	RegisterRoutes(cart, NewHandler(svc))
	return r
}

func authRequest(t *testing.T, r *gin.Engine, method, path, body string, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	token, err := userjwt.NewJWTService(testSecret).GenerateToken(userID, "user@example.com", "customer")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestHandlerGetCart(t *testing.T) {
	svc := &fakeService{getCartRes: CartResponse{Items: []CartItemResponse{
		{ID: uuid.New(), SKU: "LAP-001", Name: "16GB", Quantity: 2},
	}}}

	w := authRequest(t, newCartRouter(svc), http.MethodGet, "/api/v1/cart", "", uuid.New())

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got cartEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Cart.Items) != 1 || got.Cart.Items[0].SKU != "LAP-001" {
		t.Fatalf("cart = %v, want one LAP-001 item", got.Cart)
	}
}

func TestHandlerGetCartMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newCartRouter(&fakeService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandlerGetCartInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newCartRouter(&fakeService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandlerGetCartError(t *testing.T) {
	svc := &fakeService{getCartErr: errors.New("boom")}

	w := authRequest(t, newCartRouter(svc), http.MethodGet, "/api/v1/cart", "", uuid.New())

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerAddItem(t *testing.T) {
	variantID := uuid.New()
	svc := &fakeService{addItemRes: CartItemResponse{ID: uuid.New(), ProductVariantID: variantID, Quantity: 1, SKU: "LAP-001"}}

	w := authRequest(t, newCartRouter(svc), http.MethodPost, "/api/v1/cart/items",
		`{"product_variant_id":"`+variantID.String()+`"}`, uuid.New())

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got itemEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Item.SKU != "LAP-001" || got.Item.Quantity != 1 {
		t.Fatalf("item = %v, want LAP-001 quantity 1", got.Item)
	}
}

func TestHandlerAddItemInvalidBody(t *testing.T) {
	w := authRequest(t, newCartRouter(&fakeService{}), http.MethodPost, "/api/v1/cart/items", `{}`, uuid.New())

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerAddItemErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "variant not found", err: productvariant.ErrorProductVariantNotFound, wantStatus: http.StatusNotFound},
		{name: "variant inactive", err: ErrorProductVariantInactive, wantStatus: http.StatusConflict},
		{name: "unknown", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{addItemErr: tt.err}

			w := authRequest(t, newCartRouter(svc), http.MethodPost, "/api/v1/cart/items",
				`{"product_variant_id":"`+uuid.New().String()+`"}`, uuid.New())

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandlerIncreaseItem(t *testing.T) {
	itemID := uuid.New()
	svc := &fakeService{increaseRes: CartItemResponse{ID: itemID, Quantity: 3, SKU: "LAP-001"}}

	w := authRequest(t, newCartRouter(svc), http.MethodPost, "/api/v1/cart/items/"+itemID.String()+"/increase", "", uuid.New())

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got itemEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Item.Quantity != 3 {
		t.Fatalf("quantity = %d, want 3", got.Item.Quantity)
	}
}

func TestHandlerIncreaseItemInvalidID(t *testing.T) {
	w := authRequest(t, newCartRouter(&fakeService{}), http.MethodPost, "/api/v1/cart/items/not-a-uuid/increase", "", uuid.New())

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerIncreaseItemNotFound(t *testing.T) {
	svc := &fakeService{increaseErr: ErrorCartItemNotFound}

	w := authRequest(t, newCartRouter(svc), http.MethodPost, "/api/v1/cart/items/"+uuid.New().String()+"/increase", "", uuid.New())

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerDecreaseItem(t *testing.T) {
	itemID := uuid.New()
	svc := &fakeService{decreaseRes: CartItemResponse{ID: itemID, Quantity: 1, SKU: "LAP-001"}}

	w := authRequest(t, newCartRouter(svc), http.MethodPost, "/api/v1/cart/items/"+itemID.String()+"/decrease", "", uuid.New())

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got itemEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Item.Quantity != 1 {
		t.Fatalf("quantity = %d, want 1", got.Item.Quantity)
	}
}

func TestHandlerDecreaseItemRemoved(t *testing.T) {
	svc := &fakeService{decreaseRemoved: true}

	w := authRequest(t, newCartRouter(svc), http.MethodPost, "/api/v1/cart/items/"+uuid.New().String()+"/decrease", "", uuid.New())

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	if w.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", w.Body.String())
	}
}

func TestHandlerRemoveItem(t *testing.T) {
	w := authRequest(t, newCartRouter(&fakeService{}), http.MethodDelete, "/api/v1/cart/items/"+uuid.New().String(), "", uuid.New())

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandlerRemoveItemNotFound(t *testing.T) {
	svc := &fakeService{removeErr: ErrorCartItemNotFound}

	w := authRequest(t, newCartRouter(svc), http.MethodDelete, "/api/v1/cart/items/"+uuid.New().String(), "", uuid.New())

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
