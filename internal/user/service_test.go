package user

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	findAllResult  []models.User
	findAllErr     error
	findByIDResult models.User
	findByIDErr    error
	findByEmailRes models.User
	findByEmailErr error
	createResult   models.User
	createErr      error
	createCalled   bool
}

func (f *fakeRepo) FindAll(ctx context.Context) ([]models.User, error) {
	return f.findAllResult, f.findAllErr
}

func (f *fakeRepo) FindByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	return f.findByIDResult, f.findByIDErr
}

func (f *fakeRepo) FindByEmail(ctx context.Context, email string) (models.User, error) {
	return f.findByEmailRes, f.findByEmailErr
}

func (f *fakeRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	f.createCalled = true
	if f.createErr != nil {
		return models.User{}, f.createErr
	}
	return f.createResult, nil
}

func (f *fakeRepo) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func TestServiceFindAll(t *testing.T) {
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repo    *fakeRepo
		want    []models.User
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findAllResult: []models.User{{Email: "a@example.com"}}},
			want: []models.User{{Email: "a@example.com"}},
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

			if len(got) != 1 || got[0].Email != tt.want[0].Email {
				t.Fatalf("FindAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceFindByID(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name    string
		repo    *fakeRepo
		want    models.User
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findByIDResult: models.User{ID: id, Email: "a@example.com"}},
			want: models.User{ID: id, Email: "a@example.com"},
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorUserNotFound},
			wantErr: ErrorUserNotFound,
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

			if got.Email != tt.want.Email {
				t.Fatalf("FindByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceFindByEmail(t *testing.T) {
	tests := []struct {
		name    string
		repo    *fakeRepo
		want    string
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findByEmailRes: models.User{Email: "a@example.com"}},
			want: "a@example.com",
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByEmailErr: ErrorUserNotFound},
			wantErr: ErrorUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

			got, err := svc.FindByEmail(context.Background(), "a@example.com")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindByEmail() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.Email != tt.want {
				t.Fatalf("FindByEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceCreate(t *testing.T) {
	u := models.User{ID: uuid.New(), Email: "a@example.com"}
	createErr := errors.New("create failed")

	tests := []struct {
		name       string
		repo       *fakeRepo
		wantErr    error
		wantCalled bool
	}{
		{
			name:       "success",
			repo:       &fakeRepo{createResult: u},
			wantCalled: true,
		},
		{
			name:       "repo error",
			repo:       &fakeRepo{createErr: createErr},
			wantErr:    createErr,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

			got, err := svc.Create(context.Background(), u)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if !tt.repo.createCalled {
				t.Fatal("Create() did not call repository")
			}

			if got.ID != u.ID {
				t.Fatalf("Create() = %v, want %v", got, u)
			}
		})
	}
}
