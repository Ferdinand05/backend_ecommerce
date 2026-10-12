package biteship

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ferdinand/ecommerce/internal/shipping"
)

func TestProviderSearchAreasMapsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"areas": [
				{"id": "IDJA0100", "name": "Jakarta Pusat", "postal_code": 10110},
				{"id": "", "name": "tanpa id", "postal_code": 12345},
				{"id": "IDX", "name": "  ", "postal_code": 12345}
			]
		}`))
	}))
	defer server.Close()

	provider := NewProvider(NewClient(server.URL, "key", 0))

	areas, err := provider.SearchAreas(context.Background(), "jak")
	if err != nil {
		t.Fatalf("SearchAreas() error = %v", err)
	}

	// hasil tanpa id/name kosong dibuang.
	if len(areas) != 1 {
		t.Fatalf("areas = %v, want only valid area", areas)
	}

	if areas[0].ID != "IDJA0100" || areas[0].Name != "Jakarta Pusat" || areas[0].PostalCode != 10110 {
		t.Fatalf("area = %+v, want mapped area", areas[0])
	}
}

func TestProviderSearchAreasWrapsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewProvider(NewClient(server.URL, "key", 0))

	_, err := provider.SearchAreas(context.Background(), "jak")
	if !errors.Is(err, shipping.ErrAreaProvider) {
		t.Fatalf("error = %v, want wrapped ErrAreaProvider", err)
	}
}
