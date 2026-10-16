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

func TestProviderCalculateRatesMapsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"pricing": [
				{
					"courier_code": "jne",
					"courier_name": "JNE",
					"courier_service_code": "reg",
					"courier_service_name": "Reguler",
					"description": "Layanan reguler",
					"price": 18000,
					"shipment_duration_range": "2 - 3",
					"shipment_duration_unit": "days"
				},
				{
					"courier_code": "sicepat",
					"courier_name": "SiCepat",
					"courier_service_code": "best",
					"courier_service_name": "Besok Sampai",
					"description": "Overnight",
					"price": 25000,
					"shipment_duration_range": "1",
					"shipment_duration_unit": "days"
				}
			]
		}`))
	}))
	defer server.Close()

	provider := NewProvider(NewClient(server.URL, "key", 0))

	rates, err := provider.CalculateRates(context.Background(), shipping.RatesProviderRequest{
		OriginAreaID:      "IDNP6",
		DestinationAreaID: "IDNP3",
		Couriers:          "jne,sicepat",
		Items: []shipping.RatesProviderItem{
			{Name: "Shoes", Value: 199000, Weight: 500, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("CalculateRates() error = %v", err)
	}

	if len(rates) != 2 {
		t.Fatalf("rates count = %d, want 2", len(rates))
	}

	if rates[0].CourierCode != "jne" || rates[0].Price != 18000 {
		t.Fatalf("rates[0] = %+v, want jne 18000", rates[0])
	}

	if rates[1].CourierCode != "sicepat" || rates[1].Price != 25000 {
		t.Fatalf("rates[1] = %+v, want sicepat 25000", rates[1])
	}
}

func TestProviderCalculateRatesWrapsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewProvider(NewClient(server.URL, "key", 0))

	_, err := provider.CalculateRates(context.Background(), shipping.RatesProviderRequest{})
	if !errors.Is(err, shipping.ErrAreaProvider) {
		t.Fatalf("error = %v, want wrapped ErrAreaProvider", err)
	}
}
