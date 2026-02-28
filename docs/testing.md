# Testing Documentation

This document describes the testing infrastructure for OpenPrint Cloud.

## Overview

OpenPrint Cloud uses a multi-tier testing approach:

- **Unit Tests**: Fast, isolated tests that don't require external services
- **Integration Tests**: End-to-end tests requiring running services and containers
- **Benchmark Tests**: Performance and load testing

## Test Structure

```
tests/
├── integration/           # Integration tests (requires `integration` build tag)
│   └── api_test.go       # End-to-end API tests
└── testutil/             # Shared testing utilities
    ├── database.go       # Database test helpers
    ├── http.go           # HTTP test helpers
    └── fixtures.go       # Test data fixtures
```

## Build Tags

Integration tests use the `integration` build tag and will NOT run during normal unit test execution.

### Running Unit Tests Only

```bash
# Unit tests (without integration tests)
make test-unit
go test -short ./...
```

### Running Integration Tests

```bash
# Integration tests only (requires running services)
make test-integration
go test -tags=integration ./tests/integration/...
```

### Running All Tests

```bash
# All tests
make test
```

## Test Environment Setup

Integration tests require the following services to be running:

- PostgreSQL (port 15432 or 5432)
- Redis (port 16379 or 6379)
- Auth Service (port 18001 or 8001)
- Registry Service (port 8002)
- Job Service (port 8003)
- Storage Service (port 8004)
- Notification Service (port 18005 or 8005)

### Quick Start with Docker

```bash
# Start test environment
cd deployments/docker
docker compose up -d

# Run integration tests
cd ../..
make test-integration

# Stop test environment
cd deployments/docker
docker compose down
```

### Manual Environment Setup

```bash
# Check test environment
make test-env

# Set up test environment (pulls images, sets env vars)
make test-env-setup
```

## Environment Variables

Integration tests use the following environment variables (with defaults):

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_SERVICE_URL` | `http://localhost:18001` | Auth service endpoint |
| `REGISTRY_SERVICE_URL` | `http://localhost:8002` | Registry service endpoint |
| `JOB_SERVICE_URL` | `http://localhost:8003` | Job service endpoint |
| `STORAGE_SERVICE_URL` | `http://localhost:8004` | Storage service endpoint |
| `NOTIFICATION_SERVICE_URL` | `http://localhost:18005` | Notification service endpoint |
| `DATABASE_URL` | `postgres://openprint:openprint@localhost:15432/openprint` | PostgreSQL connection string |
| `TEST_MODE` | `true` | Enables test mode in services |

## Test Utilities

The `testutil` package provides shared testing helpers:

### Database Helpers

```go
import "github.com/openprint/openprint/tests/testutil"

// Create a test database with schema
db := testutil.SetupTestDB(t)
defer testutil.TeardownTestDB(t, db)

// Run a test transaction
testutil.InTransaction(t, db, func(tx pgx.Tx) {
    // Your test code here
})
```

### HTTP Helpers

```go
// Create a test HTTP client
client := testutil.NewTestClient()
client.AuthToken = "your-token"

// Make test requests
resp := client.PostJSON("/api/endpoint", payload)
```

### Test Fixtures

```go
// Create test data
user := testutil.CreateTestUser(t, db)
printer := testutil.CreateTestPrinter(t, db, user.ID)
job := testutil.CreateTestJob(t, db, printer.ID)
```

## Writing Tests

### Unit Test Example

```go
func TestHashPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid password", "SecurePass123!", false},
        {"empty password", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hashed, err := password.Hash(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("Hash() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && !password.Verify(tt.password, hashed) {
                t.Error("Hash() produced invalid hash")
            }
        })
    }
}
```

### Integration Test Example

```go
//go:build integration
// +build integration

package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestUserRegistration(t *testing.T) {
    client := NewTestClient()

    resp, err := client.makeRequest("POST", authServiceURL+"/auth/register",
        map[string]interface{}{
            "email":    "test@example.com",
            "password": "SecurePass123!",
        }, nil)

    assert.NoError(t, err)
    assert.Equal(t, 201, resp.StatusCode)
}
```

## Coverage

### Generating Coverage Reports

```bash
# Coverage by function
make test-cover-func

# HTML coverage report
make test-cover
```

### Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| `internal/auth` | 80% | - |
| `internal/shared` | 80% | - |
| `tests/testutil` | 70% | - |
| Services | 60% | - |

## CI/CD Integration

Tests run automatically on:
- Pull requests
- Main branch pushes
- Release tags

### GitHub Actions Workflow

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: make test-env
      - run: make test-unit
      - run: make test-integration
```

## Troubleshooting

### Integration Tests Fail to Connect

Ensure services are running:
```bash
docker ps  # Check containers
curl http://localhost:18001/health  # Check auth service
```

### Database Connection Errors

Check database is accessible:
```bash
psql postgres://openprint:openprint@localhost:15432/openprint
```

### Port Already in Use

Change service ports via environment variables:
```bash
export AUTH_SERVICE_URL=http://localhost:18002
export REGISTRY_SERVICE_URL=http://localhost:8003
```

## Best Practices

1. **Keep unit tests fast** - they should run in milliseconds
2. **Use build tags** for integration tests to prevent slow CI runs
3. **Clean up resources** - use `t.Cleanup()` for teardown
4. **Table-driven tests** - for multiple test cases
5. **Test boundaries** - mock external dependencies in unit tests
6. **Use subtests** - `t.Run()` for better test organization
7. **Check errors** - never ignore errors in tests
8. **Use assertions** - `require` for critical failures, `assert` for continued testing
