// Package main provides comprehensive tests for the compliance service HTTP handlers.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openprint/openprint/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupHandlerTest creates a test server with all routes registered.
func setupHandlerTest(t *testing.T) (*testutil.TestDB, *httptest.Server) {
	t.Helper()

	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available - run with test tag")
	}

	// Clean up before each test
	ctx := context.Background()
	if err := testutil.TruncateAllTables(ctx, testDB.Pool); err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}

	svc := New(Config{DB: testDB.Pool})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/controls", listControlsHandler(svc))
	mux.HandleFunc("/api/v1/controls/", controlByIDHandler(svc))
	mux.HandleFunc("/api/v1/controls/status/", updateControlStatusHandler(svc))
	mux.HandleFunc("/api/v1/audit", auditLogHandler(svc))
	mux.HandleFunc("/api/v1/audit/export", exportAuditLogsHandler(svc))
	mux.HandleFunc("/api/v1/breaches", breachesHandler(svc))
	mux.HandleFunc("/api/v1/reviews/pending", pendingReviewsHandler(svc))
	mux.HandleFunc("/api/v1/reports/summary", summaryHandler(svc))
	mux.HandleFunc("/api/v1/reports/generate", generateReportHandler(svc))

	server := httptest.NewServer(mux)

	return testDB, server
}

// TestHandlers_HealthEndpoint tests the health endpoint.
func TestHandlers_HealthEndpoint(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "healthy", result["status"])
	assert.Equal(t, "compliance-service", result["service"])
}

// TestHandlers_HealthEndpoint_MethodNotAllowed tests that health endpoint only accepts GET.
func TestHandlers_HealthEndpoint_MethodNotAllowed(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	// Try POST
	resp, err := http.Post(server.URL+"/health", "application/json", bytes.NewBuffer([]byte("{}")))
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Try PUT
	req, _ := http.NewRequest("PUT", server.URL+"/health", bytes.NewBuffer([]byte("{}")))
	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// TestHandlers_CreateControl tests creating a new compliance control.
func TestHandlers_CreateControl(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	control := map[string]interface{}{
		"framework":         "fedramp",
		"family":            "Access Control",
		"title":             "AC-001: Access Control Policy",
		"description":       "Test access control policy",
		"implementation":    "Implemented via LDAP",
		"status":            "pending",
		"next_review":       time.Now().AddDate(0, 0, 30).Format(time.RFC3339),
		"responsible_team":  "Security",
		"risk_level":        "medium",
	}

	body, _ := json.Marshal(control)
	resp, err := http.Post(server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "fedramp", result["framework"])
	assert.Equal(t, "Access Control", result["family"])
	assert.Equal(t, "AC-001: Access Control Policy", result["title"])
}

// TestHandlers_CreateControl_InvalidFramework tests creating a control with invalid framework.
func TestHandlers_CreateControl_InvalidFramework(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	control := map[string]interface{}{
		"framework":   "invalid_framework",
		"family":      "Access Control",
		"title":       "Test Control",
		"description": "Test description",
		"status":      "pending",
		"risk_level":  "medium",
	}

	body, _ := json.Marshal(control)
	resp, err := http.Post(server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "invalid framework", result["error"])
}

// TestHandlers_CreateControl_AllFrameworks tests creating controls for all valid frameworks.
func TestHandlers_CreateControl_AllFrameworks(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	frameworks := []string{"fedramp", "hipaa", "gdpr", "soc2"}

	for _, fw := range frameworks {
		control := map[string]interface{}{
			"framework":        fw,
			"family":           fmt.Sprintf("%s Family", strings.ToUpper(fw)),
			"title":            fmt.Sprintf("%s Test Control", strings.ToUpper(fw)),
			"description":      fmt.Sprintf("Test description for %s", fw),
			"implementation":   "Test implementation",
			"status":           "pending",
			"responsible_team": "Security",
			"risk_level":       "medium",
		}

		body, _ := json.Marshal(control)
		resp, err := http.Post(server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Framework %s should be valid", fw)
	}
}

// TestHandlers_ListControls tests listing controls.
func TestHandlers_ListControls(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test controls
	_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	_, err = testutil.CreateTestComplianceControl(ctx, testDB.Pool, "hipaa")
	require.NoError(t, err)

	// List all controls
	resp, err := http.Get(server.URL + "/api/v1/controls")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, int(result["total"].(float64)), 2)
	controls, ok := result["controls"].([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, controls)
}

// TestHandlers_ListControls_WithFrameworkFilter tests listing controls with framework filter.
func TestHandlers_ListControls_WithFrameworkFilter(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test controls for different frameworks
	for i := 0; i < 3; i++ {
		_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "hipaa")
		require.NoError(t, err)
	}

	// List only fedramp controls
	resp, err := http.Get(server.URL + "/api/v1/controls?framework=fedramp")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(3), result["total"])
	controls, ok := result["controls"].([]interface{})
	require.True(t, ok)
	assert.Len(t, controls, 3)
}

// TestHandlers_ListControls_WithPagination tests listing controls with pagination.
func TestHandlers_ListControls_WithPagination(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create 5 test controls
	for i := 0; i < 5; i++ {
		_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
		require.NoError(t, err)
	}

	// First page with limit 2
	resp, err := http.Get(server.URL + "/api/v1/controls?page=1&limit=2")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["page"])
	assert.Equal(t, float64(2), result["limit"])
	assert.Equal(t, float64(5), result["total"])
	controls, ok := result["controls"].([]interface{})
	require.True(t, ok)
	assert.Len(t, controls, 2)

	// Second page
	resp2, err := http.Get(server.URL + "/api/v1/controls?page=2&limit=2")
	require.NoError(t, err)
	defer resp2.Body.Close()

	var result2 map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&result2)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result2["page"])
	controls2, ok := result2["controls"].([]interface{})
	require.True(t, ok)
	assert.Len(t, controls2, 2)
}

// TestHandlers_GetControlByID tests getting a control by ID.
func TestHandlers_GetControlByID(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create a test control
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Get the control
	resp, err := http.Get(server.URL + "/api/v1/controls/" + controlID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, controlID, result["id"])
	assert.Equal(t, "fedramp", result["framework"])
}

// TestHandlers_GetControlByID_NotFound tests getting a non-existent control.
func TestHandlers_GetControlByID_NotFound(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	// Try to get a non-existent control
	resp, err := http.Get(server.URL + "/api/v1/controls/" + uuid.New().String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestHandlers_UpdateControlStatus tests updating control status.
func TestHandlers_UpdateControlStatus(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create a test control
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Update the control status
	updateReq := map[string]interface{}{
		"status":        "compliant",
		"last_assessed": time.Now().Format(time.RFC3339),
		"next_review":   time.Now().AddDate(0, 0, 60).Format(time.RFC3339),
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/controls/status/"+controlID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, controlID, result["control_id"])
	assert.Equal(t, "compliant", result["status"])

	// Verify the update was persisted
	resp2, err2 := http.Get(server.URL + "/api/v1/controls/" + controlID)
	require.NoError(t, err2)
	defer resp2.Body.Close()

	var control map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&control)
	assert.Equal(t, "compliant", control["status"])
}

// TestHandlers_UpdateControlStatus_InvalidStatus tests updating with invalid status.
func TestHandlers_UpdateControlStatus_InvalidStatus(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create a test control
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Try to update with invalid status
	updateReq := map[string]interface{}{
		"status": "invalid_status",
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/controls/status/"+controlID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid status", result["error"])
}

// TestHandlers_UpdateControlStatus_DefaultDates tests that default dates are applied when not provided.
func TestHandlers_UpdateControlStatus_DefaultDates(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create a test control
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Update without providing dates
	updateReq := map[string]interface{}{
		"status": "non_compliant",
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/controls/status/"+controlID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the update worked with defaults
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["updated_at"])
}

// TestHandlers_UpdateControlStatus_NotFound tests updating a non-existent control.
func TestHandlers_UpdateControlStatus_NotFound(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	updateReq := map[string]interface{}{
		"status":        "compliant",
		"last_assessed": time.Now().Format(time.RFC3339),
		"next_review":   time.Now().AddDate(0, 0, 60).Format(time.RFC3339),
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/controls/status/"+uuid.New().String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestHandlers_CreateAuditEvent tests creating an audit event.
func TestHandlers_CreateAuditEvent(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test user
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	// Create audit event
	event := map[string]interface{}{
		"user_id":       userID,
		"user_name":     "Test User",
		"resource_id":   uuid.New().String(),
		"resource_type": "test_resource",
		"action":        "test_action",
		"outcome":       "success",
		"ip_address":    "127.0.0.1",
		"user_agent":    "test-agent",
		"event_type":    "test_event",
		"category":      "test_category",
		"metadata": map[string]string{
			"key": "value",
		},
	}

	body, _ := json.Marshal(event)
	resp, err := http.Post(server.URL+"/api/v1/audit", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
}

// TestHandlers_QueryAuditEvents tests querying audit events.
func TestHandlers_QueryAuditEvents(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test user
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	// Create audit events
	for i := 0; i < 3; i++ {
		_, err := testutil.CreateTestAuditEvent(ctx, testDB.Pool, userID)
		require.NoError(t, err)
	}

	// Query audit events
	resp, err := http.Get(server.URL + "/api/v1/audit?limit=10&offset=0")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, int(result["total"].(float64)), 3)
	events, ok := result["events"].([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, events)
}

// TestHandlers_QueryAuditEvents_WithFilters tests querying with user filter.
func TestHandlers_QueryAuditEvents_WithFilters(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test users and events
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID1, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	userID2, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	// Create events for user1
	for i := 0; i < 3; i++ {
		_, err := testutil.CreateTestAuditEvent(ctx, testDB.Pool, userID1)
		require.NoError(t, err)
	}

	// Create events for user2
	for i := 0; i < 2; i++ {
		_, err := testutil.CreateTestAuditEvent(ctx, testDB.Pool, userID2)
		require.NoError(t, err)
	}

	// Query for user1 only
	resp, err := http.Get(server.URL + "/api/v1/audit?user_id=" + userID1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	total := int(result["total"].(float64))
	assert.GreaterOrEqual(t, total, 3)

	events, ok := result["events"].([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, events)

	// Verify all events are for user1
	for _, e := range events {
		event := e.(map[string]interface{})
		assert.Equal(t, userID1, event["user_id"])
	}
}

// TestHandlers_ExportAuditLogs_JSON tests exporting audit logs as JSON.
func TestHandlers_ExportAuditLogs_JSON(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test user and event
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	_, err = testutil.CreateTestAuditEvent(ctx, testDB.Pool, userID)
	require.NoError(t, err)

	// Export as JSON
	resp, err := http.Get(server.URL + "/api/v1/audit/export?format=json")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Verify JSON content
	body, _ := io.ReadAll(resp.Body)
	var events []map[string]interface{}
	err = json.Unmarshal(body, &events)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
}

// TestHandlers_ExportAuditLogs_CSV tests exporting audit logs as CSV.
func TestHandlers_ExportAuditLogs_CSV(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test user and event
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	_, err = testutil.CreateTestAuditEvent(ctx, testDB.Pool, userID)
	require.NoError(t, err)

	// Export as CSV
	resp, err := http.Get(server.URL + "/api/v1/audit/export?format=csv")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))

	// Verify CSV content
	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	assert.GreaterOrEqual(t, len(lines), 2) // Header + at least one data line
}

// TestHandlers_CreateDataBreach tests creating a data breach.
func TestHandlers_CreateDataBreach(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	breach := map[string]interface{}{
		"severity":         "high",
		"affected_records": 100,
		"data_types":       []string{"email", "name"},
		"description":      "Test breach",
	}

	body, _ := json.Marshal(breach)
	resp, err := http.Post(server.URL+"/api/v1/breaches", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
}

// TestHandlers_ListBreaches tests listing data breaches.
func TestHandlers_ListBreaches(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/breaches")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	breaches, ok := result["breaches"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), result["count"])
	assert.Empty(t, breaches)
}

// TestHandlers_CreateDataBreach_AllSeverities tests creating breaches with all severity levels.
func TestHandlers_CreateDataBreach_AllSeverities(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	severities := []string{"low", "medium", "high", "critical"}

	for _, severity := range severities {
		breach := map[string]interface{}{
			"severity":         severity,
			"affected_records": 10,
			"data_types":       []string{"email"},
			"description":      fmt.Sprintf("Test %s breach", severity),
		}

		body, _ := json.Marshal(breach)
		resp, err := http.Post(server.URL+"/api/v1/breaches", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Severity %s should be valid", severity)
	}
}

// TestHandlers_PendingReviews tests getting pending reviews.
func TestHandlers_PendingReviews(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create a control with upcoming review
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Update to have review soon
	nextReview := time.Now().AddDate(0, 0, 15)
	_, err = testDB.Pool.Exec(ctx, `
		UPDATE compliance_controls
		SET next_review = $1
		WHERE id = $2
	`, nextReview, controlID)
	require.NoError(t, err)

	// Get pending reviews
	resp, err := http.Get(server.URL + "/api/v1/reviews/pending?days=30")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(30), result["days_ahead"])
	assert.GreaterOrEqual(t, int(result["count"].(float64)), 1)

	controls, ok := result["controls"].([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, controls)
}

// TestHandlers_PendingReviews_DefaultDays tests that default days is 30.
func TestHandlers_PendingReviews_DefaultDays(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	// Get pending reviews without specifying days
	resp, err := http.Get(server.URL + "/api/v1/reviews/pending")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(30), result["days_ahead"])
}

// TestHandlers_PendingReviews_CustomDays tests custom days parameter.
func TestHandlers_PendingReviews_CustomDays(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	// Get pending reviews with custom days
	resp, err := http.Get(server.URL + "/api/v1/reviews/pending?days=60")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(60), result["days_ahead"])
}

// TestHandlers_GenerateReport tests generating a compliance report.
func TestHandlers_GenerateReport(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// Create test controls
	_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Generate report
	reportReq := map[string]interface{}{
		"framework":    "fedramp",
		"period_start": time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		"period_end":   time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(reportReq)
	resp, err := http.Post(server.URL+"/api/v1/reports/generate", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "fedramp", result["framework"])
	assert.GreaterOrEqual(t, int(result["total_controls"].(float64)), 1)
}

// TestHandlers_GenerateReport_InvalidFramework tests generating report with invalid framework.
func TestHandlers_GenerateReport_InvalidFramework(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	reportReq := map[string]interface{}{
		"framework":    "invalid",
		"period_start": time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		"period_end":   time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(reportReq)
	resp, err := http.Post(server.URL+"/api/v1/reports/generate", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid framework", result["error"])
}

// TestHandlers_GenerateReport_InvalidDates tests generating report with invalid dates.
func TestHandlers_GenerateReport_InvalidDates(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	reportReq := map[string]interface{}{
		"framework":    "fedramp",
		"period_start": "invalid-date",
		"period_end":   time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(reportReq)
	resp, err := http.Post(server.URL+"/api/v1/reports/generate", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid period_start format", result["error"])
}

// TestHandlers_GenerateReport_AllFrameworks tests generating reports for all frameworks.
func TestHandlers_GenerateReport_AllFrameworks(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	frameworks := []string{"fedramp", "hipaa", "gdpr", "soc2"}

	for _, fw := range frameworks {
		// Create a test control
		_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, fw)
		require.NoError(t, err)

		// Generate report
		reportReq := map[string]interface{}{
			"framework":    fw,
			"period_start": time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
			"period_end":   time.Now().Format(time.RFC3339),
		}

		body, _ := json.Marshal(reportReq)
		resp, err := http.Post(server.URL+"/api/v1/reports/generate", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Framework %s should generate valid report", fw)
	}
}

// TestHandlers_Summary tests getting compliance summary.
func TestHandlers_Summary(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports/summary")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	frameworks, ok := result["frameworks"].([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, frameworks)

	// Verify all frameworks are present
	frameworkMap := make(map[string]bool)
	for _, fw := range frameworks {
		fwMap := fw.(map[string]interface{})
		frameworkMap[fwMap["framework"].(string)] = true
	}

	assert.True(t, frameworkMap["fedramp"])
	assert.True(t, frameworkMap["hipaa"])
	assert.True(t, frameworkMap["gdpr"])
	assert.True(t, frameworkMap["soc2"])
}

// TestHandlers_ControlByID_Delete tests DELETE endpoint for compliance controls.
func TestHandlers_ControlByID_Delete(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("successfully delete existing control", func(t *testing.T) {
		// Create a test control
		controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
		require.NoError(t, err)

		// Delete the control
		req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/controls/"+controlID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify the control is deleted
		req2, _ := http.NewRequest("GET", server.URL+"/api/v1/controls/"+controlID, nil)
		resp2, err := client.Do(req2)
		require.NoError(t, err)
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
	})

	t.Run("return not found for non-existent control", func(t *testing.T) {
		fakeID := "00000000-0000-0000-0000-000000000000"

		req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/controls/"+fakeID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestHandlers_InvalidJSON tests sending invalid JSON to POST endpoints.
func TestHandlers_InvalidJSON(t *testing.T) {
	_, server := setupHandlerTest(t)
	defer server.Close()

	// Try to create control with invalid JSON
	resp, err := http.Post(server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer([]byte("invalid json")))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Try to create breach with invalid JSON
	resp2, err := http.Post(server.URL+"/api/v1/breaches", "application/json", bytes.NewBuffer([]byte("invalid json")))
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	// Try to create audit event with invalid JSON
	resp3, err := http.Post(server.URL+"/api/v1/audit", "application/json", bytes.NewBuffer([]byte("invalid json")))
	require.NoError(t, err)
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp3.StatusCode)
}

// TestHandlers_Integration_FullWorkflow tests a complete workflow.
func TestHandlers_Integration_FullWorkflow(t *testing.T) {
	testDB, server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	// 1. Create a control
	control := map[string]interface{}{
		"framework":        "fedramp",
		"family":           "Access Control",
		"title":            "AC-001: Test Control",
		"description":      "Test description",
		"status":           "pending",
		"responsible_team": "Security",
		"risk_level":       "medium",
	}

	body, _ := json.Marshal(control)
	resp, err := http.Post(server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdControl map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&createdControl)
	controlID := createdControl["id"].(string)

	// 2. Get the control
	resp2, err := http.Get(server.URL + "/api/v1/controls/" + controlID)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// 3. Update the control status
	updateReq := map[string]interface{}{
		"status":        "compliant",
		"last_assessed": time.Now().Format(time.RFC3339),
		"next_review":   time.Now().AddDate(0, 0, 60).Format(time.RFC3339),
	}

	body2, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/controls/status/"+controlID, bytes.NewBuffer(body2))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp3, err := client.Do(req)
	require.NoError(t, err)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	// 4. List controls to verify
	resp4, err := http.Get(server.URL + "/api/v1/controls?framework=fedramp")
	require.NoError(t, err)
	defer resp4.Body.Close()

	var listResult map[string]interface{}
	json.NewDecoder(resp4.Body).Decode(&listResult)
	assert.GreaterOrEqual(t, int(listResult["total"].(float64)), 1)

	// 5. Generate a report
	reportReq := map[string]interface{}{
		"framework":    "fedramp",
		"period_start": time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		"period_end":   time.Now().Format(time.RFC3339),
	}

	body3, _ := json.Marshal(reportReq)
	resp5, err := http.Post(server.URL+"/api/v1/reports/generate", "application/json", bytes.NewBuffer(body3))
	require.NoError(t, err)
	defer resp5.Body.Close()
	assert.Equal(t, http.StatusOK, resp5.StatusCode)

	// 6. Create an audit event tracking the changes
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)
	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	event := map[string]interface{}{
		"user_id":       userID,
		"user_name":     "Test User",
		"resource_id":   controlID,
		"resource_type": "compliance_control",
		"action":        "update_status",
		"outcome":       "success",
		"ip_address":    "127.0.0.1",
		"event_type":    "compliance_update",
		"category":      "compliance",
	}

	body4, _ := json.Marshal(event)
	resp6, err := http.Post(server.URL+"/api/v1/audit", "application/json", bytes.NewBuffer(body4))
	require.NoError(t, err)
	defer resp6.Body.Close()
	assert.Equal(t, http.StatusCreated, resp6.StatusCode)
}
