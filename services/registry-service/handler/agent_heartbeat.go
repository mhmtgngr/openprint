// Package handler provides enhanced heartbeat handlers for agents.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openprint/openprint/internal/agent"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// AgentHeartbeatConfig holds agent heartbeat handler dependencies.
type AgentHeartbeatConfig struct {
	AgentRepo        *repository.AgentRepository
	AgentPrinterRepo *repository.AgentPrinterRepository
	JobAssignmentFn  func(ctx context.Context, agentID string) (int, error)
	HeartbeatTimeout time.Duration
}

// AgentHeartbeatHandler handles enhanced heartbeat requests from agents.
type AgentHeartbeatHandler struct {
	agentRepo        *repository.AgentRepository
	agentPrinterRepo *repository.AgentPrinterRepository
	jobAssignmentFn  func(ctx context.Context, agentID string) (int, error)
	heartbeatTimeout time.Duration
}

// NewAgentHeartbeatHandler creates a new agent heartbeat handler.
func NewAgentHeartbeatHandler(cfg AgentHeartbeatConfig) *AgentHeartbeatHandler {
	return &AgentHeartbeatHandler{
		agentRepo:        cfg.AgentRepo,
		agentPrinterRepo: cfg.AgentPrinterRepo,
		jobAssignmentFn:  cfg.JobAssignmentFn,
		heartbeatTimeout: cfg.HeartbeatTimeout,
	}
}

// HandleHeartbeat processes heartbeat requests from agents.
// POST /agents/{agent_id}/heartbeat
func (h *AgentHeartbeatHandler) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	agentID := extractAgentIDFromPath(r.URL.Path)
	if agentID == "" {
		respondError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	var req agent.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate agent ID matches path
	if req.AgentID != agentID {
		respondError(w, apperrors.New("agent ID mismatch", http.StatusBadRequest))
		return
	}

	// Verify agent exists
	agentInfo, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Update heartbeat and status
	now := time.Now()
	agentInfo.Status = string(req.Status)
	agentInfo.LastHeartbeat = now

	if req.SessionState != "" {
		// Store session state (would need to extend agent model or add separate table)
	}

	if err := h.agentRepo.Update(ctx, agentInfo); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update heartbeat", http.StatusInternalServerError))
		return
	}

	// Register printers if provided
	if len(req.Printers) > 0 {
		repoPrinters := make([]*repository.DiscoveredPrinter, len(req.Printers))
		for i, p := range req.Printers {
			repoPrinters[i] = repository.FromDiscoveredPrinter(&p, agentID)
		}
		// Non-blocking update
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			h.agentPrinterRepo.RegisterPrinters(ctx, repoPrinters)
		}()
	}

	// Get pending job count
	pendingJobs := 0
	if h.jobAssignmentFn != nil {
		pendingJobs, _ = h.jobAssignmentFn(ctx, agentID)
	}

	// Generate any pending commands
	commands := h.generateCommands(agentInfo, req)

	// Build response
	response := agent.HeartbeatResponse{
		Status:               "ok",
		ServerTime:           now,
		PendingJobs:          pendingJobs,
		Commands:             commands,
		ConfigurationUpdates: make(map[string]interface{}),
	}

	// Add configuration updates if needed
	if shouldSendConfig(agentInfo, req) {
		response.ConfigurationUpdates["heartbeat_interval"] = 30 // seconds
	}

	respondJSON(w, http.StatusOK, response)
}

// GetHeartbeatStatus returns the heartbeat status of an agent.
// GET /agents/{agent_id}/heartbeat/status
func (h *AgentHeartbeatHandler) GetHeartbeatStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := extractAgentIDFromPath(r.URL.Path)
	if agentID == "" {
		respondError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	agentInfo, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	now := time.Now()
	timeSinceHeartbeat := now.Sub(agentInfo.LastHeartbeat)
	isOnline := timeSinceHeartbeat < h.heartbeatTimeout

	response := map[string]interface{}{
		"agent_id":                  agentInfo.ID,
		"status":                    agentInfo.Status,
		"is_online":                 isOnline,
		"last_heartbeat":            agentInfo.LastHeartbeat.Format(time.RFC3339),
		"time_since_heartbeat":      timeSinceHeartbeat.String(),
		"heartbeat_timeout_seconds": int(h.heartbeatTimeout.Seconds()),
		"next_heartbeat_expected":   agentInfo.LastHeartbeat.Add(h.heartbeatTimeout).Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, response)
}

// BatchHeartbeat handles heartbeat from multiple agents at once.
// POST /agents/heartbeat/batch
func (h *AgentHeartbeatHandler) BatchHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Heartbeats []agent.HeartbeatRequest `json:"heartbeats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if len(req.Heartbeats) > 100 {
		respondError(w, apperrors.New("maximum 100 heartbeats per batch", http.StatusBadRequest))
		return
	}

	now := time.Now()
	responses := make([]map[string]interface{}, len(req.Heartbeats))
	successCount := 0

	for i, hb := range req.Heartbeats {
		agentInfo, err := h.agentRepo.FindByID(ctx, hb.AgentID)
		if err != nil {
			responses[i] = map[string]interface{}{
				"agent_id": hb.AgentID,
				"status":   "error",
				"error":    "agent not found",
			}
			continue
		}

		agentInfo.Status = string(hb.Status)
		agentInfo.LastHeartbeat = now

		if err := h.agentRepo.Update(ctx, agentInfo); err != nil {
			responses[i] = map[string]interface{}{
				"agent_id": hb.AgentID,
				"status":   "error",
				"error":    "failed to update",
			}
			continue
		}

		responses[i] = map[string]interface{}{
			"agent_id":    hb.AgentID,
			"status":      "ok",
			"server_time": now.Format(time.RFC3339),
		}
		successCount++
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"processed":  len(req.Heartbeats),
		"successful": successCount,
		"failed":     len(req.Heartbeats) - successCount,
		"responses":  responses,
	})
}

// GetAgentMetrics returns metrics about agent heartbeat status.
// GET /agents/heartbeat/metrics
func (h *AgentHeartbeatHandler) GetAgentMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all agents
	agents, _, err := h.agentRepo.List(ctx, 1000, 0)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get agents", http.StatusInternalServerError))
		return
	}

	now := time.Now()
	onlineCount := 0
	offlineCount := 0
	errorCount := 0
	staleCount := 0

	for _, agent := range agents {
		timeSinceHeartbeat := now.Sub(agent.LastHeartbeat)
		isStale := timeSinceHeartbeat > h.heartbeatTimeout

		switch agent.Status {
		case "online":
			if isStale {
				staleCount++
			} else {
				onlineCount++
			}
		case "offline":
			offlineCount++
		case "error":
			errorCount++
		default:
			if isStale {
				staleCount++
			}
		}
	}

	metrics := map[string]interface{}{
		"total_agents":              len(agents),
		"online_agents":             onlineCount,
		"offline_agents":            offlineCount,
		"error_agents":              errorCount,
		"stale_agents":              staleCount,
		"heartbeat_timeout_seconds": int(h.heartbeatTimeout.Seconds()),
		"timestamp":                 now.Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, metrics)
}

// MarkStaleAgents marks agents as offline if they haven't sent heartbeat.
// This is typically called by a background job.
func (h *AgentHeartbeatHandler) MarkStaleAgents(ctx context.Context) (int, error) {
	threshold := time.Now().Add(-h.heartbeatTimeout)
	count, err := h.agentRepo.MarkOfflineBefore(ctx, threshold)
	if err != nil {
		return 0, fmt.Errorf("mark stale agents: %w", err)
	}
	return int(count), nil
}

// generateCommands generates commands for an agent based on its state.
func (h *AgentHeartbeatHandler) generateCommands(agentInfo *repository.Agent, req agent.HeartbeatRequest) []agent.AgentCommand {
	commands := make([]agent.AgentCommand, 0)

	// Command to refresh printers if printer count is zero
	if req.PrinterCount == 0 {
		commands = append(commands, agent.AgentCommand{
			CommandID: fmt.Sprintf("cmd-%d", time.Now().Unix()),
			Type:      "refresh_printers",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})
	}

	// Command to update configuration if agent version is old
	// This is a placeholder for version checking logic
	if agentInfo.Version != "" && agentInfo.Version < "2.0.0" {
		commands = append(commands, agent.AgentCommand{
			CommandID: fmt.Sprintf("update-%d", time.Now().Unix()),
			Type:      "update_available",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
	}

	return commands
}

// shouldSendConfig determines if configuration should be sent to an agent.
func shouldSendConfig(agentInfo *repository.Agent, req agent.HeartbeatRequest) bool {
	// Send config if it's been a while since the agent started
	timeSinceCreation := time.Since(agentInfo.CreatedAt)
	return timeSinceCreation > 1*time.Hour || req.PrinterCount == 0
}

// Helper functions

func extractAgentIDFromPath(path string) string {
	// Path format: /agents/{agent_id}/heartbeat
	// Simple path parsing for this specific case
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "agents" {
		return parts[1]
	}
	return ""
}
