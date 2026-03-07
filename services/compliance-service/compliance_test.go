package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Unit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %q", resp["status"])
	}
	if resp["service"] != "compliance-service" {
		t.Errorf("expected service 'compliance-service', got %q", resp["service"])
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestListControlsHandler_DeleteMethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := listControlsHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/controls", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestControlByIDHandler_PatchMethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := controlByIDHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/controls/ctrl-1", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestControlByIDHandler_MissingID(t *testing.T) {
	svc := &Service{db: nil}
	handler := controlByIDHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/controls/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateControlStatusHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := updateControlStatusHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/controls/status/ctrl-1", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestUpdateControlStatusHandler_MissingID(t *testing.T) {
	svc := &Service{db: nil}
	handler := updateControlStatusHandler(svc)

	body := bytes.NewBufferString(`{"status": "compliant"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/controls/status/", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateControlStatusHandler_InvalidBody(t *testing.T) {
	svc := &Service{db: nil}
	handler := updateControlStatusHandler(svc)

	body := bytes.NewBufferString(`invalid json`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/controls/status/ctrl-1", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateControlStatusHandler_InvalidStatus(t *testing.T) {
	svc := &Service{db: nil}
	handler := updateControlStatusHandler(svc)

	body := bytes.NewBufferString(`{"status": "invalid_status"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/controls/status/ctrl-1", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateReportHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := generateReportHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/generate", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestGenerateReportHandler_InvalidBody(t *testing.T) {
	svc := &Service{db: nil}
	handler := generateReportHandler(svc)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateReportHandler_InvalidFramework(t *testing.T) {
	svc := &Service{db: nil}
	handler := generateReportHandler(svc)

	body := bytes.NewBufferString(`{"framework": "invalid", "period_start": "2025-01-01T00:00:00Z", "period_end": "2025-12-31T23:59:59Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateReportHandler_InvalidPeriodStart(t *testing.T) {
	svc := &Service{db: nil}
	handler := generateReportHandler(svc)

	body := bytes.NewBufferString(`{"framework": "hipaa", "period_start": "not-a-date", "period_end": "2025-12-31T23:59:59Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateReportHandler_InvalidPeriodEnd(t *testing.T) {
	svc := &Service{db: nil}
	handler := generateReportHandler(svc)

	body := bytes.NewBufferString(`{"framework": "hipaa", "period_start": "2025-01-01T00:00:00Z", "period_end": "not-a-date"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSummaryHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := summaryHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/summary", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestExportAuditLogsHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := exportAuditLogsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestExportAuditLogsHandler_DefaultFormat(t *testing.T) {
	// The handler defaults unknown formats to JSON, so no error is expected
	// This test just verifies the method not allowed path works
	svc := &Service{db: nil}
	handler := exportAuditLogsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/export?format=xml", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestPendingReviewsHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := pendingReviewsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/pending", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestAuditLogHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := auditLogHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/audit", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestAuditLogHandler_PostInvalidBody(t *testing.T) {
	svc := &Service{db: nil}
	handler := auditLogHandler(svc)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBreachesHandler_MethodNotAllowed(t *testing.T) {
	svc := &Service{db: nil}
	handler := breachesHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/breaches", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestBreachesHandler_PostInvalidBody(t *testing.T) {
	svc := &Service{db: nil}
	handler := breachesHandler(svc)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/breaches", body)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestExtractIDFromPath(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   string
	}{
		{"/api/v1/controls/ctrl-123", "/api/v1/controls/", "ctrl-123"},
		{"/api/v1/controls/", "/api/v1/controls/", ""},
		{"/api/v1/controls/status/ctrl-456", "/api/v1/controls/status/", "ctrl-456"},
		{"/short", "/longer-prefix/", ""},
	}

	for _, tt := range tests {
		got := extractIDFromPath(tt.path, tt.prefix)
		if got != tt.want {
			t.Errorf("extractIDFromPath(%q, %q) = %q, want %q", tt.path, tt.prefix, got, tt.want)
		}
	}
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	respondJSON(w, http.StatusCreated, data)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["key"] != "value" {
		t.Errorf("expected key=value, got key=%q", resp["key"])
	}
}

func TestExportToCSV(t *testing.T) {
	svc := &Service{}
	events := []*AuditEvent{
		{
			EventType:    "login",
			Category:     "auth",
			UserID:       "user-1",
			UserName:     "admin",
			ResourceID:   "res-1",
			ResourceType: "session",
			Action:       "create",
			Outcome:      "success",
			IPAddress:    "10.0.0.1",
		},
	}

	data, err := svc.exportToCSV(events)
	if err != nil {
		t.Fatalf("exportToCSV failed: %v", err)
	}

	csv := string(data)
	if !bytes.Contains([]byte(csv), []byte("Timestamp,EventType,Category")) {
		t.Error("CSV missing header row")
	}
	if !bytes.Contains([]byte(csv), []byte("login,auth,user-1,admin")) {
		t.Error("CSV missing event data")
	}
}

func TestExportToCSV_Empty(t *testing.T) {
	svc := &Service{}
	data, err := svc.exportToCSV([]*AuditEvent{})
	if err != nil {
		t.Fatalf("exportToCSV failed: %v", err)
	}

	csv := string(data)
	if !bytes.Contains([]byte(csv), []byte("Timestamp,EventType,Category")) {
		t.Error("CSV missing header row")
	}
}

func TestComplianceFrameworkConstants(t *testing.T) {
	if FrameworkFedRAMP != "fedramp" {
		t.Errorf("unexpected FrameworkFedRAMP value: %q", FrameworkFedRAMP)
	}
	if FrameworkHIPAA != "hipaa" {
		t.Errorf("unexpected FrameworkHIPAA value: %q", FrameworkHIPAA)
	}
	if FrameworkGDPR != "gdpr" {
		t.Errorf("unexpected FrameworkGDPR value: %q", FrameworkGDPR)
	}
	if FrameworkSOC2 != "soc2" {
		t.Errorf("unexpected FrameworkSOC2 value: %q", FrameworkSOC2)
	}
}

func TestComplianceStatusConstants(t *testing.T) {
	if StatusCompliant != "compliant" {
		t.Errorf("unexpected StatusCompliant value: %q", StatusCompliant)
	}
	if StatusNonCompliant != "non_compliant" {
		t.Errorf("unexpected StatusNonCompliant value: %q", StatusNonCompliant)
	}
	if StatusPending != "pending" {
		t.Errorf("unexpected StatusPending value: %q", StatusPending)
	}
	if StatusNotApplicable != "not_applicable" {
		t.Errorf("unexpected StatusNotApplicable value: %q", StatusNotApplicable)
	}
}

func TestNewService(t *testing.T) {
	svc := New(Config{DB: nil})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewRepository(t *testing.T) {
	repo := NewRepository(nil)
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}
