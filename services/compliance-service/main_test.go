// Package main provides tests for the compliance service handlers.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openprint/openprint/internal/testutil"
)

var (
	testDB *testutil.TestDB
	ctx    = context.Background()
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("TestMain: Starting test database setup...")

	var err error
	testDB, err = testutil.SetupPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}

	log.Println("TestMain: Database setup complete, running tests...")
	defer func() {
		log.Println("TestMain: Cleaning up...")
		testutil.Cleanup(testDB)
	}()

	exitCode := m.Run()
	log.Printf("TestMain: Tests finished with exit code: %d", exitCode)
	os.Exit(exitCode)
}

// TestServer wraps the test server and dependencies.
type TestServer struct {
	Server  *httptest.Server
	DB      testutil.TestDB
	Service *Service
	Cleanup func()
}

// SetupTestServer creates a test server with database.
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available - run with test tag")
	}

	svc := New(Config{DB: testDB.Pool})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/controls", listControlsHandler(svc))
	mux.HandleFunc("/api/v1/controls/", controlByIDHandler(svc))
	mux.HandleFunc("/api/v1/controls/status/", updateControlStatusHandler(svc))
	mux.HandleFunc("/api/v1/breaches", breachesHandler(svc))
	mux.HandleFunc("/api/v1/reviews/pending", pendingReviewsHandler(svc))
	mux.HandleFunc("/api/v1/reports/summary", summaryHandler(svc))

	server := httptest.NewServer(mux)

	cleanup := func() {
		server.Close()
	}

	return &TestServer{
		Server:  server,
		DB:      *testDB,
		Service: svc,
		Cleanup: cleanup,
	}
}

func TestHealthHandler(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %s", result["status"])
	}
	if result["service"] != "compliance-service" {
		t.Errorf("Expected service compliance-service, got %s", result["service"])
	}
}

func TestListControlsHandler_Empty(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/v1/controls")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that controls field exists
	if _, ok := result["controls"]; !ok {
		t.Error("Expected controls field in response")
	}
}

func TestCreateControlHandler(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	control := map[string]interface{}{
		"framework":        "fedramp",
		"family":           "Access Control",
		"title":            "Test Control",
		"description":      "Test description",
		"implementation":   "Test implementation",
		"status":           "pending",
		"next_review":      time.Now().AddDate(0, 0, 30).Format(time.RFC3339),
		"responsible_team": "Security",
		"risk_level":       "medium",
	}

	body, _ := json.Marshal(control)
	resp, err := http.Post(ts.Server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["id"] == nil {
		t.Error("Expected ID in response")
	}
}

func TestCreateControlHandler_InvalidFramework(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	control := map[string]interface{}{
		"framework":   "invalid",
		"family":      "Access Control",
		"title":       "Test Control",
		"description": "Test description",
		"status":      "pending",
		"risk_level":  "medium",
	}

	body, _ := json.Marshal(control)
	resp, err := http.Post(ts.Server.URL+"/api/v1/controls", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestControlByIDHandler_NotFound(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/v1/controls/" + uuid.New().String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestUpdateControlStatusHandler(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// First create a control
	controlID := uuid.New()
	const query = `
		INSERT INTO compliance_controls (id, framework, family, title, description, status, next_review, risk_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := ts.DB.Pool.Exec(ctx, query, controlID, "fedramp", "Access Control", "Test Control", "Test desc", "pending", time.Now().AddDate(0, 0, 30), "medium")
	if err != nil {
		t.Fatalf("Failed to create control: %v", err)
	}

	// Update status
	updateReq := map[string]interface{}{
		"status":        "compliant",
		"last_assessed": time.Now().Format(time.RFC3339),
		"next_review":   time.Now().AddDate(0, 0, 60).Format(time.RFC3339),
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", ts.Server.URL+"/api/v1/controls/status/"+controlID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["control_id"] != controlID.String() {
		t.Errorf("Expected control_id %s, got %v", controlID.String(), result["control_id"])
	}

	if result["status"] != "compliant" {
		t.Errorf("Expected status compliant, got %v", result["status"])
	}
}

func TestBreachesHandler_Create(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	breach := map[string]interface{}{
		"severity":         "low",
		"affected_records": 5,
		"data_types":       []string{"email", "name"},
		"description":      "Test breach",
	}

	body, _ := json.Marshal(breach)
	resp, err := http.Post(ts.Server.URL+"/api/v1/breaches", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["id"] == "" {
		t.Error("Expected non-empty ID in response")
	}
}

func TestPendingReviewsHandler(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/v1/reviews/pending?days=30")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["controls"] == nil {
		t.Error("Expected controls in response")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Try POST on health endpoint (GET only)
	resp, err := http.Post(ts.Server.URL+"/health", "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
