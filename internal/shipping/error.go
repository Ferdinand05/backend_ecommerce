package shipping

import "errors"

var (
	ErrInvalidAreaQuery = errors.New("invalid area query")
	ErrAreaProvider     = errors.New("area provider error")
)
