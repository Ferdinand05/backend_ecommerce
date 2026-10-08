package order

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes memasang endpoint customer di group `me` dan admin di group
// `admin`. Autentikasi/otorisasi (AuthMiddleware, RequireRole) dipasang di
// router pada kedua group tersebut.
func RegisterRoutes(me *gin.RouterGroup, admin *gin.RouterGroup, handler *Handler) {

	me.GET("/orders", handler.ListMyOrders)
	me.GET("/orders/:order_id", handler.GetMyOrder)

	admin.GET("/orders", handler.ListOrders)
	admin.GET("/orders/:order_id", handler.GetOrder)
	admin.PATCH("/orders/:order_id/status", handler.UpdateStatus)
}
