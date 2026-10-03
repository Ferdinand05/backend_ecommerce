package productimage

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/product"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeImageRepo struct {
	createErr error
	createArg models.ProductImage

	findByIDRes models.ProductImage
	findByIDErr error

	findAllByProductRes []models.ProductImage
	findAllByProductErr error
	findAllProductID    uuid.UUID

	findAllByVariantRes []models.ProductImage
	findAllByVariantErr error
	findAllVariantID    uuid.UUID

	deleteErr error
	deleteID  uuid.UUID
}

func (f *fakeImageRepo) Create(ctx context.Context, image models.ProductImage) error {
	f.createArg = image
	return f.createErr
}

func (f *fakeImageRepo) FindByID(ctx context.Context, id uuid.UUID) (models.ProductImage, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeImageRepo) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductImage, error) {
	f.findAllProductID = productID
	return f.findAllByProductRes, f.findAllByProductErr
}

func (f *fakeImageRepo) FindAllByVariantID(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) ([]models.ProductImage, error) {
	f.findAllProductID = productID
	f.findAllVariantID = variantID
	return f.findAllByVariantRes, f.findAllByVariantErr
}

func (f *fakeImageRepo) Delete(ctx context.Context, id uuid.UUID) error {
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
func (f *fakeProductRepo) FindAll(ctx context.Context) ([]models.Product, error) { return nil, nil }
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
func (f *fakeProductRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type fakeVariantRepo struct {
	findByIDRes models.ProductVariant
	findByIDErr error
}

func (f *fakeVariantRepo) Create(ctx context.Context, variant models.ProductVariant) error {
	return nil
}
func (f *fakeVariantRepo) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {
	return nil, nil
}
func (f *fakeVariantRepo) FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error) {
	return f.findByIDRes, f.findByIDErr
}
func (f *fakeVariantRepo) FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error) {
	return models.ProductVariant{}, nil
}
func (f *fakeVariantRepo) ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeVariantRepo) Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error {
	return nil
}
func (f *fakeVariantRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type fakeStorage struct {
	uploadErr    error
	deleteErr    error
	uploadedKeys []string
	deletedKeys  []string
	publicURL    string
}

func (f *fakeStorage) Upload(ctx context.Context, key string, file io.Reader, contentType string) error {
	f.uploadedKeys = append(f.uploadedKeys, key)
	return f.uploadErr
}

func (f *fakeStorage) Delete(ctx context.Context, key string) error {
	f.deletedKeys = append(f.deletedKeys, key)
	return f.deleteErr
}

func (f *fakeStorage) URL(key string) string {
	return f.publicURL + "/" + key
}

func TestServiceUploadParent(t *testing.T) {
	productID := uuid.New()
	repo := &fakeImageRepo{}
	store := &fakeStorage{publicURL: "https://cdn.test"}
	svc := NewService(repo, &fakeProductRepo{findByIDRes: models.Product{ID: productID}}, &fakeVariantRepo{}, store)

	alt := "front"
	got, err := svc.Upload(context.Background(), productID, nil, strings.NewReader("img"), "image/png", &alt, 2)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if repo.createArg.ProductID != productID || repo.createArg.ProductVariantID != nil {
		t.Fatalf("Upload() image = %v, want parent image for product", repo.createArg)
	}

	wantPrefix := "products/" + productID.String() + "/"
	if !strings.HasPrefix(repo.createArg.StorageKey, wantPrefix) || !strings.HasSuffix(repo.createArg.StorageKey, ".png") {
		t.Fatalf("Upload() storage key = %q, want prefix %q and .png suffix", repo.createArg.StorageKey, wantPrefix)
	}

	if len(store.uploadedKeys) != 1 || store.uploadedKeys[0] != repo.createArg.StorageKey {
		t.Fatalf("Upload() uploaded keys = %v, want %q", store.uploadedKeys, repo.createArg.StorageKey)
	}

	if got.URL != "https://cdn.test/"+repo.createArg.StorageKey {
		t.Fatalf("Upload() URL = %q, want %q", got.URL, "https://cdn.test/"+repo.createArg.StorageKey)
	}

	if got.Alt == nil || *got.Alt != alt || got.SortOrder != 2 {
		t.Fatalf("Upload() alt/sort = %v/%d, want %q/2", got.Alt, got.SortOrder, alt)
	}
}

func TestServiceUploadVariant(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()

	repo := &fakeImageRepo{}
	store := &fakeStorage{publicURL: "https://cdn.test"}
	variantRepo := &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}}
	svc := NewService(repo, &fakeProductRepo{findByIDRes: models.Product{ID: productID}}, variantRepo, store)

	got, err := svc.Upload(context.Background(), productID, &variantID, strings.NewReader("img"), "image/webp", nil, 0)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if repo.createArg.ProductVariantID == nil || *repo.createArg.ProductVariantID != variantID {
		t.Fatalf("Upload() variant id = %v, want %v", repo.createArg.ProductVariantID, variantID)
	}

	wantPrefix := "products/" + productID.String() + "/variants/" + variantID.String() + "/"
	if !strings.HasPrefix(repo.createArg.StorageKey, wantPrefix) || !strings.HasSuffix(repo.createArg.StorageKey, ".webp") {
		t.Fatalf("Upload() storage key = %q, want prefix %q and .webp suffix", repo.createArg.StorageKey, wantPrefix)
	}

	if got.ProductVariantID == nil || *got.ProductVariantID != variantID {
		t.Fatalf("Upload() response variant id = %v, want %v", got.ProductVariantID, variantID)
	}
}

func TestServiceUploadVariantNotOwned(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()

	repo := &fakeImageRepo{}
	store := &fakeStorage{}
	variantRepo := &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: uuid.New()}}
	svc := NewService(repo, &fakeProductRepo{findByIDRes: models.Product{ID: productID}}, variantRepo, store)

	_, err := svc.Upload(context.Background(), productID, &variantID, strings.NewReader("img"), "image/png", nil, 0)
	if !errors.Is(err, productvariant.ErrorProductVariantNotFound) {
		t.Fatalf("Upload() error = %v, want %v", err, productvariant.ErrorProductVariantNotFound)
	}

	if len(store.uploadedKeys) != 0 {
		t.Fatal("Upload() must not reach storage when variant is not owned")
	}
}

func TestServiceUploadRejections(t *testing.T) {
	productID := uuid.New()
	dbErr := errors.New("db down")
	storageErr := errors.New("storage down")

	tests := []struct {
		name          string
		repo          *fakeImageRepo
		product       *fakeProductRepo
		store         *fakeStorage
		contentType   string
		wantErr       error
		wantUploaded  int
		wantCompDelet int
	}{
		{
			name:        "unsupported type",
			repo:        &fakeImageRepo{},
			product:     &fakeProductRepo{},
			store:       &fakeStorage{},
			contentType: "text/plain",
			wantErr:     ErrorUnsupportedImageType,
		},
		{
			name:        "product not found",
			repo:        &fakeImageRepo{},
			product:     &fakeProductRepo{findByIDErr: product.ErrorProductNotFound},
			store:       &fakeStorage{},
			contentType: "image/png",
			wantErr:     product.ErrorProductNotFound,
		},
		{
			name:         "storage upload error",
			repo:         &fakeImageRepo{},
			product:      &fakeProductRepo{},
			store:        &fakeStorage{uploadErr: storageErr},
			contentType:  "image/png",
			wantErr:      storageErr,
			wantUploaded: 1,
		},
		{
			name:          "db create error triggers compensation delete",
			repo:          &fakeImageRepo{createErr: dbErr},
			product:       &fakeProductRepo{},
			store:         &fakeStorage{},
			contentType:   "image/png",
			wantErr:       dbErr,
			wantUploaded:  1,
			wantCompDelet: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.product, &fakeVariantRepo{}, tt.store)

			_, err := svc.Upload(context.Background(), productID, nil, strings.NewReader("img"), tt.contentType, nil, 0)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Upload() error = %v, want %v", err, tt.wantErr)
			}

			if len(tt.store.uploadedKeys) != tt.wantUploaded {
				t.Fatalf("Upload() uploaded = %d, want %d", len(tt.store.uploadedKeys), tt.wantUploaded)
			}

			if len(tt.store.deletedKeys) != tt.wantCompDelet {
				t.Fatalf("Upload() compensated deletes = %d, want %d", len(tt.store.deletedKeys), tt.wantCompDelet)
			}
		})
	}
}

func TestServiceUploadCompensationFailureSurfacesError(t *testing.T) {
	productID := uuid.New()
	repo := &fakeImageRepo{createErr: errors.New("db down")}
	store := &fakeStorage{deleteErr: errors.New("storage down")}
	svc := NewService(repo, &fakeProductRepo{}, &fakeVariantRepo{}, store)

	_, err := svc.Upload(context.Background(), productID, nil, strings.NewReader("img"), "image/png", nil, 0)
	if err == nil {
		t.Fatal("Upload() error = nil, want combined create+cleanup error")
	}

	if len(store.deletedKeys) != 1 {
		t.Fatal("Upload() must attempt compensating delete")
	}
}

func TestServiceFindByID(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	otherVariantID := uuid.New()
	imageID := uuid.New()

	parentImage := models.ProductImage{ID: imageID, ProductID: productID, StorageKey: "products/parent.png"}
	variantImage := models.ProductImage{ID: imageID, ProductID: productID, ProductVariantID: &variantID, StorageKey: "products/variant.png"}
	otherVariantImage := models.ProductImage{ID: imageID, ProductID: productID, ProductVariantID: &otherVariantID, StorageKey: "products/other.png"}

	tests := []struct {
		name      string
		image     models.ProductImage
		repoErr   error
		variantID *uuid.UUID
		wantURL   string
		wantErr   error
	}{
		{
			name:    "parent image at parent scope",
			image:   parentImage,
			wantURL: "https://cdn.test/products/parent.png",
		},
		{
			name:    "parent scope rejects variant image",
			image:   variantImage,
			wantErr: ErrorProductImageNotFound,
		},
		{
			name:      "variant image at matching scope",
			image:     variantImage,
			variantID: &variantID,
			wantURL:   "https://cdn.test/products/variant.png",
		},
		{
			name:      "variant scope rejects parent image",
			image:     parentImage,
			variantID: &variantID,
			wantErr:   ErrorProductImageNotFound,
		},
		{
			name:      "variant scope rejects other variant image",
			image:     otherVariantImage,
			variantID: &variantID,
			wantErr:   ErrorProductImageNotFound,
		},
		{
			name:    "not found",
			repoErr: ErrorProductImageNotFound,
			wantErr: ErrorProductImageNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeImageRepo{findByIDRes: tt.image, findByIDErr: tt.repoErr}
			svc := NewService(repo, &fakeProductRepo{}, &fakeVariantRepo{}, &fakeStorage{publicURL: "https://cdn.test"})

			got, err := svc.FindByID(context.Background(), productID, tt.variantID, imageID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.URL != tt.wantURL {
				t.Fatalf("FindByID() URL = %q, want %q", got.URL, tt.wantURL)
			}
		})
	}
}

func TestServiceFindAllByProductID(t *testing.T) {
	productID := uuid.New()
	repo := &fakeImageRepo{findAllByProductRes: []models.ProductImage{
		{ID: uuid.New(), ProductID: productID, StorageKey: "a.png"},
		{ID: uuid.New(), ProductID: productID, StorageKey: "b.png"},
	}}
	store := &fakeStorage{publicURL: "https://cdn.test"}
	svc := NewService(repo, &fakeProductRepo{}, &fakeVariantRepo{}, store)

	got, err := svc.FindAllByProductID(context.Background(), productID)
	if err != nil {
		t.Fatalf("FindAllByProductID() error = %v", err)
	}

	if repo.findAllProductID != productID {
		t.Fatalf("FindAllByProductID() repo productID = %v, want %v", repo.findAllProductID, productID)
	}

	if len(got) != 2 || got[0].URL != "https://cdn.test/a.png" {
		t.Fatalf("FindAllByProductID() = %v, want mapped URLs", got)
	}
}

func TestServiceFindAllByVariantID(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	repo := &fakeImageRepo{findAllByVariantRes: []models.ProductImage{
		{ID: uuid.New(), ProductID: productID, ProductVariantID: &variantID, StorageKey: "v.png"},
	}}
	store := &fakeStorage{publicURL: "https://cdn.test"}
	svc := NewService(repo, &fakeProductRepo{}, &fakeVariantRepo{}, store)

	got, err := svc.FindAllByVariantID(context.Background(), productID, variantID)
	if err != nil {
		t.Fatalf("FindAllByVariantID() error = %v", err)
	}

	if repo.findAllProductID != productID || repo.findAllVariantID != variantID {
		t.Fatalf("FindAllByVariantID() repo args = %v/%v, want %v/%v", repo.findAllProductID, repo.findAllVariantID, productID, variantID)
	}

	if len(got) != 1 || got[0].URL != "https://cdn.test/v.png" {
		t.Fatalf("FindAllByVariantID() = %v, want mapped URL", got)
	}
}

func TestServiceDelete(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	imageID := uuid.New()
	storageErr := errors.New("storage down")

	tests := []struct {
		name         string
		image        models.ProductImage
		repoErr      error
		storeErr     error
		variantID    *uuid.UUID
		wantErr      error
		wantDeletes  int
		wantRepoCall bool
	}{
		{
			name:         "success parent",
			image:        models.ProductImage{ID: imageID, ProductID: productID, StorageKey: "a.png"},
			wantDeletes:  1,
			wantRepoCall: true,
		},
		{
			name:        "parent scope rejects variant image",
			image:       models.ProductImage{ID: imageID, ProductID: productID, ProductVariantID: &variantID, StorageKey: "v.png"},
			wantErr:     ErrorProductImageNotFound,
			wantDeletes: 0,
		},
		{
			name:        "storage delete error keeps row",
			image:       models.ProductImage{ID: imageID, ProductID: productID, StorageKey: "a.png"},
			storeErr:    storageErr,
			wantErr:     storageErr,
			wantDeletes: 1,
		},
		{
			name:    "not found",
			repoErr: ErrorProductImageNotFound,
			wantErr: ErrorProductImageNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeImageRepo{findByIDRes: tt.image, findByIDErr: tt.repoErr}
			store := &fakeStorage{deleteErr: tt.storeErr}
			svc := NewService(repo, &fakeProductRepo{}, &fakeVariantRepo{}, store)

			err := svc.Delete(context.Background(), productID, tt.variantID, imageID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
				}
			}

			if len(store.deletedKeys) != tt.wantDeletes {
				t.Fatalf("Delete() storage deletes = %d, want %d", len(store.deletedKeys), tt.wantDeletes)
			}

			if tt.wantRepoCall && repo.deleteID != imageID {
				t.Fatalf("Delete() repo id = %v, want %v", repo.deleteID, imageID)
			}

			if !tt.wantRepoCall && repo.deleteID != uuid.Nil {
				t.Fatal("Delete() must not reach repository on failure")
			}
		})
	}
}
