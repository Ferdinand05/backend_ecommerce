package product

import (
	"time"

	"github.com/google/uuid"
)

type CreateProductRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required,uuid"`
	Name        string    `json:"name" binding:"required,max=150"`
	Description *string   `json:"description"`
}

type UpdateProductRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required,uuid"`
	Name        string    `json:"name" binding:"required,max=150"`
	Description *string   `json:"description"`
}

type ProductResponse struct {
	ID          uuid.UUID `json:"id"`
	CategoryID  uuid.UUID `json:"category_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
