package productvariant

import "errors"

var (
	ErrorProductVariantNotFound  = errors.New("product variant not found")
	ErrorVariantSKUAlreadyExists = errors.New("variant sku already exists")
)
