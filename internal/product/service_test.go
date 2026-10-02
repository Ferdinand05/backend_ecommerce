package product

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	findAllRes  []models.Product
	findAllErr  error
	findByIDRes models.Product
	findByIDErr error

	findBySlugRes models.Product
	findBySlugErr error

	existsBySlug    bool
	existsBySlugErr error

	createRes models.Product
	createErr error
	createArg models.Product

	updateRes models.Product
	updateErr error
	updateID  uuid.UUID
	updateArg models.Product

	deleteErr    error
	deleteCalled bool
	deleteID     uuid.UUID
}

func (f *fakeRepo) FindAll(ctx context.Context) ([]models.Product, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeRepo) FindByID(ctx context.Context, ID uuid.UUID) (models.Product, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeRepo) FindBySlug(ctx context.Context, slug string) (models.Product, error) {
	return f.findBySlugRes, f.findBySlugErr
}

func (f *fakeRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return f.existsBySlug, f.existsBySlugErr
}

func (f *fakeRepo) Create(ctx context.Context, product models.Product) (models.Product, error) {
	f.createArg = product
	if f.createErr != nil {
		return models.Product{}, f.createErr
	}
	return f.createRes, nil
}

func (f *fakeRepo) Update(ctx context.Context, ID uuid.UUID, product models.Product) (models.Product, error) {
	f.updateID = ID
	f.updateArg = product
	if f.updateErr != nil {
		return models.Product{}, f.updateErr
	}
	return f.updateRes, nil
}

func (f *fakeRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	f.deleteCalled = true
	f.deleteID = ID
	return f.deleteErr
}

type fakeCategoryRepo struct {
	findByIDRes models.Category
	findByIDErr error
}

func (f *fakeCategoryRepo) FindAll(ctx context.Context) ([]models.Category, error) {
	return nil, nil
}

func (f *fakeCategoryRepo) FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeCategoryRepo) FindBySlug(ctx context.Context, slug string) (models.Category, error) {
	return models.Category{}, nil
}

func (f *fakeCategoryRepo) Create(ctx context.Context, category models.Category) (models.Category, error) {
	return models.Category{}, nil
}

func (f *fakeCategoryRepo) Update(ctx context.Context, ID uuid.UUID, category models.Category) (models.Category, error) {
	return models.Category{}, nil
}

func (f *fakeCategoryRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	return nil
}

func TestServiceFindAll(t *testing.T) {
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantLen int
		wantErr error
	}{
		{
			name:    "success",
			repo:    &fakeRepo{findAllRes: []models.Product{{Name: "laptop"}, {Name: "keyboard"}}},
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
			svc := NewService(tt.repo, &fakeCategoryRepo{})

			got, err := svc.FindAll(context.Background())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindAll() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if len(got) != tt.wantLen {
				t.Fatalf("FindAll() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestServiceFindByID(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name    string
		repo    *fakeRepo
		want    string
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findByIDRes: models.Product{ID: id, Name: "laptop", Slug: "laptop"}},
			want: "laptop",
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorProductNotFound},
			wantErr: ErrorProductNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeCategoryRepo{})

			got, err := svc.FindByID(context.Background(), id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.Name != tt.want {
				t.Fatalf("FindByID() = %v, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceFindBySlug(t *testing.T) {
	tests := []struct {
		name    string
		repo    *fakeRepo
		want    string
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findBySlugRes: models.Product{Name: "laptop", Slug: "laptop"}},
			want: "laptop",
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findBySlugErr: ErrorProductNotFound},
			wantErr: ErrorProductNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeCategoryRepo{})

			got, err := svc.FindBySlug(context.Background(), "laptop")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindBySlug() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.Name != tt.want {
				t.Fatalf("FindBySlug() = %v, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceCreate(t *testing.T) {
	createErr := errors.New("create failed")
	categoryID := uuid.New()

	tests := []struct {
		name     string
		req      CreateProductRequest
		repo     *fakeRepo
		category *fakeCategoryRepo
		wantSlug string
		wantErr  error
		wantDesc *string
	}{
		{
			name:     "success nil description",
			req:      CreateProductRequest{CategoryID: categoryID, Name: "Gaming Laptop"},
			repo:     &fakeRepo{createRes: models.Product{Name: "Gaming Laptop", Slug: "gaming-laptop"}},
			category: &fakeCategoryRepo{},
			wantSlug: "gaming-laptop",
		},
		{
			name:     "success with description",
			req:      CreateProductRequest{CategoryID: categoryID, Name: "Electronics", Description: strPtr("gadgets")},
			repo:     &fakeRepo{createRes: models.Product{Name: "Electronics", Slug: "electronics"}},
			category: &fakeCategoryRepo{},
			wantSlug: "electronics",
			wantDesc: strPtr("gadgets"),
		},
		{
			name:     "slug already exists",
			req:      CreateProductRequest{CategoryID: categoryID, Name: "Electronics"},
			repo:     &fakeRepo{existsBySlug: true},
			category: &fakeCategoryRepo{},
			wantErr:  ErrorProductSlugAlreadyExists,
		},
		{
			name:     "category not found",
			req:      CreateProductRequest{CategoryID: uuid.New(), Name: "Electronics"},
			repo:     &fakeRepo{},
			category: &fakeCategoryRepo{findByIDErr: category.ErrorCategoryNotFound},
			wantErr:  category.ErrorCategoryNotFound,
		},
		{
			name:     "repo error",
			req:      CreateProductRequest{CategoryID: categoryID, Name: "Books"},
			repo:     &fakeRepo{createErr: createErr},
			category: &fakeCategoryRepo{},
			wantErr:  createErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.category)

			got, err := svc.Create(context.Background(), tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if tt.repo.createArg.Slug != tt.wantSlug {
				t.Fatalf("Create() slug = %q, want %q", tt.repo.createArg.Slug, tt.wantSlug)
			}

			if tt.repo.createArg.ID == uuid.Nil {
				t.Fatal("Create() should assign a new ID")
			}

			if tt.repo.createArg.CategoryID != tt.req.CategoryID {
				t.Fatalf("Create() category id = %v, want %v", tt.repo.createArg.CategoryID, tt.req.CategoryID)
			}

			if (tt.wantDesc == nil) != (tt.repo.createArg.Description == nil) {
				t.Fatalf("Create() description = %v, want %v", tt.repo.createArg.Description, tt.wantDesc)
			}

			if got.Name != tt.req.Name {
				t.Fatalf("Create() = %v, want %q", got, tt.req.Name)
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	id := uuid.New()
	categoryID := uuid.New()
	updateErr := errors.New("update failed")

	tests := []struct {
		name     string
		req      UpdateProductRequest
		repo     *fakeRepo
		category *fakeCategoryRepo
		wantSlug string
		want     string
		wantErr  error
	}{
		{
			name: "success",
			req:  UpdateProductRequest{CategoryID: categoryID, Name: "Laptop Pro"},
			repo: &fakeRepo{
				findByIDRes: models.Product{ID: id, Name: "laptop", Slug: "laptop"},
				updateRes:   models.Product{ID: id, Name: "Laptop Pro", Slug: "laptop-pro"},
			},
			category: &fakeCategoryRepo{},
			wantSlug: "laptop-pro",
			want:     "Laptop Pro",
		},
		{
			name: "success keeps own slug without conflict check",
			req:  UpdateProductRequest{CategoryID: categoryID, Name: "Laptop"},
			repo: &fakeRepo{
				findByIDRes:  models.Product{ID: id, Name: "laptop", Slug: "laptop"},
				existsBySlug: true,
				updateRes:    models.Product{ID: id, Name: "Laptop", Slug: "laptop"},
			},
			category: &fakeCategoryRepo{},
			wantSlug: "laptop",
			want:     "Laptop",
		},
		{
			name: "slug conflict with other product",
			req:  UpdateProductRequest{CategoryID: categoryID, Name: "Laptop Pro"},
			repo: &fakeRepo{
				findByIDRes:  models.Product{ID: id, Name: "laptop", Slug: "laptop"},
				existsBySlug: true,
			},
			category: &fakeCategoryRepo{},
			wantErr:  ErrorProductSlugAlreadyExists,
		},
		{
			name:     "product not found",
			req:      UpdateProductRequest{CategoryID: categoryID, Name: "Laptop Pro"},
			repo:     &fakeRepo{findByIDErr: ErrorProductNotFound},
			category: &fakeCategoryRepo{},
			wantErr:  ErrorProductNotFound,
		},
		{
			name:     "category not found",
			req:      UpdateProductRequest{CategoryID: uuid.New(), Name: "Laptop Pro"},
			repo:     &fakeRepo{findByIDRes: models.Product{ID: id, Slug: "laptop"}},
			category: &fakeCategoryRepo{findByIDErr: category.ErrorCategoryNotFound},
			wantErr:  category.ErrorCategoryNotFound,
		},
		{
			name: "repo error",
			req:  UpdateProductRequest{CategoryID: categoryID, Name: "Laptop Pro"},
			repo: &fakeRepo{
				findByIDRes: models.Product{ID: id, Slug: "laptop"},
				updateErr:   updateErr,
			},
			category: &fakeCategoryRepo{},
			wantErr:  updateErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.category)

			got, err := svc.Update(context.Background(), id, tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if tt.repo.updateID != id {
				t.Fatalf("Update() repo id = %v, want %v", tt.repo.updateID, id)
			}

			if tt.repo.updateArg.ID != uuid.Nil {
				t.Fatalf("Update() must not assign a new ID, got %v", tt.repo.updateArg.ID)
			}

			if tt.repo.updateArg.Slug != tt.wantSlug {
				t.Fatalf("Update() slug = %q, want %q", tt.repo.updateArg.Slug, tt.wantSlug)
			}

			if tt.repo.updateArg.CategoryID != tt.req.CategoryID {
				t.Fatalf("Update() category id = %v, want %v", tt.repo.updateArg.CategoryID, tt.req.CategoryID)
			}

			if got.Name != tt.want {
				t.Fatalf("Update() = %v, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceDelete(t *testing.T) {
	id := uuid.New()
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name       string
		repo       *fakeRepo
		wantErr    error
		wantCalled bool
	}{
		{
			name:       "success",
			repo:       &fakeRepo{},
			wantCalled: true,
		},
		{
			name:       "not found",
			repo:       &fakeRepo{deleteErr: ErrorProductNotFound},
			wantErr:    ErrorProductNotFound,
			wantCalled: true,
		},
		{
			name:       "repo error",
			repo:       &fakeRepo{deleteErr: deleteErr},
			wantErr:    deleteErr,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeCategoryRepo{})

			err := svc.Delete(context.Background(), id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
				}
			}

			if !tt.repo.deleteCalled || tt.repo.deleteID != id {
				t.Fatal("Delete() did not call repository with correct id")
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
