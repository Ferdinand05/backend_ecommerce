package order

import "errors"

var (
	ErrorOrderNotFound           = errors.New("order not found")
	ErrorInvalidOrderStatus      = errors.New("invalid order status")
	ErrorInvalidStatusTransition = errors.New("invalid order status transition")
	ErrorEmptyCart               = errors.New("cart is empty")
	ErrorVariantUnavailable      = errors.New("product variant is unavailable")
	ErrorInvalidOrderInput       = errors.New("invalid order input")
	ErrorShippingRateNotFound    = errors.New("selected shipping service is unavailable")
	ErrorOrderNotCancellable     = errors.New("order cannot be cancelled")
)
