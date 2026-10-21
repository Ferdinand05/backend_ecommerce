package payment

import "errors"

var (
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderNotPayable       = errors.New("order is not payable")
	ErrOrderPaymentExpired   = errors.New("order payment period has expired")
	ErrInvalidPayment        = errors.New("invalid payment")
	ErrPaymentStateAmbiguous = errors.New("payment state could not be verified")
)
