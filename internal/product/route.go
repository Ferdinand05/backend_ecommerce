package product

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes memasang endpoint product read-only untuk publik
func RegisterPublicRoutes(rg *gin.RouterGroup, handler *Handler) {
	product := rg.Group("/products")
	{
		product.GET("", handler.FindAll)
		product.GET("/:product_id", handler.FindByID)
		product.GET("/slug/:slug", handler.FindBySlug)
	}
}

// RegisterAdminRoutes memasang endpoint product write (create, update, delete) untuk admin
func RegisterAdminRoutes(admin *gin.RouterGroup, handler *Handler) {
	product := admin.Group("/products")
	{
		product.POST("", handler.Create)
		product.PUT("/:product_id", handler.Update)
		product.DELETE("/:product_id", handler.Delete)
	}
}

// RegisterRoutes deprecated — use RegisterPublicRoutes + RegisterAdminRoutes instead
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	RegisterPublicRoutes(rg, handler)
}
