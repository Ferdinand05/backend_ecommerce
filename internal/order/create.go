package order

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/shipping"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const paymentExpiration = 30 * time.Minute

// CreateOrder menyusun order dari keranjang user: validasi ulang varian,
// hitung ulang ongkir, reservasi stok, lalu simpan semuanya dalam satu
// transaksi. Keranjang dikosongkan setelah transaksi commit.
func (s *service) CreateOrder(
	ctx context.Context,
	userID uuid.UUID,
	req CreateOrderRequest,
) (models.Order, error) {

	slog.InfoContext(ctx, "order.creating", "user_id", userID)

	// 1. Cart sumber variantID + quantity; kosong = tidak bisa order.
	cart, err := s.cartService.GetCart(ctx, userID)
	if err != nil {
		return models.Order{}, fmt.Errorf("getting cart for order: %w", err)
	}

	if len(cart.Items) == 0 {
		return models.Order{}, ErrorEmptyCart
	}

	// Validasi kuantitas sebelum menyentuh repository stok.
	for _, item := range cart.Items {
		if item.Quantity <= 0 {
			return models.Order{}, fmt.Errorf(
				"%w: variant %s has quantity %d",
				ErrorInvalidOrderInput,
				item.ProductVariantID,
				item.Quantity,
			)
		}
	}

	// 2. Validasi ulang varian: ditemukan, aktif, produk induk aktif.
	//    Harga dan identitas snapshot dari hasil query terbaru.
	variantIDs := make([]uuid.UUID, 0, len(cart.Items))
	for _, item := range cart.Items {
		variantIDs = append(variantIDs, item.ProductVariantID)
	}

	variants, err := s.variantRepo.FindByIDs(ctx, variantIDs)
	if err != nil {
		return models.Order{}, fmt.Errorf("fetching variants for order: %w", err)
	}

	variantByID := make(map[uuid.UUID]models.ProductVariant, len(variants))
	for _, v := range variants {
		variantByID[v.ID] = v
	}

	subtotal := decimal.Zero
	orderItems := make([]models.OrderItem, 0, len(cart.Items))
	rateItems := make([]shipping.ShippingRateItem, 0, len(cart.Items))

	for _, item := range cart.Items {
		variant, found := variantByID[item.ProductVariantID]
		if !found || !variant.IsActive || !variant.Product.IsActive {
			return models.Order{}, fmt.Errorf(
				"%w: variant %s or its product is inactive or not found",
				ErrorVariantUnavailable,
				item.ProductVariantID,
			)
		}

		itemSubtotal := variant.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		subtotal = subtotal.Add(itemSubtotal)

		orderItems = append(orderItems, models.OrderItem{
			ID:               uuid.New(),
			ProductVariantID: item.ProductVariantID,
			ProductName:      variant.Product.Name,
			VariantName:      variant.Name,
			SKU:              variant.SKU,
			WeightGrams:      variant.WeightGrams,
			Price:            variant.Price,
			Quantity:         item.Quantity,
			Subtotal:         itemSubtotal,
		})

		rateItems = append(rateItems, shipping.ShippingRateItem{
			ProductVariantID: item.ProductVariantID.String(),
			Quantity:         item.Quantity,
		})
	}

	// 3. Hitung ulang ongkir di backend; harga dari frontend tidak dipercaya.
	rates, err := s.shippingSvc.CalculateRates(ctx, shipping.CalculateRatesRequest{
		DestinationAreaID: req.DestinationAreaID,
		Items:             rateItems,
	})
	if err != nil {
		return models.Order{}, fmt.Errorf("calculating shipping rates: %w", err)
	}

	var selected *shipping.ShippingRate
	for i := range rates {
		if rates[i].CourierCode == req.CourierCode &&
			rates[i].CourierServiceCode == req.CourierServiceCode {
			selected = &rates[i]
			break
		}
	}

	if selected == nil {
		return models.Order{}, fmt.Errorf(
			"%w: %s %s",
			ErrorShippingRateNotFound,
			req.CourierCode,
			req.CourierServiceCode,
		)
	}

	slog.InfoContext(ctx, "order.rate_matched",
		"user_id", userID,
		"courier_code", selected.CourierCode,
		"courier_service_code", selected.CourierServiceCode,
		"price", selected.Price,
	)

	// 4. Snapshot order.
	now := time.Now()
	expiresAt := now.Add(paymentExpiration)
	shippingCost := decimal.NewFromInt(selected.Price)

	order := models.Order{
		ID:                 uuid.New(),
		UserID:             userID,
		OrderNumber:        generateOrderNumber(now),
		Status:             string(OrderStatusPending),
		Subtotal:           subtotal,
		ShippingCost:       shippingCost,
		DiscountAmount:     decimal.Zero,
		TotalAmount:        subtotal.Add(shippingCost),
		CourierCode:        selected.CourierCode,
		CourierName:        selected.CourierName,
		CourierServiceCode: selected.CourierServiceCode,
		CourierServiceName: selected.CourierServiceName,
		PaymentExpiresAt:   &expiresAt,
	}

	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}

	address := models.OrderAddress{
		ID:               uuid.New(),
		OrderID:          order.ID,
		RecipientName:    req.RecipientName,
		Phone:            req.Phone,
		Address:          req.Address,
		City:             req.City,
		Province:         req.Province,
		PostalCode:       req.PostalCode,
		BiteshipAreaID:   &req.DestinationAreaID,
		BiteshipAreaName: &req.DestinationAreaName,
	}

	history := models.OrderStatusHistory{
		ID:      uuid.New(),
		OrderID: order.ID,
		Status:  string(OrderStatusPending),
	}

	// 5. Satu transaksi: lock + reservasi stok per varian, lalu simpan semua.
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)

		for _, item := range orderItems {
			inv, err := s.inventoryRepo.FindByVariantIDForUpdate(
				txCtx,
				item.ProductVariantID,
			)
			if err != nil {
				return fmt.Errorf("locking inventory for variant %s: %w", item.ProductVariantID, err)
			}

			if err := s.inventoryRepo.ReserveStock(
				txCtx,
				item.ProductVariantID,
				item.Quantity,
			); err != nil {
				return err
			}

			// Jurnal RESERVE: stok belum turun, ini komitmen pesanan.
			refType := "ORDER"
			note := "Reservasi order " + order.OrderNumber
			if err := s.movementRepo.Create(txCtx, models.StockMovement{
				ID:              uuid.New(),
				InventoryItemID: inv.ID,
				Type:            string(inventory.StockMovementReserve),
				Quantity:        -item.Quantity,
				Note:            &note,
				ReferenceType:   &refType,
				ReferenceID:     &order.ID,
			}); err != nil {
				return err
			}

			slog.InfoContext(txCtx, "order.stock_reserved",
				"order_id", order.ID,
				"variant_id", item.ProductVariantID,
				"quantity", item.Quantity,
			)
		}

		if err := s.repository.Create(txCtx, &order); err != nil {
			return err
		}

		for i := range orderItems {
			if err := s.repository.CreateItem(txCtx, &orderItems[i]); err != nil {
				return err
			}
		}

		if err := s.repository.CreateAddress(txCtx, &address); err != nil {
			return err
		}

		return s.repository.CreateStatusHistory(txCtx, &history)
	})
	if err != nil {
		return models.Order{}, err
	}

	// Sambungkan relasi untuk response; dilakukan SETELAH Create agar
	// GORM tidak auto-insert asosiasi (dobel dengan CreateItem dkk).
	order.Items = orderItems
	order.Address = &address
	order.StatusHistories = []models.OrderStatusHistory{history}

	slog.InfoContext(ctx, "order.created",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"user_id", userID,
		"total_amount", order.TotalAmount.String(),
	)

	// 6. Keranjang dikosongkan setelah commit; kegagalan di sini
	//    tidak membatalkan order yang sudah tersimpan.
	if err := s.cartService.ClearCart(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "order.clear_cart_failed",
			"order_id", order.ID,
			"user_id", userID,
			"error", err,
		)
	}

	return order, nil
}

func generateOrderNumber(now time.Time) string {
	return fmt.Sprintf(
		"ORD-%s-%s",
		now.Format("20060102"),
		strings.ToUpper(uuid.NewString()[:8]),
	)
}
