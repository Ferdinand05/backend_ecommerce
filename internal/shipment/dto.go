package shipment

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ShipmentResponse struct {
	ID              uuid.UUID       `json:"id"`
	OrderID         uuid.UUID       `json:"order_id"`
	Provider        string          `json:"provider"`
	ProviderOrderID *string         `json:"provider_order_id,omitempty"`

	CourierCompany  *string         `json:"courier_company,omitempty"`
	CourierType     *string         `json:"courier_type,omitempty"`
	TrackingNumber  *string         `json:"tracking_number,omitempty"`
	TrackingURL     *string         `json:"tracking_url,omitempty"`

	Status          string          `json:"status"`
	ProviderStatus  *string         `json:"provider_status,omitempty"`

	ShippingCost    decimal.Decimal `json:"shipping_cost"`
	Currency        string          `json:"currency"`

	ShippedAt       *time.Time      `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time      `json:"delivered_at,omitempty"`
	CancelledAt     *time.Time      `json:"cancelled_at,omitempty"`

	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CreateShipmentResult struct {
	ShipmentID    uuid.UUID `json:"shipment_id"`
	OrderID       uuid.UUID `json:"order_id"`
	Status        string    `json:"status"`
	TrackingNumber *string  `json:"tracking_number,omitempty"`
	TrackingURL    *string  `json:"tracking_url,omitempty"`
}

type ShipmentTrackingResponse struct {
	ShipmentID     string                   `json:"shipment_id"`
	OrderID        string                   `json:"order_id"`
	Status         string                   `json:"status"`
	TrackingNumber *string                  `json:"tracking_number,omitempty"`
	TrackingURL    *string                  `json:"tracking_url,omitempty"`
	Events         []ShipmentTrackingEvent  `json:"events"`
}

type ShipmentTrackingEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}