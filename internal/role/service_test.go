package role

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	findByIDRes  models.Role
	findByIDErr  error
	findAllRes   []models.Role
	findAllErr   error
	updateRes    models.Role
	updateErr    error
	deleteErr    error
	deleteCalled bool
	deleteRoleID uuid.UUID
}

func (f *fakeRepo) FindByName(ctx context.Context, name string) (models.Role, error) {
	return models.Role{}, nil
}

func (f *fakeRepo) FindByID(ctx context.Context, roleID uuid.UUID) (models.Role, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeRepo) FindAll(ctx context.Context) ([]models.Role, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeRepo) Update(ctx context.Context, role models.Role) (models.Role, error) {
	if f.updateErr != nil {
		return models.Role{}, f.updateErr
	}
	return f.updateRes, nil
}

func (f *fakeRepo) Delete(ctx context.Context, roleID uuid.UUID) error {
	f.deleteCalled = true
	f.deleteRoleID = roleID
	return f.deleteErr
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
			repo: &fakeRepo{findByIDRes: models.Role{ID: id, Name: "customer"}},
			want: "customer",
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorRoleNotFound},
			wantErr: ErrorRoleNotFound,
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

func TestServiceFindAll(t *testing.T) {
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantLen int
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findAllRes: []models.Role{{Name: "customer"}, {Name: "admin"}}},
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
			repo: &fakeRepo{updateRes: models.Role{ID: id, Name: "admin"}},
			want: "admin",
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

			got, err := svc.Update(context.Background(), models.Role{ID: id, Name: "admin"})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
				}
				return
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
				return
			}

			if !tt.repo.deleteCalled || tt.repo.deleteRoleID != id {
				t.Fatal("Delete() did not call repository with correct id")
			}
		})
	}
}
