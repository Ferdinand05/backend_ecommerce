package shipment

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

// GET /me/orders/:order_id/shipment
func (h *Handler) GetMyOrderShipment(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	orderID, ok := parseUUIDParam(c, "order_id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response, err := h.service.FindMyShipmentByOrder(ctx, userID, orderID)
	if err != nil {
		respondShipmentError(c, "shipment.get_by_order", &userID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shipment": response,
	})
}

// GET /me/shipments/:shipment_id
func (h *Handler) GetMyShipment(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	shipmentID, ok := parseUUIDParam(c, "shipment_id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response, err := h.service.FindMyShipment(ctx, userID, shipmentID)
	if err != nil {
		respondShipmentError(c, "shipment.get", &userID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shipment": response,
	})
}

// GET /me/shipments/:shipment_id/tracking
func (h *Handler) TrackMyShipment(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	shipmentID, ok := parseUUIDParam(c, "shipment_id")
	if !ok {
		return
	}

	// Longgar: menyertakan pemanggilan tracking API provider.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.service.Track(ctx, userID, shipmentID)
	if err != nil {
		respondShipmentError(c, "shipment.track", &userID, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// POST /admin/shipments/:shipment_id/retry
func (h *Handler) RetryShipment(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	shipmentID, ok := parseUUIDParam(c, "shipment_id")
	if !ok {
		return
	}

	// Longgar: mencakup pembuatan order di API provider.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	result, err := h.service.Retry(ctx, shipmentID)
	if err != nil {
		respondShipmentError(c, "shipment.retry", &userID, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func respondShipmentError(c *gin.Context, event string, userID *uuid.UUID, err error) {

	if errors.Is(err, ErrShipmentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "shipment not found",
		})
		return
	}

	if errors.Is(err, ErrInvalidStatusTransition) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "shipment is not pending",
		})
		return
	}

	if errors.Is(err, ErrProviderOrderExists) {
		slog.Error(event+".provider_order_exists", "user_id", userID, "error", err)
		c.JSON(http.StatusConflict, gin.H{
			"error": "provider order already exists, manual reconciliation required",
		})
		return
	}

	if errors.Is(err, ErrProviderStateAmbiguous) {
		slog.Error(event+".provider_state_ambiguous", "user_id", userID, "error", err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "provider state could not be verified, please retry",
		})
		return
	}

	if errors.Is(err, ErrInvalidShipmentData) {
		slog.Error(event+".invalid_data", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "shipment data is incomplete",
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

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {

	value, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid " + name,
		})
		return uuid.Nil, false
	}

	return value, true
}
