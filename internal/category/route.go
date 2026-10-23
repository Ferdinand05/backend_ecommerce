package category

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes memasang endpoint category read-only untuk publik
func RegisterPublicRoutes(rg *gin.RouterGroup, handler *Handler) {
	category := rg.Group("/categories")
	{
		category.GET("", handler.FindAll)
		category.GET("/:id", handler.FindByID)
		category.GET("/slug/:slug", handler.FindBySlug)
	}
}

// RegisterAdminRoutes memasang endpoint category write (create, update, delete) untuk admin
func RegisterAdminRoutes(admin *gin.RouterGroup, handler *Handler) {
	category := admin.Group("/categories")
	{
		category.POST("", handler.Create)
		category.PUT("/:id", handler.Update)
		category.DELETE("/:id", handler.Delete)
	}
}

// RegisterRoutes deprecated — use RegisterPublicRoutes + RegisterAdminRoutes instead
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	RegisterPublicRoutes(rg, handler)
}
