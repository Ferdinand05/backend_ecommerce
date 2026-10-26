package order

import (
	"context"
	"fmt"
	"log/slog"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// cancelNote adalah alasan pembatalan yang dicatat di history order.
const cancelNote = "order cancelled by customer"

// CancelOrder membatalkan order milik user sendiri selama statusnya masih
// PENDING: stok yang direservasi dilepas, jurnal ORDER_CANCEL ditulis, lalu
// status order diubah ke CANCELLED. Semua menulis DB dalam satu transaksi.
func (s *service) CancelOrder(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (models.Order, error) {

	err := s.inTransaction(ctx, func(txCtx context.Context) error {

		current, err := s.repository.FindByIDForUpdate(txCtx, orderID)
		if err != nil {
			return err
		}

		// Order milik user lain diperlakukan sama dengan tidak ada, agar
		// keberadaan order tidak bocor.
		if current.UserID != userID {
			return ErrorOrderNotFound
		}

		if OrderStatus(current.Status) != OrderStatusPending {
			return ErrorOrderNotCancellable
		}

		items, err := s.repository.FindItemsByOrderID(txCtx, orderID)
		if err != nil {
			return err
		}

		for _, item := range items {

			inv, err := s.inventoryRepo.FindByVariantIDForUpdate(txCtx, item.ProductVariantID)
			if err != nil {
				return fmt.Errorf("locking inventory for variant %s: %w", item.ProductVariantID, err)
			}

			if err := s.inventoryRepo.ReleaseReservation(
				txCtx,
				item.ProductVariantID,
				item.Quantity,
			); err != nil {
				return err
			}

			// Jurnal ORDER_CANCEL: reservasi dilepas, stok fisik tidak
			// berubah (berlawanan dengan RESERVE yang menambah reservasi).
			refType := "ORDER"
			note := cancelNote
			if err := s.movementRepo.Create(txCtx, models.StockMovement{
				ID:              uuid.New(),
				InventoryItemID: inv.ID,
				Type:            string(inventory.StockMovementOrderCancel),
				Quantity:        item.Quantity,
				Note:            &note,
				ReferenceType:   &refType,
				ReferenceID:     &orderID,
			}); err != nil {
				return err
			}
		}

		if err := s.repository.UpdateStatus(txCtx, orderID, string(OrderStatusCancelled)); err != nil {
			return err
		}

		historyNote := cancelNote
		return s.repository.CreateStatusHistory(txCtx, &models.OrderStatusHistory{
			ID:      uuid.New(),
			OrderID: orderID,
			Status:  string(OrderStatusCancelled),
			Note:    &historyNote,
		})
	})
	if err != nil {
		return models.Order{}, err
	}

	slog.InfoContext(ctx, "order.cancelled",
		"order_id", orderID,
		"user_id", userID,
	)

	return s.repository.FindByID(ctx, orderID)
}

// inTransaction menjalankan fn di dalam transaksi DB. Tanpa db (unit test
// dengan fake) fn dijalankan langsung tanpa transaksi.
func (s *service) inTransaction(ctx context.Context, fn func(ctx context.Context) error) error {

	if s.db == nil {
		return fn(ctx)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(database.InjectTx(ctx, tx))
	})
}
