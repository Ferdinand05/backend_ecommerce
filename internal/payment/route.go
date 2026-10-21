package payment

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang dua jenis endpoint:
//   - webhook di public group (pemanggilnya Xendit, bukan user ber-JWT);
//   - create payment di group `me` (dilindungi AuthMiddleware di router).
func RegisterRoutes(v1 *gin.RouterGroup, me *gin.RouterGroup, handler *Handler) {
	v1.POST("/webhooks/xendit", handler.HandleInvoiceCallback)
	me.POST("/orders/:order_id/payment", handler.CreatePayment)
}
