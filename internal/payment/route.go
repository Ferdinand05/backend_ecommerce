package payment

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang tiga jenis endpoint:
//   - webhook di public group (pemanggilnya Xendit, bukan user ber-JWT);
//   - create payment di group `me` (dilindungi AuthMiddleware di router);
//   - list payments di group `admin` (dilindungi Auth + RequireRole).
func RegisterRoutes(v1 *gin.RouterGroup, me *gin.RouterGroup, admin *gin.RouterGroup, handler *Handler) {
	v1.POST("/webhooks/xendit", handler.HandleInvoiceCallback)
	me.POST("/orders/:order_id/payment", handler.CreatePayment)
	admin.GET("/payments", handler.ListPayments)
}
