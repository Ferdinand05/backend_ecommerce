package productvariant

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/product"
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

func toProductVariantResponse(variant models.ProductVariant) ProductVariantResponse {
	return ProductVariantResponse{
		ID:          variant.ID,
		ProductID:   variant.ProductID,
		SKU:         variant.SKU,
		Name:        variant.Name,
		Price:       variant.Price,
		IsActive:    variant.IsActive,
		WeightGrams: variant.WeightGrams,
	}
}

func toProductVariantResponses(variants []models.ProductVariant) []ProductVariantResponse {
	responses := make([]ProductVariantResponse, len(variants))
	for i, v := range variants {
		responses[i] = toProductVariantResponse(v)
	}

	return responses
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

func (h *Handler) FindAllByProductID(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	variants, err := h.service.FindAllByProductID(ctx, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"variants": toProductVariantResponses(variants),
	})
}

func (h *Handler) FindByID(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, err := uuid.Parse(c.Param("variant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product variant id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	variant, err := h.service.FindByID(ctx, productID, variantID)
	if err != nil {

		if errors.Is(err, ErrorProductVariantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product variant not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"variant": toProductVariantResponse(variant),
	})
}

func (h *Handler) Create(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req CreateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	variant, err := h.service.Create(ctx, productID, req)
	if err != nil {

		if errors.Is(err, ErrorInvalidWeightGrams) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid weight_grams, must be between 1 and 100000",
			})
			return
		}

		if errors.Is(err, ErrorVariantSKUAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "variant sku already exists",
			})
			return
		}

		if errors.Is(err, product.ErrorProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"variant": toProductVariantResponse(variant),
	})
}

func (h *Handler) Update(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, err := uuid.Parse(c.Param("variant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product variant id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req UpdateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	variant, err := h.service.Update(ctx, productID, variantID, req)
	if err != nil {

		if errors.Is(err, ErrorInvalidWeightGrams) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid weight_grams, must be between 1 and 100000",
			})
			return
		}

		if errors.Is(err, ErrorProductVariantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product variant not found",
			})
			return
		}

		if errors.Is(err, ErrorVariantSKUAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "variant sku already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"variant": toProductVariantResponse(variant),
	})
}

func (h *Handler) Delete(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, err := uuid.Parse(c.Param("variant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product variant id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err = h.service.Delete(ctx, productID, variantID)
	if err != nil {

		if errors.Is(err, ErrorProductVariantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product variant not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
