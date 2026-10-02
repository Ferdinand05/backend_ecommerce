package product

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	product := rg.Group("/products")

	{
		product.GET("", handler.FindAll)
		product.GET("/:product_id", handler.FindByID)
		product.GET("/slug/:slug", handler.FindBySlug)

		product.POST("", handler.Create)
		product.PUT("/:product_id", handler.Update)
		product.DELETE("/:product_id", handler.Delete)
	}

}
