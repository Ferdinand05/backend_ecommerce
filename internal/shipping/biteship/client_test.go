package biteship

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientSearchAreas(t *testing.T) {
	var gotAuth, gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"areas": [
				{"id": "IDJA0100", "name": "Jakarta Pusat", "postal_code": 10110},
				{"id": "IDBD0100", "name": "Bandung", "postal_code": 40111}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", 0)

	results, err := client.SearchAreas(context.Background(), "jakarta")
	if err != nil {
		t.Fatalf("SearchAreas() error = %v", err)
	}

	if gotAuth != "test-api-key" {
		t.Fatalf("authorization header = %q, want api key", gotAuth)
	}

	if gotQuery != "countries=ID&input=jakarta&type=single" {
		t.Fatalf("query = %q, want countries/input/type", gotQuery)
	}

	if len(results) != 2 || results[0].ID != "IDJA0100" || results[0].PostalCode != 10110 {
		t.Fatalf("results = %v, want two areas", results)
	}
}

func TestClientSearchAreasNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key", 0)

	_, err := client.SearchAreas(context.Background(), "jakarta")
	if err == nil {
		t.Fatal("SearchAreas() error = nil, want error for HTTP 401")
	}
}

func TestClientSearchAreasUnsuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": false, "areas": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", 0)

	_, err := client.SearchAreas(context.Background(), "jakarta")
	if err == nil {
		t.Fatal("SearchAreas() error = nil, want error when success=false")
	}
}

func TestClientSearchAreasInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", 0)

	_, err := client.SearchAreas(context.Background(), "jakarta")
	if err == nil {
		t.Fatal("SearchAreas() error = nil, want decode error")
	}
}

func TestClientSearchAreasConnectionError(t *testing.T) {
	// URL yang pasti tidak bisa dihubungi.
	client := NewClient("http://127.0.0.1:1", "test-api-key", 0)

	_, err := client.SearchAreas(context.Background(), "jakarta")
	if err == nil {
		t.Fatal("SearchAreas() error = nil, want connection error")
	}
}

func TestClientDefaults(t *testing.T) {
	client := NewClient("", "key", 0)

	if client.baseURL != defaultBaseURL {
		t.Fatalf("baseURL = %q, want default", client.baseURL)
	}

	if client.httpClient.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", client.httpClient.Timeout)
	}
}

// CalculateRates tests

func TestClientCalculateRates(t *testing.T) {
	var gotAuth, gotContentType, gotMethod string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)

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
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", 0)

	results, err := client.CalculateRates(context.Background(), ratesAPIRequest{
		OriginAreaID:      "IDNP6",
		DestinationAreaID: "IDNP3",
		Couriers:          "jne",
		Items: []ratesAPIItem{
			{Name: "Shoes", Value: 199000, Weight: 500, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("CalculateRates() error = %v", err)
	}

	if gotAuth != "test-api-key" {
		t.Fatalf("auth = %q, want api key", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type = %q, want application/json", gotContentType)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}

	// Verifikasi body mengirim weight per unit, bukan weight*quantity.
	var sentBody ratesAPIRequest
	if err := json.Unmarshal(gotBody, &sentBody); err != nil {
		t.Fatalf("unmarshal sent body: %v", err)
	}
	if sentBody.Items[0].Weight != 500 || sentBody.Items[0].Quantity != 2 {
		t.Fatalf("body items = %+v, want weight=500 quantity=2 (tidak boleh pre-multiply)", sentBody.Items[0])
	}

	if len(results) != 1 || results[0].CourierCode != "jne" || results[0].Price != 18000 {
		t.Fatalf("results = %v, want jne 18000", results)
	}
}

func TestClientCalculateRatesNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key", 0)

	_, err := client.CalculateRates(context.Background(), ratesAPIRequest{})
	if err == nil {
		t.Fatal("want error for HTTP 401")
	}
}

func TestClientCalculateRatesUnsuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success": false, "pricing": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", 0)

	_, err := client.CalculateRates(context.Background(), ratesAPIRequest{})
	if err == nil {
		t.Fatal("want error when success=false")
	}
}

func TestClientCalculateRatesInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{broken"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", 0)

	_, err := client.CalculateRates(context.Background(), ratesAPIRequest{})
	if err == nil {
		t.Fatal("want decode error")
	}
}
