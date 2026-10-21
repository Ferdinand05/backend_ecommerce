package payment

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service      Service
	webhookToken string
}

func NewHandler(service Service, webhookToken string) *Handler {
	return &Handler{
		service:      service,
		webhookToken: webhookToken,
	}
}

// POST /webhooks/xendit
// Dipanggil Xendit, bukan user: auth pakai callback token, bukan JWT.
func (h *Handler) HandleInvoiceCallback(c *gin.Context) {

	token := c.GetHeader("x-callback-token")
	if subtle.ConstantTimeCompare(
		[]byte(token),
		[]byte(h.webhookToken),
	) != 1 {
		slog.Warn("webhook.invalid_token", "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid callback token",
		})
		return
	}

	var cb InvoiceCallback
	if err := c.ShouldBindJSON(&cb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.HandleInvoiceCallback(ctx, cb); err != nil {
		switch {
		case errors.Is(err, ErrInvalidPayment), errors.Is(err, ErrAmountMismatch):
			slog.Error("webhook.rejected",
				"invoice_id", cb.ID, "external_id", cb.ExternalID, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			// 500: Xendit akan retry; transaksi sudah rollback.
			slog.Error("webhook.processing_failed",
				"invoice_id", cb.ID, "external_id", cb.ExternalID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
