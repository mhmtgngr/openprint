// Package websocket provides WebSocket connection handlers for the notification service.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// HandlerConfig holds handler configuration.
type HandlerConfig struct {
	Hub     *Hub
	DB      *pgxpool.Pool
	Metrics interface{} // Can be *prometheus.Metrics when available
}

// Handler handles WebSocket connections.
type Handler struct {
	hub *Hub
	db  *pgxpool.Pool
}

// NewHandler creates a new WebSocket handler.
func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		hub: cfg.Hub,
		db:  cfg.DB,
	}
}

// ServeWS handles WebSocket connection requests.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Extract user info from query params (in production, use JWT auth)
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")

	if userID == "" {
		// Try to get from JWT token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// In production, validate JWT and extract user ID
			userID = "user-" + uuid.New().String()
		} else {
			userID = "anon-" + uuid.New().String()
		}
	}

	// Create client
	client := &Client{
		ID:     uuid.New().String(),
		UserID: userID,
		OrgID:  orgID,
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan *Message, 256),
	}

	// Register client
	h.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// BroadcastHandler handles HTTP broadcast requests.
func (h *Handler) BroadcastHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type    string                 `json:"type"`
		Data    map[string]interface{} `json:"data"`
		UserID  string                 `json:"user_id,omitempty"`
		OrgID   string                 `json:"org_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Route message based on target
	if req.UserID != "" {
		h.hub.SendToUser(req.UserID, req.Type, req.Data)
	} else if req.OrgID != "" {
		h.hub.BroadcastToOrg(req.OrgID, req.Type, req.Data)
	} else {
		h.hub.Broadcast(req.Type, req.Data)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// ConnectionsHandler returns connection statistics.
func (h *Handler) ConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")

	stats := map[string]interface{}{
		"total_connections": h.hub.GetConnectionCount(),
	}

	if userID != "" {
		stats["user_connections"] = h.hub.GetConnectionsByUser(userID)
	}
	if orgID != "" {
		stats["org_connections"] = h.hub.GetConnectionsByOrg(orgID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.cfg.PongTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.cfg.PongTimeout))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *Client) writePump() {
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
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Serialize message
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
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

// handleMessage processes incoming messages from a client.
func (c *Client) handleMessage(data []byte) {
	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	switch msg.Type {
	case "ping":
		// Respond with pong
		c.Send <- &Message{
			Type:      "pong",
			Data:      map[string]interface{}{},
			Timestamp: time.Now(),
		}

	case "subscribe":
		// Handle subscription to specific channels
		if userID, ok := msg.Data["user_id"].(string); ok {
			c.UserID = userID
		}
		if orgID, ok := msg.Data["org_id"].(string); ok {
			c.OrgID = orgID
		}

	case "unsubscribe":
		// Handle unsubscribe
		c.OrgID = ""

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// SendJobStatusUpdate sends a job status update to a user.
func (h *Handler) SendJobStatusUpdate(ctx context.Context, userID, jobID, status, message string) {
	h.hub.SendToUser(userID, "job.status_update", map[string]interface{}{
		"job_id":  jobID,
		"status":  status,
		"message": message,
	})
}

// SendJobProgressUpdate sends a job progress update to a user.
func (h *Handler) SendJobProgressUpdate(ctx context.Context, userID, jobID string, progress, pagesPrinted int) {
	h.hub.SendToUser(userID, "job.progress_update", map[string]interface{}{
		"job_id":        jobID,
		"progress":      progress,
		"pages_printed": pagesPrinted,
	})
}

// BroadcastPrinterStatus broadcasts a printer status update to an organization.
func (h *Handler) BroadcastPrinterStatus(ctx context.Context, orgID, printerID, status string) {
	h.hub.BroadcastToOrg(orgID, "printer.status_update", map[string]interface{}{
		"printer_id": printerID,
		"status":     status,
	})
}

// NotifyUser sends a notification to a specific user.
func (h *Handler) NotifyUser(ctx context.Context, userID, title, body string) {
	h.hub.SendToUser(userID, "notification", map[string]interface{}{
		"title": title,
		"body":  body,
	})
}

// BroadcastSystemNotification broadcasts a system-wide notification.
func (h *Handler) BroadcastSystemNotification(ctx context.Context, title, body string) {
	h.hub.Broadcast("notification", map[string]interface{}{
		"title": title,
		"body":  body,
	})
}

// GetActiveUsers returns a list of users with active connections.
func (h *Handler) GetActiveUsers(ctx context.Context) []string {
	h.hub.mu.RLock()
	defer h.hub.mu.RUnlock()

	userSet := make(map[string]bool)
	for client := range h.hub.clients {
		userSet[client.UserID] = true
	}

	users := make([]string, 0, len(userSet))
	for userID := range userSet {
		users = append(users, userID)
	}

	return users
}

// GetConnectionStats returns detailed connection statistics.
func (h *Handler) GetConnectionStats(ctx context.Context) map[string]interface{} {
	h.hub.mu.RLock()
	defer h.hub.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(h.hub.clients),
		"unique_users":      len(h.hub.clientsByUserID),
		"unique_orgs":       len(h.hub.clientsByOrgID),
	}

	return stats
}
