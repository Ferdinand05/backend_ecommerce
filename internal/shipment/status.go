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

func (s ShipmentStatus) IsValid() bool {
	switch s {
	case ShipmentStatusPending,
		ShipmentStatusConfirmed,
		ShipmentStatusAllocated,
		ShipmentStatusPickingUp,
		ShipmentStatusPicked,
		ShipmentStatusDroppingOff,
		ShipmentStatusDelivered,
		ShipmentStatusCancelled,
		ShipmentStatusReturned,
		ShipmentStatusFailed:
		return true
	default:
		return false
	}
}

// CanTransition menjaga urutan status. Status provider (ALLOCATED dst.)
// dipetakan webhook Biteship nanti; saat ini hanya PENDING -> CONFIRMED
// yang dipakai aplikasi.
func CanTransition(from, to ShipmentStatus) bool {
	switch from {
	case ShipmentStatusPending:
		return to == ShipmentStatusConfirmed ||
			to == ShipmentStatusCancelled ||
			to == ShipmentStatusFailed

	case ShipmentStatusConfirmed,
		ShipmentStatusAllocated,
		ShipmentStatusPickingUp,
		ShipmentStatusPicked,
		ShipmentStatusDroppingOff:
		return to == ShipmentStatusAllocated ||
			to == ShipmentStatusPickingUp ||
			to == ShipmentStatusPicked ||
			to == ShipmentStatusDroppingOff ||
			to == ShipmentStatusDelivered ||
			to == ShipmentStatusReturned ||
			to == ShipmentStatusCancelled ||
			to == ShipmentStatusFailed

	case ShipmentStatusDelivered,
		ShipmentStatusCancelled,
		ShipmentStatusReturned,
		ShipmentStatusFailed:
		return false

	default:
		return false
	}
}
