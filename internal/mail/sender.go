package mail

import (
	"context"
)

type Sender interface {
	SendVerificationEmail(ctx context.Context, to string, token string) error
	SendPasswordResetEmail(ctx context.Context, to string, token string) error
}
