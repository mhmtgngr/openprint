// Package testutil provides S3-compatible storage testcontainer setup for testing.
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// DefaultS3Port is the default S3 API port.
	DefaultS3Port = "9000"
	// DefaultS3ConsolePort is the default S3 console port.
	DefaultS3ConsolePort = "9001"
	// DefaultS3AccessKey is the default S3 access key.
	DefaultS3AccessKey = "minioadmin"
	// DefaultS3SecretKey is the default S3 secret key.
	DefaultS3SecretKey = "minioadmin"
	// DefaultS3Region is the default S3 region.
	DefaultS3Region = "us-east-1"
)

// TestS3 holds resources for an S3-compatible test container.
type TestS3 struct {
	Container testcontainers.Container
	Client    *minio.Client
	Host      string
	Port      string
	AccessKey string
	SecretKey string
	Region    string
	// UseHTTPS indicates whether to use HTTPS for connections.
	UseHTTPS bool
}

// SetupS3Container creates and starts a MinIO container for S3-compatible testing.
// It returns a TestS3 struct containing the container, client, and connection details.
//
// Usage in tests:
//
//	func TestMain(m *testing.M) {
//	    testS3, err := testutil.SetupS3Container(context.Background())
//	    if err != nil {
//	        log.Fatalf("Failed to setup test S3: %v", err)
//	    }
//	    defer testutil.CleanupS3(testS3)
//	    os.Exit(m.Run())
//	}
func SetupS3Container(ctx context.Context) (*TestS3, error) {
	return SetupS3ContainerWithConfig(ctx, S3ContainerConfig{
		AccessKey: DefaultS3AccessKey,
		SecretKey: DefaultS3SecretKey,
		Region:    DefaultS3Region,
	})
}

// S3ContainerConfig holds configuration for creating an S3 test container.
type S3ContainerConfig struct {
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string // Optional bucket to create on startup
	UseHTTPS  bool
}

// SetupS3ContainerWithConfig creates an S3 container with custom configuration.
func SetupS3ContainerWithConfig(ctx context.Context, cfg S3ContainerConfig) (*TestS3, error) {
	if cfg.AccessKey == "" {
		cfg.AccessKey = DefaultS3AccessKey
	}
	if cfg.SecretKey == "" {
		cfg.SecretKey = DefaultS3SecretKey
	}
	if cfg.Region == "" {
		cfg.Region = DefaultS3Region
	}

	// Create MinIO container request
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{DefaultS3Port + "/tcp", DefaultS3ConsolePort + "/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     cfg.AccessKey,
			"MINIO_ROOT_PASSWORD": cfg.SecretKey,
		},
		Cmd: []string{"server", "/data", "--console-address", ":" + DefaultS3ConsolePort},
		WaitingFor: wait.ForLog("API:").
			WithOccurrence(1).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio container: %w", err)
	}

	// Get the mapped port
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, DefaultS3Port)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container port: %w", err)
	}

	portStr := port.Port()

	// Build endpoint
	scheme := "http"
	if cfg.UseHTTPS {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s:%s", scheme, host, portStr)

	// Create MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseHTTPS,
		Region: cfg.Region,
	})
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// Test connection with retry
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var lastErr error
	for i := 0; i < 15; i++ {
		// Try to list buckets to verify connection
		_, err := client.ListBuckets(ctx)
		if err == nil {
			break
		}
		lastErr = err

		select {
		case <-ctx.Done():
			container.Terminate(ctx)
			return nil, fmt.Errorf("s3 connection timeout: %w", ctx.Err())
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}

	if lastErr != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("s3 connection failed: %w", lastErr)
	}

	testS3 := &TestS3{
		Container: container,
		Client:    client,
		Host:      host,
		Port:      portStr,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Region:    cfg.Region,
		UseHTTPS:  cfg.UseHTTPS,
	}

	// Create bucket if specified
	if cfg.Bucket != "" {
		if err := testS3.CreateBucket(ctx, cfg.Bucket); err != nil {
			CleanupS3(testS3)
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return testS3, nil
}

// CleanupS3 terminates the S3 container.
// It should be called in a defer statement after SetupS3Container.
func CleanupS3(ts *TestS3) {
	if ts == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if ts.Container != nil {
		if err := ts.Container.Terminate(ctx); err != nil {
			log.Printf("Warning: failed to terminate S3 container: %v", err)
		}
	}
}

// CreateBucket creates a bucket in the S3 test instance.
func (ts *TestS3) CreateBucket(ctx context.Context, bucketName string) error {
	if ts == nil || ts.Client == nil {
		return fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := ts.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
		Region: ts.Region,
	})
	if err != nil {
		return fmt.Errorf("make bucket %s: %w", bucketName, err)
	}

	return nil
}

// CreateBucketWithRetry creates a bucket with retry logic.
// Useful when the bucket might have been recently deleted.
func (ts *TestS3) CreateBucketWithRetry(ctx context.Context, bucketName string, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := ts.CreateBucket(ctx, bucketName)
		if err == nil {
			return nil
		}
		lastErr = err

		// Check if bucket already exists is okay
		if isBucketExists(err) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}
	return lastErr
}

// DeleteBucket removes a bucket from the S3 test instance.
func (ts *TestS3) DeleteBucket(ctx context.Context, bucketName string) error {
	if ts == nil || ts.Client == nil {
		return fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// First, delete all objects in the bucket
	objectsCh := make(chan minio.ObjectInfo)

	// List objects
	go func() {
		defer close(objectsCh)
		for object := range ts.Client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				log.Printf("Error listing object: %v", object.Err)
				return
			}
			objectsCh <- object
		}
	}()

	// Remove objects
	errorCh := ts.Client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{})

	for err := range errorCh {
		if err.Err != nil {
			return fmt.Errorf("remove object: %w", err.Err)
		}
	}

	// Now remove the bucket
	err := ts.Client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("remove bucket %s: %w", bucketName, err)
	}

	return nil
}

// PutObject puts an object into the S3 test instance.
func (ts *TestS3) PutObject(ctx context.Context, bucketName, objectName string, data []byte, contentType string) error {
	if ts == nil || ts.Client == nil {
		return fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := ts.Client.PutObject(ctx, bucketName, objectName,
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
			ContentType: contentType,
		})
	if err != nil {
		return fmt.Errorf("put object %s: %w", objectName, err)
	}

	return nil
}

// GetObject retrieves an object from the S3 test instance.
func (ts *TestS3) GetObject(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	if ts == nil || ts.Client == nil {
		return nil, fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	obj, err := ts.Client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", objectName, err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", objectName, err)
	}

	return data, nil
}

// ObjectExists checks if an object exists in the S3 test instance.
func (ts *TestS3) ObjectExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	if ts == nil || ts.Client == nil {
		return false, fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := ts.Client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		// Check if the error is "not found"
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ListObjects lists objects in a bucket.
func (ts *TestS3) ListObjects(ctx context.Context, bucketName string) ([]string, error) {
	if ts == nil || ts.Client == nil {
		return nil, fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var objects []string
	for object := range ts.Client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list objects: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

// DeleteObject deletes an object from the S3 test instance.
func (ts *TestS3) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	if ts == nil || ts.Client == nil {
		return fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := ts.Client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object %s: %w", objectName, err)
	}

	return nil
}

// EmptyBucket removes all objects from a bucket without deleting the bucket.
func (ts *TestS3) EmptyBucket(ctx context.Context, bucketName string) error {
	if ts == nil || ts.Client == nil {
		return fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for object := range ts.Client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				log.Printf("Error listing object: %v", object.Err)
				return
			}
			objectsCh <- object
		}
	}()

	errorCh := ts.Client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{})

	for err := range errorCh {
		if err.Err != nil {
			return fmt.Errorf("remove object: %w", err.Err)
		}
	}

	return nil
}

// GetEndpoint returns the S3 endpoint URL.
func (ts *TestS3) GetEndpoint() string {
	if ts == nil {
		return ""
	}
	scheme := "http"
	if ts.UseHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, ts.Host, ts.Port)
}

// GetEndpointURL returns the S3 endpoint as a URL struct.
func (ts *TestS3) GetEndpointURL() (*url.URL, error) {
	if ts == nil {
		return nil, fmt.Errorf("test s3 is nil")
	}
	scheme := "http"
	if ts.UseHTTPS {
		scheme = "https"
	}
	return url.Parse(fmt.Sprintf("%s://%s:%s", scheme, ts.Host, ts.Port))
}

// GetPresignedURL generates a presigned URL for an object.
func (ts *TestS3) GetPresignedURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	if ts == nil || ts.Client == nil {
		return "", fmt.Errorf("s3 client is nil")
	}

	presignedURL, err := ts.Client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presigned get object: %w", err)
	}

	return presignedURL.String(), nil
}

// SetupS3ForTest is a convenience function that sets up an S3 container
// and calls testing.Main with cleanup. Use this in TestMain for simpler setup.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    os.Exit(testutil.SetupS3ForTest(m))
//	}
func SetupS3ForTest(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	testS3, err := SetupS3Container(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test S3: %v", err)
	}
	defer CleanupS3(testS3)

	return m.Run()
}

// SetupS3WithBucket creates an S3 container with a pre-created bucket.
func SetupS3WithBucket(ctx context.Context, bucketName string) (*TestS3, error) {
	testS3, err := SetupS3Container(ctx)
	if err != nil {
		return nil, err
	}

	if err := testS3.CreateBucket(ctx, bucketName); err != nil {
		CleanupS3(testS3)
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	return testS3, nil
}

// BucketExists checks if a bucket exists in the S3 test instance.
func (ts *TestS3) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if ts == nil || ts.Client == nil {
		return false, fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exists, err := ts.Client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, fmt.Errorf("check bucket exists: %w", err)
	}

	return exists, nil
}

// isNotFound checks if an error indicates that a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Check for MinIO/S3 specific error codes
	errStr := err.Error()
	return contains(errStr, "not found") || contains(errStr, "NoSuchKey")
}

// isBucketExists checks if an error indicates that a bucket already exists.
func isBucketExists(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "already exists") || contains(errStr, "BucketAlreadyOwnedByYou") || contains(errStr, "BucketAlreadyExists")
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetObjectInfo returns metadata about an object.
func (ts *TestS3) GetObjectInfo(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	if ts == nil || ts.Client == nil {
		return minio.ObjectInfo{}, fmt.Errorf("s3 client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	info, err := ts.Client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}

	return info, nil
}
