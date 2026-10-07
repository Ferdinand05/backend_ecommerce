package cartitems

import (
	"context"

	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"

	"github.com/google/uuid"
)

type Service interface {
	GetCart(
		ctx context.Context,
		userID uuid.UUID,
	) (CartResponse, error)

	AddItem(
		ctx context.Context,
		userID uuid.UUID,
		req AddCartItemRequest,
	) (CartItemResponse, error)

	IncreaseItem(
		ctx context.Context,
		userID uuid.UUID,
		itemID uuid.UUID,
	) (CartItemResponse, error)

	// DecreaseItem menghapus item saat quantity tinggal 1; removed = true.
	DecreaseItem(
		ctx context.Context,
		userID uuid.UUID,
		itemID uuid.UUID,
	) (CartItemResponse, bool, error)

	RemoveItem(
		ctx context.Context,
		userID uuid.UUID,
		itemID uuid.UUID,
	) error
}

type service struct {
	repo        Repository
	variantRepo productvariant.Repository
}

func NewService(repo Repository, variantRepo productvariant.Repository) *service {
	return &service{
		repo:        repo,
		variantRepo: variantRepo,
	}
}

func toCartItemResponse(item models.CartItem) CartItemResponse {
	return CartItemResponse{
		ID:               item.ID,
		ProductVariantID: item.ProductVariantID,
		Quantity:         item.Quantity,
		SKU:              item.ProductVariant.SKU,
		Name:             item.ProductVariant.Name,
		Price:            item.ProductVariant.Price,
		ProductID:        item.ProductVariant.ProductID,
		ProductName:      item.ProductVariant.Product.Name,
	}
}

func (s *service) GetCart(ctx context.Context, userID uuid.UUID) (CartResponse, error) {

	items, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return CartResponse{}, err
	}

	responses := make([]CartItemResponse, len(items))
	for i, item := range items {
		responses[i] = toCartItemResponse(item)
	}

	return CartResponse{Items: responses}, nil
}

func (s *service) AddItem(ctx context.Context, userID uuid.UUID, req AddCartItemRequest) (CartItemResponse, error) {

	variant, err := s.variantRepo.FindByID(ctx, req.ProductVariantID)
	if err != nil {
		return CartItemResponse{}, err
	}

	if !variant.IsActive {
		return CartItemResponse{}, ErrorProductVariantInactive
	}

	// ponytail: stok tidak divalidasi di sini (add/increase), divalidasi penuh
	// di checkout; kalau UX butuh fail-fast, tambah cek inventory repo lagi.
	item, err := s.repo.AddQuantity(ctx, userID, req.ProductVariantID, 1)
	if err != nil {
		return CartItemResponse{}, err
	}

	return toCartItemResponse(item), nil
}

func (s *service) IncreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (CartItemResponse, error) {

	if _, err := s.ownedCartItem(ctx, userID, itemID); err != nil {
		return CartItemResponse{}, err
	}

	item, err := s.repo.IncreaseQuantity(ctx, userID, itemID, 1)
	if err != nil {
		return CartItemResponse{}, err
	}

	return toCartItemResponse(item), nil
}

func (s *service) DecreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (CartItemResponse, bool, error) {

	item, err := s.ownedCartItem(ctx, userID, itemID)
	if err != nil {
		return CartItemResponse{}, false, err
	}

	if item.Quantity == 1 {
		if err := s.repo.Delete(ctx, itemID); err != nil {
			return CartItemResponse{}, false, err
		}

		return CartItemResponse{}, true, nil
	}

	updated, err := s.repo.DecreaseQuantity(ctx, userID, itemID, 1)
	if err != nil {
		return CartItemResponse{}, false, err
	}

	return toCartItemResponse(updated), false, nil
}

func (s *service) RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {

	if _, err := s.ownedCartItem(ctx, userID, itemID); err != nil {
		return err
	}

	return s.repo.Delete(ctx, itemID)
}

func (s *service) ownedCartItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (models.CartItem, error) {

	item, err := s.repo.FindByID(ctx, itemID)
	if err != nil {
		return models.CartItem{}, err
	}

	if item.UserID != userID {
		return models.CartItem{}, ErrorCartItemNotFound
	}

	return item, nil
}
