// Package websocket provides tests for WebSocket hub.
package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockConn is a mock WebSocket connection for testing
type mockConn struct {
	id        string
	closed    bool
	writeData [][]byte
	closeChan chan struct{}
}

func newMockConn(id string) *mockConn {
	return &mockConn{
		id:        id,
		closeChan: make(chan struct{}),
	}
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.writeData = append(m.writeData, data)
	return nil
}

func (m *mockConn) WriteJSON(v interface{}) error {
	return nil
}

func (m *mockConn) ReadJSON(v interface{}) error {
	<-m.closeChan
	return &websocket.CloseError{Code: websocket.CloseNormalClosure}
}

func (m *mockConn) Close() error {
	m.closed = true
	close(m.closeChan)
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error           { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error          { return nil }
func (m *mockConn) SetPongHandler(h func(appData string) error) {}
func (m *mockConn) ReadMessage() (int, []byte, error) {
	<-m.closeChan
	return 0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
}

func TestNewHub(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map should be initialized")
	}
	if hub.clientsByUserID == nil {
		t.Error("clientsByUserID map should be initialized")
	}
	if hub.clientsByOrgID == nil {
		t.Error("clientsByOrgID map should be initialized")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel should be initialized")
	}
	if hub.register == nil {
		t.Error("register channel should be initialized")
	}
	if hub.unregister == nil {
		t.Error("unregister channel should be initialized")
	}
	if hub.sendToUser == nil {
		t.Error("sendToUser channel should be initialized")
	}
	if hub.broadcastToOrg == nil {
		t.Error("broadcastToOrg channel should be initialized")
	}
	if hub.cfg.PingInterval != 30*time.Second {
		t.Errorf("Expected PingInterval of 30s, got %v", hub.cfg.PingInterval)
	}
}

func TestHub_RegisterClient(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	mockConn := newMockConn("conn-1")
	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   mockConn,
		Send:   make(chan *Message, 256),
	}

	// Register client
	hub.registerClient(client)

	// Check client is registered
	if !hub.clients[client] {
		t.Error("Client should be registered")
	}

	// Check user index
	if len(hub.clientsByUserID[client.UserID]) != 1 {
		t.Error("Client should be in user index")
	}

	// Check org index
	if len(hub.clientsByOrgID[client.OrgID]) != 1 {
		t.Error("Client should be in org index")
	}

	// Check connection count
	if hub.GetConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", hub.GetConnectionCount())
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	mockConn := newMockConn("conn-1")
	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   mockConn,
		Send:   make(chan *Message, 256),
	}

	// Register first
	hub.registerClient(client)

	// Unregister
	hub.unregisterClient(client)

	// Check client is unregistered
	if hub.clients[client] {
		t.Error("Client should be unregistered")
	}

	// Check indexes are cleared
	if len(hub.clientsByUserID[client.UserID]) != 0 {
		t.Error("Client should be removed from user index")
	}

	if len(hub.clientsByOrgID[client.OrgID]) != 0 {
		t.Error("Client should be removed from org index")
	}

	// Check connection was closed
	if !mockConn.closed {
		t.Error("Connection should be closed")
	}

	// Check send channel was closed
	select {
	case _, ok := <-client.Send:
		if ok {
			t.Error("Send channel should be closed")
		}
	default:
	}
}

func TestHub_Broadcast(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Register multiple clients
	clients := []*Client{}
	for i := 0; i < 3; i++ {
		mockConn := newMockConn("conn-" + string(rune('1'+i)))
		client := &Client{
			ID:     "client-" + string(rune('1'+i)),
			UserID: "user-" + string(rune('1'+i)),
			OrgID:  "org-1",
			Hub:    hub,
			Conn:   mockConn,
			Send:   make(chan *Message, 256),
		}
		hub.registerClient(client)
		clients = append(clients, client)
	}

	// Broadcast message
	msg := &Message{
		Type:      "test",
		Data:      map[string]interface{}{"message": "hello"},
		Timestamp: time.Now(),
	}

	hub.broadcastMessage(msg)

	// Check all clients received the message
	for _, client := range clients {
		select {
		case received := <-client.Send:
			if received.Type != "test" {
				t.Errorf("Expected message type 'test', got '%s'", received.Type)
			}
		default:
			t.Error("Client should have received broadcast message")
		}
	}
}

func TestHub_SendToUser(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Register clients for different users
	user1Client1 := &Client{
		ID:     "client-1",
		UserID: "user-1",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   newMockConn("conn-1"),
		Send:   make(chan *Message, 256),
	}
	user1Client2 := &Client{
		ID:     "client-2",
		UserID: "user-1",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   newMockConn("conn-2"),
		Send:   make(chan *Message, 256),
	}
	user2Client := &Client{
		ID:     "client-3",
		UserID: "user-2",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   newMockConn("conn-3"),
		Send:   make(chan *Message, 256),
	}

	hub.registerClient(user1Client1)
	hub.registerClient(user1Client2)
	hub.registerClient(user2Client)

	// Send message to user-1
	userMsg := &UserMessage{
		UserID: "user-1",
		Message: &Message{
			Type:      "test",
			Data:      map[string]interface{}{"message": "hello"},
			Timestamp: time.Now(),
		},
	}

	hub.sendToUserHandler(userMsg)

	// Check user-1 clients received the message
	select {
	case <-user1Client1.Send:
		// Success
	default:
		t.Error("user1Client1 should have received message")
	}

	select {
	case <-user1Client2.Send:
		// Success
	default:
		t.Error("user1Client2 should have received message")
	}

	// user2Client should not have received the message
	select {
	case <-user2Client.Send:
		t.Error("user2Client should not have received message")
	default:
		// Success - no message for user2
	}
}

func TestHub_BroadcastToOrg(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Register clients for different orgs
	org1Client1 := &Client{
		ID:     "client-1",
		UserID: "user-1",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   newMockConn("conn-1"),
		Send:   make(chan *Message, 256),
	}
	org1Client2 := &Client{
		ID:     "client-2",
		UserID: "user-2",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   newMockConn("conn-2"),
		Send:   make(chan *Message, 256),
	}
	org2Client := &Client{
		ID:     "client-3",
		UserID: "user-3",
		OrgID:  "org-2",
		Hub:    hub,
		Conn:   newMockConn("conn-3"),
		Send:   make(chan *Message, 256),
	}

	hub.registerClient(org1Client1)
	hub.registerClient(org1Client2)
	hub.registerClient(org2Client)

	// Broadcast to org-1
	orgMsg := &OrgMessage{
		OrgID: "org-1",
		Message: &Message{
			Type:      "test",
			Data:      map[string]interface{}{"message": "hello org"},
			Timestamp: time.Now(),
		},
	}

	hub.broadcastToOrgHandler(orgMsg)

	// Check org-1 clients received the message
	select {
	case <-org1Client1.Send:
		// Success
	default:
		t.Error("org1Client1 should have received message")
	}

	select {
	case <-org1Client2.Send:
		// Success
	default:
		t.Error("org1Client2 should have received message")
	}

	// org2Client should not have received the message
	select {
	case <-org2Client.Send:
		t.Error("org2Client should not have received message")
	default:
		// Success - no message for org2
	}
}

func TestHub_GetConnectionCount(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Initially no connections
	if hub.GetConnectionCount() != 0 {
		t.Errorf("Expected 0 connections, got %d", hub.GetConnectionCount())
	}

	// Add clients
	for i := 0; i < 5; i++ {
		client := &Client{
			ID:     "client-" + string(rune('1'+i)),
			UserID: "user-" + string(rune('1'+i)),
			Hub:    hub,
			Conn:   newMockConn("conn-1"),
			Send:   make(chan *Message, 256),
		}
		hub.registerClient(client)
	}

	if hub.GetConnectionCount() != 5 {
		t.Errorf("Expected 5 connections, got %d", hub.GetConnectionCount())
	}
}

func TestHub_GetConnectionsByUser(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Register multiple clients for the same user
	for i := 0; i < 3; i++ {
		client := &Client{
			ID:     "client-" + string(rune('1'+i)),
			UserID: "user-1",
			Hub:    hub,
			Conn:   newMockConn("conn-1"),
			Send:   make(chan *Message, 256),
		}
		hub.registerClient(client)
	}

	// Register a client for different user
	client := &Client{
		ID:     "client-other",
		UserID: "user-2",
		Hub:    hub,
		Conn:   newMockConn("conn-2"),
		Send:   make(chan *Message, 256),
	}
	hub.registerClient(client)

	if hub.GetConnectionsByUser("user-1") != 3 {
		t.Errorf("Expected 3 connections for user-1, got %d", hub.GetConnectionsByUser("user-1"))
	}

	if hub.GetConnectionsByUser("user-2") != 1 {
		t.Errorf("Expected 1 connection for user-2, got %d", hub.GetConnectionsByUser("user-2"))
	}

	if hub.GetConnectionsByUser("user-3") != 0 {
		t.Errorf("Expected 0 connections for user-3, got %d", hub.GetConnectionsByUser("user-3"))
	}
}

func TestHub_GetConnectionsByOrg(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Register clients for org-1
	for i := 0; i < 4; i++ {
		client := &Client{
			ID:     "client-" + string(rune('1'+i)),
			UserID: "user-" + string(rune('1'+i)),
			OrgID:  "org-1",
			Hub:    hub,
			Conn:   newMockConn("conn-1"),
			Send:   make(chan *Message, 256),
		}
		hub.registerClient(client)
	}

	// Register clients for org-2
	for i := 0; i < 2; i++ {
		client := &Client{
			ID:     "client-" + string(rune('5'+i)),
			UserID: "user-" + string(rune('5'+i)),
			OrgID:  "org-2",
			Hub:    hub,
			Conn:   newMockConn("conn-2"),
			Send:   make(chan *Message, 256),
		}
		hub.registerClient(client)
	}

	if hub.GetConnectionsByOrg("org-1") != 4 {
		t.Errorf("Expected 4 connections for org-1, got %d", hub.GetConnectionsByOrg("org-1"))
	}

	if hub.GetConnectionsByOrg("org-2") != 2 {
		t.Errorf("Expected 2 connections for org-2, got %d", hub.GetConnectionsByOrg("org-2"))
	}
}

func TestHub_Shutdown(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Register some clients
	for i := 0; i < 3; i++ {
		mockConn := newMockConn("conn-" + string(rune('1'+i)))
		client := &Client{
			ID:     "client-" + string(rune('1'+i)),
			UserID: "user-" + string(rune('1'+i)),
			Hub:    hub,
			Conn:   mockConn,
			Send:   make(chan *Message, 256),
		}
		hub.registerClient(client)
	}

	// Shutdown
	hub.Shutdown()

	// Check all clients were removed
	if hub.GetConnectionCount() != 0 {
		t.Errorf("Expected 0 connections after shutdown, got %d", hub.GetConnectionCount())
	}

	// Check maps were cleared
	if len(hub.clientsByUserID) != 0 {
		t.Error("clientsByUserID should be cleared")
	}

	if len(hub.clientsByOrgID) != 0 {
		t.Error("clientsByOrgID should be cleared")
	}
}

func TestMessage_CreationFunctions(t *testing.T) {
	t.Run("JobStatusUpdate", func(t *testing.T) {
		msg := JobStatusUpdate("job-123", "completed", "Job finished successfully")

		if msg.Type != "job.status_update" {
			t.Errorf("Expected type 'job.status_update', got '%s'", msg.Type)
		}

		if msg.Data["job_id"] != "job-123" {
			t.Error("job_id not set correctly")
		}

		if msg.Data["status"] != "completed" {
			t.Error("status not set correctly")
		}

		if msg.Data["message"] != "Job finished successfully" {
			t.Error("message not set correctly")
		}
	})

	t.Run("JobProgressUpdate", func(t *testing.T) {
		msg := JobProgressUpdate("job-123", 50, 5)

		if msg.Type != "job.progress_update" {
			t.Errorf("Expected type 'job.progress_update', got '%s'", msg.Type)
		}

		if msg.Data["progress"] != 50 {
			t.Error("progress not set correctly")
		}

		if msg.Data["pages_printed"] != 5 {
			t.Error("pages_printed not set correctly")
		}
	})

	t.Run("PrinterStatusUpdate", func(t *testing.T) {
		msg := PrinterStatusUpdate("printer-123", "online")

		if msg.Type != "printer.status_update" {
			t.Errorf("Expected type 'printer.status_update', got '%s'", msg.Type)
		}

		if msg.Data["printer_id"] != "printer-123" {
			t.Error("printer_id not set correctly")
		}

		if msg.Data["status"] != "online" {
			t.Error("status not set correctly")
		}
	})

	t.Run("NewNotification", func(t *testing.T) {
		msg := NewNotification("Test Title", "Test Body", map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		})

		if msg.Type != "notification" {
			t.Errorf("Expected type 'notification', got '%s'", msg.Type)
		}

		if msg.Data["title"] != "Test Title" {
			t.Error("title not set correctly")
		}

		if msg.Data["body"] != "Test Body" {
			t.Error("body not set correctly")
		}

		if msg.Data["key1"] != "value1" {
			t.Error("custom key1 not set correctly")
		}

		if msg.Data["key2"] != 42 {
			t.Error("custom key2 not set correctly")
		}
	})
}

func TestHub_SendMethods(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start hub in background
	go hub.Run(ctx)

	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		OrgID:  "org-1",
		Hub:    hub,
		Conn:   newMockConn("conn-1"),
		Send:   make(chan *Message, 256),
	}
	hub.registerClient(client)

	// Give time for hub to start
	time.Sleep(10 * time.Millisecond)

	t.Run("SendToUser", func(t *testing.T) {
		hub.SendToUser("user-1", "test_type", map[string]interface{}{"key": "value"})

		select {
		case msg := <-client.Send:
			if msg.Type != "test_type" {
				t.Errorf("Expected type 'test_type', got '%s'", msg.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Should have received message")
		}
	})

	t.Run("BroadcastToOrg", func(t *testing.T) {
		// Clear the channel first
		select {
		case <-client.Send:
		default:
		}

		hub.BroadcastToOrg("org-1", "org_test", map[string]interface{}{"key": "value"})

		select {
		case msg := <-client.Send:
			if msg.Type != "org_test" {
				t.Errorf("Expected type 'org_test', got '%s'", msg.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Should have received message")
		}
	})

	t.Run("Broadcast", func(t *testing.T) {
		// Clear the channel first
		select {
		case <-client.Send:
		default:
		}

		hub.Broadcast("broadcast_test", map[string]interface{}{"key": "value"})

		select {
		case msg := <-client.Send:
			if msg.Type != "broadcast_test" {
				t.Errorf("Expected type 'broadcast_test', got '%s'", msg.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Should have received message")
		}
	})
}

func TestClient_NoOrg(t *testing.T) {
	cfg := Config{
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	// Client without org
	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		OrgID:  "",
		Hub:    hub,
		Conn:   newMockConn("conn-1"),
		Send:   make(chan *Message, 256),
	}

	hub.registerClient(client)

	// Should be in user index
	if len(hub.clientsByUserID["user-1"]) != 1 {
		t.Error("Client should be in user index")
	}

	// Should not be in org index
	if len(hub.clientsByOrgID) != 0 {
		t.Error("Client without org should not create org index entry")
	}
}

func TestHub_Run_ContextCancellation(t *testing.T) {
	cfg := Config{
		PingInterval: 10 * time.Millisecond,
		PongTimeout:  60 * time.Second,
	}

	hub := NewHub(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Start hub in a goroutine
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()

	// Cancel context to stop the hub
	cancel()

	// Wait for hub to stop
	select {
	case <-done:
		// Success - hub stopped
	case <-time.After(1 * time.Second):
		t.Error("Hub did not stop within timeout")
	}
}

func TestRemoveClient(t *testing.T) {
	clients := []*Client{
		{ID: "client-1"},
		{ID: "client-2"},
		{ID: "client-3"},
	}

	// Remove middle client
	result := removeClient(clients, clients[1])

	if len(result) != 2 {
		t.Errorf("Expected 2 clients after removal, got %d", len(result))
	}

	if result[0].ID != "client-1" {
		t.Error("First client should be client-1")
	}

	if result[1].ID != "client-3" {
		t.Error("Second client should be client-3")
	}
}

func TestRemoveClient_NotFound(t *testing.T) {
	clients := []*Client{
		{ID: "client-1"},
		{ID: "client-2"},
	}

	otherClient := &Client{ID: "client-3"}

	result := removeClient(clients, otherClient)

	if len(result) != 2 {
		t.Errorf("Expected 2 clients when removing non-existent client, got %d", len(result))
	}
}
