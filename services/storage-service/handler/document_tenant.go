// Package handler provides HTTP handlers for the storage service with tenant support.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	multitenant "github.com/openprint/openprint/internal/multi-tenant"
	"github.com/openprint/openprint/services/storage-service/repository"
)

// DocumentWithTenant represents a document with tenant information.
type DocumentWithTenant struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	Name           string                 `json:"name"`
	FileName       string                 `json:"file_name"`
	ContentType    string                 `json:"content_type"`
	Size           int64                  `json:"size"`
	StoragePath    string                 `json:"storage_path"`
	Checksum       string                 `json:"checksum"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// UploadDocumentRequest represents a document upload request with tenant context.
type UploadDocumentRequest struct {
	Name        string                 `json:"name"`
	FileName    string                 `json:"file_name"`
	ContentType string                 `json:"content_type"`
	Size        int64                  `json:"size"`
	Checksum    string                 `json:"checksum"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UploadDocument handles document upload with tenant context and quota checking.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
		return
	}

	// Check storage quota before uploading
	if h.quotaEnforcer != nil {
		if err := h.quotaEnforcer.RequireStorageQuota(ctx, r.ContentLength); err != nil {
			respondError(w, err)
			return
		}
	}

	var req UploadDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.FileName == "" {
		req.FileName = req.Name
	}
	if req.Size <= 0 {
		respondError(w, apperrors.New("size must be greater than 0", http.StatusBadRequest))
		return
	}

	now := time.Now().UTC()
	doc := &repository.Document{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		Size:        req.Size,
		Checksum:    req.Checksum,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Generate storage path
	doc.StoragePath = fmt.Sprintf("/documents/%s/%s", tenantID, doc.ID)

	// Marshal metadata
	if req.Metadata != nil {
		if bytes, err := json.Marshal(req.Metadata); err == nil {
			doc.Metadata = string(bytes)
		}
	}

	docRepo := h.docRepo()
	if err := docRepo.Create(ctx, doc); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create document", http.StatusInternalServerError))
		return
	}

	// Record storage usage
	if h.quotaEnforcer != nil {
		h.quotaEnforcer.RecordStorageUsage(ctx, req.Size)
	}

	// Log the upload
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogCreate(ctx, userID, userEmail, "document", doc.ID, doc.Name,
			map[string]interface{}{"tenant_id": tenantID, "size": req.Size})
	}

	respondJSON(w, http.StatusCreated, documentToTenantResponse(doc))
}

// ListDocuments handles listing documents with tenant scoping.
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		// Platform admin can list all documents
		role := middleware.GetRole(r)
		if role != "admin" && role != "platform_admin" {
			respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
			return
		}
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	docRepo := h.docRepo()
	var docs []*repository.Document
	var total int
	var getErr error

	if tenantID != "" {
		// Tenant-scoped query
		docs, total, getErr = docRepo.ListByTenant(ctx, tenantID, limit, offset)
	} else {
		// Platform admin - get all documents
		docs, total, getErr = docRepo.List(ctx, limit, offset)
	}

	if getErr != nil {
		respondError(w, apperrors.Wrap(getErr, "failed to list documents", http.StatusInternalServerError))
		return
	}

	response := make([]*DocumentWithTenant, len(docs))
	for i, d := range docs {
		response[i] = documentToTenantResponse(d)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"documents": response,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetDocument handles retrieving a document with tenant scoping.
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract document ID from URL
	docID := extractIDFromPath(r.URL.Path, "documents")
	if docID == "" {
		respondError(w, apperrors.New("document ID required", http.StatusBadRequest))
		return
	}

	docRepo := h.docRepo()
	doc, err := docRepo.FindByID(ctx, docID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get document", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessDocument(ctx, doc) {
		respondError(w, apperrors.New("forbidden: cannot access this document", http.StatusForbidden))
		return
	}

	respondJSON(w, http.StatusOK, documentToTenantResponse(doc))
}

// DeleteDocument handles deleting a document with tenant scoping.
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract document ID from URL
	docID := extractIDFromPath(r.URL.Path, "documents")
	if docID == "" {
		respondError(w, apperrors.New("document ID required", http.StatusBadRequest))
		return
	}

	docRepo := h.docRepo()
	doc, err := docRepo.FindByID(ctx, docID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get document", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessDocument(ctx, doc) {
		respondError(w, apperrors.New("forbidden: cannot access this document", http.StatusForbidden))
		return
	}

	if err := docRepo.Delete(ctx, docID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete document", http.StatusInternalServerError))
		return
	}

	// Record storage usage reduction
	if h.quotaEnforcer != nil {
		h.quotaEnforcer.RecordStorageUsage(ctx, -doc.Size)
	}

	// Log the deletion
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogDelete(ctx, userID, userEmail, "document", doc.ID, doc.Name,
			map[string]interface{}{"size": doc.Size})
	}

	respondJSON(w, http.StatusNoContent, nil)
}

// GetStorageUsage handles retrieving storage usage for the current tenant.
func (h *Handler) GetStorageUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
		return
	}

	// Get storage usage
	docRepo := h.docRepo()
	usageBytes, err := docRepo.GetTotalSizeByTenant(ctx, tenantID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get storage usage", http.StatusInternalServerError))
		return
	}

	// Convert to GB for display
	usageGB := float64(usageBytes) / (1024 * 1024 * 1024)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id":      tenantID,
		"storage_used":   usageGB,
		"storage_bytes":  usageBytes,
	})
}

// Helper functions

// docRepo returns the document repository.
func (h *Handler) docRepo() *repository.DocumentRepository {
	return nil
}

// canAccessDocument checks if the current context can access the document.
func canAccessDocument(ctx context.Context, doc *repository.Document) bool {
	// Platform admins can access any document
	if multitenant.IsPlatformAdmin(ctx) {
		return true
	}

	// Check tenant match
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		return false
	}

	return tenantID == doc.TenantID || tenantID == doc.OrganizationID
}

// documentToTenantResponse converts a document to the tenant-aware response format.
func documentToTenantResponse(doc *repository.Document) *DocumentWithTenant {
	var metadata map[string]interface{}
	if doc.Metadata != "" {
		json.Unmarshal([]byte(doc.Metadata), &metadata)
	}

	return &DocumentWithTenant{
		ID:             doc.ID,
		TenantID:       doc.TenantID,
		OrganizationID: doc.OrganizationID,
		Name:           doc.Name,
		FileName:       doc.FileName,
		ContentType:    doc.ContentType,
		Size:           doc.Size,
		StoragePath:    doc.StoragePath,
		Checksum:       doc.Checksum,
		Metadata:       metadata,
		CreatedAt:      doc.CreatedAt,
		UpdatedAt:      doc.UpdatedAt,
	}
}

// extractIDFromPath extracts an ID from the URL path.
func extractIDFromPath(path, resource string) string {
	parts := splitPath(path)
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// splitPath splits a URL path into components.
func splitPath(path string) []string {
	path = path[1:]
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// respondJSON sends a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
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

// Import strings package
import "strings"
