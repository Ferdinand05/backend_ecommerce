package router

import (
	"testing"

	userjwt "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
)

func TestRouteRegistrationNoConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("router.New panicked while registering routes: %v", rec)
		}
	}()

	New(RouteHandlers{}, userjwt.NewJWTService("test-secret"))
}
