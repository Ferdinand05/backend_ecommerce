package user

import (
	"errors"
	"ferdinand/ecommerce/internal/role"
)

var ErrorUserNotFound = errors.New("user not found")

// ErrorInvalidUserStatus dipakai saat status user di luar nilai yang
// diizinkan database ('active' atau 'blocked').
var ErrorInvalidUserStatus = errors.New("invalid user status")

// ErrorRoleNotFound memakai sentinel dari package role agar konsisten
// dengan pemanggilan role repository.
var ErrorRoleNotFound = role.ErrorRoleNotFound