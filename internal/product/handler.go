package product

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
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

func toProductResponse(product models.Product) ProductResponse {
	return ProductResponse{
		ID:          product.ID,
		CategoryID:  product.CategoryID,
		Name:        product.Name,
		Slug:        product.Slug,
		Description: product.Description,
		IsActive:    product.IsActive,
		Category: &CategoryResponse{
			ID:   product.Category.ID,
			Name: product.Category.Name,
			Slug: product.Category.Slug,
		},
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}
}

func toProductVariantResponse(variant models.ProductVariant) ProductVariantResponse {
	return ProductVariantResponse{
		ID:       variant.ID,
		SKU:      variant.SKU,
		Name:     variant.Name,
		Price:    variant.Price,
		IsActive: variant.IsActive,
	}
}

func toProductDetailResponse(product models.Product) ProductDetailResponse {
	variants := make([]ProductVariantResponse, len(product.Variants))
	for i, v := range product.Variants {
		variants[i] = toProductVariantResponse(v)
	}

	return ProductDetailResponse{
		ProductResponse: toProductResponse(product),
		Variants:        variants,
	}
}

func (h *Handler) FindAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	products, err := h.service.FindAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	responses := make([]ProductResponse, len(products))
	for i, p := range products {
		responses[i] = toProductResponse(p)
	}

	c.JSON(http.StatusOK, gin.H{
		"products": responses,
	})
}

func (h *Handler) FindByID(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product id",
		})
		return
	}

	product, err := h.service.FindByID(ctx, id)
	if err != nil {

		if errors.Is(err, ErrorProductNotFound) {
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

	c.JSON(http.StatusOK, gin.H{
		"product": toProductDetailResponse(product),
	})
}

func (h *Handler) FindBySlug(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	slug := c.Param("slug")

	product, err := h.service.FindBySlug(ctx, slug)
	if err != nil {

		if errors.Is(err, ErrorProductNotFound) {
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

	c.JSON(http.StatusOK, gin.H{
		"product": toProductDetailResponse(product),
	})
}

func (h *Handler) Create(c *gin.Context) {

	var req CreateProductRequest

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	product, err := h.service.Create(ctx, req)
	if err != nil {

		if errors.Is(err, ErrorProductSlugAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "product slug already exists",
			})
			return
		}

		if errors.Is(err, category.ErrorCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "category not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"product": toProductResponse(product),
	})
}

func (h *Handler) Update(c *gin.Context) {

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	product, err := h.service.Update(ctx, id, req)
	if err != nil {

		if errors.Is(err, ErrorProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product not found",
			})
			return
		}

		if errors.Is(err, ErrorProductSlugAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "product slug already exists",
			})
			return
		}

		if errors.Is(err, category.ErrorCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "category not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product": toProductResponse(product),
	})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err = h.service.Delete(ctx, id)
	if err != nil {

		if errors.Is(err, ErrorProductNotFound) {
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

	c.JSON(http.StatusNoContent, gin.H{})
}
