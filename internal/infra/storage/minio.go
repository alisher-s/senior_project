package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioService struct {
	client    *minio.Client
	transport *http.Transport
	bucket    string
	publicURL string
}

func NewMinIO(ctx context.Context) (Service, error) {
	endpoint := strings.TrimSpace(os.Getenv("MINIO_ENDPOINT"))
	accessKey := strings.TrimSpace(os.Getenv("MINIO_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("MINIO_SECRET_KEY"))
	bucket := strings.TrimSpace(os.Getenv("MINIO_BUCKET"))
	useSSLStr := strings.TrimSpace(os.Getenv("MINIO_USE_SSL"))
	publicURL := strings.TrimSpace(os.Getenv("MINIO_PUBLIC_URL"))

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" || publicURL == "" {
		return nil, errors.New("missing MINIO_* configuration")
	}

	useSSL := false
	if useSSLStr != "" {
		b, err := strconv.ParseBool(useSSLStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MINIO_USE_SSL: %w", err)
		}
		useSSL = b
	}

	transport, err := minio.DefaultTransport(useSSL)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:    useSSL,
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	policy := fmt.Sprintf(
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`,
		bucket,
	)
	if err := client.SetBucketPolicy(ctx, bucket, policy); err != nil {
		return nil, err
	}

	return &minioService{
		client:    client,
		transport: transport,
		bucket:    bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
	}, nil
}

func (s *minioService) UploadImage(ctx context.Context, objectName string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, objectName), nil
}

func (s *minioService) DeleteImage(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}

func (s *minioService) Close() error {
	if s.transport != nil {
		s.transport.CloseIdleConnections()
	}
	return nil
}

