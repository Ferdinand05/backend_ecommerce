package productimage

import (
	"bytes"
	"context"
	"errors"
	"ferdinand/ecommerce/internal/product"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxImageSize = 5 << 20

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

func (h *Handler) upload(c *gin.Context, variantID *uuid.UUID) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageSize)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "image file too large",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file is required",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cannot read image file",
		})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "image file too large",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cannot read image file",
		})
		return
	}

	contentType := http.DetectContentType(content)

	var alt *string
	if value := strings.TrimSpace(c.PostForm("alt")); value != "" {
		alt = &value
	}

	sortOrder := 0
	if value := strings.TrimSpace(c.PostForm("sort_order")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid sort_order",
			})
			return
		}
		sortOrder = parsed
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	image, err := h.service.Upload(ctx, productID, variantID, bytes.NewReader(content), contentType, alt, sortOrder)
	if err != nil {

		if errors.Is(err, ErrorUnsupportedImageType) {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "unsupported image type",
			})
			return
		}

		if errors.Is(err, product.ErrorProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product not found",
			})
			return
		}

		if errors.Is(err, productvariant.ErrorProductVariantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product variant not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		slog.Error("product_image.upload_failed", "product_id", productID, "variant_id", variantID, "error", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"image": image,
	})
}

func (h *Handler) UploadProductImage(c *gin.Context) {
	h.upload(c, nil)
}

func (h *Handler) UploadVariantImage(c *gin.Context) {

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	h.upload(c, &variantID)
}

func (h *Handler) findAll(c *gin.Context, images []ProductImageResponse, err error) {

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		slog.Error("product_image.list_failed", "error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
	})
}

func (h *Handler) FindAllByProductID(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	images, err := h.service.FindAllByProductID(ctx, productID)

	h.findAll(c, images, err)
}

func (h *Handler) FindAllByVariantID(c *gin.Context) {

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

	images, err := h.service.FindAllByVariantID(ctx, productID, variantID)

	h.findAll(c, images, err)
}

func (h *Handler) findOne(c *gin.Context, productID uuid.UUID, variantID *uuid.UUID) {

	imageID, err := uuid.Parse(c.Param("image_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product image id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	image, err := h.service.FindByID(ctx, productID, variantID, imageID)
	if err != nil {

		if errors.Is(err, ErrorProductImageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product image not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		slog.Error("product_image.find_failed", "product_id", productID, "variant_id", variantID, "image_id", imageID, "error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image": image,
	})
}

func (h *Handler) FindProductImageByID(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	h.findOne(c, productID, nil)
}

func (h *Handler) FindVariantImageByID(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	h.findOne(c, productID, &variantID)
}

func (h *Handler) remove(c *gin.Context, productID uuid.UUID, variantID *uuid.UUID) {

	imageID, err := uuid.Parse(c.Param("image_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid product image id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err = h.service.Delete(ctx, productID, variantID, imageID)
	if err != nil {

		if errors.Is(err, ErrorProductImageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product image not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		slog.Error("product_image.delete_failed", "product_id", productID, "variant_id", variantID, "image_id", imageID, "error", err)
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

func (h *Handler) DeleteProductImage(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	h.remove(c, productID, nil)
}

func (h *Handler) DeleteVariantImage(c *gin.Context) {

	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := parseVariantID(c)
	if !ok {
		return
	}

	h.remove(c, productID, &variantID)
}
