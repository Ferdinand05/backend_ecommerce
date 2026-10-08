package order

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"ferdinand/ecommerce/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100

	dateLayout = "2006-01-02"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GET /me/orders
func (h *Handler) ListMyOrders(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	page, pageSize, ok := parsePagination(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.service.FindMyOrders(ctx, userID, page, pageSize)
	if err != nil {
		slog.Error("order.list_my_orders", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	respondOrderList(c, result)
}

// GET /me/orders/:order_id
func (h *Handler) GetMyOrder(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	orderID, ok := parseOrderID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	order, err := h.service.FindMyOrder(ctx, userID, orderID)
	if err != nil {
		respondOrderError(c, "order.get_my_order", &userID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": toOrderDetailResponse(order),
	})
}

// GET /admin/orders
func (h *Handler) ListOrders(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	filter, ok := parseOrderFilter(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.service.FindAll(ctx, filter)
	if err != nil {
		slog.Error("order.list_orders", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	respondOrderList(c, result)
}

// GET /admin/orders/:order_id
func (h *Handler) GetOrder(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	orderID, ok := parseOrderID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	order, err := h.service.FindByID(ctx, orderID)
	if err != nil {
		respondOrderError(c, "order.get_order", &userID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": toOrderDetailResponse(order),
	})
}

// PATCH /admin/orders/:order_id/status
func (h *Handler) UpdateStatus(c *gin.Context) {

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	orderID, ok := parseOrderID(c)
	if !ok {
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	order, err := h.service.UpdateStatus(ctx, orderID, OrderStatus(req.Status), req.Note)
	if err != nil {
		respondOrderError(c, "order.update_status", &userID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": toOrderDetailResponse(order),
	})
}

func respondOrderList(c *gin.Context, result OrderListResult) {

	orders := make([]OrderResponse, len(result.Orders))
	for i, order := range result.Orders {
		orders[i] = toOrderResponse(order)
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"meta": OrderListMeta{
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

func respondOrderError(c *gin.Context, event string, userID *uuid.UUID, err error) {

	if errors.Is(err, ErrorOrderNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "order not found",
		})
		return
	}

	if errors.Is(err, ErrorInvalidOrderStatus) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid order status",
		})
		return
	}

	if errors.Is(err, ErrorInvalidStatusTransition) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "invalid order status transition",
		})
		return
	}

	slog.Error(event, "user_id", userID, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}

// parsePagination membaca page/page_size: default 1/10, maksimum 100.
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

		pageSize = value
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize, true
}

func parseOrderID(c *gin.Context) (uuid.UUID, bool) {

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid order id",
		})
		return uuid.Nil, false
	}

	return orderID, true
}

func parseOrderFilter(c *gin.Context) (OrderFilter, bool) {

	page, pageSize, ok := parsePagination(c)
	if !ok {
		return OrderFilter{}, false
	}

	filter := OrderFilter{Page: page, PageSize: pageSize}

	if raw := c.Query("status"); raw != "" {

		status := OrderStatus(raw)
		if !status.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid status filter",
			})
			return OrderFilter{}, false
		}

		filter.Status = &status
	}

	if raw := c.Query("order_number"); raw != "" {
		filter.OrderNumber = &raw
	}

	if raw := c.Query("user_id"); raw != "" {

		userID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user_id filter",
			})
			return OrderFilter{}, false
		}

		filter.UserID = &userID
	}

	if raw := c.Query("start_date"); raw != "" {

		value, err := time.Parse(dateLayout, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid start_date, use YYYY-MM-DD",
			})
			return OrderFilter{}, false
		}

		filter.StartDate = &value
	}

	if raw := c.Query("end_date"); raw != "" {

		value, err := time.Parse(dateLayout, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid end_date, use YYYY-MM-DD",
			})
			return OrderFilter{}, false
		}

		// geser +1 hari: EndDate di repository eksklusif, input date-only inklusif.
		value = value.AddDate(0, 0, 1)
		filter.EndDate = &value
	}

	if filter.StartDate != nil && filter.EndDate != nil && filter.StartDate.After(*filter.EndDate) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_date must not be after end_date",
		})
		return OrderFilter{}, false
	}

	return filter, true
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

func toOrderResponse(order models.Order) OrderResponse {
	return OrderResponse{
		ID:             order.ID,
		OrderNumber:    order.OrderNumber,
		Status:         order.Status,
		Subtotal:       order.Subtotal,
		ShippingCost:   order.ShippingCost,
		DiscountAmount: order.DiscountAmount,
		TotalAmount:    order.TotalAmount,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
}

func toOrderDetailResponse(order models.Order) OrderDetailResponse {

	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemResponse{
			ID:               item.ID,
			ProductVariantID: item.ProductVariantID,
			ProductName:      item.ProductName,
			VariantName:      item.VariantName,
			SKU:              item.SKU,
			Price:            item.Price,
			Quantity:         item.Quantity,
			Subtotal:         item.Subtotal,
		}
	}

	address := OrderAddressResponse{}
	if order.Address != nil {
		address = toOrderAddressResponse(*order.Address)
	}

	histories := make([]OrderStatusHistoryResponse, len(order.StatusHistories))
	for i, history := range order.StatusHistories {
		histories[i] = OrderStatusHistoryResponse{
			ID:        history.ID,
			Status:    history.Status,
			Note:      history.Note,
			CreatedAt: history.CreatedAt,
		}
	}

	return OrderDetailResponse{
		ID:              order.ID,
		OrderNumber:     order.OrderNumber,
		Status:          order.Status,
		Subtotal:        order.Subtotal,
		ShippingCost:    order.ShippingCost,
		DiscountAmount:  order.DiscountAmount,
		TotalAmount:     order.TotalAmount,
		Items:           items,
		Address:         address,
		StatusHistories: histories,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}

func toOrderAddressResponse(address models.OrderAddress) OrderAddressResponse {
	return OrderAddressResponse{
		ID:             address.ID,
		RecipientName:  address.RecipientName,
		Phone:          address.Phone,
		Address:        address.Address,
		City:           address.City,
		Province:       address.Province,
		PostalCode:     address.PostalCode,
		Latitude:       address.Latitude,
		Longitude:      address.Longitude,
		BiteshipAreaID: address.BiteshipAreaID,
	}
}
