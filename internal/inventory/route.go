package inventory

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	inventory := rg.Group("/products/:product_id/variants/:variant_id/inventory")

	{
		inventory.GET("", handler.GetInventory)

		inventory.GET("/movements", handler.ListMovements)
		inventory.POST("/movements", handler.CreateMovement)
		inventory.GET("/movements/:movement_id", handler.GetMovement)
	}
}
