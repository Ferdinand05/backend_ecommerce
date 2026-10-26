package product

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
	storage storage.Provider
}

func NewHandler(service Service, storage storage.Provider) *Handler {
	return &Handler{
		service: service,
		storage: storage,
	}
}

func toProductImageResponse(image models.ProductImage, provider storage.Provider) ProductImageResponse {
	return ProductImageResponse{
		ID:        image.ID,
		URL:       provider.URL(image.StorageKey),
		Alt:       image.Alt,
		SortOrder: image.SortOrder,
	}
}

func toProductImageResponses(images []models.ProductImage, provider storage.Provider) []ProductImageResponse {
	responses := make([]ProductImageResponse, len(images))
	for i, img := range images {
		responses[i] = toProductImageResponse(img, provider)
	}

	return responses
}

func toProductResponse(product models.Product, provider storage.Provider) ProductResponse {
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
		Images:    toProductImageResponses(product.Images, provider),
		Variants:  toProductVariantResponses(product.Variants),
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}
}

func toProductVariantResponses(variants []models.ProductVariant) []ProductVariantResponse {
	responses := make([]ProductVariantResponse, len(variants))
	for i, v := range variants {
		responses[i] = ProductVariantResponse{
			ID:       v.ID,
			SKU:      v.SKU,
			Name:     v.Name,
			Price:    v.Price,
			IsActive: v.IsActive,
		}
	}

	return responses
}

func toProductVariantDetailResponse(variant models.ProductVariant, provider storage.Provider) ProductVariantDetailResponse {
	response := ProductVariantDetailResponse{
		ProductVariantResponse: ProductVariantResponse{
			ID:       variant.ID,
			SKU:      variant.SKU,
			Name:     variant.Name,
			Price:    variant.Price,
			IsActive: variant.IsActive,
		},
		WeightGrams: variant.WeightGrams,
		Images:      toProductImageResponses(variant.Images, provider),
	}

	// Zero value (tanpa Preload) = null, bukan stok nol palsu.
	if variant.Inventory != nil {
		response.Inventory = &ProductVariantStockResponse{
			Quantity:         variant.Inventory.Quantity,
			ReservedQuantity: variant.Inventory.ReservedQuantity,
		}
	}

	return response
}

func toProductDetailResponse(product models.Product, provider storage.Provider) ProductDetailResponse {
	activeVariants := filterActiveVariants(product.Variants)

	variants := make([]ProductVariantDetailResponse, len(activeVariants))
	for i, v := range activeVariants {
		variants[i] = toProductVariantDetailResponse(v, provider)
	}

	response := toProductResponse(product, provider)
	return ProductDetailResponse{
		ID:          response.ID,
		CategoryID:  response.CategoryID,
		Name:        response.Name,
		Slug:        response.Slug,
		Description: response.Description,
		IsActive:    response.IsActive,
		Category:    response.Category,
		Images:      response.Images,
		Variants:    variants,
		CreatedAt:   response.CreatedAt,
		UpdatedAt:   response.UpdatedAt,
	}
}

// filterActiveVariants membuang varian non-aktif dari daftar varian produk.
func filterActiveVariants(variants []models.ProductVariant) []models.ProductVariant {
	active := make([]models.ProductVariant, 0, len(variants))

	for _, v := range variants {
		if v.IsActive {
			active = append(active, v)
		}
	}

	return active
}

// FindAll handler list produk untuk publik: hanya produk aktif, dengan
// pencarian, filter kategori, sorting, dan pagination.
func (h *Handler) FindAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	filter, ok := parseProductFilter(c)
	if !ok {
		return
	}

	result, err := h.service.FindPublicList(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	responses := make([]ProductResponse, len(result.Products))
	for i, p := range result.Products {
		p.Variants = filterActiveVariants(p.Variants)
		responses[i] = toProductResponse(p, h.storage)
	}

	c.JSON(http.StatusOK, gin.H{
		"products": responses,
		"meta": ProductListMeta{
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// FindAllAdmin handler list produk untuk admin: semua produk, filter
// opsional is_active, dengan pagination.
func (h *Handler) FindAllAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	filter, ok := parseProductFilter(c)
	if !ok {
		return
	}

	if raw := c.Query("is_active"); raw != "" {
		value := raw == "true" || raw == "1"
		filter.IsActive = &value
	}

	result, err := h.service.FindAll(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	responses := make([]ProductResponse, len(result.Products))
	for i, p := range result.Products {
		responses[i] = toProductResponse(p, h.storage)
	}

	c.JSON(http.StatusOK, gin.H{
		"products": responses,
		"meta": ProductListMeta{
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// parseProductFilter membaca query parameter list produk.
func parseProductFilter(c *gin.Context) (ProductFilter, bool) {
	filter := ProductFilter{
		Q:    c.Query("q"),
		Sort: c.Query("sort"),
	}

	if raw := c.Query("category_id"); raw != "" {
		categoryID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid category id",
			})
			return ProductFilter{}, false
		}

		filter.CategoryID = &categoryID
	}

	if raw := c.Query("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid page",
			})
			return ProductFilter{}, false
		}

		filter.Page = page
	}

	if raw := c.Query("page_size"); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil || pageSize < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid page size",
			})
			return ProductFilter{}, false
		}

		filter.PageSize = pageSize
	}

	return filter.Normalize(), true
}

func (h *Handler) FindByID(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := uuid.Parse(c.Param("product_id"))
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
		"product": toProductDetailResponse(product, h.storage),
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
		"product": toProductDetailResponse(product, h.storage),
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
		"product": toProductResponse(product, h.storage),
	})
}

func (h *Handler) Update(c *gin.Context) {

	id, err := uuid.Parse(c.Param("product_id"))
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
		"product": toProductResponse(product, h.storage),
	})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("product_id"))
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
