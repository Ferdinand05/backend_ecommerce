package cartitems

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes memasang endpoint ke group `/cart` yang sudah dibungkus
// middleware auth di router. Package ini tidak menyentuh middleware.
func RegisterRoutes(cart *gin.RouterGroup, handler *Handler) {

	cart.GET("", handler.GetCart)
	cart.POST("/items", handler.AddItem)
	cart.POST("/items/:item_id/increase", handler.IncreaseItem)
	cart.POST("/items/:item_id/decrease", handler.DecreaseItem)
	cart.DELETE("/items/:item_id", handler.RemoveItem)
}
