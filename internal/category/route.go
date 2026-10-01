package category

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	category := rg.Group("/categories")

	{
		category.GET("", handler.FindAll)
		category.GET("/:id", handler.FindByID)
		category.GET("/slug/:slug", handler.FindBySlug)

		category.POST("", handler.Create)
		category.PUT("/:id", handler.Update)
		category.DELETE("/:id", handler.Delete)
	}

}
