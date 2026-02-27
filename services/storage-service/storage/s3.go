// Package storage provides storage backend implementations for the storage service.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Backend defines the storage interface.
type Backend interface {
	Put(ctx context.Context, path string, data []byte) error
	Get(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	List(ctx context.Context, prefix string) ([]string, error)
	GetURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}

// S3Config holds S3 backend configuration.
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// S3Backend implements Backend using S3-compatible storage.
type S3Backend struct {
	client *minio.Client
	bucket string
}

// NewS3Backend creates a new S3 backend.
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	// Initialize minio client
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{
			Region: cfg.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &S3Backend{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Put stores data at the given path.
func (b *S3Backend) Put(ctx context.Context, path string, data []byte) error {
	reader := &dataBuffer{data: data}
	_, err := b.client.PutObject(ctx, b.bucket, path,
		io.NopCloser(reader),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})

	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}

	return nil
}

// Get retrieves data from the given path.
func (b *S3Backend) Get(ctx context.Context, path string) ([]byte, error) {
	obj, err := b.client.GetObject(ctx, b.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}

	return data, nil
}

// Delete removes data at the given path.
func (b *S3Backend) Delete(ctx context.Context, path string) error {
	return b.client.RemoveObject(ctx, b.bucket, path, minio.RemoveObjectOptions{})
}

// Exists checks if data exists at the given path.
func (b *S3Backend) Exists(ctx context.Context, path string) (bool, error) {
	_, err := b.client.StatObject(ctx, b.bucket, path, minio.StatObjectOptions{})
	if err != nil {
		// Check if it's a not found error
		if errorResponse, ok := err.(minio.ErrorResponse); ok && errorResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns all objects with the given prefix.
func (b *S3Backend) List(ctx context.Context, prefix string) ([]string, error) {
	objectsCh := b.client.ListObjects(ctx, b.bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	})

	var paths []string
	for object := range objectsCh {
		if object.Err != nil {
			return nil, object.Err
		}
		paths = append(paths, object.Key)
	}

	return paths, nil
}

// GetURL returns a presigned URL for temporary access.
func (b *S3Backend) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	url, err := b.client.PresignedGetObject(ctx, b.bucket, path, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// LocalBackend implements Backend using local filesystem.
type LocalBackend struct {
	baseDir string
}

// NewLocalStorage creates a new local filesystem backend.
func NewLocalStorage(baseDir string) (*LocalBackend, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}

	return &LocalBackend{baseDir: baseDir}, nil
}

// Put stores data at the given path.
func (b *LocalBackend) Put(ctx context.Context, path string, data []byte) error {
	fullPath := filepath.Join(b.baseDir, path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	return os.WriteFile(fullPath, data, 0644)
}

// Get retrieves data from the given path.
func (b *LocalBackend) Get(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(b.baseDir, path)
	return os.ReadFile(fullPath)
}

// Delete removes data at the given path.
func (b *LocalBackend) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(b.baseDir, path)
	return os.Remove(fullPath)
}

// Exists checks if data exists at the given path.
func (b *LocalBackend) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(b.baseDir, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns all files with the given prefix.
func (b *LocalBackend) List(ctx context.Context, prefix string) ([]string, error) {
	fullPath := filepath.Join(b.baseDir, prefix)

	var paths []string
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(b.baseDir, path)
			paths = append(paths, relPath)
		}
		return nil
	})

	return paths, err
}

// GetURL returns a file URL (not presigned, for local only).
func (b *LocalBackend) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "file://" + filepath.Join(b.baseDir, path), nil
}

// dataBuffer wraps a byte slice for io.Reader.
type dataBuffer struct {
	data   []byte
	offset int
}

func (b *dataBuffer) Read(p []byte) (int, error) {
	if b.offset >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.offset:])
	b.offset += n
	return n, nil
}
