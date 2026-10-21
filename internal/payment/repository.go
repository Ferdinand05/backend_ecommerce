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
)

type Repository interface {
	Create(ctx context.Context, payment *models.Payment) error
	FindByID(ctx context.Context, paymentID uuid.UUID) (models.Payment, error)
	FindByExternalID(ctx context.Context, externalID string) (models.Payment, error)
	FindByProviderInvoiceID(ctx context.Context, invoiceID string) (models.Payment, error)
	FindLatestByOrderID(ctx context.Context, orderID uuid.UUID) (models.Payment, error)

	UpdateInvoice(ctx context.Context, paymentID uuid.UUID, invoiceID string, checkoutURL string, expiresAt *time.Time) error
	UpdateStatus(ctx context.Context, paymentID uuid.UUID, status string, paidAt *time.Time) error
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
