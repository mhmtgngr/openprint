// Package handler provides HTTP handlers for the registry service.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// Config holds handler dependencies.
type Config struct {
	AgentRepo        *repository.AgentRepository
	PrinterRepo      *repository.PrinterRepository
	HeartbeatTimeout time.Duration
	Metrics          *prometheus.Metrics
	ServiceName      string
}

// Handler provides registry service HTTP handlers.
type Handler struct {
	agentRepo        *repository.AgentRepository
	printerRepo      *repository.PrinterRepository
	heartbeatTimeout time.Duration
	metrics          *prometheus.Metrics
	serviceName      string
}

// New creates a new handler instance.
func New(cfg Config) *Handler {
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "registry-service"
	}
	return &Handler{
		agentRepo:        cfg.AgentRepo,
		printerRepo:      cfg.PrinterRepo,
		heartbeatTimeout: cfg.HeartbeatTimeout,
		metrics:          cfg.Metrics,
		serviceName:      serviceName,
	}
}

// RegisterAgentRequest represents an agent registration request.
type RegisterAgentRequest struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	OS             string `json:"os"`
	Architecture   string `json:"architecture"`
	Hostname       string `json:"hostname"`
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
		if h.metrics != nil {
			prometheus.RecordPrinterMetric(h.metrics, h.serviceName, agent.OrganizationID, agent.ID, "register_failed")
		}
		respondError(w, apperrors.Wrap(err, "failed to register agent", http.StatusInternalServerError))
		return
	}

	// Record agent registration metric
	if h.metrics != nil {
		prometheus.RecordPrinterMetric(h.metrics, h.serviceName, agent.OrganizationID, agent.ID, "register")
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"agent_id":   agentID,
		"name":       agent.Name,
		"status":     agent.Status,
		"created_at": agent.CreatedAt.Format(time.RFC3339),
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
		} else if strings.Contains(r.URL.Path, "/printers/discover") {
			h.registerDiscoveredPrinters(w, r, ctx, agentID)
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

	// Record heartbeat metric
	if h.metrics != nil {
		prometheus.RecordPrinterMetric(h.metrics, h.serviceName, "", agentID, "heartbeat")
	}

	// Return 200 OK with status confirmation
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agent_id": agentID,
		"status":   "online",
	})
}

// registerDiscoveredPrinters handles bulk printer registration from an agent.
func (h *Handler) registerDiscoveredPrinters(w http.ResponseWriter, r *http.Request, ctx context.Context, agentID string) {
	// Verify agent exists
	agent, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.New("agent not found", http.StatusNotFound))
		return
	}

	var req struct {
		Printers []struct {
			Name           string      `json:"name"`
			DisplayName    string      `json:"display_name"`
			Driver         string      `json:"driver"`
			Port           string      `json:"port"`
			ConnectionType string      `json:"connection_type"`
			Status         string      `json:"status"`
			IsDefault      bool        `json:"is_default"`
			IsShared       bool        `json:"is_shared"`
			ShareName      string      `json:"share_name"`
			Location       string      `json:"location"`
			Capabilities   interface{} `json:"capabilities"`
		} `json:"printers"`
		Replace bool `json:"replace"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	registered := 0
	printerIDs := make(map[string]string)

	for _, p := range req.Printers {
		printerID := uuid.New().String()

		// Marshal capabilities to JSON string
		capsJSON := ""
		if p.Capabilities != nil {
			if capsBytes, err := json.Marshal(p.Capabilities); err == nil {
				capsJSON = string(capsBytes)
			}
		}

		printer := &repository.Printer{
			ID:             printerID,
			Name:           p.Name,
			AgentID:        agentID,
			OrganizationID: agent.OrganizationID,
			Status:         "online",
			Capabilities:   capsJSON,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := h.printerRepo.Create(ctx, printer); err != nil {
			fmt.Printf("[WARN] Failed to register printer %s: %v\n", p.Name, err)
			continue
		}

		// Record printer registration metric
		if h.metrics != nil {
			prometheus.RecordPrinterMetric(h.metrics, h.serviceName, agent.OrganizationID, printerID, "register")
		}

		printerIDs[p.Name] = printerID
		registered++
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"registered":  registered,
		"printer_ids": printerIDs,
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
		if h.metrics != nil {
			prometheus.RecordPrinterMetric(h.metrics, h.serviceName, printer.OrganizationID, printerID, "register_failed")
		}
		respondError(w, apperrors.Wrap(err, "failed to register printer", http.StatusInternalServerError))
		return
	}

	// Record printer registration metric
	if h.metrics != nil {
		prometheus.RecordPrinterMetric(h.metrics, h.serviceName, printer.OrganizationID, printerID, "register")
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
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Helper functions

func agentToResponse(agent *repository.Agent) map[string]interface{} {
	return map[string]interface{}{
		"id":              agent.ID,
		"name":            agent.Name,
		"orgId":           agent.OrganizationID,
		"status":          agent.Status,
		"platform":        agent.OS,
		"platformVersion": agent.Version,
		"agentVersion":    agent.Version,
		"ipAddress":       "",
		"lastHeartbeat":   agent.LastHeartbeat.Format(time.RFC3339),
		"createdAt":       agent.CreatedAt.Format(time.RFC3339),
		"capabilities": map[string]interface{}{
			"supportedFormats": []string{"PDF", "PNG", "JPEG"},
			"maxJobSize":       104857600,
			"supportsColor":    true,
			"supportsDuplex":   true,
		},
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

// AgentStatusUpdate represents a real-time agent status update sent over WebSocket.
type AgentStatusUpdate struct {
	AgentID        string                 `json:"agent_id"`
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	LastHeartbeat  time.Time              `json:"last_heartbeat"`
	Version        string                 `json:"version"`
	OS             string                 `json:"os"`
	Hostname       string                 `json:"hostname"`
	Architecture   string                 `json:"architecture"`
	OnlinePrinters int                    `json:"online_printers"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// WebSocketAgentHub manages WebSocket connections for agent status updates.
type WebSocketAgentHub struct {
	clients    map[string][]*AgentWSConnection
	broadcast  chan AgentStatusUpdate
	register   chan *AgentWSConnection
	unregister chan *AgentWSConnection
	mu         sync.RWMutex
}

// AgentWSConnection represents a WebSocket connection for agent status.
type AgentWSConnection struct {
	AgentID      string
	UserID       string
	Organization string
	Conn         *websocket.Conn
	Send         chan AgentStatusUpdate
	Hub          *WebSocketAgentHub
	mu           sync.Mutex
}

// NewWebSocketAgentHub creates a new WebSocket hub for agent updates.
func NewWebSocketAgentHub() *WebSocketAgentHub {
	hub := &WebSocketAgentHub{
		clients:    make(map[string][]*AgentWSConnection),
		broadcast:  make(chan AgentStatusUpdate, 256),
		register:   make(chan *AgentWSConnection),
		unregister: make(chan *AgentWSConnection),
	}
	go hub.run()
	return hub
}

// run processes hub events.
func (h *WebSocketAgentHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = []*AgentWSConnection{client}
			} else {
				h.clients[client.UserID] = append(h.clients[client.UserID], client)
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			clients := h.clients[client.UserID]
			for i, c := range clients {
				if c == client {
					h.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			if len(h.clients[client.UserID]) == 0 {
				delete(h.clients, client.UserID)
			}
			h.mu.Unlock()
			close(client.Send)

		case update := <-h.broadcast:
			h.mu.RLock()
			// Broadcast to all interested clients
			for _, clients := range h.clients {
				for _, client := range clients {
					select {
					case client.Send <- update:
					default:
						// Client channel full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToUser sends an update to all connections for a specific user.
func (h *WebSocketAgentHub) BroadcastToUser(userID string, update AgentStatusUpdate) {
	h.mu.RLock()
	clients := h.clients[userID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- update:
		default:
		}
	}
}

// BroadcastToOrganization sends an update to all connections in an organization.
func (h *WebSocketAgentHub) BroadcastToOrganization(orgID string, update AgentStatusUpdate) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for _, client := range clients {
			if client.Organization == orgID {
				select {
				case client.Send <- update:
				default:
				}
			}
		}
	}
}

// GetConnectionCount returns the number of active connections.
func (h *WebSocketAgentHub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// AgentWebSocketHandler handles WebSocket connections for real-time agent status.
func (h *Handler) AgentWebSocketHandler(hub *WebSocketAgentHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade to WebSocket
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In production, validate origin properly
				return true
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Failed to upgrade to WebSocket", http.StatusBadRequest)
			return
		}

		// Extract user info from context (set by auth middleware)
		userID := ""
		if userIDValue := r.Context().Value("user_id"); userIDValue != nil {
			if uid, ok := userIDValue.(string); ok {
				userID = uid
			}
		}

		orgID := ""
		if orgIDValue := r.Context().Value("org_id"); orgIDValue != nil {
			if oid, ok := orgIDValue.(string); ok {
				orgID = oid
			}
		}

		// Create connection
		client := &AgentWSConnection{
			Conn:         conn,
			Send:         make(chan AgentStatusUpdate, 256),
			Hub:          hub,
			UserID:       userID,
			Organization: orgID,
		}

		// Register client
		hub.register <- client

		// Start goroutines
		go client.writePump()
		go client.readPump()
	}
}

// readPump reads messages from the WebSocket connection.
func (c *AgentWSConnection) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *AgentWSConnection) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case update, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.mu.Lock()
			err := c.Conn.WriteJSON(update)
			c.mu.Unlock()

			if err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client.
func (c *AgentWSConnection) handleMessage(data []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	msgType, _ := msg["type"].(string)

	switch msgType {
	case "subscribe_agent":
		// Subscribe to updates for a specific agent
		if agentID, ok := msg["agent_id"].(string); ok {
			c.AgentID = agentID
		}
	case "subscribe_organization":
		// Already filtered by organization in hub broadcast
	case "ping":
		// Respond with pong
		c.Send <- AgentStatusUpdate{
			Timestamp: time.Now(),
		}
	}
}

// BroadcastAgentStatus broadcasts an agent status update.
func (h *Handler) BroadcastAgentStatus(hub *WebSocketAgentHub, ctx context.Context, agentID string) error {
	agent, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	// Get online printer count for this agent
	printers, err := h.printerRepo.FindByAgent(ctx, agentID)
	if err != nil {
		printers = []*repository.Printer{}
	}

	onlinePrinters := 0
	for _, p := range printers {
		if p.Status == "online" {
			onlinePrinters++
		}
	}

	update := AgentStatusUpdate{
		AgentID:        agent.ID,
		Name:           agent.Name,
		Status:         agent.Status,
		LastHeartbeat:  agent.LastHeartbeat,
		Version:        agent.Version,
		OS:             agent.OS,
		Hostname:       agent.Hostname,
		Architecture:   agent.Architecture,
		OnlinePrinters: onlinePrinters,
		Timestamp:      time.Now(),
	}

	// Broadcast to organization members
	if agent.OrganizationID != "" {
		hub.BroadcastToOrganization(agent.OrganizationID, update)
	}

	return nil
}

// BroadcastAllAgents broadcasts status for all agents.
func (h *Handler) BroadcastAllAgents(hub *WebSocketAgentHub, ctx context.Context) error {
	agents, _, err := h.agentRepo.List(ctx, 1000, 0)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		if err := h.BroadcastAgentStatus(hub, ctx, agent.ID); err != nil {
			// Log error but continue with other agents
			continue
		}
	}

	return nil
}

// AgentEventsHandler provides SSE endpoint for agent status changes.
func (h *Handler) AgentEventsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
	flusher.Flush()

	// Keep connection alive and send updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Get agent list to track changes
	lastStatuses := make(map[string]string)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send keepalive
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

			// Check for status changes
			agents, _, err := h.agentRepo.List(ctx, 1000, 0)
			if err != nil {
				continue
			}

			for _, agent := range agents {
				lastStatus := lastStatuses[agent.ID]
				if lastStatus != agent.Status {
					// Status changed, send event
					update := AgentStatusUpdate{
						AgentID:       agent.ID,
						Name:          agent.Name,
						Status:        agent.Status,
						LastHeartbeat: agent.LastHeartbeat,
						Timestamp:     time.Now(),
					}

					data, _ := json.Marshal(update)
					fmt.Fprintf(w, "event: agent_status_changed\ndata: %s\n\n", string(data))
					flusher.Flush()

					lastStatuses[agent.ID] = agent.Status
				}
			}
		}
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
