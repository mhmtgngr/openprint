//go:build integration

package testutil

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupS3Container(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	require.NotNil(t, ts)
	defer CleanupS3(ts)

	// Verify container is running
	assert.NotNil(t, ts.Container)

	// Verify MinIO client is created
	assert.NotNil(t, ts.Client)

	// Verify connection details
	assert.NotEmpty(t, ts.Host)
	assert.NotEmpty(t, ts.Port)
	assert.Equal(t, DefaultS3AccessKey, ts.AccessKey)
	assert.Equal(t, DefaultS3SecretKey, ts.SecretKey)
	assert.Equal(t, DefaultS3Region, ts.Region)
	assert.False(t, ts.UseHTTPS)
}

func TestSetupS3Container_ListBuckets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// List buckets should work
	buckets, err := ts.Client.ListBuckets(ctx)
	require.NoError(t, err)
	assert.NotNil(t, buckets)
}

func TestSetupS3ContainerWithConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	customConfig := S3ContainerConfig{
		AccessKey: "customaccess",
		SecretKey: "customsecret",
		Region:    "eu-west-1",
		Bucket:    "test-bucket",
	}

	ts, err := SetupS3ContainerWithConfig(ctx, customConfig)
	require.NoError(t, err)
	defer CleanupS3(ts)

	assert.Equal(t, "customaccess", ts.AccessKey)
	assert.Equal(t, "customsecret", ts.SecretKey)
	assert.Equal(t, "eu-west-1", ts.Region)

	// Verify bucket was created
	exists, err := ts.BucketExists(ctx, "test-bucket")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSetupS3ContainerWithConfig_DefaultValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3ContainerWithConfig(ctx, S3ContainerConfig{})
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Should use default values
	assert.Equal(t, DefaultS3AccessKey, ts.AccessKey)
	assert.Equal(t, DefaultS3SecretKey, ts.SecretKey)
	assert.Equal(t, DefaultS3Region, ts.Region)
}

func TestCleanupS3_NilTestS3(t *testing.T) {
	// Should not panic with nil
	CleanupS3(nil)
}

func TestCleanupS3_NilContainer(t *testing.T) {
	ts := &TestS3{Container: nil}
	CleanupS3(ts)
}

func TestCreateBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Create a bucket
	err = ts.CreateBucket(ctx, "test-bucket")
	require.NoError(t, err)

	// Verify bucket exists
	exists, err := ts.BucketExists(ctx, "test-bucket")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateBucket_NilTestS3(t *testing.T) {
	var ts *TestS3
	err := ts.CreateBucket(context.Background(), "bucket")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestCreateBucketWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Create bucket with retry
	err = ts.CreateBucketWithRetry(ctx, "retry-bucket", 3)
	require.NoError(t, err)

	// Verify bucket exists
	exists, err := ts.BucketExists(ctx, "retry-bucket")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestDeleteBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Create and populate a bucket
	err = ts.CreateBucket(ctx, "delete-bucket")
	require.NoError(t, err)

	err = ts.PutObject(ctx, "delete-bucket", "test.txt", []byte("test data"), "text/plain")
	require.NoError(t, err)

	// Delete bucket
	err = ts.DeleteBucket(ctx, "delete-bucket")
	require.NoError(t, err)

	// Verify bucket is gone
	exists, err := ts.BucketExists(ctx, "delete-bucket")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDeleteBucket_WithObjects(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Create bucket and add multiple objects
	err = ts.CreateBucket(ctx, "objects-bucket")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = ts.PutObject(ctx, "objects-bucket", "file.txt", []byte("data"), "text/plain")
		require.NoError(t, err)
	}

	// Delete bucket should remove all objects first
	err = ts.DeleteBucket(ctx, "objects-bucket")
	require.NoError(t, err)

	exists, err := ts.BucketExists(ctx, "objects-bucket")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPutObject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Create bucket
	err = ts.CreateBucket(ctx, "put-bucket")
	require.NoError(t, err)

	// Put object
	data := []byte("test content")
	err = ts.PutObject(ctx, "put-bucket", "test.txt", data, "text/plain")
	require.NoError(t, err)

	// Verify object exists
	exists, err := ts.ObjectExists(ctx, "put-bucket", "test.txt")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestPutObject_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "large-bucket")
	require.NoError(t, err)

	// Create a 5MB data chunk
	data := make([]byte, 5*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err = ts.PutObject(ctx, "large-bucket", "large.bin", data, "application/octet-stream")
	require.NoError(t, err)

	// Verify size
	info, err := ts.GetObjectInfo(ctx, "large-bucket", "large.bin")
	require.NoError(t, err)
	assert.Equal(t, int64(5*1024*1024), info.Size)
}

func TestPutObject_NilTestS3(t *testing.T) {
	var ts *TestS3
	err := ts.PutObject(context.Background(), "bucket", "key", []byte("data"), "type")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestGetObject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "get-bucket")
	require.NoError(t, err)

	// Put object first
	originalData := []byte("test content for get")
	err = ts.PutObject(ctx, "get-bucket", "get-test.txt", originalData, "text/plain")
	require.NoError(t, err)

	// Get object
	retrievedData, err := ts.GetObject(ctx, "get-bucket", "get-test.txt")
	require.NoError(t, err)
	assert.Equal(t, originalData, retrievedData)
}

func TestGetObject_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "get-bucket")
	require.NoError(t, err)

	// Try to get non-existent object
	_, err = ts.GetObject(ctx, "get-bucket", "non-existent.txt")
	assert.Error(t, err)
}

func TestGetObject_NilTestS3(t *testing.T) {
	var ts *TestS3
	_, err := ts.GetObject(context.Background(), "bucket", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestObjectExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "exists-bucket")
	require.NoError(t, err)

	// Object doesn't exist yet
	exists, err := ts.ObjectExists(ctx, "exists-bucket", "test.txt")
	require.NoError(t, err)
	assert.False(t, exists)

	// Put object
	err = ts.PutObject(ctx, "exists-bucket", "test.txt", []byte("data"), "text/plain")
	require.NoError(t, err)

	// Now it exists
	exists, err = ts.ObjectExists(ctx, "exists-bucket", "test.txt")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestObjectExists_NilTestS3(t *testing.T) {
	var ts *TestS3
	_, err := ts.ObjectExists(context.Background(), "bucket", "key")
	assert.Error(t, err)
}

func TestListObjects(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "list-bucket")
	require.NoError(t, err)

	// Empty bucket
	objects, err := ts.ListObjects(ctx, "list-bucket")
	require.NoError(t, err)
	assert.Empty(t, objects)

	// Add objects
	for i := 0; i < 5; i++ {
		err = ts.PutObject(ctx, "list-bucket", "file.txt", []byte("data"), "text/plain")
		require.NoError(t, err)
	}

	// List objects
	objects, err = ts.ListObjects(ctx, "list-bucket")
	require.NoError(t, err)
	assert.Len(t, objects, 5)
}

func TestListObjects_NilTestS3(t *testing.T) {
	var ts *TestS3
	_, err := ts.ListObjects(context.Background(), "bucket")
	assert.Error(t, err)
}

func TestDeleteObject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "delete-obj-bucket")
	require.NoError(t, err)

	// Put object
	err = ts.PutObject(ctx, "delete-obj-bucket", "delete-me.txt", []byte("data"), "text/plain")
	require.NoError(t, err)

	// Verify exists
	exists, err := ts.ObjectExists(ctx, "delete-obj-bucket", "delete-me.txt")
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete object
	err = ts.DeleteObject(ctx, "delete-obj-bucket", "delete-me.txt")
	require.NoError(t, err)

	// Verify gone
	exists, err = ts.ObjectExists(ctx, "delete-obj-bucket", "delete-me.txt")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDeleteObject_NilTestS3(t *testing.T) {
	var ts *TestS3
	err := ts.DeleteObject(context.Background(), "bucket", "key")
	assert.Error(t, err)
}

func TestEmptyBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "empty-bucket")
	require.NoError(t, err)

	// Add objects
	for i := 0; i < 5; i++ {
		err = ts.PutObject(ctx, "empty-bucket", "file.txt", []byte("data"), "text/plain")
		require.NoError(t, err)
	}

	// List objects
	objects, err := ts.ListObjects(ctx, "empty-bucket")
	require.NoError(t, err)
	assert.NotEmpty(t, objects)

	// Empty bucket
	err = ts.EmptyBucket(ctx, "empty-bucket")
	require.NoError(t, err)

	// Verify empty
	objects, err = ts.ListObjects(ctx, "empty-bucket")
	require.NoError(t, err)
	assert.Empty(t, objects)
}

func TestGetEndpoint(t *testing.T) {
	t.Run("nil TestS3 returns empty string", func(t *testing.T) {
		var ts *TestS3
		endpoint := ts.GetEndpoint()
		assert.Empty(t, endpoint)
	})

	t.Run("http endpoint", func(t *testing.T) {
		ts := &TestS3{Host: "localhost", Port: "9000", UseHTTPS: false}
		endpoint := ts.GetEndpoint()
		assert.Equal(t, "http://localhost:9000", endpoint)
	})

	t.Run("https endpoint", func(t *testing.T) {
		ts := &TestS3{Host: "localhost", Port: "9000", UseHTTPS: true}
		endpoint := ts.GetEndpoint()
		assert.Equal(t, "https://localhost:9000", endpoint)
	})
}

func TestGetEndpointURL(t *testing.T) {
	t.Run("nil TestS3 returns error", func(t *testing.T) {
		var ts *TestS3
		_, err := ts.GetEndpointURL()
		assert.Error(t, err)
	})

	t.Run("valid TestS3 returns URL", func(t *testing.T) {
		ts := &TestS3{Host: "localhost", Port: "9000", UseHTTPS: false}
		url, err := ts.GetEndpointURL()
		require.NoError(t, err)
		assert.Equal(t, "http", url.Scheme)
		assert.Equal(t, "localhost", url.Hostname())
		assert.Equal(t, "9000", url.Port())
	})
}

func TestGetPresignedURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "presign-bucket")
	require.NoError(t, err)

	// Put object
	err = ts.PutObject(ctx, "presign-bucket", "presign.txt", []byte("data"), "text/plain")
	require.NoError(t, err)

	// Get presigned URL
	url, err := ts.GetPresignedURL(ctx, "presign-bucket", "presign.txt", time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "presign-bucket")
	assert.Contains(t, url, "presign.txt")
}

func TestGetPresignedURL_NilTestS3(t *testing.T) {
	var ts *TestS3
	_, err := ts.GetPresignedURL(context.Background(), "bucket", "key", time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestSetupS3WithBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3WithBucket(ctx, "auto-bucket")
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Verify bucket exists
	exists, err := ts.BucketExists(ctx, "auto-bucket")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestBucketExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	// Non-existent bucket
	exists, err := ts.BucketExists(ctx, "non-existent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Create bucket
	err = ts.CreateBucket(ctx, "test-bucket")
	require.NoError(t, err)

	// Now it exists
	exists, err = ts.BucketExists(ctx, "test-bucket")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestBucketExists_NilTestS3(t *testing.T) {
	var ts *TestS3
	_, err := ts.BucketExists(context.Background(), "bucket")
	assert.Error(t, err)
}

func TestGetObjectInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "info-bucket")
	require.NoError(t, err)

	// Put object
	data := []byte("test data")
	err = ts.PutObject(ctx, "info-bucket", "info.txt", data, "text/plain")
	require.NoError(t, err)

	// Get object info
	info, err := ts.GetObjectInfo(ctx, "info-bucket", "info.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), info.Size)
	assert.Equal(t, "text/plain", info.ContentType)
}

func TestGetObjectInfo_NilTestS3(t *testing.T) {
	var ts *TestS3
	_, err := ts.GetObjectInfo(context.Background(), "bucket", "key")
	assert.Error(t, err)
}

func TestSetupS3ForTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Smoke test for SetupS3ForTest
	done := make(chan int, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		ts, err := SetupS3Container(ctx)
		require.NoError(t, err)
		CleanupS3(ts)
		cancel()
		done <- 1
	}()

	select {
	case <-done:
		// Success
	case <-time.After(4 * time.Minute):
		t.Fatal("timeout waiting for test setup")
	}
}

func TestTestS3_HierarchicalKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "hierarchy-bucket")
	require.NoError(t, err)

	// Put objects with hierarchical keys
	objects := []string{
		"folder1/file1.txt",
		"folder1/file2.txt",
		"folder2/subfolder/file3.txt",
		"root.txt",
	}

	for _, obj := range objects {
		err = ts.PutObject(ctx, "hierarchy-bucket", obj, []byte("data"), "text/plain")
		require.NoError(t, err)
	}

	// List all objects
	listed, err := ts.ListObjects(ctx, "hierarchy-bucket")
	require.NoError(t, err)
	assert.Len(t, listed, len(objects))
}

func TestTestS3_ObjectWithSpecialChars(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "special-bucket")
	require.NoError(t, err)

	// Objects with special characters
	specialNames := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
		"文件名.txt", // Unicode
	}

	for _, name := range specialNames {
		err = ts.PutObject(ctx, "special-bucket", name, []byte("data"), "text/plain")
		require.NoError(t, err)
	}

	// Verify all can be retrieved
	for _, name := range specialNames {
		data, err := ts.GetObject(ctx, "special-bucket", name)
		require.NoError(t, err)
		assert.Equal(t, []byte("data"), data)
	}
}

func TestTestS3_MultipartUploadSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)
	defer CleanupS3(ts)

	err = ts.CreateBucket(ctx, "multipart-bucket")
	require.NoError(t, err)

	// Create a large object using direct client (simulating multipart)
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Upload directly using MinIO client for large file
	_, err = ts.Client.PutObject(ctx, "multipart-bucket", "large-file.dat",
		bytes.NewReader(largeData), int64(len(largeData)), minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
	require.NoError(t, err)

	// Verify by getting object info
	info, err := ts.GetObjectInfo(ctx, "multipart-bucket", "large-file.dat")
	require.NoError(t, err)
	assert.Equal(t, int64(10*1024*1024), info.Size)

	// Download and verify
	obj, err := ts.Client.GetObject(ctx, "multipart-bucket", "large-file.dat", minio.GetObjectOptions{})
	require.NoError(t, err)
	defer obj.Close()

	downloaded, err := io.ReadAll(obj)
	require.NoError(t, err)
	assert.Equal(t, len(largeData), len(downloaded))
}

func TestCleanupS3_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ts, err := SetupS3Container(ctx)
	require.NoError(t, err)

	// Multiple cleanup calls should not panic
	CleanupS3(ts)
	CleanupS3(ts)
	CleanupS3(nil)
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"not found error", &testError{msg: "The specified key does not exist"}, true},
		{"other error", &testError{msg: "access denied"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBucketExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"already exists error", &testError{msg: "BucketAlreadyExists"}, true},
		{"owned by you error", &testError{msg: "BucketAlreadyOwnedByYou"}, true},
		{"other error", &testError{msg: "access denied"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBucketExists(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// testError is a helper for testing error detection functions
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
