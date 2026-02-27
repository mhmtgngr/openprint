// Package handler provides HTTP handlers for the storage service.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/storage-service/storage"
)

// Config holds handler dependencies.
type Config struct {
	Backend       storage.Backend
	DB            *pgxpool.Pool
	MaxUploadSize int64
}

// Handler provides storage service HTTP handlers.
type Handler struct {
	backend       storage.Backend
	db            *pgxpool.Pool
	maxUploadSize int64
}

// New creates a new handler instance.
func New(cfg Config) *Handler {
	return &Handler{
		backend:       cfg.Backend,
		db:            cfg.DB,
		maxUploadSize: cfg.MaxUploadSize,
	}
}

// DocumentMetadata represents stored document metadata.
type DocumentMetadata struct {
	ID          string
	Name        string
	ContentType string
	Size        int64
	Checksum    string
	UserEmail   string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

// DocumentsHandler handles document list and creation.
func (h *Handler) DocumentsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.listDocuments(w, r, ctx)
	case http.MethodPost:
		h.createDocument(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	// Parse multipart form
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to parse form", http.StatusBadRequest))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, apperrors.Wrap(err, "file is required", http.StatusBadRequest))
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > h.maxUploadSize {
		respondError(w, apperrors.New(fmt.Sprintf("file size exceeds limit of %d bytes", h.maxUploadSize), http.StatusRequestEntityTooLarge))
		return
	}

	// Generate document ID
	docID := uuid.New().String()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to read file", http.StatusInternalServerError))
		return
	}

	// Store file
	storagePath := fmt.Sprintf("documents/%s/%s", docID, header.Filename)
	if err := h.backend.Put(ctx, storagePath, content); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store file", http.StatusInternalServerError))
		return
	}

	// Store metadata in database
	metadata := &DocumentMetadata{
		ID:          docID,
		Name:        header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        int64(len(content)),
		Checksum:    computeChecksum(content),
		UserEmail:   r.FormValue("user_email"),
		CreatedAt:   time.Now(),
	}

	if err := h.storeMetadata(ctx, metadata); err != nil {
		// Clean up stored file
		h.backend.Delete(ctx, storagePath)
		respondError(w, apperrors.Wrap(err, "failed to store metadata", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"document_id":  docID,
		"name":         metadata.Name,
		"content_type": metadata.ContentType,
		"size":         metadata.Size,
		"checksum":     metadata.Checksum,
		"created_at":   metadata.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	userEmail := r.URL.Query().Get("user_email")
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Build query
	query := `
		SELECT id, name, content_type, size, checksum, user_email, created_at, expires_at
		FROM documents
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if userEmail != "" {
		query += fmt.Sprintf(" AND user_email = $%d", argNum)
		args = append(args, userEmail)
		argNum++
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query documents", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	documents := make([]map[string]interface{}, 0)
	for rows.Next() {
		var doc DocumentMetadata
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.ContentType, &doc.Size, &doc.Checksum, &doc.UserEmail, &doc.CreatedAt, &doc.ExpiresAt); err != nil {
			continue
		}

		documents = append(documents, map[string]interface{}{
			"document_id":  doc.ID,
			"name":         doc.Name,
			"content_type": doc.ContentType,
			"size":         doc.Size,
			"created_at":   doc.CreatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
	})
}

// DocumentHandler handles individual document operations.
func (h *Handler) DocumentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract document ID from path
	docID := strings.TrimPrefix(r.URL.Path, "/documents/")
	if docID == "" {
		respondError(w, apperrors.New("document ID is required", http.StatusBadRequest))
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getDocument(w, r, ctx, docID)
	case http.MethodDelete:
		h.deleteDocument(w, r, ctx, docID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request, ctx context.Context, docID string) {
	// Get metadata
	metadata, err := h.getMetadata(ctx, docID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Get file from storage
	content, err := h.backend.Get(ctx, fmt.Sprintf("documents/%s/%s", docID, metadata.Name))
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to retrieve file", http.StatusInternalServerError))
		return
	}

	// Set headers
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", metadata.Name))

	w.Write(content)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request, ctx context.Context, docID string) {
	// Get metadata first
	metadata, err := h.getMetadata(ctx, docID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Delete from storage
	storagePath := fmt.Sprintf("documents/%s/%s", docID, metadata.Name)
	if err := h.backend.Delete(ctx, storagePath); err != nil {
		// Log but continue with database cleanup
		fmt.Printf("Failed to delete file from storage: %v", err)
	}

	// Delete from database
	if _, err := h.db.Exec(ctx, "DELETE FROM documents WHERE id = $1", docID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete document", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UploadHandler handles file uploads.
func (h *Handler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce max upload size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to parse form", http.StatusBadRequest))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, apperrors.Wrap(err, "file is required", http.StatusBadRequest))
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to read file", http.StatusInternalServerError))
		return
	}

	// Generate document ID
	docID := uuid.New().String()

	// Store file
	storagePath := fmt.Sprintf("uploads/%s/%s", docID, header.Filename)
	if err := h.backend.Put(ctx, storagePath, content); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store file", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"document_id": docID,
		"filename":    header.Filename,
		"size":        len(content),
		"content_type": header.Header.Get("Content-Type"),
	})
}

// DownloadHandler handles file downloads.
func (h *Handler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL
	docPath := strings.TrimPrefix(r.URL.Path, "/download/")
	if docPath == "" {
		respondError(w, apperrors.New("document path is required", http.StatusBadRequest))
		return
	}

	// Get file from storage
	content, err := h.backend.Get(ctx, "uploads/"+docPath)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to retrieve file", http.StatusInternalServerError))
		return
	}

	// Determine content type
	contentType := "application/octet-stream"
	ext := path.Ext(docPath)
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".txt":
		contentType = "text/plain"
	case ".doc", ".docx":
		contentType = "application/msword"
	case ".xls", ".xlsx":
		contentType = "application/vnd.ms-excel"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", path.Base(docPath)))

	w.Write(content)
}

// Helper functions

func (h *Handler) storeMetadata(ctx context.Context, metadata *DocumentMetadata) error {
	query := `
		INSERT INTO documents (id, name, content_type, size, checksum, user_email, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := h.db.Exec(ctx, query,
		metadata.ID,
		metadata.Name,
		metadata.ContentType,
		metadata.Size,
		metadata.Checksum,
		metadata.UserEmail,
		metadata.CreatedAt,
		metadata.ExpiresAt,
	)

	return err
}

func (h *Handler) getMetadata(ctx context.Context, docID string) (*DocumentMetadata, error) {
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

func computeChecksum(data []byte) string {
	// Simple checksum - in production use SHA256
	h := 0
	for _, b := range data {
		h = int(b) + (h << 5) - h
	}
	return fmt.Sprintf("%x", h)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
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
