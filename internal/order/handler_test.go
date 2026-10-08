package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ferdinand/ecommerce/internal/middleware"
	"ferdinand/ecommerce/internal/models"

	userjwt "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeService struct {
	findMyOrdersRes  OrderListResult
	findMyOrdersErr  error
	findMyOrdersPage int
	findMyOrdersSize int

	findMyOrderRes models.Order
	findMyOrderErr error

	findByIDRes models.Order
	findByIDErr error

	findAllRes    OrderListResult
	findAllErr    error
	findAllFilter OrderFilter

	updateStatusRes models.Order
	updateStatusErr error
}

func (f *fakeService) FindMyOrders(ctx context.Context, userID uuid.UUID, page int, pageSize int) (OrderListResult, error) {
	f.findMyOrdersPage = page
	f.findMyOrdersSize = pageSize
	return f.findMyOrdersRes, f.findMyOrdersErr
}

func (f *fakeService) FindMyOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (models.Order, error) {
	return f.findMyOrderRes, f.findMyOrderErr
}

func (f *fakeService) FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeService) FindAll(ctx context.Context, filter OrderFilter) (OrderListResult, error) {
	f.findAllFilter = filter
	return f.findAllRes, f.findAllErr
}

func (f *fakeService) UpdateStatus(ctx context.Context, orderID uuid.UUID, status OrderStatus, note *string) (models.Order, error) {
	return f.updateStatusRes, f.updateStatusErr
}

type ordersEnvelope struct {
	Orders []OrderResponse `json:"orders"`
	Meta   OrderListMeta   `json:"meta"`
}

type orderEnvelope struct {
	Order OrderDetailResponse `json:"order"`
}

const testSecret = "test-secret"

func newOrderRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	jwt := userjwt.NewJWTService(testSecret)

	me := r.Group("/api/v1/me", middleware.AuthMiddleware(jwt))
	admin := r.Group("/api/v1/admin", middleware.AuthMiddleware(jwt), middleware.RequireRole("admin"))

	RegisterRoutes(me, admin, NewHandler(svc))
	return r
}

func orderRequest(t *testing.T, r *gin.Engine, method, path, body string, userID uuid.UUID, role string) *httptest.ResponseRecorder {
	t.Helper()

	token, err := userjwt.NewJWTService(testSecret).GenerateToken(userID, "user@example.com", role)
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

func sampleOrder() models.Order {
	return models.Order{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		OrderNumber: "ORD-2026-0001",
		Status:      "PENDING",
		Subtotal:    decimal.NewFromInt(100000),
		TotalAmount: decimal.NewFromInt(100000),
	}
}

func TestListMyOrders(t *testing.T) {
	svc := &fakeService{findMyOrdersRes: OrderListResult{
		Orders:   []models.Order{sampleOrder()},
		Total:    1,
		Page:     2,
		PageSize: 20,
	}}

	w := orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/me/orders?page=2&page_size=20", "", uuid.New(), "customer")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got ordersEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Orders) != 1 || got.Orders[0].OrderNumber != "ORD-2026-0001" {
		t.Fatalf("orders = %v, want one ORD-2026-0001", got.Orders)
	}

	if got.Meta.Total != 1 || got.Meta.Page != 2 || got.Meta.PageSize != 20 {
		t.Fatalf("meta = %+v, want total 1 page 2 size 20", got.Meta)
	}

	if svc.findMyOrdersPage != 2 || svc.findMyOrdersSize != 20 {
		t.Fatalf("service args = (%d, %d), want (2, 20)", svc.findMyOrdersPage, svc.findMyOrdersSize)
	}
}

func TestListMyOrdersPaginationDefaultsAndCaps(t *testing.T) {
	svc := &fakeService{}

	orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/me/orders", "", uuid.New(), "customer")
	if svc.findMyOrdersPage != 1 || svc.findMyOrdersSize != 10 {
		t.Fatalf("defaults = (%d, %d), want (1, 10)", svc.findMyOrdersPage, svc.findMyOrdersSize)
	}

	orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/me/orders?page_size=500", "", uuid.New(), "customer")
	if svc.findMyOrdersSize != 100 {
		t.Fatalf("capped size = %d, want 100", svc.findMyOrdersSize)
	}
}

func TestListMyOrdersInvalidPagination(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "bad page", path: "/api/v1/me/orders?page=abc"},
		{name: "zero page", path: "/api/v1/me/orders?page=0"},
		{name: "bad page_size", path: "/api/v1/me/orders?page_size=x"},
		{name: "zero page_size", path: "/api/v1/me/orders?page_size=0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := orderRequest(t, newOrderRouter(&fakeService{}), http.MethodGet, tt.path, "", uuid.New(), "customer")

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestListMyOrdersRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newOrderRouter(&fakeService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetMyOrder(t *testing.T) {
	order := sampleOrder()
	svc := &fakeService{findMyOrderRes: order}

	w := orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/me/orders/"+order.ID.String(), "", uuid.New(), "customer")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got orderEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Order.ID != order.ID || got.Order.OrderNumber != order.OrderNumber {
		t.Fatalf("order = %+v, want %v", got.Order, order.OrderNumber)
	}
}

func TestGetMyOrderInvalidID(t *testing.T) {
	w := orderRequest(t, newOrderRouter(&fakeService{}), http.MethodGet, "/api/v1/me/orders/not-a-uuid", "", uuid.New(), "customer")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetMyOrderNotFound(t *testing.T) {
	svc := &fakeService{findMyOrderErr: ErrorOrderNotFound}

	w := orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/me/orders/"+uuid.New().String(), "", uuid.New(), "customer")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListOrders(t *testing.T) {
	svc := &fakeService{findAllRes: OrderListResult{Orders: []models.Order{sampleOrder()}, Total: 1, Page: 1, PageSize: 10}}

	w := orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/admin/orders", "", uuid.New(), "admin")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got ordersEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Meta.Total != 1 {
		t.Fatalf("meta = %+v, want total 1", got.Meta)
	}
}

func TestListOrdersForbiddenForCustomer(t *testing.T) {
	w := orderRequest(t, newOrderRouter(&fakeService{}), http.MethodGet, "/api/v1/admin/orders", "", uuid.New(), "customer")

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestListOrdersInvalidFilters(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "bad status", path: "/api/v1/admin/orders?status=JUNK"},
		{name: "bad user_id", path: "/api/v1/admin/orders?user_id=abc"},
		{name: "bad start_date", path: "/api/v1/admin/orders?start_date=01-02-2026"},
		{name: "bad end_date", path: "/api/v1/admin/orders?end_date=yesterday"},
		{name: "start after end", path: "/api/v1/admin/orders?start_date=2026-05-01&end_date=2026-04-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := orderRequest(t, newOrderRouter(&fakeService{}), http.MethodGet, tt.path, "", uuid.New(), "admin")

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestGetOrder(t *testing.T) {
	order := sampleOrder()
	svc := &fakeService{findByIDRes: order}

	w := orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/admin/orders/"+order.ID.String(), "", uuid.New(), "admin")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetOrderNotFound(t *testing.T) {
	svc := &fakeService{findByIDErr: ErrorOrderNotFound}

	w := orderRequest(t, newOrderRouter(svc), http.MethodGet, "/api/v1/admin/orders/"+uuid.New().String(), "", uuid.New(), "admin")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateStatus(t *testing.T) {
	order := sampleOrder()
	order.Status = "PAID"
	svc := &fakeService{updateStatusRes: order}

	w := orderRequest(t, newOrderRouter(svc), http.MethodPatch,
		"/api/v1/admin/orders/"+order.ID.String()+"/status",
		`{"status":"PAID","note":"payment confirmed"}`, uuid.New(), "admin")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got orderEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got.Order.Status != "PAID" {
		t.Fatalf("status = %q, want PAID", got.Order.Status)
	}
}

func TestUpdateStatusErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		err        error
		wantStatus int
	}{
		{name: "invalid body", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "invalid status", body: `{"status":"JUNK"}`, err: ErrorInvalidOrderStatus, wantStatus: http.StatusBadRequest},
		{name: "invalid transition", body: `{"status":"PAID"}`, err: ErrorInvalidStatusTransition, wantStatus: http.StatusConflict},
		{name: "not found", body: `{"status":"PAID"}`, err: ErrorOrderNotFound, wantStatus: http.StatusNotFound},
		{name: "unknown", body: `{"status":"PAID"}`, err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{updateStatusErr: tt.err}

			w := orderRequest(t, newOrderRouter(svc), http.MethodPatch,
				"/api/v1/admin/orders/"+uuid.New().String()+"/status", tt.body, uuid.New(), "admin")

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
