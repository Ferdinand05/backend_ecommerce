package payment

import (
	"time"

	"github.com/google/uuid"
)

type CreatePaymentResult struct {
	PaymentID   uuid.UUID
	OrderID     uuid.UUID
	Status      string
	Amount      string
	Currency    string
	CheckoutURL string
	ExpiresAt   *time.Time
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
