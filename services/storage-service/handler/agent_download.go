// Package handler provides optimized document download handlers for agents.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/storage-service/storage"
)

// AgentDownloadConfig holds agent download handler dependencies.
type AgentDownloadConfig struct {
	Backend         storage.Backend
	DB              *pgxpool.Pool
	MaxDownloadSize int64
	ResumeTimeout   time.Duration
}

// AgentDownloadHandler handles optimized document downloads for agents.
type AgentDownloadHandler struct {
	backend         storage.Backend
	db              *pgxpool.Pool
	maxDownloadSize int64
	resumeTimeout   time.Duration
}

// NewAgentDownloadHandler creates a new agent download handler.
func NewAgentDownloadHandler(cfg AgentDownloadConfig) *AgentDownloadHandler {
	return &AgentDownloadHandler{
		backend:         cfg.Backend,
		db:              cfg.DB,
		maxDownloadSize: cfg.MaxDownloadSize,
		resumeTimeout:   cfg.ResumeTimeout,
	}
}

// DownloadDocument handles document download requests from agents.
// Supports range requests for resume capability.
// GET /agents/documents/{document_id}
func (h *AgentDownloadHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract document ID from path
	docID := extractDocumentID(r.URL.Path)
	if docID == "" {
		respondAgentError(w, apperrors.New("document ID is required", http.StatusBadRequest))
		return
	}

	// Get document metadata
	metadata, err := h.getDocumentMetadata(ctx, docID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondAgentError(w, apperrors.ErrNotFound)
		} else {
			respondAgentError(w, apperrors.Wrap(err, "failed to get metadata", http.StatusInternalServerError))
		}
		return
	}

	// Check if document has expired
	if metadata.ExpiresAt != nil && metadata.ExpiresAt.Before(time.Now()) {
		respondAgentError(w, apperrors.New("document has expired", http.StatusGone))
		return
	}

	// Get storage path
	storagePath := fmt.Sprintf("documents/%s/%s", docID, metadata.Name)

	// Handle range requests
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(w, r, storagePath, metadata, rangeHeader)
		return
	}

	// Get file content
	content, err := h.backend.Get(ctx, storagePath)
	if err != nil {
		respondAgentError(w, apperrors.Wrap(err, "failed to retrieve document", http.StatusInternalServerError))
		return
	}

	// Verify checksum before sending
	computedChecksum := computeChecksum(content)
	if computedChecksum != metadata.Checksum {
		respondAgentError(w, apperrors.New("document checksum mismatch", http.StatusInternalServerError))
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(metadata.Size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", metadata.Name))
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Document-Checksum", metadata.Checksum)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", metadata.Checksum))

	// For HEAD requests, don't send body
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Write(content)
}

// DownloadDocumentByJob handles document download by job ID.
// GET /agents/jobs/{job_id}/document
func (h *AgentDownloadHandler) DownloadDocumentByJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path
	jobID := extractIDFromPath(r.URL.Path, "agents", "jobs", "document")
	if jobID == "" {
		respondAgentError(w, apperrors.New("job ID is required", http.StatusBadRequest))
		return
	}

	// Get job and document info
	var docID string

	query := `
		SELECT document_id
		FROM print_jobs
		WHERE id = $1
	`

	err := h.db.QueryRow(ctx, query, jobID).Scan(&docID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondAgentError(w, apperrors.ErrNotFound)
		} else {
			respondAgentError(w, apperrors.Wrap(err, "failed to get job document", http.StatusInternalServerError))
		}
		return
	}

	// Redirect to document download endpoint
	docURL := fmt.Sprintf("/agents/documents/%s", docID)
	http.Redirect(w, r, docURL, http.StatusFound)
}

// GetDocumentMetadata returns metadata for a document without downloading it.
// GET /agents/documents/{document_id}/metadata
func (h *AgentDownloadHandler) GetDocumentMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docID := extractDocumentID(strings.TrimSuffix(r.URL.Path, "/metadata"))
	if docID == "" {
		respondAgentError(w, apperrors.New("document ID is required", http.StatusBadRequest))
		return
	}

	metadata, err := h.getDocumentMetadata(ctx, docID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondAgentError(w, apperrors.ErrNotFound)
		} else {
			respondAgentError(w, apperrors.Wrap(err, "failed to get metadata", http.StatusInternalServerError))
		}
		return
	}

	response := map[string]interface{}{
		"document_id":     metadata.ID,
		"name":            metadata.Name,
		"content_type":    metadata.ContentType,
		"size":            metadata.Size,
		"checksum":        metadata.Checksum,
		"checksum_algo":   "sha256",
		"supports_resume": true,
		"created_at":      metadata.CreatedAt.Format(time.RFC3339),
	}

	if metadata.ExpiresAt != nil {
		response["expires_at"] = metadata.ExpiresAt.Format(time.RFC3339)
	}

	respondAgentJSON(w, http.StatusOK, response)
}

// BatchDownloadInfo returns metadata for multiple documents.
// POST /agents/documents/batch
func (h *AgentDownloadHandler) BatchDownloadInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DocumentIDs []string `json:"document_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if len(req.DocumentIDs) > 100 {
		respondAgentError(w, apperrors.New("maximum 100 documents per request", http.StatusBadRequest))
		return
	}

	documents := make([]map[string]interface{}, 0)

	for _, docID := range req.DocumentIDs {
		metadata, err := h.getDocumentMetadata(ctx, docID)
		if err != nil {
			continue
		}

		doc := map[string]interface{}{
			"document_id":  metadata.ID,
			"name":         metadata.Name,
			"content_type": metadata.ContentType,
			"size":         metadata.Size,
			"checksum":     metadata.Checksum,
			"download_url": fmt.Sprintf("/agents/documents/%s", metadata.ID),
			"created_at":   metadata.CreatedAt.Format(time.RFC3339),
		}

		documents = append(documents, doc)
	}

	respondAgentJSON(w, http.StatusOK, map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
	})
}

// VerifyDocument verifies a document's checksum after download.
// POST /agents/documents/{document_id}/verify
func (h *AgentDownloadHandler) VerifyDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docID := extractDocumentID(strings.TrimSuffix(r.URL.Path, "/verify"))
	if docID == "" {
		respondAgentError(w, apperrors.New("document ID is required", http.StatusBadRequest))
		return
	}

	var req struct {
		Checksum string `json:"checksum"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Get stored checksum
	metadata, err := h.getDocumentMetadata(ctx, docID)
	if err != nil {
		respondAgentError(w, apperrors.ErrNotFound)
		return
	}

	matches := (req.Checksum == metadata.Checksum)

	respondAgentJSON(w, http.StatusOK, map[string]interface{}{
		"valid":     matches,
		"expected":  metadata.Checksum,
		"received":  req.Checksum,
		"algorithm": "sha256",
	})
}

// handleRangeRequest handles HTTP range requests for resume support.
func (h *AgentDownloadHandler) handleRangeRequest(w http.ResponseWriter, r *http.Request, storagePath string, metadata *DocumentMetadata, rangeHeader string) {
	ctx := r.Context()

	// Parse range header: "bytes=start-end"
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		respondAgentError(w, apperrors.New("invalid range header", http.StatusRequestedRangeNotSatisfiable))
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")

	if len(parts) != 2 {
		respondAgentError(w, apperrors.New("invalid range format", http.StatusRequestedRangeNotSatisfiable))
		return
	}

	var start, end int64
	var err error

	if parts[0] != "" {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= metadata.Size {
			respondAgentError(w, apperrors.New("invalid range start", http.StatusRequestedRangeNotSatisfiable))
			return
		}
	}

	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start || end >= metadata.Size {
			respondAgentError(w, apperrors.New("invalid range end", http.StatusRequestedRangeNotSatisfiable))
			return
		}
	} else {
		end = metadata.Size - 1
	}

	contentLength := end - start + 1

	// Get file content
	content, err := h.backend.Get(ctx, storagePath)
	if err != nil {
		respondAgentError(w, apperrors.Wrap(err, "failed to retrieve document", http.StatusInternalServerError))
		return
	}

	if start >= int64(len(content)) {
		respondAgentError(w, apperrors.New("range not satisfiable", http.StatusRequestedRangeNotSatisfiable))
		return
	}

	if end >= int64(len(content)) {
		end = int64(len(content)) - 1
	}

	// Set range response headers
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, metadata.Size))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("X-Document-Checksum", metadata.Checksum)
	w.Header().Set("Cache-Control", "private, max-age=3600")

	w.WriteHeader(http.StatusPartialContent)

	// Write partial content
	w.Write(content[start : end+1])
}

// StreamDocument streams a document with chunked transfer encoding.
// This is useful for large files.
func (h *AgentDownloadHandler) StreamDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docID := extractDocumentID(strings.Replace(r.URL.Path, "/stream", "", 1))
	if docID == "" {
		respondAgentError(w, apperrors.New("document ID is required", http.StatusBadRequest))
		return
	}

	metadata, err := h.getDocumentMetadata(ctx, docID)
	if err != nil {
		respondAgentError(w, apperrors.ErrNotFound)
		return
	}

	storagePath := fmt.Sprintf("documents/%s/%s", docID, metadata.Name)

	// Set headers for chunked transfer
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Document-Checksum", metadata.Checksum)

	// Enable flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondAgentError(w, apperrors.New("streaming not supported", http.StatusInternalServerError))
		return
	}

	// Get and stream file in chunks
	content, err := h.backend.Get(ctx, storagePath)
	if err != nil {
		respondAgentError(w, apperrors.Wrap(err, "failed to retrieve document", http.StatusInternalServerError))
		return
	}

	const chunkSize = 32 * 1024 // 32KB chunks
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}

		w.Write(content[i:end])
		flusher.Flush()
	}
}

// getDocumentMetadata retrieves metadata for a document.
func (h *AgentDownloadHandler) getDocumentMetadata(ctx context.Context, docID string) (*DocumentMetadata, error) {
	var metadata DocumentMetadata

	query := `
		SELECT id, name, content_type, size, checksum, user_email, created_at, expires_at
		FROM documents
		WHERE id = $1
	`

	err := h.db.QueryRow(ctx, query, docID).Scan(
		&metadata.ID,
		&metadata.Name,
		&metadata.ContentType,
		&metadata.Size,
		&metadata.Checksum,
		&metadata.UserEmail,
		&metadata.CreatedAt,
		&metadata.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

// Helper functions

func extractDocumentID(path string) string {
	parts := splitPath(path)
	for i, part := range parts {
		if part == "documents" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func extractIDFromPath(path, resource, subresource, action string) string {
	parts := splitPath(path)
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			if subresource == "" || (i+2 < len(parts) && parts[i+2] == subresource) {
				if action == "" || (i+3 < len(parts) && parts[i+3] == action) {
					return parts[i+1]
				}
			}
		}
	}
	return ""
}

func splitPath(path string) []string {
	path = trimPath(path)
	if path == "" {
		return []string{}
	}
	parts := make([]string, 0)
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func trimPath(path string) string {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	for len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}

func respondAgentJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondAgentError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
