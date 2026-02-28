// Package websocket provides WebSocket connection handlers for agents.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// AgentHub manages agent WebSocket connections.
type AgentHub struct {
	// Registered agent connections
	agents map[*AgentClient]bool

	// Agent connections by agent ID
	agentsByID map[string][]*AgentClient

	// Inbound messages from agents
	broadcast chan *AgentMessage

	// Register requests from agents
	register chan *AgentClient

	// Unregister requests from agents
	unregister chan *AgentClient

	// Direct message to a specific agent
	sendToAgent chan *AgentMessage

	// Broadcast to all agents
	broadcastToAll chan *AgentMessage

	// Configuration
	cfg AgentHubConfig

	mu sync.RWMutex
}

// AgentHubConfig holds agent hub configuration.
type AgentHubConfig struct {
	PingInterval time.Duration
	PongTimeout  time.Duration
	MessageBufferSize int
}

// AgentClient represents a WebSocket client connection for an agent.
type AgentClient struct {
	ID        string
	AgentID   string
	Hostname  string
	Hub       *AgentHub
	Conn      *websocket.Conn
	Send      chan *AgentMessage
	mu        sync.Mutex
	ConnectedAt time.Time
	LastActivity time.Time
}

// AgentMessage represents a message for/from an agent.
type AgentMessage struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewAgentHub creates a new agent WebSocket hub.
func NewAgentHub(cfg AgentHubConfig) *AgentHub {
	if cfg.PingInterval == 0 {
		cfg.PingInterval = 30 * time.Second
	}
	if cfg.PongTimeout == 0 {
		cfg.PongTimeout = 60 * time.Second
	}
	if cfg.MessageBufferSize == 0 {
		cfg.MessageBufferSize = 256
	}

	return &AgentHub{
		agents:         make(map[*AgentClient]bool),
		agentsByID:     make(map[string][]*AgentClient),
		broadcast:      make(chan *AgentMessage, cfg.MessageBufferSize),
		register:       make(chan *AgentClient),
		unregister:     make(chan *AgentClient),
		sendToAgent:    make(chan *AgentMessage, cfg.MessageBufferSize),
		broadcastToAll: make(chan *AgentMessage, cfg.MessageBufferSize),
		cfg:           cfg,
	}
}

// Run starts the agent hub's message processing loop.
func (h *AgentHub) Run(ctx context.Context) {
	ticker := time.NewTicker(h.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.registerAgent(client)

		case client := <-h.unregister:
			h.unregisterAgent(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case agentMsg := <-h.sendToAgent:
			h.sendToAgentHandler(agentMsg)

		case message := <-h.broadcastToAll:
			h.broadcastToAllHandler(message)

		case <-ticker.C:
			h.pingAgents()
		}
	}
}

// Shutdown gracefully shuts down the agent hub.
func (h *AgentHub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.agents {
		client.Conn.Close()
		close(client.Send)
	}

	h.agents = make(map[*AgentClient]bool)
	h.agentsByID = make(map[string][]*AgentClient)
}

// registerAgent adds a new agent to the hub.
func (h *AgentHub) registerAgent(client *AgentClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.agents[client] = true
	h.agentsByID[client.AgentID] = append(h.agentsByID[client.AgentID], client)

	log.Printf("[AgentHub] Agent registered: %s (agent_id: %s, hostname: %s)", client.ID, client.AgentID, client.Hostname)
}

// unregisterAgent removes an agent from the hub.
func (h *AgentHub) unregisterAgent(client *AgentClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.agents[client]; !ok {
		return
	}

	delete(h.agents, client)

	// Remove from agent ID index
	clients := h.agentsByID[client.AgentID]
	for i, c := range clients {
		if c == client {
			h.agentsByID[client.AgentID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(h.agentsByID[client.AgentID]) == 0 {
		delete(h.agentsByID, client.AgentID)
	}

	close(client.Send)
	client.Conn.Close()

	log.Printf("[AgentHub] Agent unregistered: %s (agent_id: %s)", client.ID, client.AgentID)
}

// broadcastMessage sends a message to all connected agents.
func (h *AgentHub) broadcastMessage(message *AgentMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.agents {
		select {
		case client.Send <- message:
		default:
			// Client channel full, close connection
			h.unregister <- client
		}
	}
}

// sendToAgentHandler sends a message to a specific agent.
func (h *AgentHub) sendToAgentHandler(agentMsg *AgentMessage) {
	agentID, ok := agentMsg.Data["agent_id"].(string)
	if !ok {
		return
	}

	h.mu.RLock()
	clients := h.agentsByID[agentID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- agentMsg:
		default:
			h.unregister <- client
		}
	}
}

// broadcastToAllHandler broadcasts a message to all agents.
func (h *AgentHub) broadcastToAllHandler(message *AgentMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.agents {
		select {
		case client.Send <- message:
		default:
			h.unregister <- client
		}
	}
}

// SendToAgent sends a message to all connections for an agent.
func (h *AgentHub) SendToAgent(agentID string, msgType string, data map[string]interface{}) {
	data["agent_id"] = agentID
	h.sendToAgent <- &AgentMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// BroadcastToAll broadcasts a message to all connected agents.
func (h *AgentHub) BroadcastToAll(msgType string, data map[string]interface{}) {
	h.broadcastToAll <- &AgentMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// GetConnectionCount returns the number of active agent connections.
func (h *AgentHub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.agents)
}

// GetConnectionsByAgent returns the number of connections for an agent.
func (h *AgentHub) GetConnectionsByAgent(agentID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.agentsByID[agentID])
}

// GetConnectedAgents returns a list of connected agent IDs.
func (h *AgentHub) GetConnectedAgents() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agentIDs := make([]string, 0, len(h.agentsByID))
	for agentID := range h.agentsByID {
		agentIDs = append(agentIDs, agentID)
	}
	return agentIDs
}

// pingAgents sends ping messages to all agents.
func (h *AgentHub) pingAgents() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.agents {
		client.mu.Lock()
		err := client.Conn.WriteMessage(websocket.PingMessage, nil)
		client.mu.Unlock()

		if err != nil {
			h.unregister <- client
		}
	}
}

// AgentWebSocketHandler handles WebSocket connections from agents.
type AgentWebSocketHandler struct {
	hub *AgentHub
}

// NewAgentWebSocketHandler creates a new agent WebSocket handler.
func NewAgentWebSocketHandler(hub *AgentHub) *AgentWebSocketHandler {
	return &AgentWebSocketHandler{hub: hub}
}

var agentUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
	HandshakeTimeout: 10 * time.Second,
}

// ServeWS handles WebSocket connection requests from agents.
func (h *AgentWebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := agentUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[AgentWS] Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Extract agent info from query params
	agentID := r.URL.Query().Get("agent_id")
	hostname := r.URL.Query().Get("hostname")

	// Try to get from Authorization header (JWT)
	if agentID == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Parse JWT to get agent_id
			// For now, generate a temp ID
			agentID = "agent-" + uuid.New().String()
		} else {
			agentID = "anon-" + uuid.New().String()
		}
	}

	// Create client
	client := &AgentClient{
		ID:          uuid.New().String(),
		AgentID:     agentID,
		Hostname:    hostname,
		Hub:         h.hub,
		Conn:        conn,
		Send:        make(chan *AgentMessage, h.hub.cfg.MessageBufferSize),
		ConnectedAt: time.Now(),
		LastActivity: time.Now(),
	}

	// Register agent
	h.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *AgentClient) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.cfg.PongTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.cfg.PongTimeout))
		c.LastActivity = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[AgentWS] WebSocket error: %v", err)
			}
			break
		}

		c.LastActivity = time.Now()
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *AgentClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("[AgentWS] Failed to marshal message: %v", err)
				continue
			}

			c.mu.Lock()
			err = c.Conn.WriteMessage(websocket.TextMessage, data)
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

// handleMessage processes incoming messages from an agent.
func (c *AgentClient) handleMessage(data []byte) {
	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("[AgentWS] Failed to unmarshal message: %v", err)
		return
	}

	switch msg.Type {
	case "ping":
		c.Send <- &AgentMessage{
			Type:      "pong",
			Data:      map[string]interface{}{},
			Timestamp: time.Now(),
		}

	case "job_update":
		// Forward job status update to notification service
		log.Printf("[AgentWS] Job update from agent %s: %+v", c.AgentID, msg.Data)

	case "heartbeat":
		// Handle agent heartbeat
		c.Send <- &AgentMessage{
			Type: "heartbeat_ack",
			Data: map[string]interface{}{
				"server_time": time.Now().Unix(),
			},
			Timestamp: time.Now(),
		}

	case "printer_status":
		// Handle printer status update
		log.Printf("[AgentWS] Printer status update from agent %s: %+v", c.AgentID, msg.Data)

	case "error":
		// Handle error report from agent
		log.Printf("[AgentWS] Error report from agent %s: %+v", c.AgentID, msg.Data)

	default:
		log.Printf("[AgentWS] Unknown message type from agent %s: %s", c.AgentID, msg.Type)
	}
}

// Message type constants for agent WebSocket communication
const (
	// AgentMessageTypeJobNew indicates a new job assignment
	AgentMessageTypeJobNew = "job.new"
	// AgentMessageTypeJobUpdate indicates a job status update
	AgentMessageTypeJobUpdate = "job.update"
	// AgentMessageTypeJobCancel indicates a job cancellation
	AgentMessageTypeJobCancel = "job.cancel"
	// AgentMessageTypeCommand indicates a server command
	AgentMessageTypeCommand = "agent.command"
	// AgentMessageTypeConfig indicates a configuration update
	AgentMessageTypeConfig = "agent.config"
	// AgentMessageTypeShutdown indicates a graceful shutdown request
	AgentMessageTypeShutdown = "agent.shutdown"
	// AgentMessageTypePrinterUpdate indicates a printer status update
	AgentMessageTypePrinterUpdate = "printer.update"
)

// NewJobMessage creates a new job assignment message for an agent.
func NewJobMessage(jobID, printerID, documentURL string, options map[string]interface{}) *AgentMessage {
	data := map[string]interface{}{
		"job_id":       jobID,
		"printer_id":   printerID,
		"document_url": documentURL,
	}
	for k, v := range options {
		data[k] = v
	}

	return &AgentMessage{
		Type:      AgentMessageTypeJobNew,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewCancelJobMessage creates a job cancellation message for an agent.
func NewCancelJobMessage(jobID, reason string) *AgentMessage {
	return &AgentMessage{
		Type: AgentMessageTypeJobCancel,
		Data: map[string]interface{}{
			"job_id": jobID,
			"reason": reason,
		},
		Timestamp: time.Now(),
	}
}

// NewCommandMessage creates a command message for an agent.
func NewCommandMessage(commandID, commandType string, payload map[string]interface{}) *AgentMessage {
	data := map[string]interface{}{
		"command_id": commandID,
		"type":       commandType,
	}
	if payload != nil {
		data["payload"] = payload
	}

	return &AgentMessage{
		Type:      AgentMessageTypeCommand,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewConfigMessage creates a configuration update message for an agent.
func NewConfigMessage(config map[string]interface{}) *AgentMessage {
	return &AgentMessage{
		Type:      AgentMessageTypeConfig,
		Data:      config,
		Timestamp: time.Now(),
	}
}

// NewShutdownMessage creates a shutdown message for an agent.
func NewShutdownMessage(reason string, timeout time.Duration) *AgentMessage {
	return &AgentMessage{
		Type: AgentMessageTypeShutdown,
		Data: map[string]interface{}{
			"reason":  reason,
			"timeout": timeout.Seconds(),
		},
		Timestamp: time.Now(),
	}
}

// MarshalJSON implements custom JSON marshaling for AgentMessage.
func (m *AgentMessage) MarshalJSON() ([]byte, error) {
	type Alias AgentMessage
	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: m.Timestamp.Unix(),
		Alias:     (*Alias)(m),
	})
}
