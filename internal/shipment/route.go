package shipment

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes memasang endpoint customer di group `me` dan operasi
// pemulihan di group `admin`. Autentikasi/otorisasi dipasang di router.
func RegisterRoutes(me *gin.RouterGroup, admin *gin.RouterGroup, handler *Handler) {

	me.GET("/orders/:order_id/shipment", handler.GetMyOrderShipment)
	me.GET("/shipments/:shipment_id", handler.GetMyShipment)
	me.GET("/shipments/:shipment_id/tracking", handler.TrackMyShipment)

	admin.POST("/shipments/:shipment_id/retry", handler.RetryShipment)
}
