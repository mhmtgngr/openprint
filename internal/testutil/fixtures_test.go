//go:build integration

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTestOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)
	assert.NotEmpty(t, orgID)

	// Verify organization was created
	var name string
	err = db.Pool.QueryRow(ctx, "SELECT name FROM organizations WHERE id = $1", orgID).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Test Organization", name)
}

func TestCreateTestOrganization_Multiple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create multiple organizations
	orgIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		orgIDs[i], err = CreateTestOrganization(ctx, db.Pool)
		require.NoError(t, err)
	}

	// Verify all have unique IDs
	uniqueIDs := make(map[string]bool)
	for _, id := range orgIDs {
		uniqueIDs[id] = true
	}
	assert.Len(t, uniqueIDs, 5)

	// Verify count in database
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestCreateTestUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create organization first
	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// Create user
	userID, err := CreateTestUser(ctx, db.Pool, orgID)
	require.NoError(t, err)
	assert.NotEmpty(t, userID)

	// Verify user was created
	var email, firstName, lastName, retrievedOrgID string
	err = db.Pool.QueryRow(ctx,
		"SELECT email, first_name, last_name, organization_id FROM users WHERE id = $1",
		userID).Scan(&email, &firstName, &lastName, &retrievedOrgID)
	require.NoError(t, err)
	assert.Contains(t, email, "@example.com")
	assert.Equal(t, "Test", firstName)
	assert.Equal(t, "User", lastName)
	assert.Equal(t, orgID, retrievedOrgID)
}

func TestCreateTestUser_NoOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Try to create user with invalid organization ID
	_, err = CreateTestUser(ctx, db.Pool, uuid.New().String())
	assert.Error(t, err)
}

func TestCreateTestAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create organization first
	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// Create agent
	agentID, err := CreateTestAgent(ctx, db.Pool, orgID)
	require.NoError(t, err)
	assert.NotEmpty(t, agentID)

	// Verify agent was created
	var name, version, status, retrievedOrgID string
	err = db.Pool.QueryRow(ctx,
		"SELECT name, version, status, organization_id FROM agents WHERE id = $1",
		agentID).Scan(&name, &version, &status, &retrievedOrgID)
	require.NoError(t, err)
	assert.Equal(t, "test-agent", name)
	assert.Equal(t, "1.0.0", version)
	assert.Equal(t, "online", status)
	assert.Equal(t, orgID, retrievedOrgID)
}

func TestCreateTestPrinter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create organization and agent
	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)
	agentID, err := CreateTestAgent(ctx, db.Pool, orgID)
	require.NoError(t, err)

	// Create printer
	printerID, err := CreateTestPrinter(ctx, db.Pool, agentID)
	require.NoError(t, err)
	assert.NotEmpty(t, printerID)

	// Verify printer was created
	var name, status, retrievedAgentID, capabilities string
	err = db.Pool.QueryRow(ctx,
		"SELECT name, status, agent_id, capabilities FROM printers WHERE id = $1",
		printerID).Scan(&name, &status, &retrievedAgentID, &capabilities)
	require.NoError(t, err)
	assert.Equal(t, "Test Printer", name)
	assert.Equal(t, "online", status)
	assert.Equal(t, agentID, retrievedAgentID)
	assert.Contains(t, capabilities, "color")
}

func TestCreateTestDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create document
	userEmail := "test@example.com"
	documentID, err := CreateTestDocument(ctx, db.Pool, userEmail)
	require.NoError(t, err)
	assert.NotEmpty(t, documentID)

	// Verify document was created
	var name, contentType, retrievedUserEmail string
	var size int
	err = db.Pool.QueryRow(ctx,
		"SELECT name, content_type, size, user_email FROM documents WHERE id = $1",
		documentID).Scan(&name, &contentType, &size, &retrievedUserEmail)
	require.NoError(t, err)
	assert.Equal(t, "test-document.pdf", name)
	assert.Equal(t, "application/pdf", contentType)
	assert.Equal(t, 1024, size)
	assert.Equal(t, userEmail, retrievedUserEmail)
}

func TestCreateTestPrintJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create full setup
	_, userID, _, printerID, documentID, _, err := CreateFullTestSetup(ctx, db.Pool)
	require.NoError(t, err)

	// Get user email
	var userEmail string
	err = db.Pool.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&userEmail)
	require.NoError(t, err)

	// Create another print job
	jobID, err := CreateTestPrintJob(ctx, db.Pool, documentID, printerID, userEmail)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Verify print job was created
	var title, status, retrievedDocumentID, retrievedPrinterID string
	var copies int
	err = db.Pool.QueryRow(ctx,
		"SELECT title, copies, status, document_id, printer_id FROM print_jobs WHERE id = $1",
		jobID).Scan(&title, &copies, &status, &retrievedDocumentID, &retrievedPrinterID)
	require.NoError(t, err)
	assert.Equal(t, "Test Job", title)
	assert.Equal(t, 1, copies)
	assert.Equal(t, "queued", status)
	assert.Equal(t, documentID, retrievedDocumentID)
	assert.Equal(t, printerID, retrievedPrinterID)

	// Cleanup
	_ = CleanupTestDataByUser(ctx, db.Pool, userEmail)
}

func TestCreateTestJobAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create full setup
	fixture, err := SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)

	// Create another job assignment
	assignmentID, err := CreateTestJobAssignment(ctx, db.Pool, fixture.JobID, fixture.AgentID)
	require.NoError(t, err)
	assert.NotEmpty(t, assignmentID)

	// Verify assignment was created
	var status, retrievedJobID, retrievedAgentID string
	err = db.Pool.QueryRow(ctx,
		"SELECT status, job_id, agent_id FROM job_assignments WHERE id = $1",
		assignmentID).Scan(&status, &retrievedJobID, &retrievedAgentID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", status)
	assert.Equal(t, fixture.JobID, retrievedJobID)
	assert.Equal(t, fixture.AgentID, retrievedAgentID)
}

func TestCreateTestJobHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create full setup
	fixture, err := SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)

	// Create another job history entry
	historyID, err := CreateTestJobHistory(ctx, db.Pool, fixture.JobID)
	require.NoError(t, err)
	assert.NotEmpty(t, historyID)

	// Verify history was created
	var status, message, retrievedJobID string
	err = db.Pool.QueryRow(ctx,
		"SELECT status, message, job_id FROM job_history WHERE id = $1",
		historyID).Scan(&status, &message, &retrievedJobID)
	require.NoError(t, err)
	assert.Equal(t, "queued", status)
	assert.Equal(t, "Job created", message)
	assert.Equal(t, fixture.JobID, retrievedJobID)
}

func TestCreateFullTestSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create full test setup
	orgID, userID, agentID, printerID, documentID, jobID, err := CreateFullTestSetup(ctx, db.Pool)
	require.NoError(t, err)

	// Verify all IDs are non-empty
	assert.NotEmpty(t, orgID)
	assert.NotEmpty(t, userID)
	assert.NotEmpty(t, agentID)
	assert.NotEmpty(t, printerID)
	assert.NotEmpty(t, documentID)
	assert.NotEmpty(t, jobID)

	// Verify organization exists
	var orgName string
	err = db.Pool.QueryRow(ctx, "SELECT name FROM organizations WHERE id = $1", orgID).Scan(&orgName)
	require.NoError(t, err)
	assert.Equal(t, "Test Organization", orgName)

	// Verify user exists and is linked to organization
	var userOrgID string
	err = db.Pool.QueryRow(ctx, "SELECT organization_id FROM users WHERE id = $1", userID).Scan(&userOrgID)
	require.NoError(t, err)
	assert.Equal(t, orgID, userOrgID)

	// Verify agent is linked to organization
	var agentOrgID string
	err = db.Pool.QueryRow(ctx, "SELECT organization_id FROM agents WHERE id = $1", agentID).Scan(&agentOrgID)
	require.NoError(t, err)
	assert.Equal(t, orgID, agentOrgID)

	// Verify printer is linked to agent
	var printerAgentID string
	err = db.Pool.QueryRow(ctx, "SELECT agent_id FROM printers WHERE id = $1", printerID).Scan(&printerAgentID)
	require.NoError(t, err)
	assert.Equal(t, agentID, printerAgentID)

	// Verify print job is linked to document and printer
	var jobDocID, jobPrinterID string
	err = db.Pool.QueryRow(ctx, "SELECT document_id, printer_id FROM print_jobs WHERE id = $1", jobID).Scan(&jobDocID, &jobPrinterID)
	require.NoError(t, err)
	assert.Equal(t, documentID, jobDocID)
	assert.Equal(t, printerID, jobPrinterID)
}

func TestSetupTestFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create test fixture
	fixture, err := SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)
	require.NotNil(t, fixture)

	// Verify all fields are populated
	assert.NotEmpty(t, fixture.OrganizationID)
	assert.NotEmpty(t, fixture.UserID)
	assert.NotEmpty(t, fixture.UserEmail)
	assert.NotEmpty(t, fixture.AgentID)
	assert.NotEmpty(t, fixture.PrinterID)
	assert.NotEmpty(t, fixture.DocumentID)
	assert.NotEmpty(t, fixture.JobID)
	assert.NotEmpty(t, fixture.AssignmentID)

	// Verify email format
	assert.Contains(t, fixture.UserEmail, "@example.com")

	// Verify assignment links job and agent
	var assignmentJobID, assignmentAgentID string
	err = db.Pool.QueryRow(ctx,
		"SELECT job_id, agent_id FROM job_assignments WHERE id = $1",
		fixture.AssignmentID).Scan(&assignmentJobID, &assignmentAgentID)
	require.NoError(t, err)
	assert.Equal(t, fixture.JobID, assignmentJobID)
	assert.Equal(t, fixture.AgentID, assignmentAgentID)
}

func TestCleanupTestData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create test data
	_, err = SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)

	// Verify data exists
	var orgCount int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&orgCount)
	require.NoError(t, err)
	assert.Greater(t, orgCount, 0)

	// Cleanup
	err = CleanupTestData(ctx, db.Pool)
	require.NoError(t, err)

	// Verify all tables are empty
	tables := []string{
		"organizations", "users", "agents", "printers", "documents",
		"print_jobs", "job_assignments", "job_history",
	}
	for _, table := range tables {
		var count int
		err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "table %s should be empty", table)
	}
}

func TestCleanupTestDataByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create two users with data
	fixture1, err := SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)

	fixture2, err := SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)

	// Verify both users have data
	var jobCount1 int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM print_jobs WHERE user_email = $1", fixture1.UserEmail).Scan(&jobCount1)
	require.NoError(t, err)
	assert.Greater(t, jobCount1, 0)

	var jobCount2 int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM print_jobs WHERE user_email = $1", fixture2.UserEmail).Scan(&jobCount2)
	require.NoError(t, err)
	assert.Greater(t, jobCount2, 0)

	// Cleanup first user's data
	err = CleanupTestDataByUser(ctx, db.Pool, fixture1.UserEmail)
	require.NoError(t, err)

	// Verify first user's data is gone
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM print_jobs WHERE user_email = $1", fixture1.UserEmail).Scan(&jobCount1)
	require.NoError(t, err)
	assert.Equal(t, 0, jobCount1)

	// Verify second user's data still exists
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM print_jobs WHERE user_email = $1", fixture2.UserEmail).Scan(&jobCount2)
	require.NoError(t, err)
	assert.Greater(t, jobCount2, 0)
}

func TestCleanupTestDataByUser_NoData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Cleanup non-existent user should not error
	err = CleanupTestDataByUser(ctx, db.Pool, "nonexistent@example.com")
	require.NoError(t, err)
}

func TestTestFixture_MultipleFixtures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create multiple fixtures
	fixtures := make([]*TestFixture, 3)
	for i := 0; i < 3; i++ {
		fixtures[i], err = SetupTestFixture(ctx, db.Pool)
		require.NoError(t, err)
	}

	// Verify all have unique IDs
	orgIDs := make(map[string]bool)
	for _, f := range fixtures {
		orgIDs[f.OrganizationID] = true
	}
	assert.Len(t, orgIDs, 3)
}

func TestCreateTestOrganization_AfterTruncate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create organization
	orgID1, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// Truncate
	err = TruncateAllTables(ctx, db.Pool)
	require.NoError(t, err)

	// Create another organization
	orgID2, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// IDs should be different (UUIDs are random)
	assert.NotEqual(t, orgID1, orgID2)
}

func TestCreateTestUser_EmailFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// Create multiple users and verify email format
	for i := 0; i < 5; i++ {
		userID, err := CreateTestUser(ctx, db.Pool, orgID)
		require.NoError(t, err)

		var email string
		err = db.Pool.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
		require.NoError(t, err)

		assert.Contains(t, email, "@example.com")
		assert.Regexp(t, `^test-[a-f0-9]+@example\.com$`, email)
	}
}

func TestTestFixture_IntegrityConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create fixture
	fixture, err := SetupTestFixture(ctx, db.Pool)
	require.NoError(t, err)

	// Try to create printer with non-existent agent (should fail)
	_, err = CreateTestPrinter(ctx, db.Pool, uuid.New().String())
	assert.Error(t, err)

	// Try to create print job with non-existent document (should fail)
	_, err = CreateTestPrintJob(ctx, db.Pool, uuid.New().String(), fixture.PrinterID, fixture.UserEmail)
	assert.Error(t, err)

	// Try to create assignment with non-existent job (should fail)
	_, err = CreateTestJobAssignment(ctx, db.Pool, uuid.New().String(), fixture.AgentID)
	assert.Error(t, err)
}

func TestCreateTestAgent_MultipleAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// Create multiple agents for the same organization
	agentIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		agentIDs[i], err = CreateTestAgent(ctx, db.Pool, orgID)
		require.NoError(t, err)
	}

	// Verify all are unique
	uniqueIDs := make(map[string]bool)
	for _, id := range agentIDs {
		uniqueIDs[id] = true
	}
	assert.Len(t, uniqueIDs, 3)

	// Verify count
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM agents WHERE organization_id = $1", orgID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestTestFixture_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create fixtures concurrently
	errChan := make(chan error, 5)
	fixtures := make([]*TestFixture, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			fixture, err := SetupTestFixture(ctx, db.Pool)
			fixtures[idx] = fixture
			errChan <- err
		}(i)
	}

	// Collect results
	for i := 0; i < 5; i++ {
		err := <-errChan
		require.NoError(t, err)
	}

	// Verify all fixtures are valid
	for _, f := range fixtures {
		assert.NotNil(t, f)
		assert.NotEmpty(t, f.OrganizationID)
	}
}
