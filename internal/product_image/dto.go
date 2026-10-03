package productimage

import "github.com/google/uuid"

type ProductImageResponse struct {
	ID               uuid.UUID  `json:"id"`
	ProductID        uuid.UUID  `json:"product_id"`
	ProductVariantID *uuid.UUID `json:"product_variant_id"`
	URL              string     `json:"url"`
	Alt              *string    `json:"alt"`
	SortOrder        int        `json:"sort_order"`
}