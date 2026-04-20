package storage

import (
	"context"
	"errors"
	"io"
)

var ErrNotConfigured = errors.New("storage not configured")

type Service interface {
	UploadImage(ctx context.Context, objectName string, r io.Reader, size int64, contentType string) (string, error)
	DeleteImage(ctx context.Context, objectName string) error
	Close() error
}

