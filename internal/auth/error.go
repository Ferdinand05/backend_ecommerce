package auth

import "errors"

var (
	ErrorEmailExists        = errors.New("email already exists")
	ErrorInvalidCredentials = errors.New("invalid credentials")

	ErrorInvalidVerificationToken = errors.New(
		"invalid verification token",
	)

	ErrorVerificationTokenExpired = errors.New(
		"verification token expired",
	)

	ErrorEmailAlreadyVerified = errors.New(
		"email already verified",
	)

	ErrorEmailNotVerified = errors.New(
		"email not verified",
	)

	ErrorInvalidRefreshToken = errors.New(
		"invalid refresh token",
	)

	ErrorRefreshTokenExpired = errors.New(
		"refresh token expired",
	)

	ErrorInvalidResetToken = errors.New(
		"invalid reset token",
	)

	ErrorResetTokenExpired = errors.New(
		"reset token expired",
	)

	ErrorResetTokenUsed = errors.New(
		"reset token used",
	)
)
