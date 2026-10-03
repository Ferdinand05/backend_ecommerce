package productimage

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	products := rg.Group("/products")

	productImages := products.Group("/:product_id/images")
	{
		productImages.POST("", handler.UploadProductImage)
		productImages.GET("", handler.FindAllByProductID)
		productImages.GET("/:image_id", handler.FindProductImageByID)
		productImages.DELETE("/:image_id", handler.DeleteProductImage)
	}

	variantImages := products.Group("/:product_id/variants/:variant_id/images")
	{
		variantImages.POST("", handler.UploadVariantImage)
		variantImages.GET("", handler.FindAllByVariantID)
		variantImages.GET("/:image_id", handler.FindVariantImageByID)
		variantImages.DELETE("/:image_id", handler.DeleteVariantImage)
	}

}
