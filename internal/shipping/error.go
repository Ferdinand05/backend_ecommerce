package shipping

import "errors"

var (
	ErrInvalidAreaQuery    = errors.New("invalid area query")
	ErrAreaProvider        = errors.New("area provider error")
	ErrVariantNotAvailable = errors.New("variant not available")
	ErrInvalidRatesInput   = errors.New("invalid rates input")
)
