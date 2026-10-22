package payment

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// currentUserID mengambil userID yang dipasang AuthMiddleware.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {

	raw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return uuid.Nil, false
	}

	value, ok := raw.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(value)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return uuid.Nil, false
	}

	return userID, true
}

// POST /me/orders/:order_id/payment
// Tanpa request body: nominal dan kelayakan diambil dari database,
// bukan dari client. Semantik get-or-create: attempt PENDING yang
// masih hidup dikembalikan apa adanya.
func (h *Handler) CreatePayment(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid order id",
		})
		return
	}

	// Longgar: mencakup pembuatan invoice di API Xendit.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	result, err := h.service.CreatePayment(ctx, userID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "order not found",
			})

		case errors.Is(err, ErrOrderNotPayable):
			slog.Warn("payment.create_not_payable", "order_id", orderID, "user_id", userID)
			c.JSON(http.StatusConflict, gin.H{
				"error": "order is not payable",
			})

		case errors.Is(err, ErrOrderPaymentExpired):
			c.JSON(http.StatusGone, gin.H{
				"error": "order payment period has expired",
			})

		case errors.Is(err, ErrPaymentStateAmbiguous):
			// Status tak bisa diverifikasi; aman untuk dicoba lagi.
			slog.Error("payment.create_state_ambiguous", "order_id", orderID, "user_id", userID, "error", err)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "payment state could not be verified, please retry",
			})

		default:
			slog.Error("payment.create_failed", "order_id", orderID, "user_id", userID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

// parsePagination: pola sama dengan package lain
// (default 1/10, page_size dibatasi 100).
func parsePagination(c *gin.Context) (int, int, bool) {

	page := defaultPage
	pageSize := defaultPageSize

	if raw := c.Query("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid page",
			})
			return 0, 0, false
		}
		page = value
	}

	if raw := c.Query("page_size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid page_size",
			})
			return 0, 0, false
		}
		if value > maxPageSize {
			value = maxPageSize
		}
		pageSize = value
	}

	return page, pageSize, true
}

// GET /admin/payments?status=&page=&page_size=
func (h *Handler) ListPayments(c *gin.Context) {

	page, pageSize, ok := parsePagination(c)
	if !ok {
		return
	}

	var status *PaymentStatus
	if raw := c.Query("status"); raw != "" {
		s := PaymentStatus(raw)
		switch s {
		case PaymentStatusPending,
			PaymentStatusPaid,
			PaymentStatusExpired,
			PaymentStatusFailed:
			status = &s
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid status filter",
			})
			return
		}
	}

	// Longgar tidak perlu: murni query database lokal.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.service.ListPayments(ctx, status, page, pageSize)
	if err != nil {
		slog.Error("payment.list_failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
