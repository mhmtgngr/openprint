// Package handler provides HTTP handlers for agent group management.
package handler

import (
	"encoding/json"
	"fmt"
	stderrors "errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// ListAgentGroups handles listing all agent groups.
func (h *Handler) ListAgentGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limit := 100
	offset := 0
	orgID := r.URL.Query().Get("organization_id")
	ownerUserID := r.URL.Query().Get("owner_user_id")

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var groups []*repository.AgentGroup
	var total int
	var err error

	// Filter based on query parameters
	if orgID != "" {
		groups, err = h.agentGroupRepo.FindByOrganization(ctx, orgID)
		total = len(groups)
		// Apply pagination
		if offset < len(groups) {
			end := offset + limit
			if end > len(groups) {
				end = len(groups)
			}
			groups = groups[offset:end]
		} else {
			groups = []*repository.AgentGroup{}
		}
	} else if ownerUserID != "" {
		groups, err = h.agentGroupRepo.FindByOwner(ctx, ownerUserID)
		total = len(groups)
		// Apply pagination
		if offset < len(groups) {
			end := offset + limit
			if end > len(groups) {
				end = len(groups)
			}
			groups = groups[offset:end]
		} else {
			groups = []*repository.AgentGroup{}
		}
	} else {
		groups, total, err = h.agentGroupRepo.List(ctx, limit, offset)
	}

	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list agent groups", http.StatusInternalServerError))
		return
	}

	// Convert to response
	response := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		response[i] = agentGroupToResponse(group)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"groups": response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateAgentGroup handles creating a new agent group.
func (h *Handler) CreateAgentGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name           string                 `json:"name"`
		Description    string                 `json:"description,omitempty"`
		OrganizationID string                 `json:"organization_id,omitempty"`
		OwnerUserID    string                 `json:"owner_user_id,omitempty"`
		Type           string                 `json:"type"`
		Location       string                 `json:"location,omitempty"`
		Tags           []string               `json:"tags,omitempty"`
		PolicyID       string                 `json:"policy_id,omitempty"`
		Config         map[string]interface{} `json:"config,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.Type == "" {
		req.Type = "custom" // Default to custom type
	}

	group := &repository.AgentGroup{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Description:    req.Description,
		OrganizationID: req.OrganizationID,
		OwnerUserID:    req.OwnerUserID,
		Type:           req.Type,
		Location:       req.Location,
		Tags:           req.Tags,
		PolicyID:       req.PolicyID,
	}

	if err := h.agentGroupRepo.Create(ctx, group); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create agent group", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, agentGroupToResponse(group))
}

// GetAgentGroup handles retrieving a single agent group.
func (h *Handler) GetAgentGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID from path
	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-groups/")
	groupID = strings.Split(groupID, "/")[0]

	group, err := h.agentGroupRepo.FindByID(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Get agents in this group
	agentIDs, err := h.agentGroupRepo.GetAgentsInGroup(ctx, groupID)
	if err != nil {
		agentIDs = []string{}
	}

	response := agentGroupToResponse(group)
	response["agent_ids"] = agentIDs
	response["agent_count"] = len(agentIDs)

	respondJSON(w, http.StatusOK, response)
}

// UpdateAgentGroup handles updating an agent group.
func (h *Handler) UpdateAgentGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID from path
	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-groups/")
	groupID = strings.Split(groupID, "/")[0]

	group, err := h.agentGroupRepo.FindByID(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	var req struct {
		Name        string   `json:"name,omitempty"`
		Description string   `json:"description,omitempty"`
		Location    string   `json:"location,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		PolicyID    string   `json:"policy_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update only provided fields
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if req.Location != "" {
		group.Location = req.Location
	}
	if req.Tags != nil {
		group.Tags = req.Tags
	}
	if req.PolicyID != "" {
		group.PolicyID = req.PolicyID
	}

	if err := h.agentGroupRepo.Update(ctx, group); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update agent group", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, agentGroupToResponse(group))
}

// DeleteAgentGroup handles deleting an agent group.
func (h *Handler) DeleteAgentGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID from path
	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-groups/")
	groupID = strings.Split(groupID, "/")[0]

	// Remove all agents from the group first
	h.agentGroupRepo.RemoveAllAgentsFromGroup(ctx, groupID)

	// Delete the group
	if err := h.agentGroupRepo.Delete(ctx, groupID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete agent group", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignAgentsToGroup handles assigning agents to a group.
func (h *Handler) AssignAgentsToGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID from path
	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-groups/")
	groupID = strings.Split(groupID, "/")[0]

	// Verify group exists
	_, err := h.agentGroupRepo.FindByID(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	var req struct {
		AgentIDs []string `json:"agent_ids"`
		Replace  bool     `json:"replace,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Get user ID from context for "added_by" field
	var addedBy string
	if userIDValue := ctx.Value("user_id"); userIDValue != nil {
		addedBy, _ = userIDValue.(string)
	}

	var assigned, removed int
	var failedAgents []string

	if req.Replace {
		// Get current agents
		currentAgents, err := h.agentGroupRepo.GetAgentsInGroup(ctx, groupID)
		if err != nil {
			currentAgents = []string{}
		}
		removed = len(currentAgents)

		// Replace all agents
		if err := h.agentGroupRepo.SetAgentsForGroup(ctx, groupID, req.AgentIDs, addedBy); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to assign agents to group", http.StatusInternalServerError))
			return
		}
		assigned = len(req.AgentIDs)
	} else {
		// Add each agent to the group
		for _, agentID := range req.AgentIDs {
			// Verify agent exists
			if _, err := h.agentRepo.FindByID(ctx, agentID); err != nil {
				failedAgents = append(failedAgents, agentID)
				continue
			}

			if err := h.agentGroupRepo.AddAgentToGroup(ctx, groupID, agentID, addedBy); err != nil {
				failedAgents = append(failedAgents, agentID)
			} else {
				assigned++
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"assigned":      assigned,
		"removed":       removed,
		"failed_agents": failedAgents,
	})
}

// RemoveAgentFromGroup handles removing an agent from a group.
func (h *Handler) RemoveAgentFromGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID and agent ID from path
	// Path format: /api/v1/agent-groups/{groupID}/agents/{agentID}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}

	groupID := parts[3]
	agentID := parts[4]

	if err := h.agentGroupRepo.RemoveAgentFromGroup(ctx, groupID, agentID); err != nil {
		if stderrors.Is(err, fmt.Errorf("agent not in group")) {
			respondError(w, apperrors.ErrNotFound)
		} else {
			respondError(w, apperrors.Wrap(err, "failed to remove agent from group", http.StatusInternalServerError))
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetGroupAgents handles retrieving all agents in a group.
func (h *Handler) GetGroupAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID from path
	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-groups/")
	groupID = strings.Split(groupID, "/")[0]

	// Verify group exists
	_, err := h.agentGroupRepo.FindByID(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Get agent IDs in the group
	agentIDs, err := h.agentGroupRepo.GetAgentsInGroup(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get group agents", http.StatusInternalServerError))
		return
	}

	// Fetch full agent details
	agents := make([]map[string]interface{}, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		agent, err := h.agentRepo.FindByID(ctx, agentID)
		if err == nil {
			agents = append(agents, agentToResponse(agent))
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	})
}

// GetAgentGroups handles retrieving all groups an agent belongs to.
func (h *Handler) GetAgentGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	agentID := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	agentID = strings.Split(agentID, "/")[0]

	// Verify agent exists
	_, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Get groups for this agent
	groups, err := h.agentGroupRepo.GetGroupsForAgent(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get agent groups", http.StatusInternalServerError))
		return
	}

	// Convert to response
	response := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		response[i] = agentGroupToResponse(group)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"groups": response,
		"total":  len(response),
	})
}

// GetGroupStatus handles retrieving the status of all agents in a group.
func (h *Handler) GetGroupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract group ID from path
	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-groups/")
	groupID = strings.Split(groupID, "/")[0]

	// Verify group exists
	_, err := h.agentGroupRepo.FindByID(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Get agent IDs in the group
	agentIDs, err := h.agentGroupRepo.GetAgentsInGroup(ctx, groupID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get group agents", http.StatusInternalServerError))
		return
	}

	status := map[string]interface{}{
		"group_id":    groupID,
		"total_agents": len(agentIDs),
		"online_agents": 0,
		"offline_agents": 0,
		"error_agents": 0,
		"total_printers": 0,
		"online_printers": 0,
		"active_jobs": 0,
		"last_updated": time.Now().Format(time.RFC3339),
	}

	// Count agent statuses
	for _, agentID := range agentIDs {
		agent, err := h.agentRepo.FindByID(ctx, agentID)
		if err != nil {
			continue
		}

		switch agent.Status {
		case "online":
			status["online_agents"] = status["online_agents"].(int) + 1
		case "offline":
			status["offline_agents"] = status["offline_agents"].(int) + 1
		case "error":
			status["error_agents"] = status["error_agents"].(int) + 1
		}

		// Count printers for this agent
		printers, _ := h.printerRepo.FindByAgent(ctx, agentID)
		status["total_printers"] = status["total_printers"].(int) + len(printers)
		for _, p := range printers {
			if p.Status == "online" {
				status["online_printers"] = status["online_printers"].(int) + 1
			}
		}
	}

	respondJSON(w, http.StatusOK, status)
}

func agentGroupToResponse(group *repository.AgentGroup) map[string]interface{} {
	return map[string]interface{}{
		"id":              group.ID,
		"name":            group.Name,
		"description":     group.Description,
		"organization_id": group.OrganizationID,
		"owner_user_id":   group.OwnerUserID,
		"type":            group.Type,
		"location":        group.Location,
		"tags":            group.Tags,
		"policy_id":       group.PolicyID,
		"created_at":      group.CreatedAt.Format(time.RFC3339),
		"updated_at":      group.UpdatedAt.Format(time.RFC3339),
	}
}
