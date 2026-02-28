// Package handler provides printer-specific HTTP handlers for the registry service.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// PrinterCapabilities represents the capabilities of a printer.
type PrinterCapabilities struct {
	SupportsColor      bool     `json:"supports_color"`
	SupportsDuplex     bool     `json:"supports_duplex"`
	SupportsStapling   bool     `json:"supports_stapling"`
	SupportedMedia     []string `json:"supported_media"`
	SupportedQuality   []string `json:"supported_quality"`
	MaxPaperSize       string   `json:"max_paper_size"`
	MinPaperSize       string   `json:"min_paper_size"`
}

// UpdatePrinterCapabilitiesRequest represents a request to update printer capabilities.
type UpdatePrinterCapabilitiesRequest struct {
	Capabilities PrinterCapabilities `json:"capabilities"`
}

// UpdateCapabilities updates a printer's capabilities.
func (h *Handler) UpdateCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract printer ID from path
	// Path format: /printers/{id}/capabilities
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid printer path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdatePrinterCapabilitiesRequest
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

// GetPrintersByAgent retrieves all printers registered to a specific agent.
func (h *Handler) GetPrintersByAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	// Path format: /agents/{id}/printers
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid agent path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]

	printers, err := h.printerRepo.FindByAgent(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to find printers", http.StatusInternalServerError))
		return
	}

	// Convert to response
	response := make([]map[string]interface{}, len(printers))
	for i, printer := range printers {
		response[i] = printerToResponse(printer)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": response,
		"count":    len(printers),
	})
}

// SetPrinterStatus updates the status of a printer.
func (h *Handler) SetPrinterStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract printer ID from path
	// Path format: /printers/{id}/status
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid printer path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
		respondError(w, apperrors.ErrNotFound)
		return
	}

	printer.Status = req.Status

	if err := h.printerRepo.Update(ctx, printer); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update printer status", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, printerToResponse(printer))
}

// GetAgentPrinters retrieves all printers and their associated agent information.
func (h *Handler) GetAgentPrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all agents
	agents, _, err := h.agentRepo.List(ctx, 1000, 0)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list agents", http.StatusInternalServerError))
		return
	}

	// Build response with agent and printer information
	result := make([]map[string]interface{}, 0, len(agents))

	for _, agent := range agents {
		printers, _ := h.printerRepo.FindByAgent(ctx, agent.ID)

		printerInfos := make([]map[string]interface{}, len(printers))
		for i, printer := range printers {
			printerInfos[i] = map[string]interface{}{
				"printer_id": printer.ID,
				"name":       printer.Name,
				"status":     printer.Status,
			}
		}

		result = append(result, map[string]interface{}{
			"agent_id":      agent.ID,
			"agent_name":    agent.Name,
			"agent_status":  agent.Status,
			"hostname":      agent.Hostname,
			"last_heartbeat": agent.LastHeartbeat.Format(time.RFC3339),
			"printers":      printerInfos,
			"printer_count": len(printers),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": result,
		"count":  len(result),
	})
}

// AgentStatusResponse represents the combined status of an agent and its printers.
type AgentStatusResponse struct {
	AgentID       string               `json:"agent_id"`
	AgentName     string               `json:"agent_name"`
	AgentStatus   string               `json:"agent_status"`
	Hostname      string               `json:"hostname"`
	LastHeartbeat string               `json:"last_heartbeat"`
	Printers      []PrinterInfo        `json:"printers"`
	PrinterCount  int                  `json:"printer_count"`
}

// PrinterInfo represents basic printer information.
type PrinterInfo struct {
	PrinterID string `json:"printer_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

// GetOnlineAgents returns all currently online agents.
func (h *Handler) GetOnlineAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents, err := h.agentRepo.FindByStatus(ctx, "online")
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to find online agents", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(agents))
	for i, agent := range agents {
		response[i] = map[string]interface{}{
			"agent_id":      agent.ID,
			"name":          agent.Name,
			"hostname":      agent.Hostname,
			"last_heartbeat": agent.LastHeartbeat.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": response,
		"count":  len(agents),
	})
}

// GetAvailablePrinters returns all printers that are currently online and available.
func (h *Handler) GetAvailablePrinters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	printers, total, err := h.printerRepo.List(ctx, 1000, 0)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list printers", http.StatusInternalServerError))
		return
	}

	// Filter for online printers
	available := make([]*repository.Printer, 0)
	for _, printer := range printers {
		if printer.Status == "online" {
			available = append(available, printer)
		}
	}

	response := make([]map[string]interface{}, len(available))
	for i, printer := range available {
		response[i] = printerToResponse(printer)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": response,
		"count":    len(available),
		"total":    total,
	})
}
