package productimage

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes mounts read-only product/variant image endpoints.
func RegisterPublicRoutes(rg *gin.RouterGroup, handler *Handler) {
	products := rg.Group("/products")

	productImages := products.Group("/:product_id/images")
	productImages.GET("", handler.FindAllByProductID)
	productImages.GET("/:image_id", handler.FindProductImageByID)

	variantImages := products.Group("/:product_id/variants/:variant_id/images")
	variantImages.GET("", handler.FindAllByVariantID)
	variantImages.GET("/:image_id", handler.FindVariantImageByID)
}

// RegisterAdminRoutes mounts image upload and delete endpoints for admins.
func RegisterAdminRoutes(admin *gin.RouterGroup, handler *Handler) {
	products := admin.Group("/products")

	productImages := products.Group("/:product_id/images")
	productImages.POST("", handler.UploadProductImage)
	productImages.DELETE("/:image_id", handler.DeleteProductImage)

	variantImages := products.Group("/:product_id/variants/:variant_id/images")
	variantImages.POST("", handler.UploadVariantImage)
	variantImages.DELETE("/:image_id", handler.DeleteVariantImage)
}

// RegisterRoutes is deprecated. Use RegisterPublicRoutes and RegisterAdminRoutes.
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	RegisterPublicRoutes(rg, handler)
}
