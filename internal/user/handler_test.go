package user

import (
	"context"
	"encoding/json"
	"ferdinand/ecommerce/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeService struct {
	findAllResult  []models.User
	findAllErr     error
	findByIDResult models.User
	findByIDErr    error
	findByEmailRes models.User
	findByEmailErr error
	createResult   models.User
	createErr      error

	updateProfileResult models.User
	updateProfileErr    error
	updateStatusResult  models.User
	updateStatusErr     error
	updateRoleResult    models.User
	updateRoleErr       error
}

func (f *fakeService) FindAll(ctx context.Context) ([]models.User, error) {
	return f.findAllResult, f.findAllErr
}

func (f *fakeService) FindByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	return f.findByIDResult, f.findByIDErr
}

func (f *fakeService) FindByEmail(ctx context.Context, email string) (models.User, error) {
	return f.findByEmailRes, f.findByEmailErr
}

func (f *fakeService) Create(ctx context.Context, u models.User) (models.User, error) {
	return f.createResult, f.createErr
}

func (f *fakeService) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (models.User, error) {
	return f.updateProfileResult, f.updateProfileErr
}

func (f *fakeService) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) (models.User, error) {
	return f.updateStatusResult, f.updateStatusErr
}

func (f *fakeService) UpdateRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) (models.User, error) {
	return f.updateRoleResult, f.updateRoleErr
}

func newTestRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	RegisterRoutes(r.Group("/admin"), h)

	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	return rec
}

func TestHandlerUpdateStatusOK(t *testing.T) {
	userID := uuid.New()

	svc := &fakeService{
		updateStatusResult: models.User{
			ID:        userID,
			Email:     "a@example.com",
			FirstName: "Test",
			Status:    "blocked",
		},
	}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/status",
		`{"status":"blocked"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if resp.Status != "blocked" {
		t.Fatalf("Status = %q, want %q", resp.Status, "blocked")
	}
}

func TestHandlerUpdateStatusInvalidStatus(t *testing.T) {
	userID := uuid.New()

	r := newTestRouter(NewHandler(&fakeService{}))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/status",
		`{"status":"inactive"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandlerUpdateStatusInvalidUserID(t *testing.T) {
	r := newTestRouter(NewHandler(&fakeService{}))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/not-a-uuid/status",
		`{"status":"active"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandlerUpdateStatusUserNotFound(t *testing.T) {
	userID := uuid.New()

	svc := &fakeService{updateStatusErr: ErrorUserNotFound}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/status",
		`{"status":"active"}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestHandlerUpdateRoleOK(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	svc := &fakeService{
		updateRoleResult: models.User{
			ID:        userID,
			Email:     "a@example.com",
			FirstName: "Test",
			Status:    "active",
			Role:      models.Role{ID: roleID, Name: "admin"},
		},
	}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/role",
		`{"role_id":"`+roleID.String()+`"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if resp.Role == nil || resp.Role.ID != roleID {
		t.Fatalf("Role = %v, want role %v", resp.Role, roleID)
	}
}

func TestHandlerUpdateRoleRoleNotFound(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	svc := &fakeService{updateRoleErr: ErrorRoleNotFound}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/role",
		`{"role_id":"`+roleID.String()+`"}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestHandlerUpdateRoleInvalidBody(t *testing.T) {
	userID := uuid.New()

	r := newTestRouter(NewHandler(&fakeService{}))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/role",
		`{"role_id":"bukan-uuid"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandlerUpdateRoleUserNotFound(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	svc := &fakeService{updateRoleErr: ErrorUserNotFound}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/role",
		`{"role_id":"`+roleID.String()+`"}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestHandlerUpdateProfileOK(t *testing.T) {
	userID := uuid.New()

	svc := &fakeService{
		updateProfileResult: models.User{
			ID:        userID,
			Email:     "a@example.com",
			FirstName: "Baru",
			Status:    "active",
		},
	}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/profile",
		`{"first_name":"Baru"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if resp.FirstName != "Baru" {
		t.Fatalf("FirstName = %q, want %q", resp.FirstName, "Baru")
	}
}

func TestHandlerUpdateProfileInvalidBody(t *testing.T) {
	userID := uuid.New()

	r := newTestRouter(NewHandler(&fakeService{}))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/profile",
		`{"last_name":"TanpaNamaDepan"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandlerUpdateProfileUserNotFound(t *testing.T) {
	userID := uuid.New()

	svc := &fakeService{updateProfileErr: ErrorUserNotFound}

	r := newTestRouter(NewHandler(svc))

	rec := doRequest(t, r, http.MethodPatch,
		"/admin/users/"+userID.String()+"/profile",
		`{"first_name":"Baru"}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
