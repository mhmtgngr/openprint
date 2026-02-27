// Package storage provides tests for client-side encryption.
package storage

import (
	"context"
	"testing"
	"time"
)

// mockBackend is a mock storage backend for testing
type mockBackend struct {
	data map[string][]byte
	err  error
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		data: make(map[string][]byte),
	}
}

func (m *mockBackend) Put(ctx context.Context, path string, data []byte) error {
	m.data[path] = data
	return m.err
}

func (m *mockBackend) Get(ctx context.Context, path string) ([]byte, error) {
	data, ok := m.data[path]
	if !ok {
		return nil, &testError{"not found"}
	}
	return data, m.err
}

func (m *mockBackend) Delete(ctx context.Context, path string) error {
	delete(m.data, path)
	return m.err
}

func (m *mockBackend) Exists(ctx context.Context, path string) (bool, error) {
	_, ok := m.data[path]
	return ok, m.err
}

func (m *mockBackend) List(ctx context.Context, prefix string) ([]string, error) {
	var paths []string
	for p := range m.data {
		if len(p) >= len(prefix) && p[:len(prefix)] == prefix {
			paths = append(paths, p)
		}
	}
	return paths, m.err
}

func (m *mockBackend) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "http://example.com/" + path, m.err
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNewEncryptedBackend(t *testing.T) {
	backend := newMockBackend()

	encBackend, err := NewEncryptedBackend(backend, "test-key-12345")
	if err != nil {
		t.Fatalf("NewEncryptedBackend() returned error: %v", err)
	}

	if encBackend == nil {
		t.Fatal("NewEncryptedBackend() returned nil")
	}

	if encBackend.backend != backend {
		t.Error("Backend not set correctly")
	}
}

func TestNewEncryptedBackend_KeyDerivation(t *testing.T) {
	backend := newMockBackend()

	key1 := "test-key"
	key2 := "test-key"
	key3 := "different-key"

	enc1, _ := NewEncryptedBackend(backend, key1)
	enc2, _ := NewEncryptedBackend(backend, key2)
	enc3, _ := NewEncryptedBackend(backend, key3)

	// Same input key should derive same encryption key
	if enc1.key != enc2.key {
		t.Error("Same input key should derive same encryption key")
	}

	// Different input key should derive different encryption key
	if enc1.key == enc3.key {
		t.Error("Different input keys should derive different encryption keys")
	}
}

func TestEncryptedBackend_PutAndGet(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	originalData := []byte("sensitive data to encrypt")
	path := "test/file.txt"

	// Put encrypted data
	err := encBackend.Put(ctx, path, originalData)
	if err != nil {
		t.Fatalf("Put() returned error: %v", err)
	}

	// Verify data in backend is encrypted (not equal to original)
	storedData := backend.data[path]
	if string(storedData) == string(originalData) {
		t.Error("Stored data should be encrypted, not plaintext")
	}

	// Encrypted data should be longer than original (nonce + overhead)
	if len(storedData) <= len(originalData) {
		t.Error("Encrypted data should be longer than original")
	}

	// Get and decrypt
	retrievedData, err := encBackend.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	// Retrieved data should match original
	if string(retrievedData) != string(originalData) {
		t.Errorf("Retrieved data doesn't match original.\nGot: %s\nWant: %s", string(retrievedData), string(originalData))
	}
}

func TestEncryptedBackend_Get_NotFound(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	_, err := encBackend.Get(ctx, "nonexistent/file.txt")
	if err == nil {
		t.Error("Get() of nonexistent file should return error")
	}
}

func TestEncryptedBackend_Get_TooShort(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	// Put data that's too short to be valid encrypted data (less than nonce size)
	backend.data["short"] = []byte("too short")

	_, err := encBackend.Get(ctx, "short")
	if err == nil {
		t.Error("Get() of too-short data should return error")
	}
}

func TestEncryptedBackend_Get_WrongKey(t *testing.T) {
	backend1 := newMockBackend()
	encBackend1, _ := NewEncryptedBackend(backend1, "key-1")
	ctx := context.Background()

	originalData := []byte("secret data")
	path := "test/encrypted.txt"

	// Encrypt with key-1
	encBackend1.Put(ctx, path, originalData)

	// Try to decrypt with key-2
	backend2 := newMockBackend()
	backend2.data[path] = backend1.data[path]
	encBackend2, _ := NewEncryptedBackend(backend2, "key-2")

	_, err := encBackend2.Get(ctx, path)
	if err == nil {
		t.Error("Decrypting with wrong key should fail")
	}
}

func TestEncryptedBackend_Delete(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	data := []byte("data to delete")
	path := "test/delete.txt"

	encBackend.Put(ctx, path, data)

	// Verify exists
	exists, _ := backend.Exists(ctx, path)
	if !exists {
		t.Error("File should exist before delete")
	}

	// Delete
	err := encBackend.Delete(ctx, path)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify deleted
	exists, _ = backend.Exists(ctx, path)
	if exists {
		t.Error("File should not exist after delete")
	}
}

func TestEncryptedBackend_Exists(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	data := []byte("test data")
	path := "test/exists.txt"

	// Not exists initially
	exists, _ := encBackend.Exists(ctx, path)
	if exists {
		t.Error("File should not exist initially")
	}

	// Put data
	encBackend.Put(ctx, path, data)

	// Now exists
	exists, _ = encBackend.Exists(ctx, path)
	if !exists {
		t.Error("File should exist after Put")
	}
}

func TestEncryptedBackend_List(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	// Put some files
	encBackend.Put(ctx, "docs/file1.txt", []byte("data1"))
	encBackend.Put(ctx, "docs/file2.txt", []byte("data2"))
	encBackend.Put(ctx, "other/file3.txt", []byte("data3"))

	// List with prefix
	paths, err := encBackend.List(ctx, "docs/")
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}
}

func TestEncryptedBackend_GetURL(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	_, err := encBackend.GetURL(ctx, "test/file.txt", 1*time.Hour)
	if err == nil {
		t.Error("GetURL() should return error for encrypted backend")
	}
}

func TestEncryptedBackend_EncryptString(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")

	plaintext := "sensitive string data"

	encrypted, err := encBackend.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString() returned error: %v", err)
	}

	// Encrypted should be different from plaintext
	if encrypted == plaintext {
		t.Error("Encrypted string should differ from plaintext")
	}

	// Should be base64 encoded (only contains valid base64 chars)
	validBase64 := true
	for _, c := range encrypted {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			validBase64 = false
			break
		}
	}
	if !validBase64 {
		t.Error("Encrypted string should be valid base64")
	}

	// Decrypt
	decrypted, err := encBackend.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString() returned error: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted string doesn't match original.\nGot: %s\nWant: %s", decrypted, plaintext)
	}
}

func TestEncryptedBackend_DecryptString_InvalidBase64(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")

	_, err := encBackend.DecryptString("not valid base64!!!")
	if err == nil {
		t.Error("DecryptString() with invalid base64 should return error")
	}
}

func TestEncryptedBackend_DecryptString_TooShort(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")

	_, err := encBackend.DecryptString("YWJj") // Short base64
	if err == nil {
		t.Error("DecryptString() with too-short data should return error")
	}
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() returned error: %v", err)
	}

	if len(key1) == 0 {
		t.Error("Generated key should not be empty")
	}

	// Second key should be different
	key2, _ := GenerateKey()
	if key1 == key2 {
		t.Error("Each generated key should be unique")
	}

	// Should be valid base64
	validBase64 := true
	for _, c := range key1 {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			validBase64 = false
			break
		}
	}
	if !validBase64 {
		t.Error("Generated key should be valid base64")
	}
}

func TestDeriveKey(t *testing.T) {
	password := "test-password"
	salt := "test-salt"

	key1 := DeriveKey(password, salt)
	key2 := DeriveKey(password, salt)
	key3 := DeriveKey(password, "different-salt")

	// Same inputs should derive same key
	if key1 != key2 {
		t.Error("Same password and salt should derive same key")
	}

	// Different salt should derive different key
	if key1 == key3 {
		t.Error("Different salt should derive different key")
	}

	// Key should be 32 bytes
	if len(key1) != 32 {
		t.Errorf("Derived key should be 32 bytes, got %d", len(key1))
	}
}

func TestEncryptedBackend_EmptyData(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	emptyData := []byte("")
	path := "test/empty.txt"

	// Put empty data
	err := encBackend.Put(ctx, path, emptyData)
	if err != nil {
		t.Fatalf("Put() with empty data returned error: %v", err)
	}

	// Get empty data
	retrieved, err := encBackend.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get() with empty data returned error: %v", err)
	}

	if len(retrieved) != 0 {
		t.Errorf("Expected empty data, got %d bytes", len(retrieved))
	}
}

func TestEncryptedBackend_LargeData(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	// Create 1MB of data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	path := "test/large.bin"

	// Put large data
	err := encBackend.Put(ctx, path, largeData)
	if err != nil {
		t.Fatalf("Put() with large data returned error: %v", err)
	}

	// Get large data
	retrieved, err := encBackend.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get() with large data returned error: %v", err)
	}

	if len(retrieved) != len(largeData) {
		t.Errorf("Large data size mismatch: got %d, want %d", len(retrieved), len(largeData))
	}

	// Spot check some bytes
	for i := 0; i < 100; i++ {
		if retrieved[i] != largeData[i] {
			t.Errorf("Byte %d mismatch: got %d, want %d", i, retrieved[i], largeData[i])
			break
		}
	}
}

func TestEncryptedBackend_MultipleFiles(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	files := map[string][]byte{
		"file1.txt": []byte("content 1"),
		"file2.txt": []byte("content 2"),
		"file3.txt": []byte("content 3"),
	}

	// Put all files
	for path, data := range files {
		err := encBackend.Put(ctx, path, data)
		if err != nil {
			t.Fatalf("Put() %s returned error: %v", path, err)
		}
	}

	// Get all files and verify
	for path, originalData := range files {
		retrieved, err := encBackend.Get(ctx, path)
		if err != nil {
			t.Fatalf("Get() %s returned error: %v", path, err)
		}

		if string(retrieved) != string(originalData) {
			t.Errorf("Data mismatch for %s", path)
		}
	}
}

func TestEncryptedBackend_SpecialCharacters(t *testing.T) {
	backend := newMockBackend()
	encBackend, _ := NewEncryptedBackend(backend, "test-key-12345")
	ctx := context.Background()

	testData := []string{
		"Simple text",
		"Text with emoji 🎉🔐",
		"Text with \"quotes\"",
		"Text with 'apostrophes'",
		"Text with\nnewlines\nand\ttabs",
		"Text with special chars: !@#$%^&*()",
		"Unicode: 你好世界",
	}

	for i, data := range testData {
		path := "test/special" + string(rune('1'+i))
		original := []byte(data)

		err := encBackend.Put(ctx, path, original)
		if err != nil {
			t.Fatalf("Put() returned error: %v", err)
		}

		retrieved, err := encBackend.Get(ctx, path)
		if err != nil {
			t.Fatalf("Get() returned error: %v", err)
		}

		if string(retrieved) != data {
			t.Errorf("Special character data mismatch for %s", path)
		}
	}
}

func TestLocalBackend(t *testing.T) {
	// Note: These tests use actual filesystem in temp dir
	t.Skip("Skipping LocalBackend tests in unit test environment")
}

func TestS3Backend(t *testing.T) {
	// Note: These tests would require actual S3/minio setup
	t.Skip("Skipping S3Backend tests - requires external dependencies")
}
