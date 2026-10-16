package checkout

import "errors"

var (
	ErrCartEmpty           = errors.New("cart is empty")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrVariantNotAvailable = errors.New("variant not available")
)
