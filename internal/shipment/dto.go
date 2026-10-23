package shipment

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ShipmentResponse struct {
	ID              uuid.UUID `json:"id"`
	OrderID         uuid.UUID `json:"order_id"`
	Provider        string    `json:"provider"`
	ProviderOrderID *string   `json:"provider_order_id,omitempty"`

	CourierCompany *string `json:"courier_company,omitempty"`
	CourierType    *string `json:"courier_type,omitempty"`
	TrackingNumber *string `json:"tracking_number,omitempty"`
	TrackingURL    *string `json:"tracking_url,omitempty"`

	Status         string  `json:"status"`
	ProviderStatus *string `json:"provider_status,omitempty"`

	ShippingCost decimal.Decimal `json:"shipping_cost"`
	Currency     string          `json:"currency"`

	ShippedAt   *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateShipmentResult struct {
	ShipmentID     uuid.UUID `json:"shipment_id"`
	OrderID        uuid.UUID `json:"order_id"`
	Status         string    `json:"status"`
	TrackingNumber *string   `json:"tracking_number,omitempty"`
	TrackingURL    *string   `json:"tracking_url,omitempty"`
}

type ShipmentTrackingResponse struct {
	ShipmentID     string                  `json:"shipment_id"`
	OrderID        string                  `json:"order_id"`
	Status         string                  `json:"status"`
	TrackingNumber *string                 `json:"tracking_number,omitempty"`
	TrackingURL    *string                 `json:"tracking_url,omitempty"`
	Events         []ShipmentTrackingEvent `json:"events"`
}

type ShipmentTrackingEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// ShipmentFilter menyaring daftar shipment untuk admin. Nil nol =
// tanpa filter.
type ShipmentFilter struct {
	Status   *ShipmentStatus
	Page     int
	PageSize int
}

type ShipmentListResponse struct {
	Shipments []ShipmentResponse `json:"shipments"`
	Meta      ShipmentListMeta   `json:"meta"`
}

type ShipmentListMeta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// ShipmentDetailResponse: shipment + ringkasan order terkait.
type ShipmentDetailResponse struct {
	Shipment ShipmentResponse `json:"shipment"`
	Order    OrderSummary     `json:"order"`
}

type OrderSummary struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	OrderNumber string          `json:"order_number"`
	Status      string          `json:"status"`
	TotalAmount decimal.Decimal `json:"total_amount"`
	CreatedAt   time.Time       `json:"created_at"`
}
