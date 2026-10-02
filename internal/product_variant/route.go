package productvariant

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	variants := rg.Group("/products/:product_id/variants")

	{
		variants.GET("", handler.FindAllByProductID)
		variants.GET("/:variant_id", handler.FindByID)

		variants.POST("", handler.Create)
		variants.PUT("/:variant_id", handler.Update)
		variants.DELETE("/:variant_id", handler.Delete)
	}

}
