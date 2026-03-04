// Package testutil provides test fixture creation helpers for tests.
package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateTestOrganization creates a test organization in the database.
func CreateTestOrganization(ctx context.Context, db *pgxpool.Pool) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO organizations (id, name, slug, plan)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.Exec(ctx, query, id, "Test Organization", "test-org-"+id[:8], "free")
	if err != nil {
		return "", fmt.Errorf("create test organization: %w", err)
	}

	return id, nil
}

// CreateTestUser creates a test user in the database.
func CreateTestUser(ctx context.Context, db *pgxpool.Pool, organizationID string) (string, error) {
	id := uuid.New().String()
	email := fmt.Sprintf("test-%s@example.com", id[:8])

	query := `
		INSERT INTO users (id, email, password, first_name, last_name, organization_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Use a bcrypt hash for "password" (simplified for tests)
	hashedPassword := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	_, err := db.Exec(ctx, query, id, email, hashedPassword, "Test", "User", organizationID, true)
	if err != nil {
		return "", fmt.Errorf("create test user: %w", err)
	}

	return id, nil
}

// CreateTestAgent creates a test agent in the database.
func CreateTestAgent(ctx context.Context, db *pgxpool.Pool, organizationID string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO agents (id, name, version, os, architecture, hostname, organization_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := db.Exec(ctx, query, id, "test-agent", "1.0.0", "linux", "x86_64", "test-host", organizationID, "online")
	if err != nil {
		return "", fmt.Errorf("create test agent: %w", err)
	}

	return id, nil
}

// CreateTestPrinter creates a test printer in the database.
func CreateTestPrinter(ctx context.Context, db *pgxpool.Pool, agentID string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO printers (id, name, agent_id, status, capabilities)
		VALUES ($1, $2, $3, $4, $5)
	`

	capabilities := `{"color": true, "duplex": true, "media": ["a4", "letter"]}`

	_, err := db.Exec(ctx, query, id, "Test Printer", agentID, "online", capabilities)
	if err != nil {
		return "", fmt.Errorf("create test printer: %w", err)
	}

	return id, nil
}

// CreateTestDocument creates a test document in the database.
func CreateTestDocument(ctx context.Context, db *pgxpool.Pool, userEmail string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO documents (id, name, content_type, size, checksum, storage_path, user_email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.Exec(ctx, query, id, "test-document.pdf", "application/pdf", 1024, "test-checksum", "/test/path.pdf", userEmail)
	if err != nil {
		return "", fmt.Errorf("create test document: %w", err)
	}

	return id, nil
}

// CreateTestPrintJob creates a test print job in the database.
func CreateTestPrintJob(ctx context.Context, db *pgxpool.Pool, documentID, printerID, userEmail string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO print_jobs (id, document_id, printer_id, user_name, user_email, title,
			copies, color_mode, duplex, media_type, quality, status, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	now := time.Now()

	_, err := db.Exec(ctx, query,
		id, documentID, printerID, "Test User", userEmail, "Test Job",
		1, "monochrome", false, "a4", "normal", "queued", 5, now, now)

	if err != nil {
		return "", fmt.Errorf("create test print job: %w", err)
	}

	return id, nil
}

// CreateTestJobAssignment creates a test job assignment in the database.
func CreateTestJobAssignment(ctx context.Context, db *pgxpool.Pool, jobID, agentID string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO job_assignments (id, job_id, agent_id, status, assigned_at, created_at, updated_at, last_heartbeat)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()

	_, err := db.Exec(ctx, query, id, jobID, agentID, "assigned", now, now, now, now)
	if err != nil {
		return "", fmt.Errorf("create test job assignment: %w", err)
	}

	return id, nil
}

// CreateTestJobHistory creates a test job history entry in the database.
func CreateTestJobHistory(ctx context.Context, db *pgxpool.Pool, jobID string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO job_history (id, job_id, status, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()

	_, err := db.Exec(ctx, query, id, jobID, "queued", "Job created", now)
	if err != nil {
		return "", fmt.Errorf("create test job history: %w", err)
	}

	return id, nil
}

// CreateFullTestSetup creates a complete test data setup with all entities.
// Returns IDs of the created entities in order: organizationID, userID, agentID, printerID, documentID, jobID.
func CreateFullTestSetup(ctx context.Context, db *pgxpool.Pool) (string, string, string, string, string, string, error) {
	// Create organization
	orgID, err := CreateTestOrganization(ctx, db)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("create organization: %w", err)
	}

	// Create user
	userID, err := CreateTestUser(ctx, db, orgID)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("create user: %w", err)
	}

	// Get user email for subsequent calls
	var userEmail string
	err = db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&userEmail)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("get user email: %w", err)
	}

	// Create agent
	agentID, err := CreateTestAgent(ctx, db, orgID)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("create agent: %w", err)
	}

	// Create printer
	printerID, err := CreateTestPrinter(ctx, db, agentID)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("create printer: %w", err)
	}

	// Create document
	documentID, err := CreateTestDocument(ctx, db, userEmail)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("create document: %w", err)
	}

	// Create print job
	jobID, err := CreateTestPrintJob(ctx, db, documentID, printerID, userEmail)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("create print job: %w", err)
	}

	return orgID, userID, agentID, printerID, documentID, jobID, nil
}

// CleanupTestData removes all test data from the database.
// This is faster than recreating the container for each test.
func CleanupTestData(ctx context.Context, db *pgxpool.Pool) error {
	return TruncateAllTables(ctx, db)
}

// CleanupTestDataByUser removes test data for a specific user email.
func CleanupTestDataByUser(ctx context.Context, db *pgxpool.Pool, userEmail string) error {
	// Delete job history first (due to foreign key constraints)
	_, err := db.Exec(ctx, `
		DELETE FROM job_history
		WHERE job_id IN (SELECT id FROM print_jobs WHERE user_email = $1)
	`, userEmail)
	if err != nil {
		return fmt.Errorf("delete job history: %w", err)
	}

	// Delete job assignments
	_, err = db.Exec(ctx, `
		DELETE FROM job_assignments
		WHERE job_id IN (SELECT id FROM print_jobs WHERE user_email = $1)
	`, userEmail)
	if err != nil {
		return fmt.Errorf("delete job assignments: %w", err)
	}

	// Delete print jobs
	_, err = db.Exec(ctx, "DELETE FROM print_jobs WHERE user_email = $1", userEmail)
	if err != nil {
		return fmt.Errorf("delete print jobs: %w", err)
	}

	// Delete documents
	_, err = db.Exec(ctx, "DELETE FROM documents WHERE user_email = $1", userEmail)
	if err != nil {
		return fmt.Errorf("delete documents: %w", err)
	}

	return nil
}

// TestFixture holds IDs of commonly used test entities.
type TestFixture struct {
	OrganizationID string
	UserID         string
	UserEmail      string
	AgentID        string
	PrinterID      string
	DocumentID     string
	JobID          string
	AssignmentID   string
}

// SetupTestFixture creates a complete test fixture with all entities.
// The fixture can be used across multiple tests for consistent test data.
func SetupTestFixture(ctx context.Context, db *pgxpool.Pool) (*TestFixture, error) {
	orgID, userID, agentID, printerID, documentID, jobID, err := CreateFullTestSetup(ctx, db)
	if err != nil {
		return nil, err
	}

	// Get user email
	var userEmail string
	err = db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&userEmail)
	if err != nil {
		return nil, fmt.Errorf("get user email: %w", err)
	}

	// Create an assignment
	assignmentID, err := CreateTestJobAssignment(ctx, db, jobID, agentID)
	if err != nil {
		return nil, fmt.Errorf("create assignment: %w", err)
	}

	return &TestFixture{
		OrganizationID: orgID,
		UserID:         userID,
		UserEmail:      userEmail,
		AgentID:        agentID,
		PrinterID:      printerID,
		DocumentID:     documentID,
		JobID:          jobID,
		AssignmentID:   assignmentID,
	}, nil
}

// ValidUUID returns a valid UUID string for use in tests.
// This is useful when you need a valid UUID format but don't need a real database record.
func ValidUUID() string {
	return uuid.New().String()
}

// ValidPolicyID returns a valid UUID string formatted as a policy ID.
// This is an alias for ValidUUID for semantic clarity in tests.
func ValidPolicyID() string {
	return ValidUUID()
}

// ValidUserID returns a valid UUID string formatted as a user ID.
func ValidUserID() string {
	return ValidUUID()
}

// ValidOrganizationID returns a valid UUID string formatted as an organization ID.
func ValidOrganizationID() string {
	return ValidUUID()
}

// ValidPrinterID returns a valid UUID string formatted as a printer ID.
func ValidPrinterID() string {
	return ValidUUID();
}
