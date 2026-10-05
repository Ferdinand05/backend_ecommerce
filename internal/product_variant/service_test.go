package productvariant

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/product"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeRepo struct {
	findAllRes       []models.ProductVariant
	findAllErr       error
	findAllProductID uuid.UUID

	findByIDRes models.ProductVariant
	findByIDErr error

	findBySKURes models.ProductVariant
	findBySKUErr error

	existsBySKU    bool
	existsBySKUErr error
	existsArgSKU   string
	existsArgID    uuid.UUID

	createErr error
	createArg models.ProductVariant

	updateErr error
	updateID  uuid.UUID
	updateArg models.ProductVariant

	deleteErr error
	deleteID  uuid.UUID
}

func (f *fakeRepo) Create(ctx context.Context, variant models.ProductVariant) error {
	f.createArg = variant
	return f.createErr
}

func (f *fakeRepo) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {
	f.findAllProductID = productID
	return f.findAllRes, f.findAllErr
}

func (f *fakeRepo) FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeRepo) FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error) {
	return f.findBySKURes, f.findBySKUErr
}

func (f *fakeRepo) ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error) {
	f.existsArgSKU = sku
	f.existsArgID = id
	return f.existsBySKU, f.existsBySKUErr
}

func (f *fakeRepo) Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error {
	f.updateID = id
	f.updateArg = variant
	return f.updateErr
}

func (f *fakeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.deleteID = id
	return f.deleteErr
}

type fakeProductRepo struct {
	findByIDRes models.Product
	findByIDErr error
}

func (f *fakeProductRepo) Create(ctx context.Context, product models.Product) (models.Product, error) {
	return models.Product{}, nil
}

func (f *fakeProductRepo) FindAll(ctx context.Context) ([]models.Product, error) {
	return nil, nil
}

func (f *fakeProductRepo) FindByID(ctx context.Context, id uuid.UUID) (models.Product, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeProductRepo) FindBySlug(ctx context.Context, slug string) (models.Product, error) {
	return models.Product{}, nil
}

func (f *fakeProductRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, nil
}

func (f *fakeProductRepo) Update(ctx context.Context, id uuid.UUID, product models.Product) (models.Product, error) {
	return models.Product{}, nil
}

func (f *fakeProductRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestServiceCreateChecks(t *testing.T) {
	productID := uuid.New()
	skuErr := errors.New("sku lookup failed")

	tests := []struct {
		name    string
		repo    *fakeRepo
		product *fakeProductRepo
		wantErr error
	}{
		{
			name:    "product not found",
			repo:    &fakeRepo{},
			product: &fakeProductRepo{findByIDErr: product.ErrorProductNotFound},
			wantErr: product.ErrorProductNotFound,
		},
		{
			name:    "sku already exists",
			repo:    &fakeRepo{findBySKURes: models.ProductVariant{SKU: "LAP-001"}},
			product: &fakeProductRepo{},
			wantErr: ErrorVariantSKUAlreadyExists,
		},
		{
			name:    "sku lookup error",
			repo:    &fakeRepo{findBySKUErr: skuErr},
			product: &fakeProductRepo{},
			wantErr: skuErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.product, nil, nil)

			_, err := svc.Create(context.Background(), productID, CreateProductVariantRequest{
				SKU:   "LAP-001",
				Name:  "16GB",
				Price: decimal.NewFromInt(15000000),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}

			if tt.repo.createArg.ID != uuid.Nil {
				t.Fatal("Create() must not reach repository on validation failure")
			}
		})
	}
}

func TestServiceFindAllByProductID(t *testing.T) {
	productID := uuid.New()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantLen int
		wantErr error
	}{
		{
			name:    "success",
			repo:    &fakeRepo{findAllRes: []models.ProductVariant{{SKU: "A"}, {SKU: "B"}}},
			wantLen: 2,
		},
		{
			name:    "repo error",
			repo:    &fakeRepo{findAllErr: dbErr},
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeProductRepo{}, nil, nil)

			got, err := svc.FindAllByProductID(context.Background(), productID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindAllByProductID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if tt.repo.findAllProductID != productID {
				t.Fatalf("FindAllByProductID() repo productID = %v, want %v", tt.repo.findAllProductID, productID)
			}

			if len(got) != tt.wantLen {
				t.Fatalf("FindAllByProductID() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestServiceFindByID(t *testing.T) {
	productID := uuid.New()
	id := uuid.New()

	tests := []struct {
		name    string
		repo    *fakeRepo
		want    string
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findByIDRes: models.ProductVariant{ID: id, ProductID: productID, SKU: "LAP-001"}},
			want: "LAP-001",
		},
		{
			name:    "belongs to another product",
			repo:    &fakeRepo{findByIDRes: models.ProductVariant{ID: id, ProductID: uuid.New(), SKU: "LAP-001"}},
			wantErr: ErrorProductVariantNotFound,
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorProductVariantNotFound},
			wantErr: ErrorProductVariantNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeProductRepo{}, nil, nil)

			got, err := svc.FindByID(context.Background(), productID, id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.SKU != tt.want {
				t.Fatalf("FindByID() = %v, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceUpdateChecks(t *testing.T) {
	productID := uuid.New()
	id := uuid.New()
	existsErr := errors.New("exists check failed")

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantErr error
	}{
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorProductVariantNotFound},
			wantErr: ErrorProductVariantNotFound,
		},
		{
			name:    "belongs to another product",
			repo:    &fakeRepo{findByIDRes: models.ProductVariant{ID: id, ProductID: uuid.New()}},
			wantErr: ErrorProductVariantNotFound,
		},
		{
			name:    "sku conflict with other variant",
			repo:    &fakeRepo{findByIDRes: models.ProductVariant{ID: id, ProductID: productID}, existsBySKU: true},
			wantErr: ErrorVariantSKUAlreadyExists,
		},
		{
			name:    "exists check error",
			repo:    &fakeRepo{findByIDRes: models.ProductVariant{ID: id, ProductID: productID}, existsBySKUErr: existsErr},
			wantErr: existsErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeProductRepo{}, nil, nil)

			_, err := svc.Update(context.Background(), productID, id, UpdateProductVariantRequest{
				SKU:   "LAP-002",
				Name:  "32GB",
				Price: decimal.NewFromInt(20000000),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}

			if tt.repo.updateID != uuid.Nil {
				t.Fatal("Update() must not reach repository on validation failure")
			}
		})
	}
}

func TestServiceDeleteChecks(t *testing.T) {
	productID := uuid.New()
	id := uuid.New()

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantErr error
	}{
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorProductVariantNotFound},
			wantErr: ErrorProductVariantNotFound,
		},
		{
			name:    "belongs to another product",
			repo:    &fakeRepo{findByIDRes: models.ProductVariant{ID: id, ProductID: uuid.New()}},
			wantErr: ErrorProductVariantNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeProductRepo{}, nil, nil)

			err := svc.Delete(context.Background(), productID, id)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}

			if tt.repo.deleteID != uuid.Nil {
				t.Fatal("Delete() must not reach repository on validation failure")
			}
		})
	}
}
