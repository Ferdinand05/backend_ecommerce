package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	userjwt "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeJWT struct {
	claims *userjwt.Claims
	err    error
}

func (f *fakeJWT) GenerateToken(userID uuid.UUID, email string, role string) (string, error) {
	return "", nil
}

func (f *fakeJWT) ValidateToken(tokenString string) (*userjwt.Claims, error) {
	return f.claims, f.err
}

func runMiddleware(jwtSvc userjwt.JWTManager, authHeader string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	if authHeader != "" {
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", authHeader)
	} else {
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	}

	mw := AuthMiddleware(jwtSvc)
	mw(c)

	return w, c
}

func TestAuthMiddleware(t *testing.T) {
	validClaims := &userjwt.Claims{
		Email: "user@example.com",
		Role:  "customer",
	}
	validClaims.Subject = "user-id-1"

	tests := []struct {
		name        string
		jwtSvc      *fakeJWT
		authHeader  string
		wantStatus  int
		wantNext    bool
		wantUserID  string
		wantEmail   string
		wantRole    string
	}{
		{
			name:       "missing header",
			jwtSvc:     &fakeJWT{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed header",
			jwtSvc:     &fakeJWT{},
			authHeader: "BearerToken",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong scheme",
			jwtSvc:     &fakeJWT{},
			authHeader: "Basic abc",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty token",
			jwtSvc:     &fakeJWT{},
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			jwtSvc:     &fakeJWT{err: errors.New("token invalid")},
			authHeader: "Bearer some-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token",
			jwtSvc:     &fakeJWT{claims: validClaims},
			authHeader: "Bearer valid-token",
			wantStatus: http.StatusOK,
			wantNext:   true,
			wantUserID: "user-id-1",
			wantEmail:  "user@example.com",
			wantRole:   "customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, c := runMiddleware(tt.jwtSvc, tt.authHeader)

			if tt.wantStatus == http.StatusUnauthorized {
				if w.Code != http.StatusUnauthorized {
					t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
				}

				if c.IsAborted() == tt.wantNext {
					t.Fatal("request was not aborted for unauthorized case")
				}

				return
			}

			if got := c.GetString("userID"); got != tt.wantUserID {
				t.Fatalf("userID = %q, want %q", got, tt.wantUserID)
			}

			if got := c.GetString("email"); got != tt.wantEmail {
				t.Fatalf("email = %q, want %q", got, tt.wantEmail)
			}

			if got := c.GetString("role"); got != tt.wantRole {
				t.Fatalf("role = %q, want %q", got, tt.wantRole)
			}
		})
	}
}
