package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		required   string
		userRole   string
		wantStatus int
	}{
		{name: "matching role", required: "admin", userRole: "admin", wantStatus: http.StatusOK},
		{name: "different role", required: "admin", userRole: "customer", wantStatus: http.StatusForbidden},
		{name: "missing role", required: "admin", userRole: "", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tt.userRole != "" {
					c.Set("role", tt.userRole)
				}
				c.Next()
			})
			r.Use(RequireRole(tt.required))
			r.GET("/", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
