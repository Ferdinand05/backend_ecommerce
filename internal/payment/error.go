package payment

import "errors"

var (
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderNotPayable       = errors.New("order is not payable")
	ErrOrderPaymentExpired   = errors.New("order payment period has expired")
	ErrInvalidPayment        = errors.New("invalid payment")
	ErrAmountMismatch        = errors.New("callback amount does not match payment")
	ErrPaymentStateAmbiguous = errors.New("payment state could not be verified")
	ErrInvalidInvoiceAmount  = errors.New("order total is not a valid invoice amount")
)
