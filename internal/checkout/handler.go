package checkout

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}

	// Middleware memasang claims.Subject berupa string.
	value, ok := raw.(string)
	if !ok {
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, false
	}

	return userID, true
}

// POST /checkout/preview
func (h *Handler) Preview(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	preview, err := h.service.Preview(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCartEmpty):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "cart is empty",
			})

		case errors.Is(err, ErrVariantNotAvailable):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "one or more items are no longer available",
			})

		case errors.Is(err, ErrInsufficientStock):
			c.JSON(http.StatusConflict, gin.H{
				"error": "insufficient stock for one or more items",
			})

		default:
			slog.ErrorContext(ctx, "failed to build checkout preview",
				"user_id", userID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusOK, preview)
}
