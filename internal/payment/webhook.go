package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/order"
	xenditclient "ferdinand/ecommerce/internal/payment/xendit"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// HandleInvoiceCallback memproses callback invoice Xendit.
//
// Frontend tidak pernah menentukan order PAID; status hanya berubah
// setelah callback terverifikasi (token dicek di handler) diproses
// secara atomik di sini. Callback replay bersifat idempotent.
func (s *service) HandleInvoiceCallback(
	ctx context.Context,
	cb InvoiceCallback,
) error {

	slog.InfoContext(ctx, "webhook.invoice_received",
		"invoice_id", cb.ID,
		"external_id", cb.ExternalID,
		"status", cb.Status,
	)

	// 1. Identifikasi payment: invoice ID dulu; bila belum tersimpan
	//    (kasus UpdateInvoice gagal), fallback ke external_id.
	payment, err := s.paymentRepo.FindByProviderInvoiceID(ctx, cb.ID)
	if errors.Is(err, ErrPaymentNotFound) {
		payment, err = s.paymentRepo.FindByExternalID(ctx, cb.ExternalID)
	}
	if errors.Is(err, ErrPaymentNotFound) {
		// Bukan payment kita; retry dari Xendit tidak akan mengubah apa pun.
		slog.WarnContext(ctx, "webhook.unknown_payment",
			"invoice_id", cb.ID, "external_id", cb.ExternalID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("finding payment for callback: %w", err)
	}

	// 2. Callback harus menunjuk payment yang sama.
	if payment.ExternalID != cb.ExternalID {
		slog.ErrorContext(ctx, "webhook.external_id_mismatch",
			"payment_id", payment.ID,
			"db_external_id", payment.ExternalID,
			"callback_external_id", cb.ExternalID,
		)
		return ErrInvalidPayment
	}

	// 3. Mapping status eksplisit: tidak ada status bebas masuk database.
	switch cb.Status {
	case xenditclient.InvoiceStatusPaid, xenditclient.InvoiceStatusSettled:
		return s.finalizePaidCallback(ctx, payment, cb)

	case xenditclient.InvoiceStatusExpired:
		applied, err := s.paymentRepo.MarkExpiredPending(ctx, payment.ID)
		if err != nil {
			return err
		}
		if applied {
			slog.InfoContext(ctx, "webhook.payment_expired",
				"payment_id", payment.ID, "order_id", payment.OrderID)
		}
		return nil

	case xenditclient.InvoiceStatusPending:
		// Belum ada perubahan; payment memang masih PENDING.
		return nil

	default:
		slog.WarnContext(ctx, "webhook.status_ignored",
			"payment_id", payment.ID, "status", cb.Status)
		return nil
	}
}

// finalizePaidCallback menandai payment PAID, order PAID, mencatat
// history, memfinalisasi reservasi stok, dan menutup attempt PENDING
// lain dalam satu transaksi.
func (s *service) finalizePaidCallback(
	ctx context.Context,
	p models.Payment,
	cb InvoiceCallback,
) error {

	// Nominal dan mata uang harus persis dengan yang tercatat.
	if !p.Amount.Equal(decimal.NewFromFloat(cb.Amount)) ||
		(cb.Currency != "" && cb.Currency != p.Currency) {

		slog.ErrorContext(ctx, "webhook.amount_mismatch",
			"payment_id", p.ID,
			"db_amount", p.Amount.String(),
			"callback_amount", cb.Amount,
			"db_currency", p.Currency,
			"callback_currency", cb.Currency,
		)
		return ErrAmountMismatch
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)

		// 1. Lock payment, periksa status terkini.
		locked, err := s.paymentRepo.FindByIDForUpdate(txCtx, p.ID)
		if err != nil {
			return err
		}

		if locked.Status == string(PaymentStatusPaid) {
			// Callback replay yang sudah diproses — no-op idempotent.
			slog.InfoContext(ctx, "webhook.payment_already_paid",
				"payment_id", locked.ID)

			return nil // recovery berjalan sekali, setelah transaksi selesai.
		}

		// 2. Lock order, periksa status terkini.
		ord, err := s.orderRepo.FindByIDForUpdate(txCtx, locked.OrderID)
		if err != nil {
			return err
		}

		// 3. Validasi order masih boleh diselesaikan.
		if !order.CanTransition(
			order.OrderStatus(ord.Status),
			order.OrderStatusPaid,
		) {
			// Uang sudah masuk untuk order yang tidak bisa jadi PAID
			// (mis. CANCELLED): payment tetap dicatat PAID sebagai jejak,
			// refund ditangani di luar sistem.
			if _, err := s.paymentRepo.MarkPaid(txCtx, locked.ID, cb.PaidAt); err != nil {
				return err
			}

			slog.ErrorContext(ctx, "webhook.refund_needed",
				"payment_id", locked.ID,
				"order_id", ord.ID,
				"order_number", ord.OrderNumber,
				"order_status", ord.Status,
			)
			return nil
		}

		// 4. Payment -> PAID (guard replay antar request serentak).
		applied, err := s.paymentRepo.MarkPaid(txCtx, locked.ID, cb.PaidAt)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}

		// 5. Order -> PAID + history.
		if err := s.orderRepo.UpdateStatus(
			txCtx, ord.ID, string(order.OrderStatusPaid),
		); err != nil {
			return err
		}

		note := "Pembayaran terkonfirmasi via Xendit invoice " + cb.ID
		history := models.OrderStatusHistory{
			ID:      uuid.New(),
			OrderID: ord.ID,
			Status:  string(order.OrderStatusPaid),
			Note:    &note,
		}
		if err := s.orderRepo.CreateStatusHistory(txCtx, &history); err != nil {
			return err
		}

		// 6. Finalisasi reservasi stok per item order + jurnal SALE.
		refType := "ORDER"
		for _, item := range ord.Items {
			if err := s.inventoryRepo.FinalizeReservation(
				txCtx,
				item.ProductVariantID,
				item.Quantity,
			); err != nil {
				return err
			}

			inv, err := s.inventoryRepo.FindByVariantID(
				txCtx,
				item.ProductVariantID,
			)
			if err != nil {
				return err
			}

			note := "Penjualan order " + ord.OrderNumber
			if err := s.movementRepo.Create(txCtx, models.StockMovement{
				ID:              uuid.New(),
				InventoryItemID: inv.ID,
				Type:            string(inventory.StockMovementSale),
				Quantity:        -item.Quantity,
				Note:            &note,
				ReferenceType:   &refType,
				ReferenceID:     &ord.ID,
			}); err != nil {
				return err
			}
		}

		// 7. Tutup attempt PENDING lain milik order yang sama.
		if err := s.paymentRepo.FailOtherPendingByOrderID(
			txCtx, ord.ID, locked.ID,
		); err != nil {
			return err
		}

		// 8. Catat shipment PENDING dalam transaksi yang sama: order
		// sah dibayar, konfirmasi ke provider menyusul setelah commit.
		if err := s.shipments.CreatePending(txCtx, ord); err != nil {
			return err
		}

		slog.InfoContext(ctx, "webhook.payment_finalized",
			"payment_id", locked.ID,
			"order_id", ord.ID,
			"order_number", ord.OrderNumber,
			"amount", locked.Amount.String(),
		)

		return nil
	})
	if err != nil {
		return err
	}

	// Setelah commit: buat order pengiriman di Biteship. Di luar
	// transaksi DB; kegagalan tidak menggagalkan webhook karena
	// payment sudah sah — shipment tinggal PENDING dan dipulihkan
	// lewat replay webhook atau retry admin.
	s.recoverShipment(ctx, p.OrderID)

	return nil
}

// recoverShipment best-effort: kegagalan konfirmasi provider hanya
// dicatat, tidak menggagalkan proses pemanggil.
func (s *service) recoverShipment(ctx context.Context, orderID uuid.UUID) {
	if err := s.shipments.RecoverPending(ctx, orderID); err != nil {
		slog.ErrorContext(ctx, "shipment.recover_failed",
			"order_id", orderID,
			"error", err,
		)
	}
}
