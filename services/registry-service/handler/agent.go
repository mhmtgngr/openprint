// Package handler provides HTTP handlers for the registry service.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// Config holds handler dependencies.
type Config struct {
	AgentRepo        *repository.AgentRepository
	PrinterRepo      *repository.PrinterRepository
	HeartbeatTimeout time.Duration
}

// Handler provides registry service HTTP handlers.
type Handler struct {
	agentRepo        *repository.AgentRepository
	printerRepo      *repository.PrinterRepository
	heartbeatTimeout time.Duration
}

// New creates a new handler instance.
func New(cfg Config) *Handler {
	return &Handler{
		agentRepo:        cfg.AgentRepo,
		printerRepo:      cfg.PrinterRepo,
		heartbeatTimeout: cfg.HeartbeatTimeout,
	}
}

// RegisterAgentRequest represents an agent registration request.
type RegisterAgentRequest struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Hostname     string `json:"hostname"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// RegisterAgent handles agent registration.
func (h *Handler) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}

	// Generate agent ID
	agentID := uuid.New().String()

	// Create agent
	agent := &repository.Agent{
		ID:             agentID,
		Name:           req.Name,
		Version:        req.Version,
		OS:             req.OS,
		Architecture:   req.Architecture,
		Hostname:       req.Hostname,
		OrganizationID: req.OrganizationID,
		Status:         "online",
		LastHeartbeat:  time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.agentRepo.Create(ctx, agent); err != nil {
		// Log the underlying error for debugging
		fmt.Printf("[ERROR] Failed to register agent: %v\n", err)
		respondError(w, apperrors.Wrap(err, "failed to register agent", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"agent_id":    agentID,
		"name":        agent.Name,
		"status":      agent.Status,
		"created_at":  agent.CreatedAt.Format(time.RFC3339),
	})
}

// AgentHandler handles individual agent operations.
func (h *Handler) AgentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract agent ID from path
	// Path format: /agents/{id} or /agents/{id}/heartbeat
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid agent path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]

	switch r.Method {
	case http.MethodGet:
		h.getAgent(w, r, ctx, agentID)
	case http.MethodPut:
		h.updateAgent(w, r, ctx, agentID)
	case http.MethodDelete:
		h.deleteAgent(w, r, ctx, agentID)
	case http.MethodPost:
		// Check if this is a heartbeat
		if strings.HasSuffix(r.URL.Path, "/heartbeat") {
			h.heartbeat(w, r, ctx, agentID)
		} else {
			http.Error(w, "unknown action", http.StatusNotFound)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request, ctx context.Context, agentID string) {
	agent, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, agentToResponse(agent))
}

func (h *Handler) updateAgent(w http.ResponseWriter, r *http.Request, ctx context.Context, agentID string) {
	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	agent, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Version != "" {
		agent.Version = req.Version
	}

	if err := h.agentRepo.Update(ctx, agent); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update agent", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, agentToResponse(agent))
}

func (h *Handler) deleteAgent(w http.ResponseWriter, r *http.Request, ctx context.Context, agentID string) {
	if err := h.agentRepo.Delete(ctx, agentID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete agent", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request, ctx context.Context, agentID string) {
	if err := h.agentRepo.UpdateHeartbeat(ctx, agentID, time.Now()); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update heartbeat", http.StatusInternalServerError))
		return
	}

	// Return 200 OK with status confirmation
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agent_id": agentID,
		"status":   "online",
	})
}

// ListAgents handles listing all agents.
func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse pagination params
	limit := 100
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Get agents
	agents, total, err := h.agentRepo.List(ctx, limit, offset)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list agents", http.StatusInternalServerError))
		return
	}

	// Convert to response
	response := make([]map[string]interface{}, len(agents))
	for i, agent := range agents {
		response[i] = agentToResponse(agent)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HeartbeatMonitor periodically checks for offline agents.
func (h *Handler) HeartbeatMonitor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkOfflineAgents(ctx)
		}
	}
}

func (h *Handler) checkOfflineAgents(ctx context.Context) {
	threshold := time.Now().Add(-h.heartbeatTimeout)

	// Find all agents that haven't heartbeat recently
	agents, _, err := h.agentRepo.List(ctx, 1000, 0)
	if err != nil {
		return
	}

	for _, agent := range agents {
		if agent.Status == "online" && agent.LastHeartbeat.Before(threshold) {
			// Mark as offline
			agent.Status = "offline"
			h.agentRepo.Update(ctx, agent)
		}
	}
}

// PrinterHandler handles printer operations.
func (h *Handler) PrinterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract printer ID and sub-path from path
	// Path format: /printers/{id} or /printers/{id}/status or /printers/{id}/capabilities
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid printer path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	// Check for sub-paths (status, capabilities)
	if len(parts) >= 3 {
		subPath := parts[2]
		switch subPath {
		case "status":
			if r.Method == http.MethodPut {
				// SetPrinterStatus is in printer.go, but we handle it here for routing
				h.setPrinterStatus(w, r, ctx, printerID)
				return
			}
		case "capabilities":
			if r.Method == http.MethodPut {
				// UpdateCapabilities is in printer.go, but we handle it here for routing
				h.updateCapabilities(w, r, ctx, printerID)
				return
			}
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getPrinter(w, r, ctx, printerID)
	case http.MethodPut:
		h.updatePrinter(w, r, ctx, printerID)
	case http.MethodDelete:
		h.deletePrinter(w, r, ctx, printerID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// RegisterPrinter handles printer registration.
func (h *Handler) RegisterPrinter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name           string `json:"name"`
		AgentID        string `json:"agent_id"`
		OrganizationID string `json:"organization_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Verify agent exists
	agent, err := h.agentRepo.FindByID(ctx, req.AgentID)
	if err != nil {
		respondError(w, apperrors.New("agent not found", http.StatusBadRequest))
		return
	}

	// Generate printer ID
	printerID := uuid.New().String()

	// Create printer
	printer := &repository.Printer{
		ID:             printerID,
		Name:           req.Name,
		AgentID:        req.AgentID,
		OrganizationID: req.OrganizationID,
		Status:         "online",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.printerRepo.Create(ctx, printer); err != nil {
		fmt.Printf("[ERROR] Failed to create printer: %v\n", err)
		respondError(w, apperrors.Wrap(err, "failed to register printer", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"printer_id": printerID,
		"name":       printer.Name,
		"status":     printer.Status,
		"agent":      agent.Name,
		"created_at": printer.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) getPrinter(w http.ResponseWriter, r *http.Request, ctx context.Context, printerID string) {
	printer, err := h.printerRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, printerToResponse(printer))
}

func (h *Handler) updatePrinter(w http.ResponseWriter, r *http.Request, ctx context.Context, printerID string) {
	var req struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	printer, err := h.printerRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if req.Name != "" {
		printer.Name = req.Name
	}
	if req.Status != "" {
		printer.Status = req.Status
	}

	if err := h.printerRepo.Update(ctx, printer); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update printer", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, printerToResponse(printer))
}

func (h *Handler) deletePrinter(w http.ResponseWriter, r *http.Request, ctx context.Context, printerID string) {
	if err := h.printerRepo.Delete(ctx, printerID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete printer", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setPrinterStatus handles PUT /printers/{id}/status
func (h *Handler) setPrinterStatus(w http.ResponseWriter, r *http.Request, ctx context.Context, printerID string) {
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Status != "online" && req.Status != "offline" && req.Status != "busy" && req.Status != "error" {
		respondError(w, apperrors.New("invalid status value", http.StatusBadRequest))
		return
	}

	printer, err := h.printerRepo.FindByID(ctx, printerID)
	if err != nil {
		fmt.Printf("[ERROR] Failed to find printer %s: %v\n", printerID, err)
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if printer == nil {
		fmt.Printf("[ERROR] Printer is nil after FindByID for ID %s\n", printerID)
		respondError(w, apperrors.New("printer not found", http.StatusNotFound))
		return
	}

	printer.Status = req.Status

	if err := h.printerRepo.Update(ctx, printer); err != nil {
		fmt.Printf("[ERROR] Failed to update printer %s: %v\n", printerID, err)
		respondError(w, apperrors.Wrap(err, "failed to update printer status", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, printerToResponse(printer))
}

// updateCapabilities handles PUT /printers/{id}/capabilities
func (h *Handler) updateCapabilities(w http.ResponseWriter, r *http.Request, ctx context.Context, printerID string) {
	var req struct {
		Capabilities interface{} `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	printer, err := h.printerRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Serialize capabilities to JSON
	capabilitiesJSON, err := json.Marshal(req.Capabilities)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to serialize capabilities", http.StatusInternalServerError))
		return
	}

	printer.Capabilities = string(capabilitiesJSON)

	if err := h.printerRepo.Update(ctx, printer); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update printer", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, printerToResponse(printer))
}

// ListPrinters handles listing all printers.
func (h *Handler) ListPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse pagination params
	limit := 100
	offset := 0
	orgID := r.URL.Query().Get("organization_id")

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var printers []*repository.Printer
	var total int
	var err error

	if orgID != "" {
		printers, total, err = h.printerRepo.FindByOrganization(ctx, orgID, limit, offset)
	} else {
		printers, total, err = h.printerRepo.List(ctx, limit, offset)
	}

	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list printers", http.StatusInternalServerError))
		return
	}

	// Convert to response
	response := make([]map[string]interface{}, len(printers))
	for i, printer := range printers {
		response[i] = printerToResponse(printer)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": response,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// Helper functions

func agentToResponse(agent *repository.Agent) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"version":       agent.Version,
		"os":            agent.OS,
		"architecture":  agent.Architecture,
		"hostname":      agent.Hostname,
		"status":        agent.Status,
		"last_heartbeat": agent.LastHeartbeat.Format(time.RFC3339),
		"created_at":    agent.CreatedAt.Format(time.RFC3339),
	}
}

func printerToResponse(printer *repository.Printer) map[string]interface{} {
	return map[string]interface{}{
		"printer_id": printer.ID,
		"name":       printer.Name,
		"agent_id":   printer.AgentID,
		"status":     printer.Status,
		"created_at": printer.CreatedAt.Format(time.RFC3339),
	}
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
