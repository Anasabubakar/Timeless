package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/sponsoros/backend/internal/config"
)

type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type UploadResult struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

type MinIOStorage struct {
	client *minio.Client
	bucket string
	public bool
}

func NewMinIO(cfg *config.Config) (*MinIOStorage, error) {
	if cfg.S3Endpoint == "" {
		return nil, fmt.Errorf("S3_ENDPOINT not configured")
	}

	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.S3Bucket,
	}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	objectURL := fmt.Sprintf("%s/%s/%s", s.client.EndpointURL().String(), s.bucket, key)

	return &UploadResult{
		Key:         key,
		URL:         objectURL,
		Size:        size,
		ContentType: contentType,
	}, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *MinIOStorage) GetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presigned.String(), nil
}

func GenerateKey(orgID, folder, filename string) string {
	ext := path.Ext(filename)
	ts := time.Now().UnixMilli()
	return fmt.Sprintf("orgs/%s/%s/%d%s", orgID, folder, ts, ext)
}
