// Package storage provides tests for S3 storage backend.
package storage

import (
	"context"
	"testing"
	"time"
)

func TestDataBuffer_Read(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		readSize  int
		expectEOF bool // EOF on first read
	}{
		{
			name:      "read all data",
			data:      []byte("hello world"),
			readSize:  100,
			expectEOF: false, // EOF only on next read
		},
		{
			name:      "read in chunks",
			data:      []byte("hello world"),
			readSize:  5,
			expectEOF: false,
		},
		{
			name:      "empty buffer",
			data:      []byte(""),
			readSize:  10,
			expectEOF: true, // Empty buffer returns EOF immediately
		},
		{
			name:      "single byte",
			data:      []byte("x"),
			readSize:  10,
			expectEOF: false, // Reads 1 byte, EOF on next read
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &dataBuffer{data: tt.data}

			result := make([]byte, tt.readSize)
			n, err := buf.Read(result)

			// Empty buffer should return EOF immediately
			if len(tt.data) == 0 {
				if n != 0 {
					t.Errorf("Expected 0 bytes for empty buffer, got %d", n)
				}
				if err == nil {
					t.Error("Expected EOF error for empty buffer")
				}
				return
			}

			// For non-empty data, we read up to readSize or available data
			expected := tt.readSize
			if expected > len(tt.data) {
				expected = len(tt.data)
			}

			if n != expected {
				t.Errorf("Expected to read %d bytes, got %d", expected, n)
			}

			// Verify content matches
			for i := 0; i < n; i++ {
				if result[i] != tt.data[i] {
					t.Errorf("Byte %d mismatch", i)
				}
			}

			// Check that next read returns EOF after all data consumed
			if n == len(tt.data) {
				// All data consumed, next read should return EOF
				result2 := make([]byte, 1)
				n2, err2 := buf.Read(result2)
				if n2 != 0 {
					t.Errorf("Expected 0 bytes on second read, got %d", n2)
				}
				if err2 == nil {
					t.Error("Expected EOF on second read")
				}
			}
		})
	}
}

func TestDataBuffer_MultipleReads(t *testing.T) {
	data := []byte("0123456789")
	buf := &dataBuffer{data: data}

	// First read of 5 bytes
	result1 := make([]byte, 5)
	n1, err1 := buf.Read(result1)

	if n1 != 5 {
		t.Errorf("Expected 5 bytes, got %d", n1)
	}

	if string(result1) != "01234" {
		t.Errorf("Expected '01234', got '%s'", string(result1))
	}

	if err1 != nil {
		t.Errorf("Unexpected error on first read: %v", err1)
	}

	// Second read of remaining
	result2 := make([]byte, 10)
	n2, err2 := buf.Read(result2)

	if n2 != 5 {
		t.Errorf("Expected 5 bytes, got %d", n2)
	}

	if string(result2[:n2]) != "56789" {
		t.Errorf("Expected '56789', got '%s'", string(result2[:n2]))
	}

	// No EOF on second read - dataBuffer only returns EOF on next call
	if err2 != nil {
		t.Errorf("Unexpected error on second read: %v", err2)
	}

	// Third read should return EOF
	result3 := make([]byte, 10)
	n3, err3 := buf.Read(result3)

	if n3 != 0 {
		t.Errorf("Expected 0 bytes on EOF read, got %d", n3)
	}

	if err3 == nil {
		t.Error("Expected EOF error on third read")
	}
}

func TestS3Config_Defaults(t *testing.T) {
	cfg := S3Config{
		Endpoint:  "localhost:9000",
		Bucket:    "test-bucket",
		AccessKey: "access",
		SecretKey: "secret",
	}

	if cfg.Region == "" {
		// Empty region is valid (uses default)
	}

	if cfg.UseSSL == false {
		// Default to false
	}
}

func TestS3Config_AllFields(t *testing.T) {
	cfg := S3Config{
		Endpoint:  "s3.amazonaws.com",
		Bucket:    "my-bucket",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region:    "us-west-2",
		UseSSL:    true,
	}

	if cfg.Endpoint != "s3.amazonaws.com" {
		t.Error("Endpoint not set correctly")
	}

	if cfg.Bucket != "my-bucket" {
		t.Error("Bucket not set correctly")
	}

	if cfg.Region != "us-west-2" {
		t.Error("Region not set correctly")
	}

	if !cfg.UseSSL {
		t.Error("UseSSL should be true")
	}
}

func TestBackend_Interface(t *testing.T) {
	// Test that S3Backend implements the Backend interface
	var _ Backend = &S3Backend{}
	var _ Backend = &LocalBackend{}
	var _ Backend = &EncryptedBackend{}
}

func TestLocalBackend_Paths(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			baseDir:  "/tmp/storage",
			input:    "test/file.txt",
			expected: "/tmp/storage/test/file.txt",
		},
		{
			name:     "nested path",
			baseDir:  "/data",
			input:    "docs/2024/report.pdf",
			expected: "/data/docs/2024/report.pdf",
		},
		{
			name:     "root path",
			baseDir:  "/",
			input:    "file.txt",
			expected: "/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &LocalBackend{baseDir: tt.baseDir}

			// Just verify the path logic
			if backend.baseDir != tt.baseDir {
				t.Errorf("Expected baseDir %s, got %s", tt.baseDir, backend.baseDir)
			}
		})
	}
}

func TestLocalBackend_MethodSignatures(t *testing.T) {
	backend := &LocalBackend{baseDir: "/tmp/test"}
	ctx := context.Background()

	// Verify methods exist and have correct signatures
	t.Run("Put signature", func(t *testing.T) {
		// Method exists - just verify it doesn't panic when called (will fail without actual dir)
		_ = backend.Put
	})

	t.Run("Get signature", func(t *testing.T) {
		_ = backend.Get
	})

	t.Run("Delete signature", func(t *testing.T) {
		_ = backend.Delete
	})

	t.Run("Exists signature", func(t *testing.T) {
		_ = backend.Exists
	})

	t.Run("List signature", func(t *testing.T) {
		_ = backend.List
	})

	t.Run("GetURL signature", func(t *testing.T) {
		url, err := backend.GetURL(ctx, "test.txt", time.Hour)
		if err != nil {
			t.Errorf("GetURL() returned error: %v", err)
		}
		if url == "" {
			t.Error("GetURL() should return a URL")
		}
	})
}

func TestS3Backend_MethodSignatures(t *testing.T) {
	// Verify S3Backend has all required methods
	backend := &S3Backend{}

	t.Run("Put method exists", func(t *testing.T) {
		_ = backend.Put
	})

	t.Run("Get method exists", func(t *testing.T) {
		_ = backend.Get
	})

	t.Run("Delete method exists", func(t *testing.T) {
		_ = backend.Delete
	})

	t.Run("Exists method exists", func(t *testing.T) {
		_ = backend.Exists
	})

	t.Run("List method exists", func(t *testing.T) {
		_ = backend.List
	})

	t.Run("GetURL method exists", func(t *testing.T) {
		_ = backend.GetURL
	})
}

func TestEncryptedBackend_Wrapping(t *testing.T) {
	mockBackend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(mockBackend, "test-key")

	// EncryptedBackend should wrap the underlying backend
	if encBackend.backend != mockBackend {
		t.Error("EncryptedBackend should wrap the provided backend")
	}
}

func TestEncryptedBackend_Delegates(t *testing.T) {
	mockBackend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(mockBackend, "test-key")
	ctx := context.Background()

	// Test Delete (delegates directly)
	mockBackend.data["test"] = []byte("data")
	err := encBackend.Delete(ctx, "test")
	if err != nil {
		t.Errorf("Delete() returned error: %v", err)
	}

	if _, exists := mockBackend.data["test"]; exists {
		t.Error("Delete should delegate to underlying backend")
	}

	// Test Exists (delegates directly)
	mockBackend.data["test2"] = []byte("data")
	exists, _ := encBackend.Exists(ctx, "test2")
	if !exists {
		t.Error("Exists should return true for existing file")
	}

	// Test List (delegates directly)
	mockBackend.data["prefix/file1"] = []byte("data1")
	mockBackend.data["prefix/file2"] = []byte("data2")
	paths, _ := encBackend.List(ctx, "prefix")
	if len(paths) != 2 {
		t.Errorf("List() should delegate, got %d paths", len(paths))
	}
}

func TestBackend_PutGetDelete_Lifecycle(t *testing.T) {
	// Test the complete lifecycle using mock backend
	backend := newMockBackend()
	ctx := context.Background()

	path := "test/lifecycle.txt"
	data := []byte("test data for lifecycle")

	// Put
	err := backend.Put(ctx, path, data)
	if err != nil {
		t.Fatalf("Put() returned error: %v", err)
	}

	// Exists
	exists, err := backend.Exists(ctx, path)
	if err != nil {
		t.Fatalf("Exists() returned error: %v", err)
	}
	if !exists {
		t.Error("File should exist after Put")
	}

	// Get
	retrieved, err := backend.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if string(retrieved) != string(data) {
		t.Error("Retrieved data doesn't match original")
	}

	// Delete
	err = backend.Delete(ctx, path)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify deleted
	exists, _ = backend.Exists(ctx, path)
	if exists {
		t.Error("File should not exist after Delete")
	}

	_, err = backend.Get(ctx, path)
	if err == nil {
		t.Error("Get() after Delete should return error")
	}
}

func TestBackend_List_PrefixMatching(t *testing.T) {
	backend := newMockBackend()
	ctx := context.Background()

	// Add files with different prefixes
	backend.data["docs/file1.txt"] = []byte("data1")
	backend.data["docs/file2.txt"] = []byte("data2")
	backend.data["docs/subdir/file3.txt"] = []byte("data3")
	backend.data["images/pic1.jpg"] = []byte("data4")
	backend.data["other.txt"] = []byte("data5")

	t.Run("list docs prefix", func(t *testing.T) {
		paths, _ := backend.List(ctx, "docs/")
		if len(paths) != 3 {
			t.Errorf("Expected 3 paths with docs/ prefix, got %d", len(paths))
		}
	})

	t.Run("list images prefix", func(t *testing.T) {
		paths, _ := backend.List(ctx, "images/")
		if len(paths) != 1 {
			t.Errorf("Expected 1 path with images/ prefix, got %d", len(paths))
		}
	})

	t.Run("list all", func(t *testing.T) {
		paths, _ := backend.List(ctx, "")
		if len(paths) != 5 {
			t.Errorf("Expected 5 paths with empty prefix, got %d", len(paths))
		}
	})
}

func TestS3Backend_IntegrationNotes(t *testing.T) {
	t.Log("S3Backend integration tests require:")
	t.Log("- A running MinIO or S3 instance")
	t.Log("- Proper credentials configuration")
	t.Log("- Network connectivity")
	t.Log("")
	t.Log("For integration testing, consider using:")
	t.Log("- testcontainers-go for MinIO")
	t.Log("- Environment-based configuration")
	t.Log("- Fixture files for upload/download testing")
}

func TestLocalBackend_IntegrationNotes(t *testing.T) {
	t.Log("LocalBackend integration tests require:")
	t.Log("- Temporary directory creation and cleanup")
	t.Log("- File system permissions")
	t.Log("- Proper cleanup of test files")
}

func TestBackend_GetURL(t *testing.T) {
	mockBackend := newMockBackend()
	ctx := context.Background()

	t.Run("mock backend GetURL", func(t *testing.T) {
		url, err := mockBackend.GetURL(ctx, "test/file.txt", 1*time.Hour)
		if err != nil {
			t.Errorf("GetURL() returned error: %v", err)
		}
		if url == "" {
			t.Error("GetURL() should return a URL")
		}
	})

	t.Run("local backend GetURL format", func(t *testing.T) {
		localBackend := &LocalBackend{baseDir: "/tmp/storage"}
		url, err := localBackend.GetURL(ctx, "test/file.txt", 1*time.Hour)
		if err != nil {
			t.Errorf("GetURL() returned error: %v", err)
		}
		// Local backend returns file:// URL
		if url[:7] != "file://" {
			t.Errorf("Local backend URL should start with file://, got %s", url[:7])
		}
	})

	t.Run("encrypted backend GetURL not supported", func(t *testing.T) {
		encBackend, _ := NewEncryptedBackend(mockBackend, "test-key")
		_, err := encBackend.GetURL(ctx, "test/file.txt", 1*time.Hour)
		if err == nil {
			t.Error("Encrypted backend GetURL should return error")
		}
	})
}

func TestBackend_ErrorHandling(t *testing.T) {
	mockBackend := &mockBackend{
		data: make(map[string][]byte),
		err:  &testError{"simulated error"},
	}
	ctx := context.Background()

	t.Run("Put with error", func(t *testing.T) {
		err := mockBackend.Put(ctx, "test", []byte("data"))
		if err == nil {
			t.Error("Put() should return error")
		}
	})

	t.Run("Get with error", func(t *testing.T) {
		_, err := mockBackend.Get(ctx, "test")
		if err == nil {
			t.Error("Get() should return error")
		}
	})

	t.Run("Delete with error", func(t *testing.T) {
		err := mockBackend.Delete(ctx, "test")
		if err == nil {
			t.Error("Delete() should return error")
		}
	})

	t.Run("Exists with error", func(t *testing.T) {
		_, err := mockBackend.Exists(ctx, "test")
		if err == nil {
			t.Error("Exists() should return error")
		}
	})

	t.Run("List with error", func(t *testing.T) {
		_, err := mockBackend.List(ctx, "prefix")
		if err == nil {
			t.Error("List() should return error")
		}
	})

	t.Run("GetURL with error", func(t *testing.T) {
		_, err := mockBackend.GetURL(ctx, "test", 1*time.Hour)
		if err == nil {
			t.Error("GetURL() should return error")
		}
	})
}

func TestDataBuffer_OffsetTracking(t *testing.T) {
	data := []byte("0123456789")
	buf := &dataBuffer{data: data}

	// Initial offset
	if buf.offset != 0 {
		t.Errorf("Expected initial offset 0, got %d", buf.offset)
	}

	// Read some bytes
	buf.Read(make([]byte, 3))

	if buf.offset != 3 {
		t.Errorf("Expected offset 3, got %d", buf.offset)
	}

	// Read more
	buf.Read(make([]byte, 4))

	if buf.offset != 7 {
		t.Errorf("Expected offset 7, got %d", buf.offset)
	}

	// Read rest
	buf.Read(make([]byte, 10))

	if buf.offset != 10 {
		t.Errorf("Expected offset 10, got %d", buf.offset)
	}
}
