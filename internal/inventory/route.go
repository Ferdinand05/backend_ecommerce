package inventory

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang endpoint inventory:
//   - detail & movement per variant di rg (dipasang router di v1);
//   - list global semua movement di admin (dilindungi AuthMiddleware
//   - RequireRole("admin") di router).
func RegisterRoutes(rg *gin.RouterGroup, admin *gin.RouterGroup, handler *Handler) {
	inventory := rg.Group("/products/:product_id/variants/:variant_id/inventory")

	{
		inventory.GET("", handler.GetInventory)

		inventory.GET("/movements", handler.ListMovements)
		inventory.POST("/movements", handler.CreateMovement)
		inventory.GET("/movements/:movement_id", handler.GetMovement)
	}

	admin.GET("/stock-movements", handler.ListAllMovements)
}
