package auth

import (
	"context"
	"encoding/json"
	"ferdinand/ecommerce/internal/user"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeAuthService struct {
	getMeResult    user.UserResponse
	getMeErr       error
	updateMeResult user.UserResponse
	updateMeErr    error
}

func (f *fakeAuthService) Register(ctx context.Context, req RegisterRequest) (user.UserResponse, error) {
	return user.UserResponse{}, nil
}

func (f *fakeAuthService) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func (f *fakeAuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	return nil
}

func (f *fakeAuthService) Refresh(ctx context.Context, rawRefreshToken string) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func (f *fakeAuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	return nil
}

func (f *fakeAuthService) ResendVerification(ctx context.Context, email string) error {
	return nil
}

func (f *fakeAuthService) ForgotPassword(ctx context.Context, email string) error {
	return nil
}

func (f *fakeAuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	return nil
}

func (f *fakeAuthService) GetMe(ctx context.Context, userID uuid.UUID) (user.UserResponse, error) {
	return f.getMeResult, f.getMeErr
}

func (f *fakeAuthService) UpdateMe(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (user.UserResponse, error) {
	return f.updateMeResult, f.updateMeErr
}

func newProfileRouter(h *Handler, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	// simulating AuthMiddleware: userID disimpan sebagai string uuid.
	r.Use(func(c *gin.Context) {
		if userID != uuid.Nil {
			c.Set("userID", userID.String())
		}
		c.Next()
	})

	RegisterProfileRoutes(r.Group("/auth"), h)

	return r
}

func doProfileRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	return rec
}

func TestHandlerGetMeOK(t *testing.T) {
	userID := uuid.New()

	svc := &fakeAuthService{
		getMeResult: user.UserResponse{
			ID:        userID,
			Email:     "a@example.com",
			FirstName: "Test",
			Status:    "active",
		},
	}

	r := newProfileRouter(NewHandler(svc), userID)

	rec := doProfileRequest(t, r, http.MethodGet, "/auth", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp user.UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if resp.ID != userID {
		t.Fatalf("ID = %v, want %v", resp.ID, userID)
	}
}

func TestHandlerGetMeUnauthorized(t *testing.T) {
	r := newProfileRouter(NewHandler(&fakeAuthService{}), uuid.Nil)

	rec := doProfileRequest(t, r, http.MethodGet, "/auth", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestHandlerUpdateMeOK(t *testing.T) {
	userID := uuid.New()

	svc := &fakeAuthService{
		updateMeResult: user.UserResponse{
			ID:        userID,
			Email:     "a@example.com",
			FirstName: "Baru",
			Status:    "active",
		},
	}

	r := newProfileRouter(NewHandler(svc), userID)

	rec := doProfileRequest(t, r, http.MethodPut, "/auth",
		`{"first_name":"Baru","phone":"0812"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp user.UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if resp.FirstName != "Baru" {
		t.Fatalf("FirstName = %q, want %q", resp.FirstName, "Baru")
	}
}

func TestHandlerUpdateMeInvalidBody(t *testing.T) {
	userID := uuid.New()

	r := newProfileRouter(NewHandler(&fakeAuthService{}), userID)

	rec := doProfileRequest(t, r, http.MethodPut, "/auth",
		`{"phone":"0812"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
