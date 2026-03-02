// Package handler provides HTTP handlers for the registry service with tenant support.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	multitenant "github.com/openprint/openprint/internal/multi-tenant"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// PrinterWithTenant represents a printer with tenant information.
type PrinterWithTenant struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	AgentID        string                 `json:"agent_id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	Status         string                 `json:"status"`
	Capabilities   map[string]interface{} `json:"capabilities"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ListPrinters handles listing printers with tenant scoping.
func (h *Handler) ListPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		// Platform admin can list all printers
		role := middleware.GetRole(r)
		if role != "admin" && role != "platform_admin" {
			respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
			return
		}
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	status := r.URL.Query().Get("status")

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	printerRepo := h.printerRepo()
	var printers []*repository.Printer
	var total int
	var getErr error

	if tenantID != "" {
		// Tenant-scoped query
		printers, total, getErr = printerRepo.ListByTenant(ctx, tenantID, limit, offset, status)
	} else {
		// Platform admin - get all printers
		printers, total, getErr = printerRepo.List(ctx, limit, offset, status)
	}

	if getErr != nil {
		respondError(w, apperrors.Wrap(getErr, "failed to list printers", http.StatusInternalServerError))
		return
	}

	response := make([]*PrinterWithTenant, len(printers))
	for i, p := range printers {
		response[i] = printerToTenantResponse(p)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": response,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetPrinter handles retrieving a printer with tenant scoping.
func (h *Handler) GetPrinter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from URL
	printerID := extractIDFromPath(r.URL.Path, "printers")
	if printerID == "" {
		respondError(w, apperrors.New("printer ID required", http.StatusBadRequest))
		return
	}

	printerRepo := h.printerRepo()
	printer, err := printerRepo.FindByID(ctx, printerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get printer", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessPrinter(ctx, printer) {
		respondError(w, apperrors.New("forbidden: cannot access this printer", http.StatusForbidden))
		return
	}

	respondJSON(w, http.StatusOK, printerToTenantResponse(printer))
}

// CreatePrinter handles creating a printer with tenant scoping and quota check.
func (h *Handler) CreatePrinter(w http.ResponseWriter, r *http.Request) {
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

	// Check printer quota before creating
	if h.quotaEnforcer != nil {
		if err := h.quotaEnforcer.RequirePrinterQuota(ctx, 1); err != nil {
			respondError(w, err)
			return
		}
	}

	var req struct {
		ID           string                 `json:"id"`
		Name         string                 `json:"name"`
		AgentID      string                 `json:"agent_id"`
		Capabilities map[string]interface{} `json:"capabilities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate input
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.AgentID == "" {
		respondError(w, apperrors.New("agent_id is required", http.StatusBadRequest))
		return
	}

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Marshal capabilities
	capabilitiesJSON := "{}"
	if req.Capabilities != nil {
		if bytes, err := json.Marshal(req.Capabilities); err == nil {
			capabilitiesJSON = string(bytes)
		}
	}

	printer := &repository.Printer{
		ID:             req.ID,
		Name:           req.Name,
		AgentID:        req.AgentID,
		OrganizationID: tenantID, // Use tenant_id as organization_id
		Status:         "online",
		Capabilities:   capabilitiesJSON,
	}

	printerRepo := h.printerRepo()
	if err := printerRepo.Create(ctx, printer); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create printer", http.StatusInternalServerError))
		return
	}

	// Record printer usage
	if h.quotaEnforcer != nil {
		h.quotaEnforcer.RecordPrinterUsage(ctx, 1)
	}

	// Log the creation
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogCreate(ctx, userID, userEmail, "printer", printer.ID, printer.Name,
			map[string]interface{}{"tenant_id": tenantID})
	}

	respondJSON(w, http.StatusCreated, printerToTenantResponse(printer))
}

// UpdatePrinter handles updating a printer with tenant scoping.
func (h *Handler) UpdatePrinter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from URL
	printerID := extractIDFromPath(r.URL.Path, "printers")
	if printerID == "" {
		respondError(w, apperrors.New("printer ID required", http.StatusBadRequest))
		return
	}

	printerRepo := h.printerRepo()
	printer, err := printerRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get printer", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessPrinter(ctx, printer) {
		respondError(w, apperrors.New("forbidden: cannot access this printer", http.StatusForbidden))
		return
	}

	var req struct {
		Name         string                 `json:"name,omitempty"`
		Status       string                 `json:"status,omitempty"`
		Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update fields
	if req.Name != "" {
		printer.Name = req.Name
	}
	if req.Status != "" {
		printer.Status = req.Status
	}
	if req.Capabilities != nil {
		if bytes, err := json.Marshal(req.Capabilities); err == nil {
			printer.Capabilities = string(bytes)
		}
	}
	printer.UpdatedAt = time.Now()

	if err := printerRepo.Update(ctx, printer); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update printer", http.StatusInternalServerError))
		return
	}

	// Log the update
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, userID, userEmail, "printer", printer.ID, printer.Name, nil)
	}

	respondJSON(w, http.StatusOK, printerToTenantResponse(printer))
}

// DeletePrinter handles deleting a printer with tenant scoping.
func (h *Handler) DeletePrinter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from URL
	printerID := extractIDFromPath(r.URL.Path, "printers")
	if printerID == "" {
		respondError(w, apperrors.New("printer ID required", http.StatusBadRequest))
		return
	}

	printerRepo := h.printerRepo()
	printer, err := printerRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get printer", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessPrinter(ctx, printer) {
		respondError(w, apperrors.New("forbidden: cannot access this printer", http.StatusForbidden))
		return
	}

	if err := printerRepo.Delete(ctx, printerID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete printer", http.StatusInternalServerError))
		return
	}

	// Record printer usage reduction
	if h.quotaEnforcer != nil {
		h.quotaEnforcer.RecordPrinterUsage(ctx, -1)
	}

	// Log the deletion
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogDelete(ctx, userID, userEmail, "printer", printer.ID, printer.Name, nil)
	}

	respondJSON(w, http.StatusNoContent, nil)
}

// Helper functions

// printerRepo returns the printer repository.
func (h *Handler) printerRepo() *repository.PrinterRepository {
	// In actual implementation, this would be a field on Handler
	return nil
}

// canAccessPrinter checks if the current context can access the printer.
func canAccessPrinter(ctx context.Context, printer *repository.Printer) bool {
	// Platform admins can access any printer
	if multitenant.IsPlatformAdmin(ctx) {
		return true
	}

	// Check tenant match
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		return false
	}

	return tenantID == printer.OrganizationID
}

// printerToTenantResponse converts a printer to the tenant-aware response format.
func printerToTenantResponse(printer *repository.Printer) *PrinterWithTenant {
	var capabilities map[string]interface{}
	_ = json.Unmarshal([]byte(printer.Capabilities), &capabilities)

	return &PrinterWithTenant{
		ID:             printer.ID,
		Name:           printer.Name,
		AgentID:        printer.AgentID,
		TenantID:       printer.OrganizationID,
		OrganizationID: printer.OrganizationID,
		Status:         printer.Status,
		Capabilities:   capabilities,
		CreatedAt:      printer.CreatedAt,
		UpdatedAt:      printer.UpdatedAt,
	}
}

// extractIDFromPath extracts an ID from the URL path.
func extractIDFromPath(path, resource string) string {
	// Simple path parsing - in production use proper router
	parts := splitPath(path)
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
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

// splitPath splits a URL path into components.
func splitPath(path string) []string {
	path = path[1:] // Remove leading slash
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// Import strings package
import "strings"
