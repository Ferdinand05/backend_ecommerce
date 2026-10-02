package productvariant

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/product"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func newVariantService(db *gorm.DB) *service {
	productRepo := product.NewRepository(db)
	variantRepo := NewRepository(db)
	return NewService(variantRepo, productRepo, db)
}

func TestVariantServiceCreate(t *testing.T) {
	db := setupTestDB(t)
	svc := newVariantService(db)

	category := seedCategory(t, db, "electronics", "electronics")
	prod := seedProduct(t, db, category.ID, "laptop", "laptop")

	created, err := svc.Create(context.Background(), prod.ID, CreateProductVariantRequest{
		SKU:   "LAP-001",
		Name:  "16GB",
		Price: decimal.NewFromInt(15000000),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.SKU != "LAP-001" || !created.IsActive {
		t.Fatalf("Create() = %v, want active LAP-001 variant", created)
	}

	found, err := svc.FindByID(context.Background(), prod.ID, created.ID)
	if err != nil {
		t.Fatalf("FindByID() after Create error = %v", err)
	}

	if found.SKU != "LAP-001" {
		t.Fatalf("FindByID() after Create = %v, want persisted variant", found)
	}

	_, err = svc.Create(context.Background(), prod.ID, CreateProductVariantRequest{
		SKU:   "LAP-001",
		Name:  "duplicate",
		Price: decimal.NewFromInt(1),
	})
	if !errors.Is(err, ErrorVariantSKUAlreadyExists) {
		t.Fatalf("Create() duplicate sku error = %v, want %v", err, ErrorVariantSKUAlreadyExists)
	}

	_, err = svc.Create(context.Background(), uuid.New(), CreateProductVariantRequest{
		SKU:   "LAP-00X",
		Name:  "orphan",
		Price: decimal.NewFromInt(1),
	})
	if !errors.Is(err, product.ErrorProductNotFound) {
		t.Fatalf("Create() missing product error = %v, want %v", err, product.ErrorProductNotFound)
	}
}

func TestVariantServiceFindAllByProductID(t *testing.T) {
	db := setupTestDB(t)
	svc := newVariantService(db)

	category := seedCategory(t, db, "electronics", "electronics")
	prod := seedProduct(t, db, category.ID, "laptop", "laptop")

	for _, sku := range []string{"LAP-001", "LAP-002"} {
		if _, err := svc.Create(context.Background(), prod.ID, CreateProductVariantRequest{
			SKU:   sku,
			Name:  sku,
			Price: decimal.NewFromInt(1000000),
		}); err != nil {
			t.Fatalf("Create() %s error = %v", sku, err)
		}
	}

	variants, err := svc.FindAllByProductID(context.Background(), prod.ID)
	if err != nil {
		t.Fatalf("FindAllByProductID() error = %v", err)
	}

	if len(variants) != 2 {
		t.Fatalf("FindAllByProductID() len = %d, want 2", len(variants))
	}

	if variants[0].SKU != "LAP-001" || variants[1].SKU != "LAP-002" {
		t.Fatalf("FindAllByProductID() = %v, %v, want ascending order", variants[0], variants[1])
	}
}

func TestVariantServiceUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := newVariantService(db)

	category := seedCategory(t, db, "electronics", "electronics")
	prod := seedProduct(t, db, category.ID, "laptop", "laptop")

	created, err := svc.Create(context.Background(), prod.ID, CreateProductVariantRequest{
		SKU:   "LAP-001",
		Name:  "16GB",
		Price: decimal.NewFromInt(15000000),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(context.Background(), prod.ID, created.ID, UpdateProductVariantRequest{
		SKU:   "LAP-001",
		Name:  "16GB Pro",
		Price: decimal.NewFromInt(16000000),
	})
	if err != nil {
		t.Fatalf("Update() with own sku error = %v", err)
	}

	if updated.Name != "16GB Pro" || !updated.IsActive {
		t.Fatalf("Update() = %v, want updated name and preserved IsActive", updated)
	}

	found, err := svc.FindByID(context.Background(), prod.ID, created.ID)
	if err != nil {
		t.Fatalf("FindByID() after Update error = %v", err)
	}

	if found.Name != "16GB Pro" || found.Price.String() != "16000000" {
		t.Fatalf("FindByID() after Update = %v, want persisted changes", found)
	}

	if _, err := svc.Create(context.Background(), prod.ID, CreateProductVariantRequest{
		SKU:   "LAP-002",
		Name:  "32GB",
		Price: decimal.NewFromInt(20000000),
	}); err != nil {
		t.Fatalf("Create() LAP-002 error = %v", err)
	}

	_, err = svc.Update(context.Background(), prod.ID, created.ID, UpdateProductVariantRequest{
		SKU:   "LAP-002",
		Name:  "conflict",
		Price: decimal.NewFromInt(1),
	})
	if !errors.Is(err, ErrorVariantSKUAlreadyExists) {
		t.Fatalf("Update() conflicting sku error = %v, want %v", err, ErrorVariantSKUAlreadyExists)
	}

	_, err = svc.Update(context.Background(), uuid.New(), created.ID, UpdateProductVariantRequest{
		SKU:   "LAP-001",
		Name:  "stolen",
		Price: decimal.NewFromInt(1),
	})
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("Update() other product error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}

func TestVariantServiceDelete(t *testing.T) {
	db := setupTestDB(t)
	svc := newVariantService(db)

	category := seedCategory(t, db, "electronics", "electronics")
	prod := seedProduct(t, db, category.ID, "laptop", "laptop")

	created, err := svc.Create(context.Background(), prod.ID, CreateProductVariantRequest{
		SKU:   "LAP-DEL",
		Name:  "to delete",
		Price: decimal.NewFromInt(1000),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = svc.Delete(context.Background(), prod.ID, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = svc.FindByID(context.Background(), prod.ID, created.ID)
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorProductVariantNotFound)
	}

	err = svc.Delete(context.Background(), prod.ID, created.ID)
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("Delete() after Delete error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}

