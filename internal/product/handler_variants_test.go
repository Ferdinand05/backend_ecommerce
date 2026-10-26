package product

import (
	"encoding/json"
	"net/http"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TestProductListIncludesVariantSummary memastikan list produk memuat
// variants ringkas (id, sku, name, price, is_active) tanpa stock/images.
func TestProductListIncludesVariantSummary(t *testing.T) {
	desc := "portable computer"
	svc := &fakeService{findAllRes: []models.Product{
		{ID: uuid.New(), Name: "laptop", Slug: "laptop", Description: &desc, IsActive: true,
			Variants: []models.ProductVariant{
				{ID: uuid.New(), SKU: "LAP-001", Name: "16GB", Price: decimal.NewFromInt(15000000), IsActive: true},
				{ID: uuid.New(), SKU: "LAP-002", Name: "32GB", Price: decimal.NewFromInt(25000000), IsActive: true},
			}},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var raw struct {
		Products []struct {
			Variants []map[string]any `json:"variants"`
		} `json:"products"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(raw.Products) != 1 {
		t.Fatalf("products len = %d, want 1", len(raw.Products))
	}

	if len(raw.Products[0].Variants) != 2 {
		t.Fatalf("variants len = %d, want 2 (katalog butuh jumlah variant)", len(raw.Products[0].Variants))
	}

	for _, key := range []string{"id", "sku", "name", "price", "is_active"} {
		if _, ok := raw.Products[0].Variants[0][key]; !ok {
			t.Fatalf("variant missing key %q in %s", key, w.Body.String())
		}
	}

	// Ringkas: tanpa stok dan gambar.
	for _, key := range []string{"inventory", "images", "weight_grams"} {
		if _, ok := raw.Products[0].Variants[0][key]; ok {
			t.Fatalf("variant list harus ringkas, tapi punya %q: %s", key, w.Body.String())
		}
	}
}

// TestProductDetailIncludesVariantStockAndImages memastikan detail produk
// memuat variants lengkap dengan stok dan gambar.
func TestProductDetailIncludesVariantStockAndImages(t *testing.T) {
	id := uuid.New()
	svc := &fakeService{findByIDRes: models.Product{
		ID: id, Name: "laptop", Slug: "laptop", IsActive: true,
		Variants: []models.ProductVariant{
			{ID: uuid.New(), SKU: "LAP-001", Name: "16GB", Price: decimal.NewFromInt(15000000), IsActive: true,
				WeightGrams: 1200,
				Inventory:   &models.InventoryItem{Quantity: 8, ReservedQuantity: 2},
				Images:      []models.ProductImage{{ID: uuid.New(), StorageKey: "variants/lap.jpg", SortOrder: 1}}},
		},
	}}

	w := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/products/"+id.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var raw struct {
		Product struct {
			Variants []struct {
				SKU         string `json:"sku"`
				WeightGrams int    `json:"weight_grams"`
				Inventory   *struct {
					Quantity         int `json:"quantity"`
					ReservedQuantity int `json:"reserved_quantity"`
				} `json:"inventory"`
				Images []map[string]any `json:"images"`
			} `json:"variants"`
		} `json:"product"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(raw.Product.Variants) != 1 {
		t.Fatalf("variants len = %d, want 1", len(raw.Product.Variants))
	}

	variant := raw.Product.Variants[0]
	if variant.SKU != "LAP-001" {
		t.Fatalf("sku = %q, want LAP-001", variant.SKU)
	}

	if variant.WeightGrams != 1200 {
		t.Fatalf("weight_grams = %d, want 1200", variant.WeightGrams)
	}

	if variant.Inventory == nil {
		t.Fatalf("inventory null: %s", w.Body.String())
	}

	if variant.Inventory.Quantity != 8 || variant.Inventory.ReservedQuantity != 2 {
		t.Fatalf("inventory = %+v, want quantity 8 reserved 2", variant.Inventory)
	}

	if len(variant.Images) != 1 {
		t.Fatalf("images len = %d, want 1", len(variant.Images))
	}

	if _, ok := variant.Images[0]["url"]; !ok {
		t.Fatalf("image missing url: %s", w.Body.String())
	}
}
