package product

import "errors"

var (
	ErrorProductNotFound          = errors.New("product not found")
	ErrorProductSlugAlreadyExists = errors.New("product slug already exists")
)