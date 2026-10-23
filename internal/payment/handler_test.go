package payment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeHandlerService struct {
	cb    InvoiceCallback
	res   error
	calls int

	createOrderID uuid.UUID
	createUserID  uuid.UUID
	createResult  CreatePaymentResult
	createErr     error
	createCalls   int

	listStatus *PaymentStatus
	listResult PaymentListResponse
	listErr    error
	listCalls  int
}

func (f *fakeHandlerService) CreatePayment(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (CreatePaymentResult, error) {
	f.createCalls++
	f.createUserID = userID
	f.createOrderID = orderID
	return f.createResult, f.createErr
}

func (f *fakeHandlerService) HandleInvoiceCallback(ctx context.Context, cb InvoiceCallback) error {
	f.calls++
	f.cb = cb
	return f.res
}

func (f *fakeHandlerService) ListPayments(ctx context.Context, status *PaymentStatus, page, pageSize int) (PaymentListResponse, error) {
	f.listCalls++
	f.listStatus = status
	if f.listErr != nil {
		return PaymentListResponse{}, f.listErr
	}
	return f.listResult, nil
}

func newWebhookRouter(svc Service, token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), r.Group("/api/v1/me"), r.Group("/api/v1/admin"), NewHandler(svc, token))
	return r
}

func newAuthedRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Pengganti AuthMiddleware: pasang userID langsung.
	r.Use(func(c *gin.Context) {
		c.Set("userID", uuid.NewString())
		c.Next()
	})

	RegisterRoutes(r.Group("/api/v1"), r.Group("/api/v1/me"), r.Group("/api/v1/admin"), NewHandler(svc, "secret-token"))
	return r
}

func postWebhook(r *gin.Engine, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/xendit", strings.NewReader(body))
	if token != "" {
		req.Header.Set("x-callback-token", token)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func postCreatePayment(r *gin.Engine, orderID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/orders/"+orderID+"/payment", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const webhookBody = `{
	"id": "inv-123",
	"external_id": "PAY-1",
	"status": "PAID",
	"amount": 120000,
	"currency": "IDR"
}`

// --- webhook handler ---

func TestWebhookHandlerRejectsInvalidToken(t *testing.T) {
	svc := &fakeHandlerService{}

	w := postWebhook(newWebhookRouter(svc, "secret-token"), "token-salah", webhookBody)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if svc.calls != 0 {
		t.Fatalf("service dipanggil %d kali, want 0", svc.calls)
	}
}

func TestWebhookHandlerRejectsMissingToken(t *testing.T) {
	svc := &fakeHandlerService{}

	w := postWebhook(newWebhookRouter(svc, "secret-token"), "", webhookBody)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestWebhookHandlerSuccess(t *testing.T) {
	svc := &fakeHandlerService{}

	w := postWebhook(newWebhookRouter(svc, "secret-token"), "secret-token", webhookBody)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}
	if svc.calls != 1 {
		t.Fatalf("calls = %d, want 1", svc.calls)
	}
	if svc.cb.ID != "inv-123" || svc.cb.Amount != 120000 {
		t.Fatalf("callback = %+v, tidak sesuai payload", svc.cb)
	}
}

func TestWebhookHandlerMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "nominal tidak cocok", err: ErrAmountMismatch, wantStatus: http.StatusBadRequest},
		{name: "payment tidak valid", err: ErrInvalidPayment, wantStatus: http.StatusBadRequest},
		{name: "error lain", err: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeHandlerService{res: tt.err}

			w := postWebhook(newWebhookRouter(svc, "secret-token"), "secret-token", webhookBody)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestWebhookHandlerInvalidBody(t *testing.T) {
	svc := &fakeHandlerService{}

	w := postWebhook(newWebhookRouter(svc, "secret-token"), "secret-token", `{`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if svc.calls != 0 {
		t.Fatalf("service dipanggil %d kali, want 0", svc.calls)
	}
}

// --- create payment handler ---

func TestCreatePaymentHandlerSuccess(t *testing.T) {
	paymentID := uuid.New()
	orderID := uuid.New()

	svc := &fakeHandlerService{}
	svc.createResult = CreatePaymentResult{
		PaymentID:   paymentID,
		OrderID:     orderID,
		Status:      "PENDING",
		Amount:      "120000",
		Currency:    "IDR",
		CheckoutURL: "https://checkout.xendit.co/web/inv-1",
	}

	w := postCreatePayment(newAuthedRouter(svc), orderID.String())

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}
	if svc.createCalls != 1 {
		t.Fatalf("calls = %d, want 1", svc.createCalls)
	}
	if svc.createOrderID != orderID {
		t.Fatalf("order_id = %s, want %s", svc.createOrderID, orderID)
	}
	if svc.createUserID == uuid.Nil {
		t.Fatal("userID kosong, harus dari context")
	}

	body := w.Body.String()
	for _, want := range []string{
		`"payment_id":"` + paymentID.String() + `"`,
		`"checkout_url":"https://checkout.xendit.co/web/inv-1"`,
		`"amount":"120000"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body tidak memuat %s:\n%s", want, body)
		}
	}
}

func TestCreatePaymentHandlerInvalidOrderID(t *testing.T) {
	svc := &fakeHandlerService{}

	w := postCreatePayment(newAuthedRouter(svc), "bukan-uuid")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if svc.createCalls != 0 {
		t.Fatalf("service dipanggil %d kali, want 0", svc.createCalls)
	}
}

func TestCreatePaymentHandlerMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "order tidak ditemukan", err: ErrOrderNotFound, wantStatus: http.StatusNotFound},
		{name: "order tidak bisa dibayar", err: ErrOrderNotPayable, wantStatus: http.StatusConflict},
		{name: "periode bayar habis", err: ErrOrderPaymentExpired, wantStatus: http.StatusGone},
		{name: "state ambigu", err: ErrPaymentStateAmbiguous, wantStatus: http.StatusBadGateway},
		{name: "error lain", err: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeHandlerService{createErr: tt.err}

			w := postCreatePayment(newAuthedRouter(svc), uuid.NewString())

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- list payments (admin) ---

func listPayments(r *gin.Engine, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListPaymentsHandlerRejectsInvalidStatus(t *testing.T) {
	svc := &fakeHandlerService{}

	w := listPayments(newAuthedRouter(svc), "?status=HACKED")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if svc.listCalls != 0 {
		t.Fatalf("service dipanggil %d kali, want 0", svc.listCalls)
	}
}

func TestListPaymentsHandlerSuccess(t *testing.T) {
	svc := &fakeHandlerService{listResult: PaymentListResponse{
		Payments: []PaymentResponse{{ID: uuid.New(), Status: "PAID"}},
		Meta:     PaymentListMeta{Total: 1, Page: 1, PageSize: 10},
	}}

	w := listPayments(newAuthedRouter(svc), "?status=PAID&page=1&page_size=10")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if svc.listCalls != 1 || svc.listStatus == nil || *svc.listStatus != PaymentStatusPaid {
		t.Fatalf("service status filter = %v, calls = %d", svc.listStatus, svc.listCalls)
	}
}

func TestListPaymentsHandlerDefaultPagination(t *testing.T) {
	svc := &fakeHandlerService{}

	w := listPayments(newAuthedRouter(svc), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	_ = svc // default filter: service menerima status nil (lihat unit service test)
}
