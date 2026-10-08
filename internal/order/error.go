package order

import "errors"

var (
	ErrorOrderNotFound           = errors.New("order not found")
	ErrorInvalidOrderStatus      = errors.New("invalid order status")
	ErrorInvalidStatusTransition = errors.New("invalid order status transition")
)
