// Package handler provides HTTP handlers for the notification service with tenant support.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	multitenant "github.com/openprint/openprint/internal/multi-tenant"
)

// NotificationWithTenant represents a notification with tenant information.
type NotificationWithTenant struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	UserID         string                 `json:"user_id"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Read           bool                   `json:"read"`
	CreatedAt      time.Time              `json:"created_at"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
}

// CreateNotificationRequest represents a notification creation request.
type CreateNotificationRequest struct {
	UserID  string                 `json:"user_id"`
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
	TTL     int                    `json:"ttl,omitempty"` // Time to live in seconds
}

// SendNotification handles sending a notification to a tenant's users.
func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
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

	var req CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.UserID == "" {
		// If no specific user, broadcast to all tenant users
		req.UserID = "*"
	}
	if req.Type == "" {
		req.Type = "info"
	}
	if req.Title == "" {
		respondError(w, apperrors.New("title is required", http.StatusBadRequest))
		return
	}

	now := time.Now().UTC()
	notification := &Notification{
		ID:             generateID(),
		TenantID:       tenantID,
		OrganizationID: tenantID,
		UserID:         req.UserID,
		Type:           req.Type,
		Title:          req.Title,
		Message:        req.Message,
		Read:           false,
		CreatedAt:      now,
	}

	// Set expiration if TTL provided
	if req.TTL > 0 {
		expiresAt := now.Add(time.Duration(req.TTL) * time.Second)
		notification.ExpiresAt = &expiresAt
	}

	// Marshal data
	if req.Data != nil {
		if bytes, err := json.Marshal(req.Data); err == nil {
			notification.Data = string(bytes)
		}
	}

	// Send notification via WebSocket to connected clients
	if h.hub != nil {
		h.hub.BroadcastToTenant(tenantID, notification)
	}

	// Store notification for offline users
	if h.notificationStore != nil {
		if err := h.notificationStore.Create(ctx, notification); err != nil {
			// Log but don't fail - notification was sent via WebSocket
			fmt.Printf("warning: failed to store notification: %v", err)
		}
	}

	// Log the notification
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogCreate(ctx, userID, userEmail, "notification", notification.ID, notification.Title,
			map[string]interface{}{"tenant_id": tenantID, "recipient": req.UserID})
	}

	respondJSON(w, http.StatusCreated, notificationToTenantResponse(notification))
}

// ListNotifications handles listing notifications with tenant scoping.
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
		return
	}

	// Get current user ID
	userID := middleware.GetUserID(r)

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	includeRead := r.URL.Query().Get("include_read") == "true"

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Get notifications for user in tenant
	notifications, total, err := h.listNotificationsForUser(ctx, tenantID, userID, limit, offset, includeRead)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list notifications", http.StatusInternalServerError))
		return
	}

	response := make([]*NotificationWithTenant, len(notifications))
	for i, n := range notifications {
		response[i] = notificationToTenantResponse(n)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": response,
		"total":          total,
		"limit":          limit,
		"offset":         offset,
	})
}

// MarkAsRead handles marking a notification as read.
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract notification ID from URL
	notifID := extractIDFromPath(r.URL.Path, "notifications")
	if notifID == "" {
		respondError(w, apperrors.New("notification ID required", http.StatusBadRequest))
		return
	}

	// Get tenant ID and user ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
		return
	}
	userID := middleware.GetUserID(r)

	// Verify notification belongs to user's tenant
	notification, err := h.getNotification(ctx, notifID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get notification", http.StatusInternalServerError))
		return
	}

	if notification.TenantID != tenantID {
		respondError(w, apperrors.New("forbidden: cannot access this notification", http.StatusForbidden))
		return
	}

	// Mark as read
	if err := h.markNotificationAsRead(ctx, notifID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to mark notification as read", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":   notifID,
		"read": true,
	})
}

// MarkAllAsRead handles marking all notifications as read for the current user.
func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID and user ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
		return
	}
	userID := middleware.GetUserID(r)

	// Mark all as read
	count, err := h.markAllNotificationsAsRead(ctx, tenantID, userID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to mark notifications as read", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}

// WebSocketHandler handles WebSocket connections with tenant context.
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from context
	tenantID, err := multitenant.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "tenant context required", http.StatusForbidden)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Configure appropriately for production
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("failed to upgrade to websocket: %v", err)
		return
	}

	// Register connection with hub
	client := &Client{
		ID:       generateID(),
		TenantID: tenantID,
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	if h.hub != nil {
		h.hub.Register(client)
		go client.WritePump()
		client.ReadPump(h.hub)
	}
}

// Helper types and functions

// Notification represents a notification in the system.
type Notification struct {
	ID             string
	TenantID       string
	OrganizationID string
	UserID         string
	Type           string
	Title          string
	Message        string
	Data           string
	Read           bool
	CreatedAt      time.Time
	ExpiresAt      *time.Time
}

// Client represents a WebSocket client connection.
type Client struct {
	ID       string
	TenantID string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients
	clients map[*Client]bool
	// Inbound messages from the clients
	broadcast chan []byte
	// Register requests from the clients
	register chan *Client
	// Unregister requests from clients
	unregister chan *Client
	// Tenant-indexed clients for efficient broadcasting
	tenantClients map[string]map[*Client]bool
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		clients:        make(map[*Client]bool),
		tenantClients:  make(map[string]map[*Client]bool),
	}
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			if h.tenantClients[client.TenantID] == nil {
				h.tenantClients[client.TenantID] = make(map[*Client]bool)
			}
			h.tenantClients[client.TenantID][client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if clients, ok := h.tenantClients[client.TenantID]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.tenantClients, client.TenantID)
					}
				}
				close(client.Send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					delete(h.clients, client)
				}
			}
		}
	}
}

// BroadcastToTenant sends a notification to all clients in a tenant.
func (h *Hub) BroadcastToTenant(tenantID string, notification *Notification) {
	message, _ := json.Marshal(notification)
	for client := range h.tenantClients[tenantID] {
		// Send to user-specific or broadcast
		if notification.UserID == "*" || notification.UserID == client.UserID {
			select {
			case client.Send <- message:
			default:
				// Client channel full, skip
			}
		}
	}
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
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
			w, err := c.Conn.NextWriter(websocket.TextMessage, message)
			if err != nil {
				return
			}
			w.Close()
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ErrNotFound is returned when a record is not found.
var ErrNotFound = errors.New("not found")

// Helper functions for the handler

func (h *Handler) listNotificationsForUser(ctx context.Context, tenantID, userID string, limit, offset int, includeRead bool) ([]*Notification, int, error) {
	// In production, this would query the database
	// For now, return empty list
	return []*Notification{}, 0, nil
}

func (h *Handler) getNotification(ctx context.Context, id string) (*Notification, error) {
	// In production, this would query the database
	return nil, ErrNotFound
}

func (h *Handler) markNotificationAsRead(ctx context.Context, id string) error {
	// In production, this would update the database
	return nil
}

func (h *Handler) markAllNotificationsAsRead(ctx context.Context, tenantID, userID string) (int, error) {
	// In production, this would update the database
	return 0, nil
}

func generateID() string {
	return uuid.New().String()
}

func notificationToTenantResponse(n *Notification) *NotificationWithTenant {
	var data map[string]interface{}
	if n.Data != "" {
		json.Unmarshal([]byte(n.Data), &data)
	}

	return &NotificationWithTenant{
		ID:             n.ID,
		TenantID:       n.TenantID,
		OrganizationID: n.OrganizationID,
		UserID:         n.UserID,
		Type:           n.Type,
		Title:          n.Title,
		Message:        n.Message,
		Data:           data,
		Read:           n.Read,
		CreatedAt:      n.CreatedAt,
		ExpiresAt:      n.ExpiresAt,
	}
}

func extractIDFromPath(path, resource string) string {
	parts := splitPath(path)
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func splitPath(path string) []string {
	path = path[1:]
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

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
