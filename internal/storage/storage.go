package storage

import (
	"context"
	"io"
)

type Provider interface {
    Upload(
        ctx context.Context,
        key string,
        file io.Reader,
        contentType string,
    ) error

    Delete(
        ctx context.Context,
        key string,
    ) error

    URL(key string) string
}