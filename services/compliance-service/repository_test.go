// Package main provides comprehensive tests for the compliance service repository layer.
package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openprint/openprint/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRepositoryTest creates a test database and repository for testing.
func setupRepositoryTest(t *testing.T) (*testutil.TestDB, *Repository) {
	t.Helper()

	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available - run with test tag")
	}

	// Clean up before each test
	ctx := context.Background()
	if err := testutil.TruncateAllTables(ctx, testDB.Pool); err != nil {
		// If truncate fails, the pool might be closed - try to skip gracefully
		t.Logf("Failed to truncate tables (pool may be closed): %v", err)
		t.Skip("Database pool no longer available")
	}

	repo := NewRepository(testDB.Pool)
	// Don't call Cleanup here - it's handled by TestMain
	return testDB, repo
}

// TestRepository_GetControl tests retrieving a control by ID.
func TestRepository_GetControl(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a test control
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err, "Failed to create test control")

	// Test getting the control
	control, err := repo.GetControl(ctx, controlID)
	require.NoError(t, err, "Failed to get control")
	require.NotNil(t, control, "Control should not be nil")

	// Verify control fields
	assert.Equal(t, controlID, control.ID)
	assert.Equal(t, "fedramp", string(control.Framework))
	assert.Equal(t, "compliant", string(control.Status))
	assert.NotEmpty(t, control.Title)
	assert.NotEmpty(t, control.Family)
}

// TestRepository_GetControl_NotFound tests getting a non-existent control.
func TestRepository_GetControl_NotFound(t *testing.T) {
	_, repo := setupRepositoryTest(t)

	ctx := context.Background()

	// Try to get a non-existent control
	_, err := repo.GetControl(ctx, uuid.New().String())
	assert.Error(t, err, "Expected error for non-existent control")
}

// TestRepository_ListControls tests listing controls with filters.
func TestRepository_ListControls(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create test controls for different frameworks
	_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	_, err = testutil.CreateTestComplianceControl(ctx, testDB.Pool, "hipaa")
	require.NoError(t, err)

	_, err = testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Test listing all controls
	controls, total, err := repo.ListControls(ctx, "", "", 100, 0)
	require.NoError(t, err, "Failed to list controls")
	assert.GreaterOrEqual(t, total, 3, "Should have at least 3 controls")
	assert.Len(t, controls, total, "Controls count should match total")

	// Test filtering by framework
	fedrampControls, fedrampTotal, err := repo.ListControls(ctx, FrameworkFedRAMP, "", 100, 0)
	require.NoError(t, err, "Failed to list fedramp controls")
	assert.Equal(t, 2, fedrampTotal, "Should have 2 fedramp controls")
	assert.Len(t, fedrampControls, 2, "FedRAMP controls count should match total")

	// Test filtering by status
	_, compliantTotal, err := repo.ListControls(ctx, "", StatusCompliant, 100, 0)
	require.NoError(t, err, "Failed to list compliant controls")
	assert.GreaterOrEqual(t, compliantTotal, 3, "Should have at least 3 compliant controls")
}

// TestRepository_ListControls_Pagination tests pagination for control listing.
func TestRepository_ListControls_Pagination(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create 5 test controls
	for i := 0; i < 5; i++ {
		_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
		require.NoError(t, err)
	}

	// Test pagination - first page
	page1, total, err := repo.ListControls(ctx, "", "", 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2, "First page should have 2 controls")
	assert.Equal(t, 5, total, "Total should be 5")

	// Test pagination - second page
	page2, total, err := repo.ListControls(ctx, "", "", 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2, "Second page should have 2 controls")
	assert.Equal(t, 5, total, "Total should still be 5")

	// Test pagination - last page (partial)
	page3, total, err := repo.ListControls(ctx, "", "", 2, 4)
	require.NoError(t, err)
	assert.Len(t, page3, 1, "Last page should have 1 control")
	assert.Equal(t, 5, total, "Total should still be 5")
}

// TestRepository_UpdateControlStatus tests updating control status.
func TestRepository_UpdateControlStatus(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a test control
	controlID, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Update the control status
	lastAssessed := time.Now()
	nextReview := time.Now().AddDate(0, 0, 60)
	err = repo.UpdateControlStatus(ctx, controlID, StatusNonCompliant, lastAssessed, nextReview)
	require.NoError(t, err, "Failed to update control status")

	// Verify the update
	control, err := repo.GetControl(ctx, controlID)
	require.NoError(t, err, "Failed to get updated control")
	assert.Equal(t, StatusNonCompliant, control.Status)
	assert.NotNil(t, control.LastAssessed)
	assert.NotNil(t, control.NextReview)
}

// TestRepository_UpdateControlStatus_NotFound tests updating a non-existent control.
func TestRepository_UpdateControlStatus_NotFound(t *testing.T) {
	_, repo := setupRepositoryTest(t)

	ctx := context.Background()

	// Try to update a non-existent control
	err := repo.UpdateControlStatus(ctx, uuid.New().String(), StatusCompliant, time.Now(), time.Now())
	assert.Error(t, err, "Expected error for non-existent control")
}

// TestRepository_CreateAuditEvent tests creating an audit event.
func TestRepository_CreateAuditEvent(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create test user and organization
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	// Create an audit event
	event := &AuditEvent{
		UserID:       userID,
		UserName:     "Test User",
		ResourceID:   uuid.New().String(),
		ResourceType: "test_resource",
		Action:       "test_action",
		Outcome:      "success",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
		EventType:    "test_event",
		Category:     "test_category",
		Metadata:     map[string]string{"key": "value"},
	}

	err = repo.CreateAuditEvent(ctx, event)
	require.NoError(t, err, "Failed to create audit event")

	// Verify the event was created
	assert.NotEmpty(t, event.ID, "Event ID should be set")
	assert.False(t, event.Timestamp.IsZero(), "Timestamp should be set")
	assert.NotNil(t, event.RetentionDate, "Retention date should be set")

	// Verify retention date is approximately 7 years in the future
	expectedRetention := time.Now().AddDate(7, 0, 0)
	assert.WithinDuration(t, expectedRetention, *event.RetentionDate, time.Minute)
}

// TestRepository_QueryAuditEvents tests querying audit events with filters.
func TestRepository_QueryAuditEvents(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create test organization and user
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	// Create multiple audit events
	for i := 0; i < 5; i++ {
		event := &AuditEvent{
			UserID:       userID,
			UserName:     "Test User",
			ResourceID:   uuid.New().String(),
			ResourceType: "test_resource",
			Action:       "test_action",
			Outcome:      "success",
			IPAddress:    "127.0.0.1",
			EventType:    "test_event",
			Category:     "test_category",
		}
		err := repo.CreateAuditEvent(ctx, event)
		require.NoError(t, err)
	}

	// Test querying all events
	filter := AuditFilter{Limit: 100, Offset: 0}
	events, total, err := repo.QueryAuditEvents(ctx, filter)
	require.NoError(t, err, "Failed to query audit events")
	assert.GreaterOrEqual(t, total, 5, "Should have at least 5 events")
	assert.Len(t, events, total, "Events count should match total")

	// Test filtering by user ID
	filter = AuditFilter{UserID: userID, Limit: 100, Offset: 0}
	events, total, err = repo.QueryAuditEvents(ctx, filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 5, "Should have at least 5 events for user")
	for _, event := range events {
		assert.Equal(t, userID, event.UserID, "All events should belong to the user")
	}

	// Test pagination
	filter = AuditFilter{Limit: 2, Offset: 0}
	events, total, err = repo.QueryAuditEvents(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2, "Should return 2 events")
	assert.GreaterOrEqual(t, total, 5, "Total should be at least 5")
}

// TestRepository_QueryAuditEvents_TimeRange tests querying audit events with time range filter.
func TestRepository_QueryAuditEvents_TimeRange(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create test organization and user
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	// Create an audit event
	event := &AuditEvent{
		UserID:       userID,
		UserName:     "Test User",
		ResourceID:   uuid.New().String(),
		ResourceType: "test_resource",
		Action:       "test_action",
		Outcome:      "success",
		IPAddress:    "127.0.0.1",
		EventType:    "test_event",
		Category:     "test_category",
	}
	err = repo.CreateAuditEvent(ctx, event)
	require.NoError(t, err)

	// Query with time range that includes the event
	now := time.Now()
	filter := AuditFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
		Limit:     100,
		Offset:    0,
	}
	events, total, err := repo.QueryAuditEvents(ctx, filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1, "Should have at least 1 event in time range")
	assert.NotEmpty(t, events, "Events should not be empty")

	// Query with time range that excludes the event (past)
	filter = AuditFilter{
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now.Add(-12 * time.Hour),
		Limit:     100,
		Offset:    0,
	}
	events, total, err = repo.QueryAuditEvents(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 0, total, "Should have no events in past time range")
	assert.Empty(t, events, "Events should be empty")
}

// TestRepository_RecordDataBreach tests recording a data breach.
func TestRepository_RecordDataBreach(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a data breach
	breach := &DataBreach{
		Severity:          "high",
		AffectedRecords:   100,
		DataTypes:         []string{"email", "name", "ssn"},
		Description:       "Test data breach",
		ContainmentStatus: "identifying",
		NotificationSent:  false,
	}

	err := repo.RecordDataBreach(ctx, breach)
	require.NoError(t, err, "Failed to record data breach")

	// Verify the breach was recorded
	assert.NotEmpty(t, breach.ID, "Breach ID should be set")
	assert.False(t, breach.ReportedAt.IsZero(), "Reported at should be set")

	// Verify the breach exists in the database
	var count int
	err = testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM data_breaches WHERE id = $1", breach.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Breach should exist in database")
}

// TestRepository_RecordDataBreach_WithResolution tests recording a resolved data breach.
func TestRepository_RecordDataBreach_WithResolution(t *testing.T) {
	_, repo := setupRepositoryTest(t)

	ctx := context.Background()

	// Create a resolved data breach
	resolvedAt := time.Now()
	breach := &DataBreach{
		Severity:          "low",
		AffectedRecords:   5,
		DataTypes:         []string{"email"},
		Description:       "Minor data breach",
		ContainmentStatus: "resolved",
		NotificationSent:  true,
		ResolvedAt:        &resolvedAt,
		LessonsLearned:    "Test lessons learned",
	}

	err := repo.RecordDataBreach(ctx, breach)
	require.NoError(t, err, "Failed to record resolved data breach")

	// Verify the breach was recorded with all fields
	assert.NotEmpty(t, breach.ID, "Breach ID should be set")
	assert.NotNil(t, breach.ResolvedAt, "Resolved at should be set")
	assert.NotEmpty(t, breach.LessonsLearned, "Lessons learned should be set")
}

// TestRepository_GetPendingReviews tests getting pending reviews.
func TestRepository_GetPendingReviews(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a control with upcoming review (within 30 days)
	controlID1, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Update the control to have a review due soon
	nextReview := time.Now().AddDate(0, 0, 15)
	_, err = testDB.Pool.Exec(ctx, `
		UPDATE compliance_controls
		SET next_review = $1
		WHERE id = $2
	`, nextReview, controlID1)
	require.NoError(t, err)

	// Create a control with review overdue (negative days)
	controlID2, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "hipaa")
	require.NoError(t, err)

	pastReview := time.Now().AddDate(0, 0, -5)
	_, err = testDB.Pool.Exec(ctx, `
		UPDATE compliance_controls
		SET next_review = $1
		WHERE id = $2
	`, pastReview, controlID2)
	require.NoError(t, err)

	// Create a control with review far in the future
	_, err = testutil.CreateTestComplianceControl(ctx, testDB.Pool, "gdpr")
	require.NoError(t, err)

	// Get pending reviews within 30 days
	pending, err := repo.GetPendingReviews(ctx, 30*24*time.Hour)
	require.NoError(t, err, "Failed to get pending reviews")
	assert.GreaterOrEqual(t, len(pending), 2, "Should have at least 2 pending reviews")

	// Verify that the pending controls are the ones we expect
	controlIDs := make([]string, len(pending))
	for i, control := range pending {
		controlIDs[i] = control.ID
	}
	assert.Contains(t, controlIDs, controlID1, "Should include control with upcoming review")
	assert.Contains(t, controlIDs, controlID2, "Should include control with overdue review")
}

// TestRepository_GetPendingReviews_Empty tests getting pending reviews when none are due.
func TestRepository_GetPendingReviews_Empty(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a control with review far in the future
	_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, "fedramp")
	require.NoError(t, err)

	// Update all controls to have reviews far in the future
	futureReview := time.Now().AddDate(0, 0, 90)
	_, err = testDB.Pool.Exec(ctx, `
		UPDATE compliance_controls
		SET next_review = $1
	`, futureReview)
	require.NoError(t, err)

	// Get pending reviews within 30 days
	pending, err := repo.GetPendingReviews(ctx, 30*24*time.Hour)
	require.NoError(t, err, "Failed to get pending reviews")
	assert.Empty(t, pending, "Should have no pending reviews")
}

// TestRepository_Integration_ControlLifecycle tests the full lifecycle of a control.
func TestRepository_Integration_ControlLifecycle(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a control via database
	controlID := uuid.New()
	query := `
		INSERT INTO compliance_controls (id, framework, family, title, description, implementation, status, next_review, responsible_team, risk_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := testDB.Pool.Exec(ctx, query, controlID, "fedramp", "Access Control",
		"Test Control", "Test description", "Test implementation", "pending", time.Now().AddDate(0, 0, 30), "security-team", "medium")
	require.NoError(t, err)

	// Read the control
	control, err := repo.GetControl(ctx, controlID.String())
	require.NoError(t, err)
	assert.Equal(t, controlID.String(), control.ID)
	assert.Equal(t, "pending", string(control.Status))

	// Update the control status
	lastAssessed := time.Now()
	nextReview := time.Now().AddDate(0, 0, 60)
	err = repo.UpdateControlStatus(ctx, controlID.String(), StatusCompliant, lastAssessed, nextReview)
	require.NoError(t, err)

	// Verify the update
	updatedControl, err := repo.GetControl(ctx, controlID.String())
	require.NoError(t, err)
	assert.Equal(t, "compliant", string(updatedControl.Status))
	assert.NotNil(t, updatedControl.LastAssessed)

	// List controls and verify our control is there
	controls, total, err := repo.ListControls(ctx, FrameworkFedRAMP, StatusCompliant, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)

	found := false
	for _, c := range controls {
		if c.ID == controlID.String() {
			found = true
			break
		}
	}
	assert.True(t, found, "Control should be in the list")
}

// TestRepository_MultipleFrameworks tests controls across multiple frameworks.
func TestRepository_MultipleFrameworks(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	frameworks := []string{"fedramp", "hipaa", "gdpr", "soc2"}
	createdCount := make(map[string]int)

	// Create controls for each framework
	for _, fw := range frameworks {
		for i := 0; i < 3; i++ {
			_, err := testutil.CreateTestComplianceControl(ctx, testDB.Pool, fw)
			require.NoError(t, err)
			createdCount[fw]++
		}
	}

	// Verify each framework has the correct count
	for _, fw := range frameworks {
		controls, total, err := repo.ListControls(ctx, ComplianceFramework(fw), "", 100, 0)
		require.NoError(t, err, "Failed to list controls for framework %s", fw)
		assert.Equal(t, createdCount[fw], total, "Framework %s should have %d controls", fw, createdCount[fw])
		assert.Len(t, controls, total, "Controls count should match total for framework %s", fw)
	}

	// Verify total count across all frameworks
	allControls, total, err := repo.ListControls(ctx, "", "", 100, 0)
	require.NoError(t, err)
	expectedTotal := 0
	for _, count := range createdCount {
		expectedTotal += count
	}
	assert.GreaterOrEqual(t, total, expectedTotal, "Total controls should be at least %d", expectedTotal)
	assert.Len(t, allControls, total, "All controls count should match total")
}

// TestRepository_DataTypesJSON tests that data types are properly stored as JSONB.
func TestRepository_DataTypesJSON(t *testing.T) {
	testDB, repo := setupRepositoryTest(t)
	// Note: Database cleanup is handled by TestMain

	ctx := context.Background()

	// Create a breach with specific data types
	breach := &DataBreach{
		Severity:          "critical",
		AffectedRecords:   1000,
		DataTypes:         []string{"email", "name", "ssn", "address", "phone"},
		Description:       "Major data breach",
		ContainmentStatus: "investigating",
	}

	err := repo.RecordDataBreach(ctx, breach)
	require.NoError(t, err)

	// Verify data types were stored correctly
	var storedTypes []string
	err = testDB.Pool.QueryRow(ctx, `
		SELECT data_types FROM data_breaches WHERE id = $1
	`, breach.ID).Scan(&storedTypes)
	require.NoError(t, err)

	assert.Len(t, storedTypes, 5, "Should have 5 data types")
	assert.Contains(t, storedTypes, "email")
	assert.Contains(t, storedTypes, "ssn")
}
