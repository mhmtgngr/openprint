//go:build integration
// +build integration

// Package integration provides end-to-end API tests for OpenPrint Cloud services.
// These tests require running services (via Docker Compose) and test actual HTTP requests,
// database interactions, and inter-service communication.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Service base URLs (configurable via environment)
var (
	authServiceURL        string
	registryServiceURL    string
	jobServiceURL         string
	storageServiceURL     string
	notificationServiceURL string
	databaseURL           string
)

func init() {
	// Read environment variables on init
	authServiceURL = getEnv("AUTH_SERVICE_URL", "http://localhost:18001")
	registryServiceURL = getEnv("REGISTRY_SERVICE_URL", "http://localhost:8002")
	jobServiceURL = getEnv("JOB_SERVICE_URL", "http://localhost:8003")
	storageServiceURL = getEnv("STORAGE_SERVICE_URL", "http://localhost:8004")
	notificationServiceURL = getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:18005")
	databaseURL = getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:15432/openprint")
}

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// connectDB safely connects to the database and returns the connection.
// Returns nil if connection fails (to prevent panics in cleanup code).
func connectDB(ctx context.Context) *pgx.Conn {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		// Log the error but don't panic in cleanup code
		fmt.Printf("Warning: failed to connect to database for cleanup: %v\n", err)
		return nil
	}
	return conn
}

// cleanupDB executes a cleanup function with a safe database connection.
// If the database connection fails, the cleanup function is not called.
func cleanupDB(ctx context.Context, cleanupFunc func(*pgx.Conn)) {
	conn := connectDB(ctx)
	if conn == nil {
		return
	}
	defer conn.Close(ctx)
	cleanupFunc(conn)
}

// TestMain checks if services are available before running tests
func TestMain(m *testing.M) {
	// Give services time to start if tests run right after docker-compose up
	time.Sleep(2 * time.Second)
	os.Exit(m.Run())
}

// Helper struct for HTTP test client
type TestClient struct {
	Client       *http.Client
	AuthToken    string
	RefreshToken string
	UserID       string
	AgentID      string
	PrinterID    string
	DocumentID   string
	JobID        string
}

// NewTestClient creates a new HTTP client for integration testing
func NewTestClient() *TestClient {
	return &TestClient{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest performs an HTTP request with optional auth token
func (c *TestClient) makeRequest(method, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	return c.Client.Do(req)
}

// ============================================================================
// Health Check Tests (Table-Driven)
// ============================================================================

func TestHealthChecks(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedKey string
		expectedVal string
	}{
		{
			name:        "auth service health",
			url:         authServiceURL + "/health",
			expectedKey: "service",
			expectedVal: "auth-service",
		},
		{
			name:        "registry service health",
			url:         registryServiceURL + "/health",
			expectedKey: "service",
			expectedVal: "registry-service",
		},
		{
			name:        "job service health",
			url:         jobServiceURL + "/health",
			expectedKey: "service",
			expectedVal: "job-service",
		},
		{
			name:        "storage service health",
			url:         storageServiceURL + "/health",
			expectedKey: "service",
			expectedVal: "storage-service",
		},
		{
			name:        "notification service health",
			url:         notificationServiceURL + "/health",
			expectedKey: "service",
			expectedVal: "notification-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Checking health endpoint: %s", tt.url)

			resp, err := http.Get(tt.url)
			require.NoError(t, err, "Failed to make health check request")
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200")

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err, "Failed to decode health response")

			assert.Contains(t, result, tt.expectedKey, "Response should contain service key")
			assert.Equal(t, tt.expectedVal, result[tt.expectedKey], "Service name should match")
			assert.Equal(t, "healthy", result["status"], "Status should be healthy")
		})
	}
}

// ============================================================================
// Database Connection Tests
// ============================================================================

func TestDatabaseConnection(t *testing.T) {
	t.Run("connect to postgres", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := pgx.Connect(ctx, databaseURL)
		require.NoError(t, err, "Should connect to database")
		defer conn.Close(ctx)

		var result string
		err = conn.QueryRow(ctx, "SELECT current_database()").Scan(&result)
		require.NoError(t, err, "Should query current database")
		assert.Equal(t, "openprint", result, "Database name should match")
	})

	t.Run("check required tables exist", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := pgx.Connect(ctx, databaseURL)
		require.NoError(t, err, "Should connect to database")
		defer conn.Close(ctx)

		expectedTables := []string{
			"users", "organizations", "agents", "printers",
			"documents", "print_jobs", "user_sessions",
		}

		for _, table := range expectedTables {
			t.Run("table_"+table, func(t *testing.T) {
				var exists bool
				query := `
					SELECT EXISTS (
						SELECT FROM information_schema.tables
						WHERE table_name = $1
					)`
				err = conn.QueryRow(ctx, query, table).Scan(&exists)
				require.NoError(t, err, "Should check table existence")
				assert.True(t, exists, "Table %s should exist", table)
			})
		}
	})
}

// ============================================================================
// Auth Service Tests (Table-Driven)
// ============================================================================

func TestAuthServiceEndpoints(t *testing.T) {
	client := NewTestClient()
	var testEmail string

	tests := []struct {
		name           string
		method         string
		url            string
		body           interface{}
		validateFunc   func(*testing.T, *http.Response)
		expectedStatus int
		skipAuth       bool
	}{
		{
			name:           "register new user",
			method:         "POST",
			url:            authServiceURL + "/auth/register",
			body: map[string]interface{}{
				"email":       fmt.Sprintf("test+%d@example.com", time.Now().UnixNano()),
				"password":    "SecurePassword123!",
				"first_name":  "Test",
				"last_name":   "User",
			},
			expectedStatus: http.StatusCreated,
			skipAuth:       true,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "user_id")
				assert.Contains(t, result, "access_token")
				client.UserID = result["user_id"].(string)
				client.AuthToken = result["access_token"].(string)
				if refresh, ok := result["refresh_token"].(string); ok {
					client.RefreshToken = refresh
				}
			},
		},
		{
			name:           "get current user",
			method:         "GET",
			url:            authServiceURL + "/auth/me",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "email")
			},
		},
		{
			name:           "unauthorized request",
			method:         "GET",
			url:            authServiceURL + "/auth/me",
			body:           nil,
			skipAuth:       true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "register duplicate email should fail",
			method:         "POST",
			url:            authServiceURL + "/auth/register",
			body: map[string]interface{}{
				"email":       "duplicate@example.com",
				"password":    "SecurePassword123!",
				"first_name":  "Duplicate",
				"last_name":   "Test",
			},
			expectedStatus: http.StatusCreated,
			skipAuth:       true,
			validateFunc: func(t *testing.T, resp *http.Response) {
				// First registration should succeed
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				testEmail = "duplicate@example.com"

				// Try to register again - should fail
				dupResp, _ := client.makeRequest("POST", authServiceURL+"/auth/register",
					map[string]interface{}{
						"email":    testEmail,
						"password": "AnotherPassword123!",
						"first_name": "Another",
						"last_name":  "User",
					}, nil)
				defer dupResp.Body.Close()
				// Should return conflict or bad request
				assert.Contains(t, []int{http.StatusConflict, http.StatusBadRequest}, dupResp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			if !tt.skipAuth {
				resp, err = client.makeRequest(tt.method, tt.url, tt.body, nil)
			} else {
				// Make request without auth
				var reqBody io.Reader
				if tt.body != nil {
					jsonData, _ := json.Marshal(tt.body)
					reqBody = bytes.NewReader(jsonData)
				}
				req, _ := http.NewRequest(tt.method, tt.url, reqBody)
				req.Header.Set("Content-Type", "application/json")
				resp, err = client.Client.Do(req)
			}

			require.NoError(t, err, "Request should succeed")
			defer resp.Body.Close()

			// Allow flexible status codes for certain tests
			if tt.expectedStatus > 0 {
				assert.Equal(t, tt.expectedStatus, resp.StatusCode,
					"Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.validateFunc != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				tt.validateFunc(t, resp)
			}
		})
	}

	// Cleanup test data
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cleanupDB(ctx, func(conn *pgx.Conn) {
			// Clean up test emails
			testEmails := []string{
				"duplicate@example.com",
			}
			for _, email := range testEmails {
				conn.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
			}
		})
	})
}

// ============================================================================
// Registry Service Tests
// ============================================================================

func TestRegistryServiceEndpoints(t *testing.T) {
	client := NewTestClient()

	// Setup: Get auth token
	testEmail := fmt.Sprintf("registry_test+%d@example.com", time.Now().UnixNano())
	registerResp, err := client.makeRequest("POST", authServiceURL+"/auth/register",
		map[string]interface{}{
			"email":       testEmail,
			"password":    "SecurePassword123!",
			"first_name":  "Registry",
			"last_name":   "Tester",
		}, nil)
	if err == nil && registerResp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		json.NewDecoder(registerResp.Body).Decode(&result)
		client.AuthToken = result["access_token"].(string)
		client.UserID = result["user_id"].(string)
		registerResp.Body.Close()
	} else if registerResp != nil {
		registerResp.Body.Close()
	}

	// Track cleanup IDs
	var cleanupAgentID, cleanupPrinterID string

	tests := []struct {
		name           string
		method         string
		url            string
		urlFunc        func() string
		body           interface{}
		bodyFunc       func() interface{}
		validateFunc   func(*testing.T, *http.Response)
		expectedStatus int
		skipOnFail     bool
	}{
		{
			name:       "register agent",
			method:     "POST",
			url:        registryServiceURL + "/agents/register",
			body: map[string]interface{}{
				"name":         fmt.Sprintf("test-agent-%d", time.Now().UnixNano()),
				"version":      "1.0.0",
				"os":           "linux",
				"architecture": "amd64",
				"hostname":     "test-host",
			},
			expectedStatus: http.StatusCreated,
			skipOnFail:     true, // Skip dependent tests if this fails
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "agent_id")
				client.AgentID = result["agent_id"].(string)
				cleanupAgentID = client.AgentID
			},
		},
		{
			name:           "list agents",
			method:         "GET",
			url:            registryServiceURL + "/agents",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "agents")
			},
		},
		{
			name:           "get agent by id",
			method:         "GET",
			urlFunc:        func() string { return registryServiceURL + "/agents/" + client.AgentID },
			body:           nil,
			expectedStatus: http.StatusOK,
			skipOnFail:     true,
		},
		{
			name:       "register printer",
			method:     "POST",
			url:        registryServiceURL + "/printers/register",
			bodyFunc: func() interface{} {
				return map[string]interface{}{
					"agent_id": client.AgentID,
					"name":     fmt.Sprintf("test-printer-%d", time.Now().UnixNano()),
					"model":    "HP LaserJet Pro",
					"location": "Office 1",
				}
			},
			expectedStatus: http.StatusCreated,
			skipOnFail:     true,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "printer_id")
				client.PrinterID = result["printer_id"].(string)
				cleanupPrinterID = client.PrinterID
			},
		},
		{
			name:           "list printers",
			method:         "GET",
			url:            registryServiceURL + "/printers",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "printers")
			},
		},
		{
			name:           "send agent heartbeat",
			method:         "POST",
			urlFunc:        func() string { return registryServiceURL + "/agents/" + client.AgentID + "/heartbeat" },
			body:           map[string]interface{}{"status": "online"},
			expectedStatus: http.StatusOK,
			skipOnFail:     true,
		},
		{
			name:           "update printer status",
			method:         "PUT",
			urlFunc:        func() string { return registryServiceURL + "/printers/" + client.PrinterID + "/status" },
			body:           map[string]interface{}{"status": "online"},
			expectedStatus: http.StatusOK,
			skipOnFail:     true,
		},
	}

	skipNext := false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnFail && skipNext {
				t.Skip("Skipping due to previous test failure")
			}

			url := tt.url
			if tt.urlFunc != nil {
				url = tt.urlFunc()
			}

			body := tt.body
			if tt.bodyFunc != nil {
				body = tt.bodyFunc()
			}

			resp, err := client.makeRequest(tt.method, url, body, nil)
			require.NoError(t, err, "Request should succeed")
			defer resp.Body.Close()

			statusOK := assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)

			if tt.skipOnFail && !statusOK {
				skipNext = true
			}

			if tt.validateFunc != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				tt.validateFunc(t, resp)
			}
		})
	}

	// Cleanup
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cleanupDB(ctx, func(conn *pgx.Conn) {
			if cleanupPrinterID != "" {
				conn.Exec(ctx, "DELETE FROM printers WHERE id = $1", cleanupPrinterID)
			}
			if cleanupAgentID != "" {
				conn.Exec(ctx, "DELETE FROM agents WHERE id = $1", cleanupAgentID)
			}
			conn.Exec(ctx, "DELETE FROM users WHERE email = $1", testEmail)
		})
	})
}

// ============================================================================
// Job Service Tests
// ============================================================================

func TestJobServiceEndpoints(t *testing.T) {
	client := NewTestClient()

	// Setup: Register user
	testEmail := fmt.Sprintf("job_test+%d@example.com", time.Now().UnixNano())
	registerResp, _ := client.makeRequest("POST", authServiceURL+"/auth/register",
		map[string]interface{}{
			"email":       testEmail,
			"password":    "SecurePassword123!",
			"first_name":  "Job",
			"last_name":   "Tester",
		}, nil)
	if registerResp != nil && registerResp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		json.NewDecoder(registerResp.Body).Decode(&result)
		client.AuthToken = result["access_token"].(string)
		client.UserID = result["user_id"].(string)
		registerResp.Body.Close()
	}

	// Setup: Register agent
	agentResp, _ := client.makeRequest("POST", registryServiceURL+"/agents/register",
		map[string]interface{}{
			"name":         fmt.Sprintf("job-agent-%d", time.Now().UnixNano()),
			"version":      "1.0.0",
			"os":           "linux",
			"architecture": "amd64",
			"hostname":     "job-test-host",
		}, nil)
	if agentResp != nil && agentResp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		json.NewDecoder(agentResp.Body).Decode(&result)
		client.AgentID = result["agent_id"].(string)
		agentResp.Body.Close()
	}

	// Setup: Register printer
	printerResp, _ := client.makeRequest("POST", registryServiceURL+"/printers/register",
		map[string]interface{}{
			"agent_id":  client.AgentID,
			"name":      fmt.Sprintf("job-printer-%d", time.Now().UnixNano()),
			"model":     "Test Printer",
			"location":  "Test Location",
		}, nil)
	if printerResp != nil && printerResp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		json.NewDecoder(printerResp.Body).Decode(&result)
		client.PrinterID = result["printer_id"].(string)
		printerResp.Body.Close()
	}

	// Setup: Upload a document for the job
	var cleanupDocumentID string
	{
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test-job-document.txt")
		if err == nil {
			part.Write([]byte("This is a test document for job printing."))
			writer.WriteField("title", "Test Document for Job")
			writer.Close()
			req, err := http.NewRequest("POST", storageServiceURL+"/documents", body)
			if err == nil {
				req.Header.Set("Content-Type", writer.FormDataContentType())
				if client.AuthToken != "" {
					req.Header.Set("Authorization", "Bearer "+client.AuthToken)
				}
				documentResp, err := client.Client.Do(req)
				if err == nil && documentResp.StatusCode == http.StatusCreated {
					var result map[string]interface{}
					json.NewDecoder(documentResp.Body).Decode(&result)
					client.DocumentID = result["document_id"].(string)
					cleanupDocumentID = client.DocumentID
					documentResp.Body.Close()
				}
			}
		}
	}

	// Track cleanup IDs
	var cleanupJobID string
	skipNext := false

	tests := []struct {
		name           string
		method         string
		url            string
		urlFunc        func() string
		body           interface{}
		validateFunc   func(*testing.T, *http.Response)
		expectedStatus int
		skipOnFail     bool
	}{
		{
			name:           "list jobs",
			method:         "GET",
			url:            jobServiceURL + "/jobs",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "jobs")
			},
		},
		{
			name:       "create job",
			method:     "POST",
			url:        jobServiceURL + "/jobs",
			body: map[string]interface{}{
				"document_id": client.DocumentID,
				"printer_id":  client.PrinterID,
				"user_name":   "Test User",
				"user_email":  testEmail,
				"title":       "Test Print Job",
				"copies":      1,
				"color_mode":  "monochrome",
				"duplex":      false,
				"media_type":  "a4",
				"quality":     "normal",
				"pages":       5,
			},
			expectedStatus: http.StatusCreated,
			skipOnFail:     true,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "job_id")
				client.JobID = result["job_id"].(string)
				cleanupJobID = client.JobID
			},
		},
		{
			name:           "get job by id",
			method:         "GET",
			urlFunc:        func() string { return jobServiceURL + "/jobs/" + client.JobID },
			body:           nil,
			expectedStatus: http.StatusOK,
			skipOnFail:     true,
		},
		{
			name:           "get queue stats",
			method:         "GET",
			url:            jobServiceURL + "/queue/stats",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "queued")
				assert.Contains(t, result, "processing")
				assert.Contains(t, result, "completed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnFail && skipNext {
				t.Skip("Skipping due to previous test failure")
			}

			url := tt.url
			if tt.urlFunc != nil {
				url = tt.urlFunc()
			}

			resp, err := client.makeRequest(tt.method, url, tt.body, nil)
			require.NoError(t, err, "Request should succeed")
			defer resp.Body.Close()

			statusOK := assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)

			if tt.skipOnFail && !statusOK {
				skipNext = true
			}

			if tt.validateFunc != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				tt.validateFunc(t, resp)
			}
		})
	}

	// Cleanup
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cleanupDB(ctx, func(conn *pgx.Conn) {
			if cleanupJobID != "" {
				conn.Exec(ctx, "DELETE FROM print_jobs WHERE id = $1", cleanupJobID)
			}
			if cleanupDocumentID != "" {
				conn.Exec(ctx, "DELETE FROM documents WHERE id = $1", cleanupDocumentID)
			}
			if client.PrinterID != "" {
				conn.Exec(ctx, "DELETE FROM printers WHERE id = $1", client.PrinterID)
			}
			if client.AgentID != "" {
				conn.Exec(ctx, "DELETE FROM agents WHERE id = $1", client.AgentID)
			}
			conn.Exec(ctx, "DELETE FROM users WHERE email = $1", testEmail)
		})
	})
}

// ============================================================================
// Storage Service Tests
// ============================================================================

func TestStorageServiceEndpoints(t *testing.T) {
	client := NewTestClient()

	// Setup: Get auth token
	testEmail := fmt.Sprintf("storage_test+%d@example.com", time.Now().UnixNano())
	registerResp, _ := client.makeRequest("POST", authServiceURL+"/auth/register",
		map[string]interface{}{
			"email":       testEmail,
			"password":    "SecurePassword123!",
			"first_name":  "Storage",
			"last_name":   "Tester",
		}, nil)
	if registerResp != nil && registerResp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		json.NewDecoder(registerResp.Body).Decode(&result)
		client.AuthToken = result["access_token"].(string)
		client.UserID = result["user_id"].(string)
		registerResp.Body.Close()
	}

	var cleanupDocumentID string
	skipNext := false

	tests := []struct {
		name           string
		method         string
		url            string
		urlFunc        func() string
		body           interface{}
		isMultipart    bool
		filename       string
		validateFunc   func(*testing.T, *http.Response)
		expectedStatus int
		skipOnFail     bool
	}{
		{
			name:           "list documents",
			method:         "GET",
			url:            storageServiceURL + "/documents",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "documents")
			},
		},
		{
			name:        "upload document",
			method:      "POST",
			url:         storageServiceURL + "/documents",
			isMultipart: true,
			filename:    "test-document.txt",
			expectedStatus: http.StatusCreated,
			skipOnFail:  true,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "document_id")
				client.DocumentID = result["document_id"].(string)
				cleanupDocumentID = client.DocumentID
			},
		},
		{
			name:           "get document metadata",
			method:         "GET",
			urlFunc:        func() string { return storageServiceURL + "/documents/" + client.DocumentID + "/metadata" },
			body:           nil,
			expectedStatus: http.StatusOK,
			skipOnFail:     true,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result, "document_id")
				assert.Contains(t, result, "name")
				assert.Contains(t, result, "size")
				assert.Contains(t, result, "content_type")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnFail && skipNext {
				t.Skip("Skipping due to previous test failure")
			}

			var resp *http.Response
			var err error

			if tt.isMultipart {
				// Create multipart form for file upload
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)

				part, err := writer.CreateFormFile("file", tt.filename)
				require.NoError(t, err)

				// Write test content
				content := []byte("This is a test document for integration testing.")
				_, err = part.Write(content)
				require.NoError(t, err)

				writer.WriteField("title", "Test Document")
				writer.WriteField("description", "Integration test document")

				err = writer.Close()
				require.NoError(t, err)

				req, err := http.NewRequest(tt.method, tt.url, body)
				require.NoError(t, err)

				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("Authorization", "Bearer "+client.AuthToken)

				resp, err = client.Client.Do(req)
			} else {
				// Use urlFunc if provided, otherwise use static url
				url := tt.url
				if tt.urlFunc != nil {
					url = tt.urlFunc()
				}
				resp, err = client.makeRequest(tt.method, url, tt.body, nil)
			}

			require.NoError(t, err, "Request should succeed")
			defer resp.Body.Close()

			statusOK := assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)

			if tt.skipOnFail && !statusOK {
				skipNext = true
			}

			if tt.validateFunc != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				tt.validateFunc(t, resp)
			}
		})
	}

	// Cleanup
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cleanupDB(ctx, func(conn *pgx.Conn) {
			if cleanupDocumentID != "" {
				conn.Exec(ctx, "DELETE FROM documents WHERE id = $1", cleanupDocumentID)
			}
			conn.Exec(ctx, "DELETE FROM users WHERE email = $1", testEmail)
		})
	})
}

// ============================================================================
// End-to-End Workflow Tests
// ============================================================================

func TestEndToEndWorkflow(t *testing.T) {
	t.Run("complete print job workflow", func(t *testing.T) {
		client := NewTestClient()
		testEmail := fmt.Sprintf("e2e_test+%d@example.com", time.Now().UnixNano())

		var cleanupIDs struct {
			userID    string
			agentID   string
			printerID string
			documentID string
			jobID     string
		}

		// Step 1: Register user
		t.Log("Step 1: Register user")
		registerResp, err := client.makeRequest("POST", authServiceURL+"/auth/register",
			map[string]interface{}{
				"email":       testEmail,
				"password":    "SecurePassword123!",
				"first_name":  "E2E",
				"last_name":   "Test",
			}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, registerResp.StatusCode)
		var registerResult map[string]interface{}
		json.NewDecoder(registerResp.Body).Decode(&registerResult)
		registerResp.Body.Close()
		client.UserID = registerResult["user_id"].(string)
		cleanupIDs.userID = client.UserID
		client.AuthToken = registerResult["access_token"].(string)

		// Step 2: Register agent
		t.Log("Step 2: Register agent")
		agentResp, err := client.makeRequest("POST", registryServiceURL+"/agents/register",
			map[string]interface{}{
				"name":         fmt.Sprintf("e2e-agent-%d", time.Now().UnixNano()),
				"version":      "1.0.0",
				"os":           "linux",
				"architecture": "amd64",
				"hostname":     "e2e-host",
			}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, agentResp.StatusCode)
		var agentResult map[string]interface{}
		json.NewDecoder(agentResp.Body).Decode(&agentResult)
		agentResp.Body.Close()
		client.AgentID = agentResult["agent_id"].(string)
		cleanupIDs.agentID = client.AgentID

		// Step 3: Register printer
		t.Log("Step 3: Register printer")
		printerResp, err := client.makeRequest("POST", registryServiceURL+"/printers/register",
			map[string]interface{}{
				"agent_id":  client.AgentID,
				"name":      fmt.Sprintf("e2e-printer-%d", time.Now().UnixNano()),
				"model":     "E2E Printer Model",
				"location":  "E2E Test Lab",
			}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, printerResp.StatusCode)
		var printerResult map[string]interface{}
		json.NewDecoder(printerResp.Body).Decode(&printerResult)
		printerResp.Body.Close()
		client.PrinterID = printerResult["printer_id"].(string)
		cleanupIDs.printerID = client.PrinterID

		// Step 4: Upload document
		t.Log("Step 4: Upload document")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "e2e-test.pdf")
		part.Write([]byte("%PDF-1.4\nTest PDF content"))
		writer.WriteField("title", "E2E Test Document")
		writer.Close()

		uploadReq, _ := http.NewRequest("POST", storageServiceURL+"/documents", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("Authorization", "Bearer "+client.AuthToken)
		uploadResp, err := client.Client.Do(uploadReq)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, uploadResp.StatusCode)
		var uploadResult map[string]interface{}
		json.NewDecoder(uploadResp.Body).Decode(&uploadResult)
		uploadResp.Body.Close()
		client.DocumentID = uploadResult["document_id"].(string)
		cleanupIDs.documentID = client.DocumentID

		// Step 5: Send agent heartbeat
		t.Log("Step 5: Send agent heartbeat")
		heartbeatResp, err := client.makeRequest("POST",
			registryServiceURL+"/agents/"+client.AgentID+"/heartbeat",
			map[string]interface{}{"status": "online"}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, heartbeatResp.StatusCode)
		heartbeatResp.Body.Close()

		// Step 6: Update printer status
		t.Log("Step 6: Update printer status to online")
		printerStatusResp, err := client.makeRequest("PUT",
			registryServiceURL+"/printers/"+client.PrinterID+"/status",
			map[string]interface{}{"status": "online"}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, printerStatusResp.StatusCode)
		printerStatusResp.Body.Close()

		// Step 7: Create print job
		t.Log("Step 7: Create print job")
		jobResp, err := client.makeRequest("POST", jobServiceURL+"/jobs",
			map[string]interface{}{
				"document_id": client.DocumentID,
				"printer_id":  client.PrinterID,
				"user_name":   "E2E Test User",
				"user_email":  testEmail,
				"title":       "E2E Test Print Job",
				"copies":      1,
				"color_mode":  "color",
				"duplex":      true,
				"media_type":  "a4",
				"quality":     "high",
				"pages":       3,
			}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, jobResp.StatusCode)
		var jobResult map[string]interface{}
		json.NewDecoder(jobResp.Body).Decode(&jobResult)
		jobResp.Body.Close()
		client.JobID = jobResult["job_id"].(string)
		cleanupIDs.jobID = client.JobID

		// Step 8: Verify job in database
		t.Log("Step 8: Verify job in database")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, databaseURL)
		require.NoError(t, err, "Failed to connect to database for verification")
		defer conn.Close(ctx)

		var jobCount int
		conn.QueryRow(ctx, "SELECT COUNT(*) FROM print_jobs WHERE id = $1", client.JobID).Scan(&jobCount)
		assert.Equal(t, 1, jobCount, "Job should exist in database")

		// Step 9: Get job details
		t.Log("Step 9: Get job details")
		getJobResp, err := client.makeRequest("GET", jobServiceURL+"/jobs/"+client.JobID, nil, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getJobResp.StatusCode)
		var getJobResult map[string]interface{}
		json.NewDecoder(getJobResp.Body).Decode(&getJobResult)
		getJobResp.Body.Close()
		assert.Equal(t, client.DocumentID, getJobResult["document_id"])
		assert.Equal(t, client.PrinterID, getJobResult["printer_id"])

		// Step 10: List all jobs
		t.Log("Step 10: List all jobs")
		listJobsResp, err := client.makeRequest("GET", jobServiceURL+"/jobs", nil, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listJobsResp.StatusCode)
		var listJobsResult map[string]interface{}
		json.NewDecoder(listJobsResp.Body).Decode(&listJobsResult)
		listJobsResp.Body.Close()
		jobs := listJobsResult["jobs"].([]interface{})
		assert.Greater(t, len(jobs), 0, "Should have at least one job")

		t.Log("End-to-end workflow completed successfully")

		// Cleanup
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cleanupDB(ctx, func(conn *pgx.Conn) {
				if cleanupIDs.jobID != "" {
					conn.Exec(ctx, "DELETE FROM print_jobs WHERE id = $1", cleanupIDs.jobID)
				}
				if cleanupIDs.documentID != "" {
					conn.Exec(ctx, "DELETE FROM documents WHERE id = $1", cleanupIDs.documentID)
				}
				if cleanupIDs.printerID != "" {
					conn.Exec(ctx, "DELETE FROM printers WHERE id = $1", cleanupIDs.printerID)
				}
				if cleanupIDs.agentID != "" {
					conn.Exec(ctx, "DELETE FROM agents WHERE id = $1", cleanupIDs.agentID)
				}
				if cleanupIDs.userID != "" {
					conn.Exec(ctx, "DELETE FROM users WHERE id = $1", cleanupIDs.userID)
				}
			})
		})
	})
}

// ============================================================================
// Docker Container Communication Tests
// ============================================================================

func TestDockerContainerCommunication(t *testing.T) {
	t.Run("service to service communication", func(t *testing.T) {
		client := NewTestClient()

		// Register a user through auth service
		testEmail := fmt.Sprintf("docker_test+%d@example.com", time.Now().UnixNano())
		registerResp, err := client.makeRequest("POST", authServiceURL+"/auth/register",
			map[string]interface{}{
				"email":       testEmail,
				"password":    "SecurePassword123!",
				"first_name":  "Docker",
				"last_name":   "Test",
			}, nil)
		require.NoError(t, err)
		if registerResp.StatusCode == http.StatusCreated {
			var result map[string]interface{}
			json.NewDecoder(registerResp.Body).Decode(&result)
			client.AuthToken = result["access_token"].(string)
			client.UserID = result["user_id"].(string)
			registerResp.Body.Close()

			// Use the auth token to access registry service (inter-service auth validation)
			agentResp, err := client.makeRequest("POST", registryServiceURL+"/agents/register",
				map[string]interface{}{
					"name":         fmt.Sprintf("docker-agent-%d", time.Now().UnixNano()),
					"version":      "1.0.0",
					"os":           "linux",
					"architecture": "amd64",
					"hostname":     "docker-test-host",
				}, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, agentResp.StatusCode,
				"Registry service should accept tokens from auth service")
			var agentResult map[string]interface{}
			json.NewDecoder(agentResp.Body).Decode(&agentResult)
			agentResp.Body.Close()

			var agentID string
			if id, ok := agentResult["agent_id"].(string); ok {
				agentID = id
			}

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cleanupDB(ctx, func(conn *pgx.Conn) {
				if agentID != "" {
					conn.Exec(ctx, "DELETE FROM agents WHERE id = $1", agentID)
				}
				conn.Exec(ctx, "DELETE FROM users WHERE email = $1", testEmail)
			})
		}
	})

	t.Run("database connection from services", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := pgx.Connect(ctx, databaseURL)
		require.NoError(t, err, "Should connect to database")
		defer conn.Close(ctx)

		// Check if we can query the audit log (if it exists)
		var tableExists bool
		conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = 'audit_log'
			)`).Scan(&tableExists)

		if tableExists {
			var auditCount int
			conn.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log").Scan(&auditCount)
			t.Logf("Found %d audit log entries", auditCount)
		}
	})
}

// ============================================================================
// Concurrent Request Tests
// ============================================================================

func TestConcurrentRequests(t *testing.T) {
	t.Run("concurrent health checks", func(t *testing.T) {
		endpoints := []string{
			authServiceURL + "/health",
			registryServiceURL + "/health",
			jobServiceURL + "/health",
			storageServiceURL + "/health",
			notificationServiceURL + "/health",
		}

		results := make(chan string, len(endpoints))

		for _, endpoint := range endpoints {
			go func(url string) {
				resp, err := http.Get(url)
				if err != nil {
					results <- fmt.Sprintf("ERROR: %s - %v", url, err)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					results <- fmt.Sprintf("OK: %s", url)
				} else {
					results <- fmt.Sprintf("FAIL: %s - status %d", url, resp.StatusCode)
				}
			}(endpoint)
		}

		// Collect results
		successCount := 0
		for i := 0; i < len(endpoints); i++ {
			result := <-results
			if strings.HasPrefix(result, "OK:") {
				successCount++
			}
			t.Log(result)
		}

		assert.Equal(t, len(endpoints), successCount, "All concurrent requests should succeed")
	})
}

// ============================================================================
// WebSocket Connection Test
// ============================================================================

func TestWebSocketConnection(t *testing.T) {
	t.Run("notification service websocket endpoint", func(t *testing.T) {
		// Use HTTP to check if the WebSocket endpoint exists
		// We can't make a true WebSocket connection without gorilla/websocket client
		wsHTTPURL := strings.Replace(notificationServiceURL, "http://", "http://", 1) + "/ws"

		req, err := http.NewRequest("GET", wsHTTPURL, nil)
		require.NoError(t, err)

		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// The endpoint should respond (either with upgrade or error)
		// Status 101 = Switching Protocols (successful WebSocket upgrade)
		// Status 400/401 = Bad request / Unauthorized (endpoint exists but wrong params)
		// Status 404 = Endpoint not found
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"WebSocket endpoint should exist")
	})
}

// ============================================================================
// Test Utilities
// ============================================================================

// createTestRecorder creates a response recorder for testing HTTP handlers directly
func createTestRecorder(method, path string, body io.Reader) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return httptest.NewRecorder(), req
}

// requireServicesAvailable checks if all services are available before running tests
func requireServicesAvailable(t *testing.T) {
	services := []struct {
		name string
		url  string
	}{
		{"auth", authServiceURL + "/health"},
		{"registry", registryServiceURL + "/health"},
		{"job", jobServiceURL + "/health"},
		{"storage", storageServiceURL + "/health"},
		{"notification", notificationServiceURL + "/health"},
	}

	unavailable := []string{}
	for _, svc := range services {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", svc.url, nil)
		resp, err := http.DefaultClient.Do(req)
		cancel()

		if err != nil || resp.StatusCode != http.StatusOK {
			unavailable = append(unavailable, svc.name)
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if len(unavailable) > 0 {
		t.Skipf("Services not available: %v. Start services with: cd deployments/docker && docker-compose up -d", unavailable)
	}
}

// TestServiceAvailability is a meta-test that checks if services are running
func TestServiceAvailability(t *testing.T) {
	requireServicesAvailable(t)
}
