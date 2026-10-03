package router

import (
	"ferdinand/ecommerce/internal/auth"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/product"
	"ferdinand/ecommerce/internal/product_image"
	"ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/user"
	userJWT "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
)

type RouteHandlers struct {
	AuthHandler           *auth.Handler
	UserHandler           *user.Handler
	RoleHandler           *role.Handler
	CategoryHandler       *category.Handler
	ProductHandler        *product.Handler
	ProductVariantHandler *productvariant.Handler
	ProductImageHandler   *productimage.Handler
}

func New(handlers RouteHandlers, jwtService *userJWT.JWTService) *gin.Engine {

	r := gin.Default()
	v1 := r.Group("/api/v1")

	// public
	auth.RegisterRoutes(v1, handlers.AuthHandler)
	user.RegisterRoutes(v1, handlers.UserHandler)
	role.RegisterRoutes(v1, handlers.RoleHandler)
	category.RegisterRoutes(v1, handlers.CategoryHandler)
	product.RegisterRoutes(v1, handlers.ProductHandler)
	productvariant.RegisterRoutes(v1, handlers.ProductVariantHandler)
	productimage.RegisterRoutes(v1, handlers.ProductImageHandler)
	return r
}
