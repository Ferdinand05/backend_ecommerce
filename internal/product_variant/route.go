package productvariant

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes memasang endpoint product variant read-only untuk publik
func RegisterPublicRoutes(rg *gin.RouterGroup, handler *Handler) {
	variants := rg.Group("/products/:product_id/variants")
	{
		variants.GET("", handler.FindAllByProductID)
		variants.GET("/:variant_id", handler.FindByID)
	}
}

// RegisterAdminRoutes memasang endpoint product variant write (create, update, delete) untuk admin
func RegisterAdminRoutes(admin *gin.RouterGroup, handler *Handler) {
	variants := admin.Group("/products/:product_id/variants")
	{
		variants.POST("", handler.Create)
		variants.PUT("/:variant_id", handler.Update)
		variants.DELETE("/:variant_id", handler.Delete)
	}
}

// RegisterRoutes deprecated — use RegisterPublicRoutes + RegisterAdminRoutes instead
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	RegisterPublicRoutes(rg, handler)
}
