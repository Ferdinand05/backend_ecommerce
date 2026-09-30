package role

import (
	"time"

	"github.com/google/uuid"
)

type RoleResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateRoleRequest struct {
	Name string `json:"name" binding:"required"`
}
