package inventory

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang endpoint inventory pada group admin.
// Stock movements adalah sensitive data admin-only; quantity/reserved
// sudah embed di product variant response via preload untuk publik.
func RegisterRoutes(admin *gin.RouterGroup, handler *Handler) {
	inventory := admin.Group("/products/:product_id/variants/:variant_id/inventory")
	{
		inventory.GET("", handler.GetInventory)
		inventory.GET("/movements", handler.ListMovements)
		inventory.POST("/movements", handler.CreateMovement)
		inventory.GET("/movements/:movement_id", handler.GetMovement)
	}

	admin.GET("/stock-movements", handler.ListAllMovements)
}
