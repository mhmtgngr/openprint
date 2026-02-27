// Package websocket provides real-time WebSocket communication for notifications.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Connection defines the interface for a WebSocket connection.
type Connection interface {
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
	Close() error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(appData string) error)
	SetReadDeadline(t time.Time) error
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
}

// Message represents a notification message.
type Message struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Client represents a WebSocket client connection.
type Client struct {
	ID       string
	UserID   string
	OrgID    string
	Hub      *Hub
	Conn     Connection
	Send     chan *Message
	mu       sync.Mutex
}

// Config holds hub configuration.
type Config struct {
	PingInterval time.Duration
	PongTimeout  time.Duration
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Client lookups
	clientsByUserID map[string][]*Client
	clientsByOrgID  map[string][]*Client

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Direct message to a specific user
	sendToUser chan *UserMessage

	// Broadcast to organization
	broadcastToOrg chan *OrgMessage

	// Configuration
	cfg Config

	mu sync.RWMutex
}

// UserMessage represents a message to a specific user.
type UserMessage struct {
	UserID  string
	Message *Message
}

// OrgMessage represents a message to an organization.
type OrgMessage struct {
	OrgID   string
	Message *Message
}

// NewHub creates a new WebSocket hub.
func NewHub(cfg Config) *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		clientsByUserID: make(map[string][]*Client),
		clientsByOrgID:  make(map[string][]*Client),
		broadcast:       make(chan *Message, 256),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		sendToUser:      make(chan *UserMessage, 256),
		broadcastToOrg:  make(chan *OrgMessage, 256),
		cfg:            cfg,
	}
}

// Run starts the hub's message processing loop.
func (h *Hub) Run(ctx context.Context) {
	// Start ticker for ping/pong
	ticker := time.NewTicker(h.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case userMsg := <-h.sendToUser:
			h.sendToUserHandler(userMsg)

		case orgMsg := <-h.broadcastToOrg:
			h.broadcastToOrgHandler(orgMsg)

		case <-ticker.C:
			h.pingClients()
		}
	}
}

// Shutdown gracefully shuts down the hub.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all client connections
	for client := range h.clients {
		client.Conn.Close()
		close(client.Send)
	}

	// Clear all maps
	h.clients = make(map[*Client]bool)
	h.clientsByUserID = make(map[string][]*Client)
	h.clientsByOrgID = make(map[string][]*Client)
}

// registerClient adds a new client to the hub.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// Add to user index
	h.clientsByUserID[client.UserID] = append(h.clientsByUserID[client.UserID], client)

	// Add to org index
	if client.OrgID != "" {
		h.clientsByOrgID[client.OrgID] = append(h.clientsByOrgID[client.OrgID], client)
	}

	log.Printf("Client registered: %s (user: %s, org: %s)", client.ID, client.UserID, client.OrgID)
}

// unregisterClient removes a client from the hub.
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)

	// Remove from user index
	h.clientsByUserID[client.UserID] = removeClient(h.clientsByUserID[client.UserID], client)

	// Remove from org index
	if client.OrgID != "" {
		h.clientsByOrgID[client.OrgID] = removeClient(h.clientsByOrgID[client.OrgID], client)
	}

	close(client.Send)
	client.Conn.Close()

	log.Printf("Client unregistered: %s", client.ID)
}

// broadcastMessage sends a message to all connected clients.
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.Send <- message:
		default:
			// Client channel full, close connection
			h.unregister <- client
		}
	}
}

// sendToUserHandler sends a message to a specific user.
func (h *Hub) sendToUserHandler(userMsg *UserMessage) {
	h.mu.RLock()
	clients := h.clientsByUserID[userMsg.UserID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- userMsg.Message:
		default:
			h.unregister <- client
		}
	}
}

// broadcastToOrgHandler broadcasts a message to an organization.
func (h *Hub) broadcastToOrgHandler(orgMsg *OrgMessage) {
	h.mu.RLock()
	clients := h.clientsByOrgID[orgMsg.OrgID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- orgMsg.Message:
		default:
			h.unregister <- client
		}
	}
}

// SendToUser sends a message to all connections for a user.
func (h *Hub) SendToUser(userID string, msgType string, data map[string]interface{}) {
	h.sendToUser <- &UserMessage{
		UserID: userID,
		Message: &Message{
			Type:      msgType,
			Data:      data,
			Timestamp: time.Now(),
		},
	}
}

// BroadcastToOrg broadcasts a message to all connections in an organization.
func (h *Hub) BroadcastToOrg(orgID string, msgType string, data map[string]interface{}) {
	h.broadcastToOrg <- &OrgMessage{
		OrgID: orgID,
		Message: &Message{
			Type:      msgType,
			Data:      data,
			Timestamp: time.Now(),
		},
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msgType string, data map[string]interface{}) {
	h.broadcast <- &Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// GetConnectionCount returns the number of active connections.
func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetConnectionsByUser returns the number of connections for a user.
func (h *Hub) GetConnectionsByUser(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clientsByUserID[userID])
}

// GetConnectionsByOrg returns the number of connections for an organization.
func (h *Hub) GetConnectionsByOrg(orgID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clientsByOrgID[orgID])
}

// pingClients sends ping messages to all clients.
func (h *Hub) pingClients() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.Lock()
		err := client.Conn.WriteMessage(websocket.PingMessage, nil)
		client.mu.Unlock()

		if err != nil {
			h.unregister <- client
		}
	}
}

// removeClient removes a client from a slice.
func removeClient(clients []*Client, client *Client) []*Client {
	for i, c := range clients {
		if c == client {
			return append(clients[:i], clients[i+1:]...)
		}
	}
	return clients
}

// JobStatusUpdate creates a job status update message.
func JobStatusUpdate(jobID, status, message string) *Message {
	return &Message{
		Type: "job.status_update",
		Data: map[string]interface{}{
			"job_id":  jobID,
			"status":  status,
			"message": message,
		},
		Timestamp: time.Now(),
	}
}

// JobProgressUpdate creates a job progress update message.
func JobProgressUpdate(jobID string, progress int, pagesPrinted int) *Message {
	return &Message{
		Type: "job.progress_update",
		Data: map[string]interface{}{
			"job_id":        jobID,
			"progress":      progress,
			"pages_printed": pagesPrinted,
		},
		Timestamp: time.Now(),
	}
}

// PrinterStatusUpdate creates a printer status update message.
func PrinterStatusUpdate(printerID, status string) *Message {
	return &Message{
		Type: "printer.status_update",
		Data: map[string]interface{}{
			"printer_id": printerID,
			"status":     status,
		},
		Timestamp: time.Now(),
	}
}

// NewNotification creates a generic notification message.
func NewNotification(title, body string, data map[string]interface{}) *Message {
	msgData := make(map[string]interface{})
	msgData["title"] = title
	msgData["body"] = body
	for k, v := range data {
		msgData[k] = v
	}

	return &Message{
		Type: "notification",
		Data: msgData,
		Timestamp: time.Now(),
	}
}

// MarshalJSON implements custom JSON marshaling for Message.
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: m.Timestamp.Unix(),
		Alias:     (*Alias)(m),
	})
}
