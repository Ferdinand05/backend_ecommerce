package cartitems

import (
	"ferdinand/ecommerce/internal/middleware"
	userjwt "ferdinand/ecommerce/utils/jwt"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler, jwtService *userjwt.JWTService) {
	cart := rg.Group("/cart", middleware.AuthMiddleware(jwtService))

	{
		cart.GET("", handler.GetCart)
		cart.POST("/items", handler.AddItem)
		cart.POST("/items/:item_id/increase", handler.IncreaseItem)
		cart.POST("/items/:item_id/decrease", handler.DecreaseItem)
		cart.DELETE("/items/:item_id", handler.RemoveItem)
	}
}
