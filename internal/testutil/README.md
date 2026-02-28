# Test Utilities Package

The `testutil` package provides reusable testing utilities for OpenPrint services. It uses [testcontainers-go](https://golang.testcontainers.org/) to create isolated PostgreSQL containers for testing, eliminating the need for external database dependencies.

## Features

- **Automated PostgreSQL Setup**: Spin up a fresh PostgreSQL container for each test package
- **Migration Support**: Automatically runs all migrations from the `migrations/` directory
- **Test Fixtures**: Helper functions to create test data (organizations, users, agents, printers, jobs)
- **Data Cleanup**: Utilities to truncate tables between tests without recreating containers
- **Connection Pooling**: Pre-configured pgxpool for optimal test performance

## Usage

### Basic Test Setup

The simplest way to use testutil is via `TestMain`:

```go
package repository

import (
    "context"
    "testing"
    "os"

    "github.com/openprint/openprint/internal/testutil"
)

var testDB *testutil.TestDB

func TestMain(m *testing.M) {
    ctx := context.Background()
    testDB, err = testutil.SetupPostgresContainer(ctx)
    if err != nil {
        log.Fatalf("Failed to setup test database: %v", err)
    }
    defer testutil.Cleanup(testDB)

    os.Exit(m.Run())
}

func TestMyRepository(t *testing.T) {
    // Use testDB.Pool for database operations
    repo := NewMyRepository(testDB.Pool)
    // ... test code
}
```

### Convenience Function

For even simpler setup, use `SetupPostgresForTest`:

```go
func TestMain(m *testing.M) {
    os.Exit(testutil.SetupPostgresForTest(m))
}
```

### Using Test Fixtures

The package provides helpers for creating test data:

```go
func TestJobAssignment(t *testing.T) {
    ctx := context.Background()

    // Create a complete test setup
    orgID, userID, agentID, printerID, documentID, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
    require.NoError(t, err)

    // Create an assignment
    assignmentID, err := testutil.CreateTestJobAssignment(ctx, testDB.Pool, jobID, agentID)
    require.NoError(t, err)

    // ... test code
}
```

### Using Test Fixtures Struct

For complex tests, use the `TestFixture` struct:

```go
func TestComplexWorkflow(t *testing.T) {
    ctx := context.Background()

    fixture, err := testutil.SetupTestFixture(ctx, testDB.Pool)
    require.NoError(t, err)

    // Access all fixture IDs
    t.Logf("Organization: %s", fixture.OrganizationID)
    t.Logf("User: %s (%s)", fixture.UserID, fixture.UserEmail)
    t.Logf("Agent: %s", fixture.AgentID)
    t.Logf("Printer: %s", fixture.PrinterID)
    t.Logf("Document: %s", fixture.DocumentID)
    t.Logf("Job: %s", fixture.JobID)
    t.Logf("Assignment: %s", fixture.AssignmentID)
}
```

### Cleaning Up Between Tests

To run multiple tests in sequence without recreating the container:

```go
func TestSuite(t *testing.T) {
    ctx := context.Background()

    t.Run("Test1", func(t *testing.T) {
        // Test code...
        testutil.CleanupTestData(ctx, testDB.Pool)
    })

    t.Run("Test2", func(t *testing.T) {
        // Fresh database state
        // Test code...
    })
}
```

## Available Functions

### Database Setup

| Function | Description |
|----------|-------------|
| `SetupPostgresContainer(ctx)` | Creates and starts a PostgreSQL container with migrations |
| `Cleanup(db)` | Terminates container and closes connections |
| `SetupPostgresForTest(m)` | Convenience function for TestMain |
| `GetTestDBConnection(db)` | Returns the connection string |

### Migrations

| Function | Description |
|----------|-------------|
| `RunMigrations(ctx, db, dir)` | Runs all .up.sql migration files from a directory |
| `ExecuteMigrationFile(ctx, db, dir, name)` | Executes a specific migration file |
| `ExecuteSQL(ctx, db, sql)` | Directly executes SQL |
| `ResetMigrations(ctx, db)` | Drops migrations tracking table |

### Test Fixtures

| Function | Description |
|----------|-------------|
| `CreateTestOrganization(ctx, db)` | Creates a test organization |
| `CreateTestUser(ctx, db, orgID)` | Creates a test user |
| `CreateTestAgent(ctx, db, orgID)` | Creates a test agent |
| `CreateTestPrinter(ctx, db, agentID)` | Creates a test printer |
| `CreateTestDocument(ctx, db, email)` | Creates a test document |
| `CreateTestPrintJob(ctx, db, docID, printerID, email)` | Creates a test print job |
| `CreateTestJobAssignment(ctx, db, jobID, agentID)` | Creates a test job assignment |
| `CreateTestJobHistory(ctx, db, jobID)` | Creates a test job history entry |
| `CreateFullTestSetup(ctx, db)` | Creates a complete test data hierarchy |
| `SetupTestFixture(ctx, db)` | Creates and returns a TestFixture struct |

### Data Cleanup

| Function | Description |
|----------|-------------|
| `TruncateAllTables(ctx, db)` | Truncates all tables with CASCADE |
| `CleanupTestData(ctx, db)` | Alias for TruncateAllTables |
| `CleanupTestDataByUser(ctx, db, email)` | Removes data for a specific user |

## Requirements

- Docker daemon must be running (testcontainers creates containers via Docker)
- Go 1.24 or later
- Ports are automatically assigned by Docker (no port conflicts)

## Best Practices

1. **One Container Per Test Package**: Use `TestMain` to create one container shared by all tests in the package
2. **Cleanup Between Tests**: Use `TruncateAllTables` or `CleanupTestData` between tests for isolation
3. **Use Fixture Helpers**: Leverage the fixture creation functions instead of manual INSERT statements
4. **Context Timeouts**: Always use contexts with timeouts when calling database operations
5. **Parallel Tests**: Avoid running tests in parallel (`t.Parallel()`) when using shared fixtures

## Example: Complete Test File

```go
package repository_test

import (
    "context"
    "os"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/openprint/openprint/internal/testutil"
    "github.com/openprint/openprint/services/job-service/repository"
)

var (
    testDB  *testutil.TestDB
    ctx     = context.Background()
)

func TestMain(m *testing.M) {
    testDB, err = testutil.SetupPostgresContainer(ctx)
    if err != nil {
        log.Fatalf("Failed to setup test database: %v", err)
    }
    defer testutil.Cleanup(testDB)

    os.Exit(m.Run())
}

func TestJobAssignmentRepository_AssignJob(t *testing.T) {
    // Setup test data
    fixture, err := testutil.SetupTestFixture(ctx, testDB.Pool)
    require.NoError(t, err)

    // Create repository
    repo := repository.NewJobAssignmentRepository(testDB.Pool)

    // Create a new job
    jobID, err := testutil.CreateTestPrintJob(ctx, testDB.Pool,
        fixture.DocumentID, fixture.PrinterID, fixture.UserEmail)
    require.NoError(t, err)

    // Test assignment
    assignment := &repository.JobAssignment{
        JobID:   jobID,
        AgentID: fixture.AgentID,
        Status:  "assigned",
    }

    err = repo.AssignJob(ctx, assignment)
    require.NoError(t, err)
    assert.NotEmpty(t, assignment.ID)

    // Cleanup
    testutil.CleanupTestData(ctx, testDB.Pool)
}
```

## Troubleshooting

### "Docker daemon not running"

Ensure Docker is installed and running:
```bash
docker ps
```

### "Port already in use"

Testcontainers automatically assigns available ports. If you see this error, you may have a conflicting configuration.

### "Migration file not found"

The migrations directory is located relative to `go.mod`. Ensure you're running tests from the project root.

### Slow test startup

The first test run starts the Docker container, which takes 10-20 seconds. Subsequent tests reuse the container.
