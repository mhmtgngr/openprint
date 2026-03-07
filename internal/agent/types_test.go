package agent

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPrinterConnectionType_Constants(t *testing.T) {
	tests := []struct {
		constant PrinterConnectionType
		expected string
	}{
		{ConnectionLocal, "local"},
		{ConnectionNetwork, "network"},
		{ConnectionShared, "shared"},
		{ConnectionWSD, "wsd"},
		{ConnectionLPD, "lpd"},
	}

	for _, tt := range tests {
		if string(tt.constant) != tt.expected {
			t.Errorf("PrinterConnectionType = %q, want %q", tt.constant, tt.expected)
		}
	}
}

func TestPrinterStatus_Constants(t *testing.T) {
	tests := []struct {
		constant PrinterStatus
		expected string
	}{
		{PrinterStatusIdle, "idle"},
		{PrinterStatusPrinting, "printing"},
		{PrinterStatusBusy, "busy"},
		{PrinterStatusOffline, "offline"},
		{PrinterStatusError, "error"},
		{PrinterStatusOutOfPaper, "out_of_paper"},
		{PrinterStatusLowToner, "low_toner"},
		{PrinterStatusDoorOpen, "door_open"},
	}

	for _, tt := range tests {
		if string(tt.constant) != tt.expected {
			t.Errorf("PrinterStatus = %q, want %q", tt.constant, tt.expected)
		}
	}
}

func TestAgentStatus_Constants(t *testing.T) {
	tests := []struct {
		constant AgentStatus
		expected string
	}{
		{AgentStatusOnline, "online"},
		{AgentStatusOffline, "offline"},
		{AgentStatusError, "error"},
		{AgentStatusMaintenance, "maintenance"},
		{AgentStatusInitializing, "initializing"},
	}

	for _, tt := range tests {
		if string(tt.constant) != tt.expected {
			t.Errorf("AgentStatus = %q, want %q", tt.constant, tt.expected)
		}
	}
}

func TestSessionState_Constants(t *testing.T) {
	tests := []struct {
		constant SessionState
		expected string
	}{
		{SessionStateActive, "active"},
		{SessionStateIdle, "idle"},
		{SessionStateDisconnected, "disconnected"},
	}

	for _, tt := range tests {
		if string(tt.constant) != tt.expected {
			t.Errorf("SessionState = %q, want %q", tt.constant, tt.expected)
		}
	}
}

func TestMessageType_Constants(t *testing.T) {
	if MessageTypeJobUpdate != "job.update" {
		t.Errorf("MessageTypeJobUpdate = %q, want %q", MessageTypeJobUpdate, "job.update")
	}
	if MessageTypeNewJob != "job.new" {
		t.Errorf("MessageTypeNewJob = %q, want %q", MessageTypeNewJob, "job.new")
	}
	if MessageTypeCommand != "agent.command" {
		t.Errorf("MessageTypeCommand = %q, want %q", MessageTypeCommand, "agent.command")
	}
	if MessageTypeConfigUpdate != "agent.config" {
		t.Errorf("MessageTypeConfigUpdate = %q, want %q", MessageTypeConfigUpdate, "agent.config")
	}
	if MessageTypePrinterUpdate != "printer.update" {
		t.Errorf("MessageTypePrinterUpdate = %q, want %q", MessageTypePrinterUpdate, "printer.update")
	}
	if MessageTypeHeartbeatAck != "heartbeat.ack" {
		t.Errorf("MessageTypeHeartbeatAck = %q, want %q", MessageTypeHeartbeatAck, "heartbeat.ack")
	}
}

func TestDiscoveredPrinter_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	printer := DiscoveredPrinter{
		PrinterID:      "printer-1",
		AgentID:        "agent-1",
		Name:           "HP LaserJet",
		DisplayName:    "Office Printer",
		Driver:         "HP Universal",
		Port:           "USB001",
		ConnectionType: ConnectionLocal,
		Status:         PrinterStatusIdle,
		IsDefault:      true,
		IsShared:       false,
		Location:       "Room 101",
		LastSeen:       now,
		Capabilities: &PrinterCapabilities{
			CanColor:  true,
			CanDuplex: true,
			MaxCopies: 999,
		},
	}

	data, err := json.Marshal(printer)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded DiscoveredPrinter
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.PrinterID != "printer-1" {
		t.Errorf("PrinterID = %q, want %q", decoded.PrinterID, "printer-1")
	}
	if decoded.ConnectionType != ConnectionLocal {
		t.Errorf("ConnectionType = %q, want %q", decoded.ConnectionType, ConnectionLocal)
	}
	if decoded.Status != PrinterStatusIdle {
		t.Errorf("Status = %q, want %q", decoded.Status, PrinterStatusIdle)
	}
	if decoded.Capabilities == nil || !decoded.Capabilities.CanColor {
		t.Error("Capabilities.CanColor should be true")
	}
}

func TestHeartbeatRequest_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	req := HeartbeatRequest{
		AgentID:       "agent-1",
		Status:        AgentStatusOnline,
		SessionState:  SessionStateActive,
		PrinterCount:  3,
		JobQueueDepth: 5,
		ActiveJobID:   "job-42",
		Timestamp:     now,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded HeartbeatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", decoded.AgentID, "agent-1")
	}
	if decoded.Status != AgentStatusOnline {
		t.Errorf("Status = %q, want %q", decoded.Status, AgentStatusOnline)
	}
	if decoded.PrinterCount != 3 {
		t.Errorf("PrinterCount = %d, want 3", decoded.PrinterCount)
	}
}

func TestJobStatusUpdate_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	update := JobStatusUpdate{
		JobID:        "job-1",
		AgentID:      "agent-1",
		Status:       "completed",
		Message:      "Printed successfully",
		PagesPrinted: 10,
		TotalPages:   10,
		Timestamp:    now,
	}

	data, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded JobStatusUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Status != "completed" {
		t.Errorf("Status = %q, want %q", decoded.Status, "completed")
	}
	if decoded.PagesPrinted != 10 {
		t.Errorf("PagesPrinted = %d, want 10", decoded.PagesPrinted)
	}
}

func TestAgentRegistrationRequest_JSON(t *testing.T) {
	req := AgentRegistrationRequest{
		Name:            "Test Agent",
		Version:         "1.0.0",
		OS:              "Windows 11",
		Architecture:    "amd64",
		Hostname:        "DESKTOP-123",
		Domain:          "CORP",
		OrganizationID:  "org-1",
		EnrollmentToken: "token-abc",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded AgentRegistrationRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "Test Agent" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Test Agent")
	}
	if decoded.OS != "Windows 11" {
		t.Errorf("OS = %q, want %q", decoded.OS, "Windows 11")
	}
}

func TestWebSocketMessage_JSON(t *testing.T) {
	msg := WebSocketMessage{
		Type:      MessageTypeJobUpdate,
		Data:      json.RawMessage(`{"job_id":"job-1","status":"completed"}`),
		Timestamp: time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded WebSocketMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Type != MessageTypeJobUpdate {
		t.Errorf("Type = %q, want %q", decoded.Type, MessageTypeJobUpdate)
	}
}

func TestPrinterCapabilities_JSON(t *testing.T) {
	caps := PrinterCapabilities{
		CanColor:            true,
		CanDuplex:           true,
		CanStaple:           false,
		CanPunch:            false,
		SupportedMediaTypes: []string{"plain", "glossy"},
		SupportedPaperSizes: []string{"a4", "letter"},
		MaxResolution:       1200,
		MinResolution:       300,
		SupportedBins:       []string{"tray1", "tray2"},
		MaxCopies:           999,
	}

	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded PrinterCapabilities
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !decoded.CanColor {
		t.Error("CanColor should be true")
	}
	if decoded.MaxResolution != 1200 {
		t.Errorf("MaxResolution = %d, want 1200", decoded.MaxResolution)
	}
	if len(decoded.SupportedPaperSizes) != 2 {
		t.Errorf("SupportedPaperSizes length = %d, want 2", len(decoded.SupportedPaperSizes))
	}
}
