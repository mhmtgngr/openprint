# Integration Tests

This directory contains end-to-end integration tests for OpenPrint Cloud services.

## Prerequisites

Before running integration tests, ensure all services are running:

```bash
cd deployments/docker
docker-compose up -d
```

Wait for all services to be healthy (may take 30-60 seconds):

```bash
docker-compose ps
```

## Running Tests

### Run all integration tests
```bash
go test ./tests/integration/ -v
```

### Run specific test
```bash
go test ./tests/integration/ -v -run TestHealthChecks
```

### Run with timeout
```bash
go test ./tests/integration/ -v -timeout 5m
```

### Skip slow tests
```bash
go test ./tests/integration/ -v -short
```

## Environment Variables

You can customize service URLs using environment variables:

| Variable | Default |
|----------|---------|
| `AUTH_SERVICE_URL` | `http://localhost:18001` |
| `REGISTRY_SERVICE_URL` | `http://localhost:8002` |
| `JOB_SERVICE_URL` | `http://localhost:8003` |
| `STORAGE_SERVICE_URL` | `http://localhost:8004` |
| `NOTIFICATION_SERVICE_URL` | `http://localhost:18005` |
| `DATABASE_URL` | `postgres://openprint:openprint@localhost:15432/openprint` |

Example:
```bash
AUTH_SERVICE_URL=http://localhost:8001 go test ./tests/integration/ -v
```

## Test Coverage

### Health Checks
- All service health endpoints
- Service availability verification

### Authentication Service
- User registration
- User login
- Token refresh
- Get current user
- Unauthorized request handling

### Registry Service
- Agent registration
- Printer registration
- Agent heartbeat
- List/get agents and printers
- Status updates
- Capabilities updates

### Job Service
- Job creation
- Job listing with filters
- Job status updates
- Queue statistics

### Storage Service
- Document upload (multipart)
- Document metadata retrieval
- Document listing

### End-to-End Workflows
- Complete print job workflow:
  1. Register user
  2. Register agent
  3. Register printer
  4. Upload document
  5. Send heartbeat
  6. Update printer status
  7. Create print job
  8. Verify in database
  9. Retrieve job details
  10. List all jobs

### Docker Communication
- Inter-service communication
- Database connection verification
- Concurrent request handling

### Performance Tests
- Health check response times (skipped in short mode)

### WebSocket Tests
- Notification service WebSocket upgrade

## Test Data

Tests create and clean up their own data. Each test uses unique identifiers
based on timestamps to avoid conflicts. Test data is cleaned up in `defer`
statements to ensure cleanup even on test failure.

## Troubleshooting

### Tests fail with connection refused
- Ensure services are running: `docker-compose ps`
- Check service logs: `docker-compose logs -f [service-name]`

### Tests fail with unauthorized
- Check JWT_SECRET matches between services
- Verify database is accessible

### Tests timeout
- Increase timeout: `go test ./tests/integration/ -v -timeout 10m`
- Check system resources: `docker stats`

### Database errors
- Verify database is running: `docker-compose exec postgres pg_isready`
- Check database connection string in DATABASE_URL
