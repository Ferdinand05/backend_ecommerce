package category

import (
	"time"

	"github.com/google/uuid"
)

type CreateCategoryRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
}

type UpdateCategoryRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
}

type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
