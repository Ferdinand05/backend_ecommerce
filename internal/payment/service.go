package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/order"
	xenditclient "ferdinand/ecommerce/internal/payment/xendit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Service interface {
	CreatePayment(
		ctx context.Context,
		userID uuid.UUID,
		orderID uuid.UUID,
	) (CreatePaymentResult, error)

	HandleInvoiceCallback(
		ctx context.Context,
		cb InvoiceCallback,
	) error
}

type service struct {
	// db dipakai tx webhook: finalizePaidCallback.
	db            *gorm.DB
	orderRepo     orderReader
	paymentRepo   Repository
	inventoryRepo inventory.InventoryRepository
	xendit        xenditclient.Client
}

// orderReader diambil dari method yang tersedia di order.Repository;
// webhook butuh akses ber-transaction ke order.
type orderReader interface {
	FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error)
	FindByIDForUpdate(ctx context.Context, orderID uuid.UUID) (models.Order, error)
	UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error
	CreateStatusHistory(ctx context.Context, history *models.OrderStatusHistory) error
}

func NewService(
	db *gorm.DB,
	orderRepo orderReader,
	paymentRepo Repository,
	inventoryRepo inventory.InventoryRepository,
	xendit xenditclient.Client,
) Service {
	return &service{
		db:            db,
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		xendit:        xendit,
	}
}

// CreatePayment membuat percobaan pembayaran untuk order PENDING.
//
// Aturan: satu order boleh punya banyak attempt, tapi hanya satu payment
// PENDING pada satu waktu (dijaga partial unique index). Tidak ada
// transaksi DB yang menahan pemanggilan API Xendit.
func (s *service) CreatePayment(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (CreatePaymentResult, error) {

	slog.InfoContext(ctx, "payment.creating", "user_id", userID, "order_id", orderID)

	// 1. Order harus ada dan milik user yang login.
	ord, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return CreatePaymentResult{}, fmt.Errorf("find order: %w", err)
	}

	if ord.UserID != userID {
		return CreatePaymentResult{}, ErrOrderNotFound
	}

	// 2. Hanya order PENDING yang dapat dibayar.
	if ord.Status != string(order.OrderStatusPending) {
		return CreatePaymentResult{}, ErrOrderNotPayable
	}

	now := time.Now()

	// 3. Periode pembayaran order belum boleh habis.
	if ord.PaymentExpiresAt != nil && !now.Before(*ord.PaymentExpiresAt) {
		return CreatePaymentResult{}, ErrOrderPaymentExpired
	}

	// 4. Attempt terakhir masih PENDING → resolve (reuse / dead / error).
	existing, err := s.paymentRepo.FindLatestByOrderID(ctx, orderID)
	if err != nil && !errors.Is(err, ErrPaymentNotFound) {
		return CreatePaymentResult{}, fmt.Errorf("find existing payment: %w", err)
	}

	if err == nil && existing.Status == string(PaymentStatusPending) {
		result, dead, err := s.resolvePendingPayment(ctx, existing)
		if err != nil {
			return CreatePaymentResult{}, err
		}
		if !dead {
			return result, nil
		}
		// Payment lama mati terkonfirmasi → buat attempt baru.
	}

	payment := models.Payment{
		ID:         uuid.New(),
		OrderID:    ord.ID,
		Provider:   models.PaymentProviderXendit,
		ExternalID: "PAY-" + uuid.NewString(),
		Status:     string(PaymentStatusPending),
		Amount:     ord.TotalAmount,
		Currency:   "IDR",
	}

	// 5. Simpan attempt; kalah race → resolve payment pemenang race.
	if err := s.paymentRepo.Create(ctx, &payment); err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			return CreatePaymentResult{}, fmt.Errorf("create payment record: %w", err)
		}

		winner, ferr := s.paymentRepo.FindLatestByOrderID(ctx, orderID)
		if ferr != nil {
			return CreatePaymentResult{}, fmt.Errorf("find pending payment after race: %w", ferr)
		}

		result, dead, rerr := s.resolvePendingPayment(ctx, winner)
		if rerr != nil {
			return CreatePaymentResult{}, rerr
		}
		if !dead {
			return result, nil
		}

		// ponytail: satu kali retry saja; kena 23505 dua kali berarti ada masalah lain.
		if err := s.paymentRepo.Create(ctx, &payment); err != nil {
			return CreatePaymentResult{}, fmt.Errorf("create payment record: %w", err)
		}
	}

	// 6. Masa berlaku invoice = sisa masa bayar order.
	duration := 30 * time.Minute
	if ord.PaymentExpiresAt != nil {
		duration = time.Until(*ord.PaymentExpiresAt)
	}
	if duration <= 0 {
		return CreatePaymentResult{}, ErrOrderPaymentExpired
	}

	amount, _ := payment.Amount.Float64()

	// Invoice IDR harus bilangan bulat; pecahan (mis. dari ongkir)
	// pasti ditolak Xendit dengan pesan generik.
	if !payment.Amount.IsInteger() {
		return CreatePaymentResult{}, fmt.Errorf(
			"order total %s is not an integer amount: %w",
			payment.Amount.String(), ErrInvalidInvoiceAmount,
		)
	}

	// 7. Buat invoice di Xendit.
	inv, err := s.xendit.CreateInvoice(
		ctx,
		xenditclient.CreateInvoiceRequest{
			ExternalID:      payment.ExternalID,
			Description:     "Pembayaran order " + ord.OrderNumber,
			Amount:          amount,
			Currency:        payment.Currency,
			InvoiceDuration: duration,
		},
	)
	if err != nil {
		// Timeout / error ≠ invoice tidak jadi. Verifikasi dulu berdasarkan
		// external_id; FAILED hanya bila Xendit mengkonfirmasi tidak ada.
		confirmed, verr := s.xendit.FindLatestInvoiceByExternalID(ctx, payment.ExternalID)
		if verr != nil {
			// Row tetap PENDING tanpa URL; attempt berikutnya memverifikasi lagi.
			return CreatePaymentResult{}, fmt.Errorf(
				"%w: invoice creation failed and state unverifiable",
				ErrPaymentStateAmbiguous,
			)
		}

		if confirmed == nil {
			if merr := s.markFailed(ctx, payment.ID); merr != nil {
				return CreatePaymentResult{}, merr
			}
			return CreatePaymentResult{}, fmt.Errorf("create xendit invoice: %w", err)
		}

		// Invoice ternyata sudah jadi → recover, bukan buat baru.
		inv = *confirmed
		slog.InfoContext(ctx, "payment.invoice_recovered_after_error",
			"payment_id", payment.ID, "invoice_id", inv.ID)
	}

	// 8. Simpan info invoice. Gagal di sini TIDAK menandai FAILED:
	// invoice bisa saja hidup dan bisa dibayar. Row dibiarkan PENDING
	// tanpa URL; attempt berikutnya menemukannya lewat external_id.
	if err := s.paymentRepo.UpdateInvoice(
		ctx,
		payment.ID,
		inv.ID,
		inv.CheckoutURL,
		inv.ExpiryDate,
	); err != nil {
		return CreatePaymentResult{}, fmt.Errorf("save xendit invoice: %w", err)
	}

	slog.InfoContext(ctx, "payment.invoice_created",
		"payment_id", payment.ID,
		"order_id", ord.ID,
		"invoice_id", inv.ID,
	)

	payment.ProviderInvoiceID = &inv.ID
	payment.CheckoutURL = &inv.CheckoutURL
	payment.ExpiresAt = inv.ExpiryDate

	return toCreatePaymentResult(payment), nil
}

// resolvePendingPayment menentukan nasib payment PENDING: reuse invoice
// yang hidup, mati terkonfirmasi (dead=true, aman dibuat attempt baru),
// atau error karena statusnya tak bisa dipastikan.
func (s *service) resolvePendingPayment(
	ctx context.Context,
	p models.Payment,
) (CreatePaymentResult, bool, error) {

	now := time.Now()

	// Invoice lokal masih valid → reuse tanpa menyentuh Xendit.
	if p.CheckoutURL != nil && *p.CheckoutURL != "" &&
		p.ExpiresAt != nil && now.Before(*p.ExpiresAt) {

		slog.InfoContext(ctx, "payment.invoice_reused",
			"payment_id", p.ID, "order_id", p.OrderID)
		return toCreatePaymentResult(p), false, nil
	}

	inv, err := s.xendit.FindLatestInvoiceByExternalID(ctx, p.ExternalID)
	if err != nil {
		// Status tak pasti: jangan mengubah state apa pun.
		return CreatePaymentResult{}, false,
			fmt.Errorf("%w: %v", ErrPaymentStateAmbiguous, err)
	}

	if inv == nil {
		// Xendit mengkonfirmasi invoice tidak pernah ada → FAILED aman.
		if merr := s.markFailed(ctx, p.ID); merr != nil {
			return CreatePaymentResult{}, false, merr
		}
		return CreatePaymentResult{}, true, nil
	}

	switch inv.Status {
	case xenditclient.InvoiceStatusExpired:
		if merr := s.markFailed(ctx, p.ID); merr != nil {
			return CreatePaymentResult{}, false, merr
		}
		return CreatePaymentResult{}, true, nil

	case xenditclient.InvoiceStatusPaid, xenditclient.InvoiceStatusSettled:
		// Invoice sudah dibayar: catat PAID. Finalisasi order
		// dilakukan fase webhook, bukan di sini.
		if err := s.paymentRepo.UpdateStatus(
			ctx, p.ID, string(PaymentStatusPaid), nil,
		); err != nil {
			return CreatePaymentResult{}, false, fmt.Errorf("mark payment paid: %w", err)
		}

		slog.InfoContext(ctx, "payment.paid_detected",
			"payment_id", p.ID, "order_id", p.OrderID)
		return CreatePaymentResult{}, false, ErrOrderNotPayable

	case xenditclient.InvoiceStatusPending:
		// Invoice hidup → recover info invoice ke DB bila belum tersimpan.
		if p.ProviderInvoiceID == nil || p.CheckoutURL == nil || *p.CheckoutURL == "" {
			if err := s.paymentRepo.UpdateInvoice(
				ctx, p.ID, inv.ID, inv.CheckoutURL, inv.ExpiryDate,
			); err != nil {
				return CreatePaymentResult{}, false,
					fmt.Errorf("recover invoice info: %w", err)
			}

			slog.InfoContext(ctx, "payment.invoice_recovered",
				"payment_id", p.ID, "invoice_id", inv.ID)
		}

		fresh, err := s.paymentRepo.FindByID(ctx, p.ID)
		if err != nil {
			return CreatePaymentResult{}, false, err
		}
		return toCreatePaymentResult(fresh), false, nil

	default:
		return CreatePaymentResult{}, false,
			fmt.Errorf("%w: unexpected invoice status %q",
				ErrPaymentStateAmbiguous, inv.Status)
	}
}

func (s *service) markFailed(ctx context.Context, paymentID uuid.UUID) error {
	if err := s.paymentRepo.UpdateStatus(
		ctx, paymentID, string(PaymentStatusFailed), nil,
	); err != nil {
		return fmt.Errorf("mark payment failed: %w", err)
	}

	slog.InfoContext(ctx, "payment.marked_failed", "payment_id", paymentID)
	return nil
}

func toCreatePaymentResult(
	p models.Payment,
) CreatePaymentResult {
	result := CreatePaymentResult{
		PaymentID: p.ID,
		OrderID:   p.OrderID,
		Status:    p.Status,
		Amount:    p.Amount.String(),
		Currency:  p.Currency,
		ExpiresAt: p.ExpiresAt,
	}

	if p.CheckoutURL != nil {
		result.CheckoutURL = *p.CheckoutURL
	}

	return result
}
