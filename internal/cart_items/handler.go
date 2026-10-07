package cartitems

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	productvariant "ferdinand/ecommerce/internal/product_variant"

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

func (h *Handler) GetCart(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	cart, err := h.service.GetCart(ctx, userID)
	if err != nil {
		slog.Error("cart.get_cart", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cart": cart,
	})
}

func (h *Handler) AddItem(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	item, err := h.service.AddItem(ctx, userID, req)
	if err != nil {
		h.respondItemError(c, "cart.add_item", userID, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"item": item,
	})
}

func (h *Handler) IncreaseItem(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	itemID, ok := parseItemID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	item, err := h.service.IncreaseItem(ctx, userID, itemID)
	if err != nil {
		h.respondItemError(c, "cart.increase_item", userID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

func (h *Handler) DecreaseItem(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	itemID, ok := parseItemID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	item, removed, err := h.service.DecreaseItem(ctx, userID, itemID)
	if err != nil {
		h.respondItemError(c, "cart.decrease_item", userID, err)
		return
	}

	if removed {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

func (h *Handler) RemoveItem(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	itemID, ok := parseItemID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.RemoveItem(ctx, userID, itemID); err != nil {
		h.respondItemError(c, "cart.remove_item", userID, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// respondItemError memetakan error domain ke status HTTP yang sama untuk
// seluruh endpoint item.
func (h *Handler) respondItemError(c *gin.Context, event string, userID uuid.UUID, err error) {

	if errors.Is(err, ErrorCartItemNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "cart item not found",
		})
		return
	}

	if errors.Is(err, productvariant.ErrorProductVariantNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "product variant not found",
		})
		return
	}

	if errors.Is(err, ErrorProductVariantInactive) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "product variant is not available",
		})
		return
	}

	slog.Error(event, "user_id", userID, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}

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

func parseItemID(c *gin.Context) (uuid.UUID, bool) {

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid cart item id",
		})
		return uuid.Nil, false
	}

	return itemID, true
}
