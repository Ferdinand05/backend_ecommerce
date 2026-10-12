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
}

func (f *fakeService) SearchAreas(ctx context.Context, query string) ([]Area, error) {
	// validasi query ada di service; fake harus perilakunya sama.
	if strings.TrimSpace(query) == "" {
		return nil, ErrInvalidAreaQuery
	}

	return f.areas, f.err
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
