package payment

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang endpoint webhook di public group:
// pemanggilnya Xendit, bukan user ber-JWT.
func RegisterRoutes(v1 *gin.RouterGroup, handler *Handler) {
	v1.POST("/webhooks/xendit", handler.HandleInvoiceCallback)
}
