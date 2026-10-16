package shipping

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) SearchAreas(c *gin.Context) {
	query := c.Query("q")

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	areas, err := h.service.SearchAreas(ctx, query)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAreaQuery):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "query parameter q is required",
			})

		case errors.Is(err, ErrAreaProvider):
			slog.ErrorContext(ctx, "shipping area provider failed",
				"error", err,
			)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "failed to search shipping areas",
			})

		default:
			slog.ErrorContext(ctx, "failed to search shipping areas",
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusOK, SearchAreasResponse{
		Areas: areas,
	})
}

func (h *Handler) CalculateRates(c *gin.Context) {

	var req CalculateRatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		10*time.Second,
	)
	defer cancel()

	rates, err := h.service.CalculateRates(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRatesInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid shipping rates request",
			})

		case errors.Is(err, ErrVariantNotAvailable):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "one or more product variants are not available for shipping",
			})

		case errors.Is(err, ErrAreaProvider):
			slog.ErrorContext(ctx, "shipping rates provider failed",
				"error", err,
			)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "failed to calculate shipping rates",
			})

		default:
			slog.ErrorContext(ctx, "failed to calculate shipping rates",
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusOK, CalculateRatesResponse{
		Rates: rates,
	})
}
