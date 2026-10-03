package productimage

import (
	"context"
	"fmt"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/product"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/storage"
	"io"

	"github.com/google/uuid"
)

type Service interface {
	Upload(
		ctx context.Context,
		productID uuid.UUID,
		variantID *uuid.UUID,
		file io.Reader,
		contentType string,
		alt *string,
		sortOrder int,
	) (ProductImageResponse, error)

	FindByID(
		ctx context.Context,
		productID uuid.UUID,
		variantID *uuid.UUID,
		imageID uuid.UUID,
	) (ProductImageResponse, error)

	FindAllByProductID(
		ctx context.Context,
		productID uuid.UUID,
	) ([]ProductImageResponse, error)

	FindAllByVariantID(
		ctx context.Context,
		productID uuid.UUID,
		variantID uuid.UUID,
	) ([]ProductImageResponse, error)

	Delete(
		ctx context.Context,
		productID uuid.UUID,
		variantID *uuid.UUID,
		imageID uuid.UUID,
	) error
}

type service struct {
	repo        Repository
	productRepo product.Repository
	variantRepo productvariant.Repository
	storage     storage.Provider
}

func NewService(
	repo Repository,
	productRepo product.Repository,
	variantRepo productvariant.Repository,
	storage storage.Provider,
) *service {
	return &service{
		repo:        repo,
		productRepo: productRepo,
		variantRepo: variantRepo,
		storage:     storage,
	}
}

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func buildStorageKey(productID uuid.UUID, variantID *uuid.UUID, imageID uuid.UUID, ext string) string {

	if variantID != nil {
		return fmt.Sprintf("products/%s/variants/%s/%s%s", productID, *variantID, imageID, ext)
	}

	return fmt.Sprintf("products/%s/%s%s", productID, imageID, ext)
}

func toProductImageResponse(image models.ProductImage, provider storage.Provider) ProductImageResponse {
	return ProductImageResponse{
		ID:               image.ID,
		ProductID:        image.ProductID,
		ProductVariantID: image.ProductVariantID,
		URL:              provider.URL(image.StorageKey),
		Alt:              image.Alt,
		SortOrder:        image.SortOrder,
	}
}

func toProductImageResponses(images []models.ProductImage, provider storage.Provider) []ProductImageResponse {
	responses := make([]ProductImageResponse, len(images))
	for i, img := range images {
		responses[i] = toProductImageResponse(img, provider)
	}

	return responses
}

func imageBelongsToScope(image models.ProductImage, productID uuid.UUID, variantID *uuid.UUID) bool {

	if image.ProductID != productID {
		return false
	}

	if variantID == nil {
		return image.ProductVariantID == nil
	}

	return image.ProductVariantID != nil && *image.ProductVariantID == *variantID
}

func (s *service) Upload(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, file io.Reader, contentType string, alt *string, sortOrder int) (ProductImageResponse, error) {

	_, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return ProductImageResponse{}, err
	}

	if variantID != nil {

		variant, err := s.variantRepo.FindByID(ctx, *variantID)
		if err != nil {
			return ProductImageResponse{}, err
		}

		if variant.ProductID != productID {
			return ProductImageResponse{}, productvariant.ErrorProductVariantNotFound
		}
	}

	ext, ok := allowedImageTypes[contentType]
	if !ok {
		return ProductImageResponse{}, ErrorUnsupportedImageType
	}

	imageID := uuid.New()
	key := buildStorageKey(productID, variantID, imageID, ext)

	if err := s.storage.Upload(ctx, key, file, contentType); err != nil {
		return ProductImageResponse{}, err
	}

	image := models.ProductImage{
		ID:               imageID,
		ProductID:        productID,
		ProductVariantID: variantID,
		StorageKey:       key,
		Alt:              alt,
		SortOrder:        sortOrder,
	}

	if err := s.repo.Create(ctx, image); err != nil {

		if delErr := s.storage.Delete(ctx, key); delErr != nil {
			return ProductImageResponse{}, fmt.Errorf("creating product image:%w (storage cleanup failed: %v)", err, delErr)
		}

		return ProductImageResponse{}, err
	}

	return toProductImageResponse(image, s.storage), nil
}

func (s *service) FindByID(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, imageID uuid.UUID) (ProductImageResponse, error) {

	image, err := s.repo.FindByID(ctx, imageID)
	if err != nil {
		return ProductImageResponse{}, err
	}

	if !imageBelongsToScope(image, productID, variantID) {
		return ProductImageResponse{}, ErrorProductImageNotFound
	}

	return toProductImageResponse(image, s.storage), nil
}

func (s *service) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]ProductImageResponse, error) {

	images, err := s.repo.FindAllByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return toProductImageResponses(images, s.storage), nil
}

func (s *service) FindAllByVariantID(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) ([]ProductImageResponse, error) {

	images, err := s.repo.FindAllByVariantID(ctx, productID, variantID)
	if err != nil {
		return nil, err
	}

	return toProductImageResponses(images, s.storage), nil
}

func (s *service) Delete(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, imageID uuid.UUID) error {

	image, err := s.repo.FindByID(ctx, imageID)
	if err != nil {
		return err
	}

	if !imageBelongsToScope(image, productID, variantID) {
		return ErrorProductImageNotFound
	}

	if err := s.storage.Delete(ctx, image.StorageKey); err != nil {
		return err
	}

	return s.repo.Delete(ctx, imageID)
}
