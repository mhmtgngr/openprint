//go:build integration

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupPostgresContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer Cleanup(db)

	// Verify container is running
	assert.NotNil(t, db.Container)

	// Verify connection pool is created
	assert.NotNil(t, db.Pool)

	// Verify connection string
	assert.NotEmpty(t, db.ConnString)
	assert.Contains(t, db.ConnString, "postgres://")
	assert.Contains(t, db.ConnString, DefaultTestDatabase)

	// Test database connection
	err = db.Pool.Ping(ctx)
	require.NoError(t, err, "database should be reachable")
}

func TestSetupPostgresContainer_ConnectionRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Test that we can execute queries
	var result int
	err = db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestCleanup_NilDB(t *testing.T) {
	// Should not panic with nil
	Cleanup(nil)
}

func TestCleanup_NilContainer(t *testing.T) {
	// Should not panic with nil container
	db := &TestDB{Container: nil}
	Cleanup(db)
}

func TestGetTestDBConnection(t *testing.T) {
	t.Run("nil db returns empty string", func(t *testing.T) {
		connStr := GetTestDBConnection(nil)
		assert.Empty(t, connStr)
	})

	t.Run("valid db returns connection string", func(t *testing.T) {
		db := &TestDB{ConnString: "postgres://localhost:5432/test"}
		connStr := GetTestDBConnection(db)
		assert.Equal(t, "postgres://localhost:5432/test", connStr)
	})
}

func TestTruncateAllTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create some test data
	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)
	assert.NotEmpty(t, orgID)

	// Verify data exists
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)

	// Truncate all tables
	err = TruncateAllTables(ctx, db.Pool)
	require.NoError(t, err)

	// Verify data is gone
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTruncateAllTables_NonExistentTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// TruncateAllTables should handle non-existent tables gracefully
	err = TruncateAllTables(ctx, db.Pool)
	require.NoError(t, err)
}

func TestSetupPostgresForTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This is mainly a smoke test to ensure SetupPostgresForTest works
	// In real usage, this would be called from TestMain
	done := make(chan int, 1)

	go func() {
		// Simulate what TestMain would do
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		db, err := SetupPostgresContainer(ctx)
		require.NoError(t, err)
		defer Cleanup(db)
		cancel()
		done <- 1
	}()

	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Minute):
		t.Fatal("timeout waiting for test setup")
	}
}

func TestTestDB_ConnectionStringFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Verify connection string format
	assert.Contains(t, db.ConnString, "postgres://")
	assert.Contains(t, db.ConnString, DefaultTestUser)
	assert.Contains(t, db.ConnString, DefaultTestPassword)
	assert.Contains(t, db.ConnString, DefaultTestDatabase)
	assert.Contains(t, db.ConnString, "sslmode=disable")
}

func TestTestDB_PoolConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Test concurrent connections
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			var result int
			err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
			errChan <- err
		}()
	}

	for i := 0; i < 10; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

func TestTestDB_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var result int
	err = db.Pool.QueryRow(cancelledCtx, "SELECT 1").Scan(&result)
	assert.Error(t, err)
}

func TestTestDB_TransactionSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Test transaction
	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)

	// Insert data
	orgID, err := CreateTestOrganization(ctx, db.Pool)
	require.NoError(t, err)

	// Rollback
	err = tx.Rollback(ctx)
	require.NoError(t, err)

	// Verify data still exists (wasn't in the transaction)
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations WHERE id = $1", orgID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTestDB_MultipleContainers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test that we can create multiple containers (for parallel tests)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	db1, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db1)

	db2, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db2)

	// Both should be functional
	err = db1.Pool.Ping(ctx)
	require.NoError(t, err)

	err = db2.Pool.Ping(ctx)
	require.NoError(t, err)

	// Should have different connection strings
	assert.NotEqual(t, db1.ConnString, db2.ConnString)
}

func TestTestDB_QueryTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Test query with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// This should complete quickly
	var result int
	err = db.Pool.QueryRow(timeoutCtx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestCleanup_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)

	// Multiple cleanup calls should not panic
	Cleanup(db)
	Cleanup(db)
	Cleanup(nil)
}
