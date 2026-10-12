package biteship

import (
	"context"
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
