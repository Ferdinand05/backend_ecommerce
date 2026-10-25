package router

import (
	"ferdinand/ecommerce/internal/auth"
	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/checkout"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/middleware"
	"ferdinand/ecommerce/internal/order"
	"ferdinand/ecommerce/internal/payment"
	"ferdinand/ecommerce/internal/product"
	productimage "ferdinand/ecommerce/internal/product_image"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/shipment"
	"ferdinand/ecommerce/internal/shipping"
	"ferdinand/ecommerce/internal/user"
	userJWT "ferdinand/ecommerce/utils/jwt"
	"time"

	"github.com/gin-contrib/cors"
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
	ShippingHandler       *shipping.Handler
	CheckoutHandler       *checkout.Handler
	PaymentHandler        *payment.Handler
	ShipmentHandler       *shipment.Handler
}

func New(handlers RouteHandlers, jwtService *userJWT.JWTService) *gin.Engine {

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestLogger())

	// Konfigurasi CORS
    r.Use(cors.New(cors.Config{
        // HANYA daftarkan origin secara eksplisit (hapus wildcard "*")
        AllowOrigins:     []string{"http://localhost:5173"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

	v1 := r.Group("/api/v1")

	// ============================================================
	// PUBLIC ENDPOINTS (no authentication required)
	// ============================================================
	// Auth: login, register — accessible by anyone
	auth.RegisterRoutes(v1, handlers.AuthHandler)

	// Categories: read-only list/detail — product catalog browsing
	category.RegisterPublicRoutes(v1, handlers.CategoryHandler)

	// Products: read-only list/detail — product catalog browsing
	product.RegisterPublicRoutes(v1, handlers.ProductHandler)

	// Product variants: read-only list/detail — nested under products
	productvariant.RegisterPublicRoutes(v1, handlers.ProductVariantHandler)

	// Product images: read-only list/detail — nested under products/variants
	productimage.RegisterPublicRoutes(v1, handlers.ProductImageHandler)

	// ============================================================
	// CUSTOMER ENDPOINTS (authentication required, no role check)
	// ============================================================
	// Shipping: calculate shipping rates — customer uses for checkout
	shippingRG := v1.Group("/shipping", middleware.AuthMiddleware(jwtService))
	shipping.RegisterRoutes(shippingRG, handlers.ShippingHandler)

	// Cart: manage shopping cart items — customer only
	cartRG := v1.Group("/cart", middleware.AuthMiddleware(jwtService))
	cartitems.RegisterRoutes(cartRG, handlers.CartHandler)

	// Checkout: preview order before creation — customer only
	checkoutRG := v1.Group("/checkout", middleware.AuthMiddleware(jwtService))
	checkout.RegisterRoutes(checkoutRG, handlers.CheckoutHandler)

	// Customer orders: /me/orders — create, list, detail own orders
	// Admin orders: /admin/orders — list all, detail, update status, histories
	me := v1.Group("/me", middleware.AuthMiddleware(jwtService))
	admin := v1.Group("/admin", middleware.AuthMiddleware(jwtService), middleware.RequireRole("admin"))
	order.RegisterRoutes(me, admin, handlers.OrderHandler)

	// Customer payments: create payment for own order
	// Admin payments: list all payments
	// Webhook: /api/v1/webhooks/xendit (public, token-based auth in handler)
	payment.RegisterRoutes(v1, me, admin, handlers.PaymentHandler)

	// Customer shipments: track own shipments via /me/shipments
	// Admin shipments: list/detail/tracking/retry all shipments
	shipment.RegisterRoutes(me, admin, handlers.ShipmentHandler)

	// ============================================================
	// ADMIN ENDPOINTS (authentication + role="admin" required)
	// ============================================================
	// Users: CRUD all users — admin only (public registration via /auth/register)
	user.RegisterRoutes(admin, handlers.UserHandler)

	// Roles: CRUD roles — admin only (roles typically seeded, endpoint optional)
	role.RegisterRoutes(admin, handlers.RoleHandler)

	// Categories: create, update, delete — admin only
	category.RegisterAdminRoutes(admin, handlers.CategoryHandler)

	// Products: create, update, delete — admin only
	product.RegisterAdminRoutes(admin, handlers.ProductHandler)

	// Product variants: create, update, delete — admin only
	productvariant.RegisterAdminRoutes(admin, handlers.ProductVariantHandler)

	// Product images: upload, delete — admin only
	productimage.RegisterAdminRoutes(admin, handlers.ProductImageHandler)

	// Inventory: manage stock movements — admin only
	inventory.RegisterRoutes(admin, handlers.InventoryHandler)

	// Order management: list/update all orders, view histories
	// Already registered above via order.RegisterRoutes(me, admin, ...)

	// Shipment management: list/detail/tracking/retry all shipments
	// Already registered above via shipment.RegisterRoutes(me, admin, ...)

	return r
}
