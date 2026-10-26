package order

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes memasang endpoint customer di group `me` dan admin di group
// `admin`. Autentikasi/otorisasi (AuthMiddleware, RequireRole) dipasang di
// router pada kedua group tersebut.
func RegisterRoutes(me *gin.RouterGroup, admin *gin.RouterGroup, handler *Handler) {

	me.POST("/orders", handler.CreateOrder)
	me.GET("/orders", handler.ListMyOrders)
	me.GET("/orders/:order_id", handler.GetMyOrder)
	me.POST("/orders/:order_id/cancel", handler.CancelOrder)

	admin.GET("/orders", handler.ListOrders)
	admin.GET("/orders/:order_id", handler.GetOrder)
	admin.GET("/orders/:order_id/histories", handler.ListStatusHistories)
	admin.PATCH("/orders/:order_id/status", handler.UpdateStatus)
}
