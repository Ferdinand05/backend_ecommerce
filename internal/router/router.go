package router

import (
	"ferdinand/ecommerce/internal/auth"
	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/middleware"
	"ferdinand/ecommerce/internal/order"
	"ferdinand/ecommerce/internal/product"
	productimage "ferdinand/ecommerce/internal/product_image"
	productvariant "ferdinand/ecommerce/internal/product_variant"
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
	InventoryHandler      *inventory.Handler
	CartHandler           *cartitems.Handler
	OrderHandler          *order.Handler
}

func New(handlers RouteHandlers, jwtService *userJWT.JWTService) *gin.Engine {

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestLogger())
	v1 := r.Group("/api/v1")

	// public
	auth.RegisterRoutes(v1, handlers.AuthHandler)
	user.RegisterRoutes(v1, handlers.UserHandler)
	role.RegisterRoutes(v1, handlers.RoleHandler)
	category.RegisterRoutes(v1, handlers.CategoryHandler)
	product.RegisterRoutes(v1, handlers.ProductHandler)
	productvariant.RegisterRoutes(v1, handlers.ProductVariantHandler)
	productimage.RegisterRoutes(v1, handlers.ProductImageHandler)
	inventory.RegisterRoutes(v1, handlers.InventoryHandler)

	// authenticated
	cart := v1.Group("/cart", middleware.AuthMiddleware(jwtService))
	cartitems.RegisterRoutes(cart, handlers.CartHandler)
	me := v1.Group("/me", middleware.AuthMiddleware(jwtService))
	admin := v1.Group("/admin", middleware.AuthMiddleware(jwtService), middleware.RequireRole("admin")) // admin
	order.RegisterRoutes(me, admin, handlers.OrderHandler)

	return r
}
