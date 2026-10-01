package category

import (
	"context"
	"errors"
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

func (h *Handler) FindAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	categories, err := h.service.FindAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	responses := make([]CategoryResponse, len(categories))
	for i, c := range categories {
		responses[i] = toCategoryResponse(c)
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": responses,
	})

}

func toCategoryResponse(category models.Category) CategoryResponse {
	return CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}

func (h *Handler) FindByID(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid category id",
		})
		return
	}

	category, err := h.service.FindByID(ctx, id)
	if err != nil {

		if errors.Is(err, ErrorCategoryNotFound) {
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
		"category": toCategoryResponse(category),
	})

}

func (h *Handler) FindBySlug(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	slug := c.Param("slug")

	category, err := h.service.FindBySlug(ctx, slug)
	if err != nil {

		if errors.Is(err, ErrorCategoryNotFound) {
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
		"category": toCategoryResponse(category),
	})

}

func (h *Handler) Create(c *gin.Context) {

	var req CreateCategoryRequest

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	category, err := h.service.Create(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"category": toCategoryResponse(category),
	})
}

func (h *Handler) Update(c *gin.Context) {

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid category id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	category, err := h.service.Update(ctx, id, req)
	if err != nil {

		if errors.Is(err, ErrorCategoryNotFound) {
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
		"category": toCategoryResponse(category),
	})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid category id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err = h.service.Delete(ctx, id)
	if err != nil {

		if errors.Is(err, ErrorCategoryNotFound) {
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

	c.JSON(http.StatusNoContent, gin.H{})
}
