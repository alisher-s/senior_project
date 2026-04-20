package storage

import (
	"context"
	"io"
)

type Service interface {
	UploadImage(ctx context.Context, objectName string, r io.Reader, size int64, contentType string) (string, error)
	DeleteImage(ctx context.Context, objectName string) error
	Close() error
}

