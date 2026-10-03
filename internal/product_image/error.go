package productimage

import "errors"

var (
	ErrorProductImageNotFound = errors.New("product image not found")
	ErrorUnsupportedImageType = errors.New("unsupported image type")
	ErrorImageTooLarge        = errors.New("image file too large")
)
