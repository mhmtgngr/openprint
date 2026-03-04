// Package main provides tests for the policy service handlers.
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
	Engine  *Engine
	Cleanup func()
}

// SetupTestServer creates a test server with database.
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available - run with test tag")
	}

	engine := NewEngine(Config{DB: testDB.Pool})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/policies", policiesHandler(engine))
	mux.HandleFunc("/api/v1/policies/", policyByIDHandler(engine))
	mux.HandleFunc("/api/v1/evaluate", evaluateHandler(engine))
	mux.HandleFunc("/api/v1/rules/validate", validateRulesHandler(engine))
	mux.HandleFunc("/api/v1/test", testPolicyHandler(engine))

	server := httptest.NewServer(mux)

	cleanup := func() {
		server.Close()
	}

	return &TestServer{
		Server:  server,
		DB:      *testDB,
		Engine:  engine,
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
	if result["service"] != "policy-service" {
		t.Errorf("Expected service policy-service, got %s", result["service"])
	}
}

func TestListPoliciesHandler_Empty(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/v1/policies")
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

	policies, ok := result["policies"].([]interface{})
	if !ok {
		t.Fatal("policies field missing or wrong type")
	}

	if len(policies) != 0 {
		t.Errorf("Expected 0 policies, got %d", len(policies))
	}

	// JSON numbers are decoded as float64
	total, ok := result["total"].(float64)
	if !ok || total != 0 {
		t.Errorf("Expected total 0, got %v (type %T)", result["total"], result["total"])
	}
}

func TestCreatePolicyHandler(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	policy := map[string]interface{}{
		"name":        "Test Policy",
		"description": "Test policy description",
		"type":        "quota",
		"status":      "draft",
		"priority":    50,
		"rules": []map[string]interface{}{
			{
				"id":       "rule1",
				"field":    "document.page_count",
				"operator": "less_than",
				"value":    100,
			},
		},
		"actions": []map[string]interface{}{
			{
				"type": "deny",
				"parameters": map[string]interface{}{
					"message": "Page count exceeded",
				},
			},
		},
		"scope": map[string]interface{}{
			"user_ids":       []string{},
			"group_ids":      []string{},
			"printer_ids":    []string{},
			"document_types": []string{},
		},
	}

	body, _ := json.Marshal(policy)
	resp, err := http.Post(ts.Server.URL+"/api/v1/policies", "application/json", bytes.NewBuffer(body))
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

	if result["message"] != "Policy created" {
		t.Errorf("Expected message 'Policy created', got %s", result["message"])
	}
}

func TestCreatePolicyHandler_InvalidType(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	policy := map[string]interface{}{
		"name":        "Test Policy",
		"description": "Test policy description",
		"type":        "invalid_type",
		"status":      "draft",
		"rules":       []map[string]interface{}{},
		"actions":     []map[string]interface{}{},
	}

	body, _ := json.Marshal(policy)
	resp, err := http.Post(ts.Server.URL+"/api/v1/policies", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestCreatePolicyHandler_MissingRuleID(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	policy := map[string]interface{}{
		"name":        "Test Policy",
		"description": "Test policy description",
		"type":        "quota",
		"status":      "draft",
		"rules": []map[string]interface{}{
			{
				"field":    "document.page_count",
				"operator": "less_than",
				"value":    100,
			},
		},
		"actions": []map[string]interface{}{
			{
				"type": "deny",
			},
		},
	}

	body, _ := json.Marshal(policy)
	resp, err := http.Post(ts.Server.URL+"/api/v1/policies", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestPolicyByIDHandler_NotFound(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/v1/policies/" + uuid.New().String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestEvaluateHandler_DenyPolicy(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create a test organization and user first for foreign key constraint
	orgID, err := testutil.CreateTestOrganization(ctx, ts.DB.Pool)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	userID, err := testutil.CreateTestUser(ctx, ts.DB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a deny policy using Repository.Create
	policy := &Policy{
		Name:        "Deny Large Documents",
		Description: "Denies documents over 10 pages",
		Type:        PolicyTypeContent,
		Status:      PolicyStatusActive,
		Priority:    100,
		Rules: []Rule{
			{ID: "rule1", Field: "document.page_count", Operator: OpGreaterThan, Value: 10},
		},
		Actions: []PolicyActionConfig{
			{Type: ActionDeny, Parameters: map[string]interface{}{"message": "Too many pages"}, Order: 1},
		},
		Scope:     PolicyScope{},
		CreatedBy: userID,
	}

	repo := NewRepository(ts.DB.Pool)
	if err := repo.Create(ctx, policy); err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Reload policies
	_ = ts.Engine.LoadPolicies(ctx)

	// Evaluate with a document that should be denied
	evalCtx := map[string]interface{}{
		"user_id":       uuid.New().String(),
		"printer_id":    uuid.New().String(),
		"document_name": "large.pdf",
		"document_type": "pdf",
		"page_count":    50,
		"color_mode":    "color",
		"duplex_mode":   "duplex",
		"cost":          5.50,
	}

	body, _ := json.Marshal(evalCtx)
	resp, err := http.Post(ts.Server.URL+"/api/v1/evaluate", "application/json", bytes.NewBuffer(body))
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

	if result["action"] != "deny" {
		t.Errorf("Expected action 'deny', got %v", result["action"])
	}
}

func TestEvaluateHandler_AllowPolicy(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	evalCtx := map[string]interface{}{
		"user_id":       uuid.New().String(),
		"printer_id":    uuid.New().String(),
		"document_name": "small.pdf",
		"document_type": "pdf",
		"page_count":    5,
		"color_mode":    "color",
		"duplex_mode":   "duplex",
		"cost":          0.50,
	}

	body, _ := json.Marshal(evalCtx)
	resp, err := http.Post(ts.Server.URL+"/api/v1/evaluate", "application/json", bytes.NewBuffer(body))
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

	if result["action"] != "allow" {
		t.Errorf("Expected action 'allow', got %v", result["action"])
	}
}

func TestValidateRulesHandler_Valid(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	req := map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"id":       "rule1",
				"field":    "document.page_count",
				"operator": "less_than",
				"value":    100,
			},
			{
				"id":       "rule2",
				"field":    "time.hour",
				"operator": "between",
				"value":    []int{9, 17},
			},
		},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(ts.Server.URL+"/api/v1/rules/validate", "application/json", bytes.NewBuffer(body))
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

	if result["valid"].(bool) != true {
		t.Error("Expected valid to be true")
	}
}

func TestValidateRulesHandler_Invalid(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	req := map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"field":    "document.page_count",
				"operator": "less_than",
				"value":    100,
			},
			{
				"id":       "rule2",
				"operator": "between",
				"value":    []int{9, 17},
			},
		},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(ts.Server.URL+"/api/v1/rules/validate", "application/json", bytes.NewBuffer(body))
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

	if result["valid"].(bool) != false {
		t.Error("Expected valid to be false")
	}
}

func TestTestPolicyHandler_Matched(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	policy := map[string]interface{}{
		"name":     "Test Policy",
		"type":     "access",
		"status":   "active",
		"priority": 50,
		"rules": []map[string]interface{}{
			{
				"id":       "rule1",
				"field":    "document.page_count",
				"operator": "less_than",
				"value":    100,
			},
		},
		"actions": []map[string]interface{}{
			{
				"type": "allow",
			},
		},
		"scope": map[string]interface{}{},
	}

	testCtx := map[string]interface{}{
		"user_id":       uuid.New().String(),
		"printer_id":    uuid.New().String(),
		"document_name": "test.pdf",
		"document_type": "pdf",
		"page_count":    50,
		"color_mode":    "color",
		"duplex_mode":   "duplex",
		"cost":          1.50,
	}

	req := map[string]interface{}{
		"policy":       policy,
		"test_context": testCtx,
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(ts.Server.URL+"/api/v1/test", "application/json", bytes.NewBuffer(body))
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

	if result["matched"].(bool) != true {
		t.Error("Expected policy to match")
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

	// Try DELETE on list endpoint
	req, _ := http.NewRequest("DELETE", ts.Server.URL+"/api/v1/policies", nil)
	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp2.StatusCode)
	}
}
