package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeService struct {
	areas []Area
	err   error

	rates     []ShippingRate
	ratesErr  error
	ratesCall int
	lastReq   CalculateRatesRequest
}

func (f *fakeService) SearchAreas(ctx context.Context, query string) ([]Area, error) {
	// validasi query ada di service; fake harus perilakunya sama.
	if strings.TrimSpace(query) == "" {
		return nil, ErrInvalidAreaQuery
	}

	return f.areas, f.err
}

func (f *fakeService) CalculateRates(ctx context.Context, req CalculateRatesRequest) ([]ShippingRate, error) {
	f.ratesCall++
	f.lastReq = req
	return f.rates, f.ratesErr
}

func newShippingRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	RegisterRoutes(r.Group("/api/v1/shipping"), NewHandler(svc))
	return r
}

func TestHandlerSearchAreas(t *testing.T) {
	svc := &fakeService{areas: []Area{{ID: "IDJA", Name: "Jakarta", PostalCode: 10110}}}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/areas?q=jakarta", nil)
	newShippingRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got SearchAreasResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Areas) != 1 || got.Areas[0].Name != "Jakarta" {
		t.Fatalf("areas = %v, want Jakarta", got.Areas)
	}
}

func TestHandlerSearchAreasErrors(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		err        error
		wantStatus int
	}{
		{name: "missing query", query: "", wantStatus: http.StatusBadRequest},
		{name: "invalid query", query: "q=", wantStatus: http.StatusBadRequest},
		{name: "provider error", query: "q=jakarta", err: ErrAreaProvider, wantStatus: http.StatusBadGateway},
		{name: "unknown error", query: "q=jakarta", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{err: tt.err}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/areas?"+tt.query, nil)
			newShippingRouter(svc).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandlerCalculateRates(t *testing.T) {
	svc := &fakeService{
		rates: []ShippingRate{{CourierCode: "jne", CourierName: "JNE", Price: 18000}},
	}

	body := `{
		"destination_area_id": "DEST-AREA",
		"items": [
			{"product_variant_id": "11111111-1111-1111-1111-111111111111", "quantity": 2}
		]
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/rates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newShippingRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got CalculateRatesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Rates) != 1 || got.Rates[0].CourierCode != "jne" {
		t.Fatalf("rates = %v, want jne", got.Rates)
	}
}

func TestHandlerCalculateRatesErrors(t *testing.T) {
	validBody := `{
		"destination_area_id": "DEST-AREA",
		"items": [{"product_variant_id": "11111111-1111-1111-1111-111111111111", "quantity": 1}]
	}`

	tests := []struct {
		name       string
		body       string
		err        error
		wantStatus int
	}{
		{
			name:       "invalid json body",
			body:       `{broken`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid rates input",
			body:       validBody,
			err:        ErrInvalidRatesInput,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "variant not available",
			body:       validBody,
			err:        ErrVariantNotAvailable,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "provider error",
			body:       validBody,
			err:        ErrAreaProvider,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "unknown error",
			body:       validBody,
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{ratesErr: tt.err}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/rates", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			newShippingRouter(svc).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
