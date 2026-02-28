// Package testutil provides reusable testing utilities for OpenPrint services.
// It uses testcontainers-go to create isolated PostgreSQL containers for testing,
// eliminating the need for external database dependencies.
package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// DefaultTestDatabase is the default database name to create in the test container.
	DefaultTestDatabase = "openprint_test"
	// DefaultTestUser is the default PostgreSQL user for test containers.
	DefaultTestUser = "testuser"
	// DefaultTestPassword is the default PostgreSQL password for test containers.
	DefaultTestPassword = "testpass"
)

// TestDB holds resources for a test database container.
type TestDB struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
	ConnString string
}

// SetupPostgresContainer creates and starts a PostgreSQL container for testing.
// It returns a TestDB struct containing the container, connection pool, and connection string.
// The container is automatically configured with:
// - A database named DefaultTestDatabase
// - A user with credentials from DefaultTestUser/DefaultTestPassword
// - The uuid-ossp extension enabled
//
// Usage in tests:
//
//	func TestMain(m *testing.M) {
//	    testDB, err := SetupPostgresContainer(context.Background())
//	    if err != nil {
//	        log.Fatalf("Failed to setup test database: %v", err)
//	    }
//	    defer Cleanup(testDB)
//	    os.Exit(m.Run())
//	}
func SetupPostgresContainer(ctx context.Context) (*TestDB, error) {
	// Get the project root directory to find migrations
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("find project root: %w", err)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations directory not found at %s", migrationsDir)
	}

	// Create PostgreSQL container request
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       DefaultTestDatabase,
			"POSTGRES_USER":     DefaultTestUser,
			"POSTGRES_PASSWORD": DefaultTestPassword,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("create postgres container: %w", err)
	}

	// Get the mapped port
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container port: %w", err)
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		DefaultTestUser, DefaultTestPassword, host, port.Port(), DefaultTestDatabase)

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	// Configure pool for testing
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.HealthCheckPeriod = 30 * time.Second
	poolConfig.MaxConnLifetime = time.Hour

	// Retry connection with backoff
	var pool *pgxpool.Pool
	var lastErr error
	for i := 0; i < 10; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			break
		}
		lastErr = err
		select {
		case <-ctx.Done():
			container.Terminate(ctx)
			return nil, ctx.Err()
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}

	if pool == nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("create connection pool after retries: %w", lastErr)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		container.Terminate(ctx)
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Run migrations
	if err := RunMigrations(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		container.Terminate(ctx)
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &TestDB{
		Container:   container,
		Pool:        pool,
		ConnString: connString,
	}, nil
}

// Cleanup terminates the test database container and closes the connection pool.
// It should be called in a defer statement after SetupPostgresContainer.
func Cleanup(db *TestDB) {
	if db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if db.Pool != nil {
		db.Pool.Close()
	}

	if db.Container != nil {
		if err := db.Container.Terminate(ctx); err != nil {
			log.Printf("Warning: failed to terminate test container: %v", err)
		}
	}
}

// GetTestDBConnection returns a connection string for the test database.
// This is useful for tests that need the connection string directly.
func GetTestDBConnection(db *TestDB) string {
	if db == nil {
		return ""
	}
	return db.ConnString
}

// TruncateAllTables truncates all tables in the test database.
// This is useful for cleaning up between tests without re-creating the container.
func TruncateAllTables(ctx context.Context, db *pgxpool.Pool) error {
	// Get all table names
	tables := []string{
		"job_assignments",
		"job_history",
		"print_jobs",
		"documents",
		"user_sessions",
		"audit_log",
		"printers",
		"agents",
		"users",
		"organizations",
		"api_keys",
		"webhooks",
		"invitations",
		"devices",
		"discovered_printers",
		"agent_events",
		"agent_certificates",
		"enrollment_tokens",
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Disable triggers for faster truncation
	if _, err := tx.Exec(ctx, "SET session_replication_role = 'replica'"); err != nil {
		return fmt.Errorf("disable triggers: %w", err)
	}

	// Truncate each table with CASCADE
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := tx.Exec(ctx, query); err != nil {
			// Table might not exist, continue
			continue
		}
	}

	// Re-enable triggers
	if _, err := tx.Exec(ctx, "SET session_replication_role = 'origin'"); err != nil {
		return fmt.Errorf("enable triggers: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// findProjectRoot finds the project root directory by looking for go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gomod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found (go.mod not found)")
		}
		dir = parent
	}
}

// SetupPostgresForTest is a convenience function that sets up a test database
// and calls testing.Main with cleanup. Use this in TestMain for simpler setup.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    os.Exit(testutil.SetupPostgresForTest(m))
//	}
func SetupPostgresForTest(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	testDB, err := SetupPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}
	defer Cleanup(testDB)

	return m.Run()
}
