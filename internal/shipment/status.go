package shipment

type ShipmentStatus string

var (
	ShipmentStatusPending     ShipmentStatus = "PENDING"
	ShipmentStatusConfirmed   ShipmentStatus = "CONFIRMED"
	ShipmentStatusAllocated   ShipmentStatus = "ALLOCATED"
	ShipmentStatusPickingUp   ShipmentStatus = "PICKING_UP"
	ShipmentStatusPicked      ShipmentStatus = "PICKED"
	ShipmentStatusDroppingOff ShipmentStatus = "DROPPING_OFF"
	ShipmentStatusDelivered   ShipmentStatus = "DELIVERED"
	ShipmentStatusCancelled   ShipmentStatus = "CANCELLED"
	ShipmentStatusReturned    ShipmentStatus = "RETURNED"
	ShipmentStatusFailed      ShipmentStatus = "FAILED"
)