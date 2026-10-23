package shipment

import (
	"context"
	"errors"
	"fmt"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, shipment *models.Shipment) error

	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*models.Shipment, error)

	FindByOrderID(
		ctx context.Context,
		orderID uuid.UUID,
	) (*models.Shipment, error)

	FindByIDForUpdate(
		ctx context.Context,
		id uuid.UUID,
	) (*models.Shipment, error)

	Update(ctx context.Context, shipment *models.Shipment) error

	// UpdateIfStatus menyimpan perubahan hanya bila status masih
	// `fromStatus`. RowsAffected == 0 berarti proses lain sudah
	// mengubah baris (race dua confirm serentak).
	UpdateIfStatus(
		ctx context.Context,
		shipment *models.Shipment,
		fromStatus ShipmentStatus,
	) (bool, error)

	FindByTrackingNumber(
		ctx context.Context,
		trackingNumber string,
	) (*models.Shipment, error)

	FindByProviderOrderID(
		ctx context.Context,
		providerOrderID string,
	) (*models.Shipment, error)

	// FindAll untuk data table admin: filter status opsional +
	// pagination, urut terbaru.
	FindAll(ctx context.Context, filter ShipmentFilter) (ShipmentListResponse, error)

	// FindByIDWithOrder memuat relasi Order untuk halaman detail.
	FindByIDWithOrder(ctx context.Context, id uuid.UUID) (*models.Shipment, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, shipment *models.Shipment) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(shipment).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateShipment
		}
		return fmt.Errorf("creating shipment: %w", err)
	}

	return nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipmentNotFound
		}
		return nil, fmt.Errorf("finding shipment: %w", err)
	}

	return &shipment, nil
}

func (r *repository) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Shipment, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var shipment models.Shipment
	if err := db.First(&shipment, "order_id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipmentNotFound
		}
		return nil, fmt.Errorf("finding shipment by order: %w", err)
	}

	return &shipment, nil
}

func (r *repository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var shipment models.Shipment
	if err := db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&shipment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipmentNotFound
		}
		return nil, fmt.Errorf("finding shipment for update: %w", err)
	}

	return &shipment, nil
}

// UpdateIfStatus menyimpan field hasil konfirmasi provider dengan guard
// status. Map dipakai (bukan struct) agar nilai nil ikut tersimpan.
func (r *repository) UpdateIfStatus(
	ctx context.Context,
	shipment *models.Shipment,
	fromStatus ShipmentStatus,
) (bool, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Model(&models.Shipment{}).
		Where("id = ? AND status = ?", shipment.ID, string(fromStatus)).
		Updates(map[string]any{
			"status":            shipment.Status,
			"provider_order_id": shipment.ProviderOrderID,
			"provider_status":   shipment.ProviderStatus,
			"tracking_number":   shipment.TrackingNumber,
			"tracking_url":      shipment.TrackingURL,
		})

	if result.Error != nil {
		return false, fmt.Errorf("updating shipment: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r *repository) Update(ctx context.Context, shipment *models.Shipment) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Save(shipment).Error; err != nil {
		return fmt.Errorf("saving shipment: %w", err)
	}

	return nil
}

func (r *repository) FindByTrackingNumber(ctx context.Context, trackingNumber string) (*models.Shipment, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var shipment models.Shipment
	if err := db.First(&shipment, "tracking_number = ?", trackingNumber).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipmentNotFound
		}
		return nil, fmt.Errorf("finding shipment by tracking number: %w", err)
	}

	return &shipment, nil
}

func (r *repository) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*models.Shipment, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var shipment models.Shipment
	if err := db.First(&shipment, "provider_order_id = ?", providerOrderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipmentNotFound
		}
		return nil, fmt.Errorf("finding shipment by provider order id: %w", err)
	}

	return &shipment, nil
}

func (r *repository) FindAll(ctx context.Context, filter ShipmentFilter) (ShipmentListResponse, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	query := db.Model(&models.Shipment{})

	if filter.Status != nil {
		query = query.Where("status = ?", string(*filter.Status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return ShipmentListResponse{}, fmt.Errorf("counting shipments: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize

	var shipments []models.Shipment
	if err := query.
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&shipments).Error; err != nil {
		return ShipmentListResponse{}, fmt.Errorf("finding shipments: %w", err)
	}

	items := make([]ShipmentResponse, len(shipments))
	for i, s := range shipments {
		items[i] = toShipmentResponse(s)
	}

	return ShipmentListResponse{
		Shipments: items,
		Meta: ShipmentListMeta{
			Total:    total,
			Page:     filter.Page,
			PageSize: filter.PageSize,
		},
	}, nil
}

func (r *repository) FindByIDWithOrder(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var shipment models.Shipment
	if err := db.Preload("Order").First(&shipment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipmentNotFound
		}
		return nil, fmt.Errorf("finding shipment with order: %w", err)
	}

	return &shipment, nil
}

func toShipmentResponse(shp models.Shipment) ShipmentResponse {
	return ShipmentResponse{
		ID:              shp.ID,
		OrderID:         shp.OrderID,
		Provider:        shp.Provider,
		ProviderOrderID: shp.ProviderOrderID,

		CourierCompany: shp.CourierCompany,
		CourierType:    shp.CourierType,
		TrackingNumber: shp.TrackingNumber,
		TrackingURL:    shp.TrackingURL,

		Status:         shp.Status,
		ProviderStatus: shp.ProviderStatus,

		ShippingCost: shp.ShippingCost,
		Currency:     shp.Currency,

		ShippedAt:   shp.ShippedAt,
		DeliveredAt: shp.DeliveredAt,
		CancelledAt: shp.CancelledAt,

		CreatedAt: shp.CreatedAt,
		UpdatedAt: shp.UpdatedAt,
	}
}
