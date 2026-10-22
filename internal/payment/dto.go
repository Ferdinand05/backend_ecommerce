package payment

import (
	"time"

	"github.com/google/uuid"
)

type CreatePaymentResult struct {
	PaymentID   uuid.UUID  `json:"payment_id"`
	OrderID     uuid.UUID  `json:"order_id"`
	Status      string     `json:"status"`
	Amount      string     `json:"amount"`
	Currency    string     `json:"currency"`
	CheckoutURL string     `json:"checkout_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// PaymentResponse untuk list admin: tidak membocorkan data sensitif
// selain identifier invoice/checkout.
type PaymentResponse struct {
	ID                uuid.UUID  `json:"id"`
	OrderID           uuid.UUID  `json:"order_id"`
	Provider          string     `json:"provider"`
	ExternalID        string     `json:"external_id"`
	ProviderInvoiceID *string    `json:"provider_invoice_id"`
	Status            string     `json:"status"`
	Amount            string     `json:"amount"`
	Currency          string     `json:"currency"`
	CheckoutURL       *string    `json:"checkout_url"`
	ExpiresAt         *time.Time `json:"expires_at"`
	PaidAt            *time.Time `json:"paid_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PaymentListMeta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type PaymentListResponse struct {
	Payments []PaymentResponse `json:"payments"`
	Meta     PaymentListMeta   `json:"meta"`
}

// InvoiceCallback adalah payload callback invoice Xendit. Hanya field
// yang dipakai backend yang diambil; amount berupa number di JSON.
type InvoiceCallback struct {
	ID            string     `json:"id"`
	ExternalID    string     `json:"external_id"`
	Status        string     `json:"status"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	PaidAt        *time.Time `json:"paid_at"`
	PaymentMethod string     `json:"payment_method"`
}
