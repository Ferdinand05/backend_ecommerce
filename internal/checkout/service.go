package checkout

import (
	"context"
	"errors"
	"fmt"

	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service interface {
	Preview(
		ctx context.Context,
		userID uuid.UUID,
	) (PreviewResponse, error)
}

type service struct {
	cartService   cartitems.Service
	inventoryRepo inventory.InventoryRepository
	variantRepo   productvariant.Repository
}

func NewService(
	cartService cartitems.Service,
	inventoryRepo inventory.InventoryRepository,
	variantRepo productvariant.Repository,
) Service {
	return &service{
		cartService:   cartService,
		inventoryRepo: inventoryRepo,
		variantRepo:   variantRepo,
	}
}

// Preview menyusun ringkasan checkout milik user: item, harga terkini,
// berat, ketersediaan stok, dan subtotal.
//
// ponytail: cek stok dilakukan per item (N+1); keranjang realistis < 10 item.
// Kalau nanti jadi bottleneck, tambah FindByVariantIDs di inventory repo.
func (s *service) Preview(
	ctx context.Context,
	userID uuid.UUID,
) (PreviewResponse, error) {
	cart, err := s.cartService.GetCart(ctx, userID)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("getting cart for checkout: %w", err)
	}

	if len(cart.Items) == 0 {
		return PreviewResponse{}, ErrCartEmpty
	}

	// Validasi varian: semua ditemukan dan masih aktif.
	// Status produk induk belum diperiksa pada tahap ini.
	variantIDs := make([]uuid.UUID, 0, len(cart.Items))
	for _, item := range cart.Items {
		variantIDs = append(variantIDs, item.ProductVariantID)
	}

	variants, err := s.variantRepo.FindByIDs(ctx, variantIDs)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("fetching variants for checkout: %w", err)
	}

	variantByID := make(map[uuid.UUID]models.ProductVariant, len(variants))
	for _, v := range variants {
		variantByID[v.ID] = v
	}

	items := make([]PreviewItemResponse, 0, len(cart.Items))
	subtotal := decimal.Zero

	for _, item := range cart.Items {
		// Cart hanya menyumbang ProductVariantID + Quantity.
		// Harga, nama, SKU, dan berat diambil dari hasil query varian terbaru
		// agar tidak bergantung pada data cart yang bisa tidak sinkron.
		variant, found := variantByID[item.ProductVariantID]
		if !found || !variant.IsActive {
			return PreviewResponse{}, fmt.Errorf(
				"%w: variant %s is inactive or not found",
				ErrVariantNotAvailable,
				item.ProductVariantID,
			)
		}

		inventoryItem, err := s.inventoryRepo.FindByVariantID(
			ctx,
			item.ProductVariantID,
		)
		if err != nil {
			// Variant tanpa baris inventory sama sekali tidak tersedia.
			if errors.Is(err, inventory.ErrorInventoryNotFound) {
				return PreviewResponse{}, fmt.Errorf(
					"%w: variant %s is out of stock",
					ErrInsufficientStock,
					item.ProductVariantID,
				)
			}

			return PreviewResponse{}, fmt.Errorf(
				"getting inventory for variant %s: %w",
				item.ProductVariantID,
				err,
			)
		}

		// Stok fisik dikurangi yang sedang direservasi order lain.
		availableStock := inventoryItem.Quantity - inventoryItem.ReservedQuantity

		if availableStock < item.Quantity {
			return PreviewResponse{}, fmt.Errorf(
				"%w: variant %s, requested %d, available %d",
				ErrInsufficientStock,
				item.ProductVariantID,
				item.Quantity,
				availableStock,
			)
		}

		unitPrice := variant.Price

		itemSubtotal := unitPrice.Mul(
			decimal.NewFromInt(int64(item.Quantity)),
		)
		subtotal = subtotal.Add(itemSubtotal)

		items = append(items, PreviewItemResponse{
			ProductVariantID: item.ProductVariantID,
			ProductName:      variant.Product.Name,
			VariantName:      variant.Name,
			SKU:              variant.SKU,
			Quantity:         item.Quantity,
			UnitPrice:        unitPrice,
			Subtotal:         itemSubtotal,
			WeightGrams:      variant.WeightGrams,
			AvailableStock:   availableStock,
		})
	}

	return PreviewResponse{
		Items:    items,
		Subtotal: subtotal,
	}, nil
}
