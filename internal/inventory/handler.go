package inventory

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

func parseProductID(c *gin.Context) (uuid.UUID, bool) {
	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product id",
		})
		return uuid.Nil, false
	}

	return productID, true
}

func parseVariantID(c *gin.Context) (uuid.UUID, bool) {
	variantID, err := uuid.Parse(c.Param("variant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product variant id",
		})
		return uuid.Nil, false
	}

	return variantID, true
}

func (h *Handler) GetInventory(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	inventory, err := h.service.GetInventory(ctx, productID, variantID)
	if err != nil {
		h.respondError(c, err, "inventory.get_failed", variantID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"inventory": inventory,
	})
}

func (h *Handler) ListMovements(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	movements, err := h.service.ListMovements(ctx, productID, variantID)
	if err != nil {
		h.respondError(c, err, "inventory.list_movements_failed", variantID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"movements": movements,
	})
}

func (h *Handler) GetMovement(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	movementID, err := uuid.Parse(c.Param("movement_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid stock movement id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	movement, err := h.service.GetMovement(ctx, productID, variantID, movementID)
	if err != nil {
		h.respondError(c, err, "inventory.get_movement_failed", variantID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"movement": movement,
	})
}

func (h *Handler) CreateMovement(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req CreateStockMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	movement, err := h.service.CreateMovement(ctx, productID, variantID, req)
	if err != nil {
		h.respondError(c, err, "inventory.create_movement_failed", variantID)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"movement": movement,
	})
}

func (h *Handler) respondError(c *gin.Context, err error, event string, variantID uuid.UUID) {

	switch {
	case errors.Is(err, ErrorInsufficientStock):
		c.JSON(http.StatusConflict, gin.H{
			"error": "insufficient stock",
		})
	case errors.Is(err, ErrorInvalidMovementType):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid stock movement type",
		})
	case errors.Is(err, ErrorInvalidMovementQuantity):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "stock movement quantity must be greater than zero",
		})
	case errors.Is(err, ErrorInventoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "inventory item not found",
		})
	case errors.Is(err, ErrorStockMovementNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "stock movement not found",
		})
	case errors.Is(err, productvariant.ErrorProductVariantNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "product variant not found",
		})
	default:
		slog.Error(event, "variant_id", variantID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	}
}
