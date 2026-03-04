// Package handler provides HTTP handlers for printer discovery by agents.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openprint/openprint/internal/agent"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// AgentDiscoveryConfig holds agent discovery handler dependencies.
type AgentDiscoveryConfig struct {
	AgentRepo        *repository.AgentRepository
	AgentPrinterRepo *repository.AgentPrinterRepository
}

// AgentDiscoveryHandler handles printer discovery from agents.
type AgentDiscoveryHandler struct {
	agentRepo        *repository.AgentRepository
	agentPrinterRepo *repository.AgentPrinterRepository
}

// NewAgentDiscoveryHandler creates a new agent discovery handler.
func NewAgentDiscoveryHandler(cfg AgentDiscoveryConfig) *AgentDiscoveryHandler {
	return &AgentDiscoveryHandler{
		agentRepo:        cfg.AgentRepo,
		agentPrinterRepo: cfg.AgentPrinterRepo,
	}
}

// RegisterPrinters handles printer discovery registration from agents.
// POST /agents/{agent_id}/printers/discovery
func (h *AgentDiscoveryHandler) RegisterPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	agentID := extractAgentID(r.URL.Path)
	if agentID == "" {
		respondError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	var req agent.PrinterDiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Verify agent exists
	_, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Convert printers to repository format
	repoPrinters := make([]*repository.DiscoveredPrinter, len(req.Printers))
	for i, p := range req.Printers {
		repoPrinters[i] = repository.FromDiscoveredPrinter(&p, agentID)
	}

	// If replace is true, delete existing printers first
	if req.Replace {
		printerNames := make([]string, len(req.Printers))
		for i, p := range req.Printers {
			printerNames[i] = p.Name
		}
		if err := h.agentPrinterRepo.DeleteExcept(ctx, agentID, printerNames); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to clean old printers", http.StatusInternalServerError))
			return
		}
	}

	// Register printers
	if err := h.agentPrinterRepo.RegisterPrinters(ctx, repoPrinters); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to register printers", http.StatusInternalServerError))
		return
	}

	// Get updated printer list for response
	printerIDs := make(map[string]string)
	for _, p := range req.Printers {
		if p.PrinterID != "" {
			printerIDs[p.Name] = p.PrinterID
		} else {
			// Try to find the printer by name to get its ID
			found, err := h.agentPrinterRepo.FindByAgentAndName(ctx, agentID, p.Name)
			if err == nil {
				printerIDs[p.Name] = found.ID
			}
		}
	}

	response := agent.PrinterDiscoveryResponse{
		Registered:   len(req.Printers),
		Updated:      0,
		Unregistered: 0,
		PrinterIDs:   printerIDs,
	}

	respondJSON(w, http.StatusOK, response)
}

// GetDiscoveredPrinters returns all printers discovered by an agent.
// GET /agents/{agent_id}/printers
func (h *AgentDiscoveryHandler) GetDiscoveredPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := extractAgentID(r.URL.Path)
	if agentID == "" {
		respondError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	// Verify agent exists
	_, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	printers, err := h.agentPrinterRepo.FindByAgent(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get printers", http.StatusInternalServerError))
		return
	}

	// Convert to response format
	response := make([]*agent.DiscoveredPrinter, len(printers))
	for i, p := range printers {
		response[i] = p.ToDiscoveredPrinter()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": response,
		"count":    len(response),
	})
}

// GetDiscoveredPrinter returns a specific discovered printer.
// GET /agents/{agent_id}/printers/{printer_id}
func (h *AgentDiscoveryHandler) GetDiscoveredPrinter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID, printerID := extractAgentAndPrinterID(r.URL.Path)
	if agentID == "" || printerID == "" {
		respondError(w, apperrors.New("agent ID and printer ID are required", http.StatusBadRequest))
		return
	}

	printer, err := h.agentPrinterRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Verify printer belongs to agent
	if printer.AgentID != agentID {
		respondError(w, apperrors.New("printer not found for this agent", http.StatusNotFound))
		return
	}

	respondJSON(w, http.StatusOK, printer.ToDiscoveredPrinter())
}

// UpdatePrinterStatus updates a discovered printer's status.
// PUT /agents/{agent_id}/printers/{printer_id}/status
func (h *AgentDiscoveryHandler) UpdatePrinterStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID, printerID := extractAgentAndPrinterID(r.URL.Path)
	if agentID == "" || printerID == "" {
		respondError(w, apperrors.New("agent ID and printer ID are required", http.StatusBadRequest))
		return
	}

	var req struct {
		Status agent.PrinterStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate status
	switch req.Status {
	case agent.PrinterStatusIdle, agent.PrinterStatusPrinting, agent.PrinterStatusBusy,
		agent.PrinterStatusOffline, agent.PrinterStatusError, agent.PrinterStatusOutOfPaper,
		agent.PrinterStatusLowToner, agent.PrinterStatusDoorOpen:
		// Valid statuses
	default:
		respondError(w, apperrors.New("invalid printer status", http.StatusBadRequest))
		return
	}

	printer, err := h.agentPrinterRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Verify printer belongs to agent
	if printer.AgentID != agentID {
		respondError(w, apperrors.New("printer not found for this agent", http.StatusNotFound))
		return
	}

	if err := h.agentPrinterRepo.UpdateStatus(ctx, printerID, req.Status); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update printer status", http.StatusInternalServerError))
		return
	}

	// Fetch updated printer
	printer, _ = h.agentPrinterRepo.FindByID(ctx, printerID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printer_id": printerID,
		"status":     req.Status,
		"updated_at": time.Now().Format(time.RFC3339),
	})
}

// UpdatePrinterCapabilities updates a discovered printer's capabilities.
// PUT /agents/{agent_id}/printers/{printer_id}/capabilities
func (h *AgentDiscoveryHandler) UpdatePrinterCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID, printerID := extractAgentAndPrinterID(r.URL.Path)
	if agentID == "" || printerID == "" {
		respondError(w, apperrors.New("agent ID and printer ID are required", http.StatusBadRequest))
		return
	}

	var req struct {
		Capabilities *agent.PrinterCapabilities `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	printer, err := h.agentPrinterRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Verify printer belongs to agent
	if printer.AgentID != agentID {
		respondError(w, apperrors.New("printer not found for this agent", http.StatusNotFound))
		return
	}

	if err := h.agentPrinterRepo.UpdateCapabilities(ctx, printerID, req.Capabilities); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update printer capabilities", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printer_id":   printerID,
		"capabilities": req.Capabilities,
		"updated_at":   time.Now().Format(time.RFC3339),
	})
}

// DeleteDiscoveredPrinter removes a discovered printer.
// DELETE /agents/{agent_id}/printers/{printer_id}
func (h *AgentDiscoveryHandler) DeleteDiscoveredPrinter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID, printerID := extractAgentAndPrinterID(r.URL.Path)
	if agentID == "" || printerID == "" {
		respondError(w, apperrors.New("agent ID and printer ID are required", http.StatusBadRequest))
		return
	}

	printer, err := h.agentPrinterRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Verify printer belongs to agent
	if printer.AgentID != agentID {
		respondError(w, apperrors.New("printer not found for this agent", http.StatusNotFound))
		return
	}

	if err := h.agentPrinterRepo.Delete(ctx, printerID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete printer", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAllDiscoveredPrinters returns all discovered printers across all agents.
// GET /printers/discovered
func (h *AgentDiscoveryHandler) GetAllDiscoveredPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse pagination params
	limit := 100
	offset := 0
	status := r.URL.Query().Get("status")

	if l := r.URL.Query().Get("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			respondError(w, apperrors.New("invalid limit parameter", http.StatusBadRequest))
			return
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil {
			respondError(w, apperrors.New("invalid offset parameter", http.StatusBadRequest))
			return
		}
	}

	var printers []*repository.DiscoveredPrinter
	var total int
	var err error

	if status != "" {
		printers, err = h.agentPrinterRepo.FindByStatus(ctx, agent.PrinterStatus(status))
		total = len(printers)
	} else {
		printers, total, err = h.agentPrinterRepo.List(ctx, limit, offset)
	}

	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get printers", http.StatusInternalServerError))
		return
	}

	// Convert to response format
	response := make([]*agent.DiscoveredPrinter, len(printers))
	for i, p := range printers {
		response[i] = p.ToDiscoveredPrinter()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": response,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// SyncAgentPrinters performs a full sync of printers for an agent.
// POST /agents/{agent_id}/printers/sync
func (h *AgentDiscoveryHandler) SyncAgentPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := extractAgentID(r.URL.Path)
	if agentID == "" {
		respondError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	// Verify agent exists
	agentInfo, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	var req agent.PrinterDiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	req.Replace = true // Sync always replaces

	// Get existing printers to calculate diff
	existingPrinters, _ := h.agentPrinterRepo.FindByAgent(ctx, agentID)
	existingMap := make(map[string]*repository.DiscoveredPrinter)
	for _, p := range existingPrinters {
		existingMap[p.Name] = p
	}

	// Convert and register new printers
	repoPrinters := make([]*repository.DiscoveredPrinter, len(req.Printers))
	for i, p := range req.Printers {
		repoPrinters[i] = repository.FromDiscoveredPrinter(&p, agentID)
	}

	// Get list of new printer names
	printerNames := make([]string, len(req.Printers))
	for i, p := range req.Printers {
		printerNames[i] = p.Name
	}

	// Count printers that will be unregistered
	unregisteredCount := 0
	for _, p := range existingPrinters {
		found := false
		for _, newName := range printerNames {
			if p.Name == newName {
				found = true
				break
			}
		}
		if !found {
			unregisteredCount++
		}
	}

	// Delete printers not in new list
	_ = h.agentPrinterRepo.DeleteExcept(ctx, agentID, printerNames)

	// Register new printers
	if err := h.agentPrinterRepo.RegisterPrinters(ctx, repoPrinters); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to register printers", http.StatusInternalServerError))
		return
	}

	// Calculate stats
	registered := 0
	updated := 0
	printerIDs := make(map[string]string)

	for _, p := range req.Printers {
		if _, exists := existingMap[p.Name]; exists {
			updated++
		} else {
			registered++
		}

		// Get the server-assigned printer ID
		found, _ := h.agentPrinterRepo.FindByAgentAndName(ctx, agentID, p.Name)
		if found != nil {
			printerIDs[p.Name] = found.ID
		}
	}

	// Update agent printer count
	agentInfo.Status = string(agent.AgentStatusOnline)
	// Note: Update printer count in agent info (would need to extend agent repo)

	response := agent.PrinterDiscoveryResponse{
		Registered:   registered,
		Updated:      updated,
		Unregistered: unregisteredCount,
		PrinterIDs:   printerIDs,
	}

	respondJSON(w, http.StatusOK, response)
}

// GetPrinterCapabilities returns a specific printer's capabilities.
// GET /agents/{agent_id}/printers/{printer_id}/capabilities
func (h *AgentDiscoveryHandler) GetPrinterCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID, printerID := extractAgentAndPrinterID(r.URL.Path)
	if agentID == "" || printerID == "" {
		respondError(w, apperrors.New("agent ID and printer ID are required", http.StatusBadRequest))
		return
	}

	printer, err := h.agentPrinterRepo.FindByID(ctx, printerID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Verify printer belongs to agent
	if printer.AgentID != agentID {
		respondError(w, apperrors.New("printer not found for this agent", http.StatusNotFound))
		return
	}

	// Parse and return capabilities
	var capabilities *agent.PrinterCapabilities
	if printer.Capabilities != "" {
		json.Unmarshal([]byte(printer.Capabilities), &capabilities)
	}

	if capabilities == nil {
		capabilities = &agent.PrinterCapabilities{}
	}

	respondJSON(w, http.StatusOK, capabilities)
}

// Helper functions

func extractAgentID(path string) string {
	// Path format: /agents/{agent_id}/printers/...
	// or: /agents/{agent_id}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "agents" {
		return parts[1]
	}
	return ""
}

func extractAgentAndPrinterID(path string) (agentID, printerID string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 && parts[0] == "agents" && parts[2] == "printers" {
		return parts[1], parts[3]
	}
	return "", ""
}
