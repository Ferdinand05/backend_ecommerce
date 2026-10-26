package product

import (
	"context"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func publicProduct(name, description string, price int64, active bool) models.Product {
	return models.Product{
		ID:          uuid.New(),
		Name:        name,
		Slug:        name,
		Description: &description,
		IsActive:    active,
		Variants: []models.ProductVariant{
			{ID: uuid.New(), SKU: name + "-SKU", Name: name, Price: decimal.NewFromInt(price), IsActive: active},
		},
	}
}

func TestFindPublicListFiltersByKeyword(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Product{
		publicProduct("laptop", "portable computer", 1000, true),
		publicProduct("mouse", "clicky device", 200, true),
	}}

	svc := newTestService(repo, &fakeCache{data: map[string][]byte{}})

	result, err := svc.FindPublicList(context.Background(), ProductFilter{Q: "mouse"})
	if err != nil {
		t.Fatalf("FindPublicList() error = %v", err)
	}

	if len(result.Products) != 1 || result.Products[0].Name != "mouse" {
		t.Fatalf("FindPublicList() = %v, want only mouse", result.Products)
	}

	if result.Total != 1 {
		t.Fatalf("FindPublicList() total = %d, want 1", result.Total)
	}
}

func TestFindPublicListFiltersByCategory(t *testing.T) {
	categoryID := uuid.New()
	other := uuid.New()

	first := publicProduct("laptop", "computer", 1000, true)
	first.CategoryID = categoryID

	second := publicProduct("mouse", "mouse", 200, true)
	second.CategoryID = other

	repo := &fakeRepo{findAllRes: []models.Product{first, second}}
	svc := newTestService(repo, &fakeCache{data: map[string][]byte{}})

	result, err := svc.FindPublicList(context.Background(), ProductFilter{CategoryID: &categoryID})
	if err != nil {
		t.Fatalf("FindPublicList() error = %v", err)
	}

	if len(result.Products) != 1 || result.Products[0].Name != "laptop" {
		t.Fatalf("FindPublicList() = %v, want only laptop", result.Products)
	}
}

func TestFindPublicListPaginates(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Product{
		publicProduct("a", "a", 100, true),
		publicProduct("b", "b", 200, true),
		publicProduct("c", "c", 300, true),
	}}

	svc := newTestService(repo, &fakeCache{data: map[string][]byte{}})

	result, err := svc.FindPublicList(context.Background(), ProductFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("FindPublicList() error = %v", err)
	}

	if len(result.Products) != 1 || result.Products[0].Name != "c" {
		t.Fatalf("FindPublicList() = %v, want page 2 with 1 item", result.Products)
	}

	if result.Total != 3 || result.Page != 2 || result.PageSize != 2 {
		t.Fatalf("FindPublicList() meta = (%d, %d, %d), want (3, 2, 2)", result.Total, result.Page, result.PageSize)
	}
}

func TestFindPublicListSortsByPrice(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Product{
		publicProduct("expensive", "x", 900, true),
		publicProduct("cheap", "y", 100, true),
	}}

	svc := newTestService(repo, &fakeCache{data: map[string][]byte{}})

	result, err := svc.FindPublicList(context.Background(), ProductFilter{Sort: "price_asc"})
	if err != nil {
		t.Fatalf("FindPublicList() error = %v", err)
	}

	if result.Products[0].Name != "cheap" {
		t.Fatalf("FindPublicList() = %v, want cheap first", result.Products)
	}
}

func TestProductFilterNormalize(t *testing.T) {
	filter := ProductFilter{}.Normalize()

	if filter.Page != 1 || filter.PageSize != defaultProductPageSize {
		t.Fatalf("Normalize() = (%d, %d), want (1, %d)", filter.Page, filter.PageSize, defaultProductPageSize)
	}

	filter = ProductFilter{Page: -3, PageSize: 5000}.Normalize()

	if filter.Page != 1 || filter.PageSize != maxProductPageSize {
		t.Fatalf("Normalize() = (%d, %d), want (1, %d)", filter.Page, filter.PageSize, maxProductPageSize)
	}
}
