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
}

func (f *fakeHandlerService) CreatePayment(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (CreatePaymentResult, error) {
	return CreatePaymentResult{}, errors.New("not implemented")
}

func (f *fakeHandlerService) HandleInvoiceCallback(ctx context.Context, cb InvoiceCallback) error {
	f.calls++
	f.cb = cb
	return f.res
}

func newWebhookRouter(svc Service, token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(svc, token))
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

const webhookBody = `{
	"id": "inv-123",
	"external_id": "PAY-1",
	"status": "PAID",
	"amount": 120000,
	"currency": "IDR"
}`

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
