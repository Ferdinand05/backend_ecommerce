package inventory

import "errors"

var (
	ErrorInventoryNotFound       = errors.New("inventory item not found")
	ErrorStockMovementNotFound   = errors.New("stock movement not found")
	ErrorInsufficientStock       = errors.New("insufficient stock")
	ErrorInvalidMovementType     = errors.New("invalid stock movement type")
	ErrorInvalidMovementQuantity = errors.New("stock movement quantity must be greater than zero")
)
