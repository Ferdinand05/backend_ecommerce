package category

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	findAllRes  []models.Category
	findAllErr  error
	findByIDRes models.Category
	findByIDErr error

	findBySlugRes models.Category
	findBySlugErr error

	createRes models.Category
	createErr error
	createArg models.Category

	updateRes models.Category
	updateErr error
	updateID  uuid.UUID
	updateArg models.Category

	deleteErr    error
	deleteCalled bool
	deleteID     uuid.UUID
}

func (f *fakeRepo) FindAll(ctx context.Context) ([]models.Category, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeRepo) FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeRepo) FindBySlug(ctx context.Context, slug string) (models.Category, error) {
	return f.findBySlugRes, f.findBySlugErr
}

func (f *fakeRepo) Create(ctx context.Context, category models.Category) (models.Category, error) {
	f.createArg = category
	if f.createErr != nil {
		return models.Category{}, f.createErr
	}
	return f.createRes, nil
}

func (f *fakeRepo) Update(ctx context.Context, ID uuid.UUID, category models.Category) (models.Category, error) {
	f.updateID = ID
	f.updateArg = category
	if f.updateErr != nil {
		return models.Category{}, f.updateErr
	}
	return f.updateRes, nil
}

func (f *fakeRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	f.deleteCalled = true
	f.deleteID = ID
	return f.deleteErr
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
			repo:    &fakeRepo{findAllRes: []models.Category{{Name: "electronics"}, {Name: "fashion"}}},
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
			svc := NewService(tt.repo)

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
			repo: &fakeRepo{findByIDRes: models.Category{ID: id, Name: "electronics"}},
			want: "electronics",
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorCategoryNotFound},
			wantErr: ErrorCategoryNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

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
			repo: &fakeRepo{findBySlugRes: models.Category{Name: "electronics", Slug: "electronics"}},
			want: "electronics",
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findBySlugErr: ErrorCategoryNotFound},
			wantErr: ErrorCategoryNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

			got, err := svc.FindBySlug(context.Background(), "electronics")
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

	tests := []struct {
		name     string
		req      CreateCategoryRequest
		repo     *fakeRepo
		wantSlug string
		wantErr  error
		wantDesc *string
	}{
		{
			name:     "success nil description",
			req:      CreateCategoryRequest{Name: "Home & Garden"},
			repo:     &fakeRepo{createRes: models.Category{Name: "Home & Garden", Slug: "home-and-garden"}},
			wantSlug: "home-and-garden",
		},
		{
			name:     "success with description",
			req:      CreateCategoryRequest{Name: "Electronics", Description: strPtr("gadgets")},
			repo:     &fakeRepo{createRes: models.Category{Name: "Electronics", Slug: "electronics"}},
			wantSlug: "electronics",
			wantDesc: strPtr("gadgets"),
		},
		{
			name:    "repo error",
			req:     CreateCategoryRequest{Name: "Books"},
			repo:    &fakeRepo{createErr: createErr},
			wantErr: createErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

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
	updateErr := errors.New("update failed")

	tests := []struct {
		name    string
		repo    *fakeRepo
		want    string
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{updateRes: models.Category{ID: id, Name: "electronics-updated", Slug: "electronics-updated"}},
			want: "electronics-updated",
		},
		{
			name:    "repo error",
			repo:    &fakeRepo{updateErr: updateErr},
			wantErr: updateErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

			got, err := svc.Update(context.Background(), id, UpdateCategoryRequest{Name: "Electronics Updated"})
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

			if tt.repo.updateArg.Slug != "electronics-updated" {
				t.Fatalf("Update() slug = %q, want %q", tt.repo.updateArg.Slug, "electronics-updated")
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
			name:       "repo error",
			repo:       &fakeRepo{deleteErr: deleteErr},
			wantErr:    deleteErr,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

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
