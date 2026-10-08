package order

import (
	"time"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
)

// OrderFilter hanya untuk query admin. Field pointer = filter opsional.
// EndDate bersifat eksklusif; handler menggeser +1 hari dari input date-only.
type OrderFilter struct {
	UserID *uuid.UUID

	Status *OrderStatus

	OrderNumber *string

	StartDate *time.Time
	EndDate   *time.Time

	Page     int
	PageSize int
}

type OrderListResult struct {
	Orders   []models.Order
	Total    int64
	Page     int
	PageSize int
}
