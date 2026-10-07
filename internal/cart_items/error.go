package cartitems

import "errors"

var (
	ErrorCartItemNotFound       = errors.New("cart item not found")
	ErrorProductVariantInactive = errors.New("product variant is not available")
)
