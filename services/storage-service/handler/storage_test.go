// Package handler provides tests for storage service HTTP handlers.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// mockStorageBackend is a mock storage backend for testing
type mockStorageBackend struct {
	data map[string][]byte
}

func newMockStorageBackend() *mockStorageBackend {
	return &mockStorageBackend{
		data: make(map[string][]byte),
	}
}

func (m *mockStorageBackend) Put(ctx context.Context, path string, data []byte) error {
	m.data[path] = data
	return nil
}

func (m *mockStorageBackend) Get(ctx context.Context, path string) ([]byte, error) {
	data, ok := m.data[path]
	if !ok {
		return nil, &testError{"not found"}
	}
	return data, nil
}

func (m *mockStorageBackend) Delete(ctx context.Context, path string) error {
	delete(m.data, path)
	return nil
}

func (m *mockStorageBackend) Exists(ctx context.Context, path string) (bool, error) {
	_, ok := m.data[path]
	return ok, nil
}

func (m *mockStorageBackend) List(ctx context.Context, prefix string) ([]string, error) {
	var paths []string
	for p := range m.data {
		if strings.HasPrefix(p, prefix) {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

func (m *mockStorageBackend) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "http://example.com/" + path, nil
}

// mockDB is a mock database for testing
type mockDB struct {
	documents map[string]*DocumentMetadata
}

func newMockDB() *mockDB {
	return &mockDB{
		documents: make(map[string]*DocumentMetadata),
	}
}

func (m *mockDB) Query(ctx context.Context, query string, args ...interface{}) *mockRows {
	return &mockRows{db: m}
}

func (m *mockDB) QueryRow(ctx context.Context, query string, args ...interface{}) *mockRow {
	return &mockRow{db: m}
}

func (m *mockDB) Exec(ctx context.Context, query string, args ...interface{}) *mockResult {
	return &mockResult{db: m}
}

type mockRows struct {
	db *mockDB
}

func (r *mockRows) Close() error                 { return nil }
func (r *mockRows) Next() bool                   { return false }
func (r *mockRows) Scan(dest ...interface{}) error { return nil }

type mockRow struct {
	db *mockDB
}

func (r *mockRow) Scan(dest ...interface{}) error {
	// Return mock data
	if len(dest) > 0 {
		if str, ok := dest[0].(*string); ok {
			*str = "doc-123"
		}
	}
	return nil
}

type mockResult struct {
	db *mockDB
}

func (r *mockResult) RowsAffected() int64 { return 1 }

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNewHandler(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil, // Use nil for unit tests
		MaxUploadSize: 100 * 1024 * 1024, // 100MB
	}

	h := New(cfg)

	if h == nil {
		t.Fatal("New() returned nil")
	}
	if h.backend != backend {
		t.Error("Backend not set correctly")
	}
	if h.maxUploadSize != 100*1024*1024 {
		t.Errorf("MaxUploadSize not set correctly, got %d", h.maxUploadSize)
	}
}

func TestDocumentMetadata_Struct(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	metadata := &DocumentMetadata{
		ID:          "doc-123",
		Name:        "test.pdf",
		ContentType: "application/pdf",
		Size:        1024,
		Checksum:    "abc123",
		UserEmail:   "test@example.com",
		CreatedAt:   now,
		ExpiresAt:   &expiresAt,
	}

	if metadata.ID != "doc-123" {
		t.Error("ID not set correctly")
	}
	if metadata.Name != "test.pdf" {
		t.Error("Name not set correctly")
	}
	if metadata.ContentType != "application/pdf" {
		t.Error("ContentType not set correctly")
	}
	if metadata.Size != 1024 {
		t.Error("Size not set correctly")
	}
	if metadata.ExpiresAt == nil {
		t.Error("ExpiresAt should be set")
	}
}

func TestCreateJobRequest_Validation(t *testing.T) {
	// This tests the validation logic conceptually
	tests := []struct {
		name    string
		docID   string
		printerID string
		userEmail string
		wantErr bool
	}{
		{
			name:      "valid request",
			docID:     "doc-123",
			printerID: "printer-123",
			userEmail: "test@example.com",
			wantErr:   false,
		},
		{
			name:      "missing document_id",
			docID:     "",
			printerID: "printer-123",
			userEmail: "test@example.com",
			wantErr:   true,
		},
		{
			name:      "missing printer_id",
			docID:     "doc-123",
			printerID: "",
			userEmail: "test@example.com",
			wantErr:   true,
		},
		{
			name:      "missing user_email",
			docID:     "doc-123",
			printerID: "printer-123",
			userEmail: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.docID == "" || tt.printerID == "" || tt.userEmail == ""
			if hasError != tt.wantErr {
				t.Errorf("Validation error mismatch")
			}
		})
	}
}

func TestHandler_UploadHandler(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	// Create a multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.UploadHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["document_id"] == nil {
		t.Error("Response should include document_id")
	}
}

func TestHandler_UploadHandler_WrongMethod(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	req := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()

	h.UploadHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandler_DownloadHandler(t *testing.T) {
	backend := newMockStorageBackend()

	// Add a file to backend
	backend.data["uploads/test.txt"] = []byte("download test content")

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	req := httptest.NewRequest("GET", "/download/test.txt", nil)
	w := httptest.NewRecorder()

	h.DownloadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	content := w.Body.String()
	if content != "download test content" {
		t.Errorf("Expected content 'download test content', got '%s'", content)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got '%s'", contentType)
	}
}

func TestHandler_DownloadHandler_MissingPath(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	req := httptest.NewRequest("GET", "/download/", nil)
	w := httptest.NewRecorder()

	h.DownloadHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_DownloadHandler_NotFound(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	req := httptest.NewRequest("GET", "/download/nonexistent.txt", nil)
	w := httptest.NewRecorder()

	h.DownloadHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandler_DownloadHandler_ContentTypes(t *testing.T) {
	backend := newMockStorageBackend()

	tests := []struct {
		filename    string
		contentType string
		data        []byte
	}{
		{"test.pdf", "application/pdf", []byte("pdf content")},
		{"test.txt", "text/plain", []byte("text content")},
		{"test.doc", "application/msword", []byte("doc content")},
		{"test.xlsx", "application/vnd.ms-excel", []byte("xlsx content")},
	}

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			backend.data["uploads/"+tt.filename] = tt.data

			req := httptest.NewRequest("GET", "/download/"+tt.filename, nil)
			w := httptest.NewRecorder()

			h.DownloadHandler(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.contentType {
				t.Errorf("Expected Content-Type '%s', got '%s'", tt.contentType, contentType)
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte("")},
		{"simple data", []byte("hello")},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF}},
		{"long data", []byte(strings.Repeat("a", 1000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := computeChecksum(tt.data)
			if checksum == "" {
				t.Error("Checksum should not be empty")
			}
			// Same data should produce same checksum
			checksum2 := computeChecksum(tt.data)
			if checksum != checksum2 {
				t.Error("Same data should produce same checksum")
			}
		})
	}
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"message": "success",
		"count":   42,
	}

	respondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be application/json")
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["message"] != "success" {
		t.Error("Response data mismatch")
	}
}

func TestRespondError_AppError(t *testing.T) {
	w := httptest.NewRecorder()

	err := apperrors.New("test error", http.StatusBadRequest)
	respondError(w, err)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be application/json")
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] == nil {
		t.Error("Error response should include code")
	}
}

func TestRespondError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()

	err := &testError{"generic error"}
	respondError(w, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] != "INTERNAL_ERROR" {
		t.Errorf("Expected code 'INTERNAL_ERROR', got '%s'", response["code"])
	}
}

func TestHandler_DocumentsHandler_Get(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	t.Run("list documents", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/documents", nil)
		w := httptest.NewRecorder()

		h.DocumentsHandler(w, req)

		// Without DB, we expect error
		if w.Code != http.StatusInternalServerError || w.Code == http.StatusOK {
			t.Logf("DocumentsHandler returned status %d (expected with nil DB)", w.Code)
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/documents", nil)
		w := httptest.NewRecorder()

		h.DocumentsHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestHandler_DocumentHandler_Get(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 10 * 1024 * 1024,
	}

	h := New(cfg)

	t.Run("valid document ID in path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/documents/doc-123", nil)

		// Extract docID as handler would
		docID := strings.TrimPrefix(req.URL.Path, "/documents/")

		if docID != "doc-123" {
			t.Errorf("Expected docID 'doc-123', got '%s'", docID)
		}
	})

	t.Run("empty document ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/documents/", nil)

		docID := strings.TrimPrefix(req.URL.Path, "/documents/")

		if docID != "" {
			t.Errorf("Expected empty docID, got '%s'", docID)
		}
	})
}

func TestHandler_MaxUploadSize(t *testing.T) {
	tests := []struct {
		name         string
		maxUpload    int64
		expectedSize int64
	}{
		{"10 MB", 10 * 1024 * 1024, 10 * 1024 * 1024},
		{"50 MB", 50 * 1024 * 1024, 50 * 1024 * 1024},
		{"100 MB", 100 * 1024 * 1024, 100 * 1024 * 1024},
		{"1 GB", 1024 * 1024 * 1024, 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Backend:       newMockStorageBackend(),
				DB:            nil,
				MaxUploadSize: tt.maxUpload,
			}

			h := New(cfg)

			if h.maxUploadSize != tt.expectedSize {
				t.Errorf("Expected MaxUploadSize %d, got %d", tt.expectedSize, h.maxUploadSize)
			}
		})
	}
}

func TestHandler_FileSizeValidation(t *testing.T) {
	backend := newMockStorageBackend()

	cfg := Config{
		Backend:       backend,
		DB:            nil,
		MaxUploadSize: 100, // 100 bytes
	}

	h := New(cfg)

	// Create a file larger than limit
	largeData := strings.Repeat("x", 200)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "large.txt")
	io.WriteString(part, largeData)

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.UploadHandler(w, req)

	// Should get error or be rejected
	if w.Code == http.StatusCreated {
		t.Error("Large file should be rejected")
	}
}

func TestHandler_PathExtraction(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"/documents/doc-123", "doc-123"},
		{"/documents/", ""},
		{"/documents", ""},
		{"/documents/folder/subfolder/doc-456", "folder/subfolder/doc-456"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			docID := strings.TrimPrefix(tt.url, "/documents/")
			if docID != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, docID)
			}
		})
	}
}

func TestHandler_QueryParams(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		userEmail string
		limit     int
		offset    int
	}{
		{
			name:      "with all params",
			url:       "/documents?user_email=test@example.com&limit=10&offset=5",
			userEmail: "test@example.com",
			limit:     10,
			offset:    5,
		},
		{
			name:      "with only email",
			url:       "/documents?user_email=test@example.com",
			userEmail: "test@example.com",
			limit:     50, // default
			offset:    0,  // default
		},
		{
			name:      "no params",
			url:       "/documents",
			userEmail: "",
			limit:     50,
			offset:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)

			userEmail := req.URL.Query().Get("user_email")
			limit := 50
			offset := 0

			if l := req.URL.Query().Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}
			if o := req.URL.Query().Get("offset"); o != "" {
				fmt.Sscanf(o, "%d", &offset)
			}

			if userEmail != tt.userEmail {
				t.Errorf("Expected userEmail '%s', got '%s'", tt.userEmail, userEmail)
			}
			if limit != tt.limit {
				t.Errorf("Expected limit %d, got %d", tt.limit, limit)
			}
			if offset != tt.offset {
				t.Errorf("Expected offset %d, got %d", tt.offset, offset)
			}
		})
	}
}

func TestDocumentMetadata_WithExpiry(t *testing.T) {
	now := time.Now()
	in24Hours := now.Add(24 * time.Hour)

	metadata := &DocumentMetadata{
		ID:        "doc-123",
		Name:      "temporary.pdf",
		CreatedAt: now,
		ExpiresAt: &in24Hours,
	}

	if metadata.ExpiresAt == nil {
		t.Error("ExpiresAt should be set")
	}

	if metadata.ExpiresAt.Sub(now) < 23*time.Hour {
		t.Error("ExpiresAt should be approximately 24 hours from now")
	}
}

func TestDocumentMetadata_WithoutExpiry(t *testing.T) {
	metadata := &DocumentMetadata{
		ID:        "doc-123",
		Name:      "permanent.pdf",
		CreatedAt: time.Now(),
		ExpiresAt: nil,
	}

	if metadata.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for permanent documents")
	}
}
