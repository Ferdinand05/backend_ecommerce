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
