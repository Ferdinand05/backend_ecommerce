package router

import (
	"ferdinand/ecommerce/internal/auth"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/user"
	userJWT "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
)

type RouteHandlers struct {
	AuthHandler *auth.Handler
	UserHandler *user.Handler
	RoleHandler *role.Handler
}

func New(handlers RouteHandlers, jwtService *userJWT.JWTService) *gin.Engine {

	r := gin.Default()
	v1 := r.Group("/api/v1")

	// public
	auth.RegisterRoutes(v1, handlers.AuthHandler)
	user.RegisterRoutes(v1, handlers.UserHandler)
	role.RegisterRoutes(v1, handlers.RoleHandler)

	return r
}
