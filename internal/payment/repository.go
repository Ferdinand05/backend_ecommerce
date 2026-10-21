package payment

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, payment *models.Payment) error
	FindByID(ctx context.Context, paymentID uuid.UUID) (models.Payment, error)
	FindByExternalID(ctx context.Context, externalID string) (models.Payment, error)
	FindByProviderInvoiceID(ctx context.Context, invoiceID string) (models.Payment, error)
	FindLatestByOrderID(ctx context.Context, orderID uuid.UUID) (models.Payment, error)

	UpdateInvoice(ctx context.Context, paymentID uuid.UUID, invoiceID string, checkoutURL string, expiresAt *time.Time) error
	UpdateStatus(ctx context.Context, paymentID uuid.UUID, status string, paidAt *time.Time) error

	FindByIDForUpdate(ctx context.Context, paymentID uuid.UUID) (models.Payment, error)
	MarkPaid(ctx context.Context, paymentID uuid.UUID, paidAt *time.Time) (bool, error)
	MarkExpiredPending(ctx context.Context, paymentID uuid.UUID) (bool, error)
	FailOtherPendingByOrderID(ctx context.Context, orderID uuid.UUID, excludePaymentID uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Create(ctx context.Context, payment *models.Payment) error {

	db := database.GetDB(ctx, r.db)
	err := db.Create(&payment).Error

	if err != nil {
		return fmt.Errorf("create payment:%w", err)
	}

	return nil
}

func (r *repository) FindByID(ctx context.Context, paymentID uuid.UUID) (models.Payment, error) {
	var payment models.Payment

	db := database.GetDB(ctx, r.db)
	err := db.First(&payment, "id = ?", paymentID).Error

	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Payment{}, ErrPaymentNotFound
		}
		return models.Payment{}, err
	}

	return payment, nil
}

func (r *repository) FindByExternalID(
	ctx context.Context,
	externalID string,
) (models.Payment, error) {
	var payment models.Payment

	err := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Where("external_id = ?", externalID).
		First(&payment).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Payment{}, ErrPaymentNotFound
	}
	if err != nil {
		return models.Payment{}, err
	}

	return payment, nil
}

func (r *repository) FindByProviderInvoiceID(
	ctx context.Context,
	invoiceID string,
) (models.Payment, error) {
	var payment models.Payment

	err := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Where("provider_invoice_id = ?", invoiceID).
		First(&payment).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Payment{}, ErrPaymentNotFound
	}
	if err != nil {
		return models.Payment{}, err
	}

	return payment, nil
}

// FindLatestByOrderID mengambil percobaan pembayaran terbaru
// untuk order tertentu.
func (r *repository) FindLatestByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
) (models.Payment, error) {
	var payment models.Payment

	err := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Order("id DESC").
		First(&payment).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Payment{}, ErrPaymentNotFound
	}
	if err != nil {
		return models.Payment{}, err
	}

	return payment, nil
}

func (r *repository) UpdateInvoice(
	ctx context.Context,
	paymentID uuid.UUID,
	invoiceID string,
	checkoutURL string,
	expiresAt *time.Time,
) error {
	result := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Updates(map[string]interface{}{
			"provider_invoice_id": invoiceID,
			"checkout_url":        checkoutURL,
			"expires_at":          expiresAt,
			"updated_at":          time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentNotFound
	}

	return nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	paymentID uuid.UUID,
	status string,
	paidAt *time.Time,
) error {
	result := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Updates(map[string]interface{}{
			"status":     status,
			"paid_at":    paidAt,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentNotFound
	}

	return nil
}

// FindByIDForUpdate mengambil payment dengan SELECT ... FOR UPDATE.
// Wajib dipanggil di dalam transaksi (pola database.InjectTx).
func (r *repository) FindByIDForUpdate(
	ctx context.Context,
	paymentID uuid.UUID,
) (models.Payment, error) {
	var payment models.Payment

	err := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&payment, "id = ?", paymentID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Payment{}, ErrPaymentNotFound
	}
	if err != nil {
		return models.Payment{}, fmt.Errorf("locking payment: %w", err)
	}

	return payment, nil
}

// MarkPaid mengubah payment menjadi PAID dari status apa pun kecuali
// PAID itu sendiri. Mengembalikan false bila payment sudah PAID
// (callback replay) sehingga proses menjadi no-op yang aman.
func (r *repository) MarkPaid(
	ctx context.Context,
	paymentID uuid.UUID,
	paidAt *time.Time,
) (bool, error) {
	result := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Model(&models.Payment{}).
		Where("id = ? AND status <> ?", paymentID, string(PaymentStatusPaid)).
		Updates(map[string]interface{}{
			"status":     string(PaymentStatusPaid),
			"paid_at":    paidAt,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return false, fmt.Errorf("marking payment paid: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// MarkExpiredPending mengubah payment PENDING menjadi EXPIRED.
// Payment PAID/FAILED tidak disentuh; false berarti tidak ada perubahan.
func (r *repository) MarkExpiredPending(
	ctx context.Context,
	paymentID uuid.UUID,
) (bool, error) {
	result := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Model(&models.Payment{}).
		Where("id = ? AND status = ?", paymentID, string(PaymentStatusPending)).
		Update("status", string(PaymentStatusExpired))

	if result.Error != nil {
		return false, fmt.Errorf("marking payment expired: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// FailOtherPendingByOrderID menutup attempt PENDING lain milik order
// yang sama setelah satu payment terkonfirmasi PAID, agar invoice lama
// tidak menggantung sebagai PENDING di database.
func (r *repository) FailOtherPendingByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
	excludePaymentID uuid.UUID,
) error {
	result := database.GetDB(ctx, r.db).
		WithContext(ctx).
		Model(&models.Payment{}).
		Where(
			"order_id = ? AND status = ? AND id <> ?",
			orderID, string(PaymentStatusPending), excludePaymentID,
		).
		Update("status", string(PaymentStatusFailed))

	if result.Error != nil {
		return fmt.Errorf("failing other pending payments: %w", result.Error)
	}

	return nil
}
