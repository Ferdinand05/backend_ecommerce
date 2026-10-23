package shipment

import "errors"

var (
	ErrShipmentNotFound        = errors.New("shipment not found")
	ErrDuplicateShipment       = errors.New("shipment already exists for this order")
	ErrInvalidShipmentData     = errors.New("shipment data is incomplete")
	ErrInvalidStatusTransition = errors.New("invalid shipment status transition")
	// Provider sudah punya order dengan reference_id yang sama, tetapi
	// ID-nya tidak tersimpan lokal (timeout ambigu). Resolusi manual.
	ErrProviderOrderExists = errors.New("provider order already exists")
)

// ErrProviderOrder sudah dibuat di provider tapi status lokal tak
// terverifikasi; aman dicoba lagi.
var ErrProviderStateAmbiguous = errors.New("provider order state could not be verified")
