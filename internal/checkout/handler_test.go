package checkout

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeHandlerService struct {
	preview PreviewResponse
	err     error
	calls   int
}

func (f *fakeHandlerService) Preview(ctx context.Context, userID uuid.UUID) (PreviewResponse, error) {
	f.calls++
	return f.preview, f.err
}

func newCheckoutRouter(svc Service, withAuth bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	handler := NewHandler(svc)

	if withAuth {
		// Simulasi middleware auth yang sudah set userID (string, seperti claims.Subject).
		r.Use(func(c *gin.Context) {
			c.Set("userID", "99999999-9999-9999-9999-999999999999")
			c.Next()
		})
	}

	r.POST("/api/v1/checkout/preview", handler.Preview)
	return r
}

func TestHandlerPreviewSuccess(t *testing.T) {
	svc := &fakeHandlerService{preview: PreviewResponse{
		Items: []PreviewItemResponse{{
			ProductVariantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			ProductName:      "Kaos Polos",
			Quantity:         2,
			UnitPrice:        decimal.NewFromInt(95000),
			Subtotal:         decimal.NewFromInt(190000),
			AvailableStock:   5,
		}},
		Subtotal: decimal.NewFromInt(190000),
	}}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", nil)
	newCheckoutRouter(svc, true).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got PreviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Items) != 1 || !got.Subtotal.Equal(decimal.NewFromInt(190000)) {
		t.Fatalf("response = %+v, want 1 item subtotal 190000", got)
	}
}

func TestHandlerPreviewWithoutAuth(t *testing.T) {
	svc := &fakeHandlerService{}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", nil)
	newCheckoutRouter(svc, false).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	if svc.calls != 0 {
		t.Fatal("service tidak boleh dipanggil tanpa auth")
	}
}

func TestHandlerPreviewErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "cart kosong", err: ErrCartEmpty, wantStatus: http.StatusBadRequest},
		{name: "varian tidak tersedia", err: ErrVariantNotAvailable, wantStatus: http.StatusUnprocessableEntity},
		{name: "stok kurang", err: ErrInsufficientStock, wantStatus: http.StatusConflict},
		{name: "error tak dikenal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeHandlerService{err: tt.err}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", strings.NewReader(""))
			newCheckoutRouter(svc, true).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
