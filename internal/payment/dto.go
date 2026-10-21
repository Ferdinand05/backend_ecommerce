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
