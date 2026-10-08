package order

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusPaid       OrderStatus = "PAID"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusShipped    OrderStatus = "SHIPPED"
	OrderStatusDelivered  OrderStatus = "DELIVERED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending,
		OrderStatusPaid,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func CanTransition(from, to OrderStatus) bool {
	switch from {
	case OrderStatusPending:
		return to == OrderStatusPaid ||
			to == OrderStatusCancelled

	case OrderStatusPaid:
		return to == OrderStatusProcessing ||
			to == OrderStatusCancelled

	case OrderStatusProcessing:
		return to == OrderStatusShipped ||
			to == OrderStatusCancelled

	case OrderStatusShipped:
		return to == OrderStatusDelivered

	case OrderStatusDelivered:
		return false

	case OrderStatusCancelled:
		return false

	default:
		return false
	}
}
