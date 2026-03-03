# OpenPrint Cloud - AI Agent Instructions

This document provides guidance for AI assistants (Claude, Cursor, etc.) when working on the OpenPrint Cloud project.

## Project Context

You are working on OpenPrint Cloud, a microservices-based print management platform with 12+ services.

## Key Principles

1. **Code Quality**: All code must be production-ready
2. **Testing**: Every feature needs tests
3. **Documentation**: Update docs with code changes
4. **Security**: Never expose secrets, validate inputs

5. **Performance**: Consider performance implications

## Project Structure

```
/home/cmit/openprint/
├── services/           # 12 microservices
│   ├── api-gateway/   # Main entry point (:8000)
│   ├── analytics-service/   # Usage analytics (:8006)
│   ├── auth-service/   # Authentication (:8001)
│   ├── compliance-service/   # Compliance tracking (:8008)
│   ├── job-service/   # Print job management (:8003)
│   ├── m365-integration-service/   # M365 integration (:8011)
│   ├── notification-service/   # WebSocket notifications (:8005)
│   ├── organization-service/   # Organization management (:8009)
│   ├── policy-service/   # Print policy engine (:8010)
│   ├── registry-service/   # Printer/agent registry (:8002)
│   ├── storage-service/   # Document storage (:8004)
├── internal/           # Shared packages
│   ├── agent/       # Agent utilities
│   ├── auth/       # Authentication (JWT, OIDC, SAML, password)
│   ├── middleware/   # HTTP middleware
│   ├── multi-tenant/   # Multi-tenancy support
│   ├── shared/       # Shared utilities
│   └── testutil/     # Testing utilities
├── web/dashboard/     # React frontend
├── migrations/         # Database migrations
├── deployments/docker/  # Docker configuration
└── tests/             # Test suites
    ├── integration/
    └── load/
```

## Code Standards

### Go Services

#### Structure
```
services/my-service/
├── main.go           # Entry point, health check
├── handler/           # HTTP handlers
│   ├── my_handler.go
│   └── another_handler.go
├── repository/       # Database operations
│   ├── my_repo.go
│   └── another_repo.go
└── service/          # Business logic (optional)
```

#### Handler Pattern
```go
type MyHandler struct {
    service *MyService
}

func (h *MyHandler) HandleEndpoint(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 1. Parse and validate request
    var req MyRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request")
        return
    }
    
    // 2. Business logic
    result, err := h.service.DoSomething(ctx, &req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "internal error")
        return
    }
    
    // 3. Return response
    respondJSON(w, http.StatusOK, result)
}
```

#### Repository Pattern
```go
type MyRepository struct {
    db *pgxpool.Pool
}

func (r *MyRepository) GetByID(ctx context.Context, id string) (*MyEntity, error) {
    query := `SELECT * FROM my_table WHERE id = $1`
    
    var entity MyEntity
    err := r.db.QueryRow(ctx, query, id).Scan(&entity)
    if err != nil {
        return nil, err
    }
    
    return &entity, nil
}
```

### Database

#### Migrations
- Location: `migrations/`
- Naming: `0000XX_description.up.sql`
- Sequential numbering (000001, 000002, etc.)
- Always include `.down.sql` for rollback

#### Query Guidelines
- **Always use parameterized queries**
```go
// Good
rows, err := db.Query(ctx, "SELECT * FROM users WHERE email = $1", email)

// Bad - SQL injection vulnerable
rows, err := db.Query(ctx, fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email))
```

- **Use context with timeout**
```go
ctx, cancel := context.WithTimeout(5 * time.Second)
defer cancel()
```

#### Transaction Handling
```go
func (r *MyRepository) ComplexOperation(ctx context.Context) error {
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Multiple operations
    if err := r.operation1(ctx, tx); err != nil {
        return err
    }
    if err := r.operation2(ctx, tx); err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}
```

### Testing

#### Unit Tests
- File naming: `*_test.go`
- Run with: `go test ./...`
- Use table-driven tests
- Mock external dependencies

#### Integration Tests
- Build tag: `// +build integration`
- Run with: `go test -tags=integration ./...`
- Use `internal/testutil` helpers
- Test against real database

#### Test Coverage
- Minimum 60% coverage for services
- Use `go test -cover ./...`
- Focus on critical paths

### Error Handling

#### Always Check Errors
```go
result, err := service.DoSomething(ctx)
if err != nil {
    // Handle error appropriately
    log.Printf("Operation failed: %v", err)
    return err
}
// Use result
```

#### Error Types
- **Validation errors**: Return HTTP 400
- **Not found**: Return HTTP 404
- **Unauthorized**: Return HTTP 401
- **Internal errors**: Return HTTP 500
- **Service unavailable**: Return HTTP 503

## Environment Variables

### Required
```bash
JWT_SECRET=your-secret-key-min-32-characters
DATABASE_URL=postgres://openprint:openprint@localhost:5432/openprint
```

### Service URLs (for api-gateway)
```bash
AUTH_SERVICE_URL=http://localhost:8001
REGISTRY_SERVICE_URL=http://localhost:8002
JOB_SERVICE_URL=http://localhost:8003
STORAGE_SERVICE_URL=http://localhost:8004
NOTIFICATION_SERVICE_URL=http://localhost:8005
ANALYTICS_SERVICE_URL=http://localhost:8006
ORGANIZATION_SERVICE_URL=http://localhost:8007
```

### Optional
```bash
SERVER_ADDR=:8001                    # Override default port
S3_ENDPOINT=                         # Object storage
S3_BUCKET=openprint-documents
ENCRYPTION_KEY=                      # AES-256 key
JAEGER_ENDPOINT=                     # Tracing
LOG_LEVEL=info                       # debug, info, warn, error
TEST_MODE=true                       # Enable test mode
```

## Common Tasks

### Adding a New Service

1. **Create directory structure**
   ```bash
   mkdir -p services/my-service/{handler,repository}
   touch services/my-service/main.go
   ```

2. **Create main.go**
   ```go
   package main
   
   import (
       "context"
       "log"
       "net/http"
       "os"
       "github.com/openprint/openprint/internal/shared/middleware"
   )
   
   func main() {
       cfg := loadConfig()
       
       mux := http.NewServeMux()
       mux.HandleFunc("/health", healthHandler)
       
       // Add your routes
       
       server := &http.Server{
           Addr:    cfg.ServerAddr,
           Handler: middleware.Chain(
               middleware.LoggingMiddleware(log.New(os.Stdout, "[MY-SERVICE] ", log.LstdFlags)),
               middleware.RecoveryMiddleware(log.New(os.Stdout, "[MY-SERVICE] ", log.LstdFlags)),
           )(mux),
       }
       
       log.Printf("Service starting on %s", cfg.ServerAddr)
       if err := server.ListenAndServe(); err != nil {
           log.Fatal(err)
       }
   }
   ```

3. **Add to Makefile**
   ```makefile
   SERVICES += my-service
   ```

4. **Create Dockerfile**
   ```dockerfile
   FROM golang:1.24-alpine AS builder
   WORKDIR /build
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   RUN CGO_ENABLED=0 GOOS=linux go build -o my-service ./services/my-service
   
   FROM alpine:3.19
   RUN apk --no-cache add ca-certificates curl
   WORKDIR /app
   COPY --from=builder /build/my-service .
   EXPOSE 8001
   HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
       CMD curl -f http://localhost:8001/health || exit 1
   USER nobody
   ENTRYPOINT ["./my-service"]
   ```

5. **Write tests**
   - Unit tests in `*_test.go`
   - Integration tests with build tag

6. **Update API Gateway routes** (if public API)
   - Edit `services/api-gateway/main.go`
   - Add route mapping

7. **Update CLAUDE.md**
   - Add service to services table
   - Document purpose and port

### Adding a New API Endpoint

1. **Define route in main.go**
   ```go
   mux.HandleFunc("/api/v1/my-resource", h.ListMyResources)
   mux.HandleFunc("/api/v1/my-resource/", h.GetMyResource)
   ```

2. **Create handler**
   ```go
   func (h *Handler) ListMyResources(w http.ResponseWriter, r *http.Request) {
       ctx := r.Context()
       // Implementation
   }
   ```

3. **Add repository method** (if needed)
   ```go
   func (r *MyRepository) ListMyResources(ctx context.Context) ([]MyResource, error) {
       query := `SELECT * FROM my_resources ORDER BY created_at DESC`
       // Implementation
   }
   ```

4. **Write tests**
   ```go
   func TestListMyResources(t *testing.T) {
       // Test implementation
   }
   ```

5. **Update API Gateway** (if new service)
   - Add route in api-gateway

6. **Document in CLAUDE.md**
   - Add route to API Gateway Routes section

### Updating Database Schema

1. **Create migration**
   ```bash
   # Find next number
   ls migrations/*.up.sql | tail -1
   # Create migration files
   touch migrations/000029_description.up.sql
   touch migrations/000029_description.down.sql
   ```

2. **Write migration SQL**
   ```sql
   -- up.sql
   CREATE TABLE IF NOT EXISTS my_table (
       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
       created_at TIMESTAMPTZ DEFAULT NOW()
   );
   
   -- down.sql
   DROP TABLE IF EXISTS my_table;
   ```

3. **Run migration**
   ```bash
   migrate -path migrations -database "$DATABASE_URL" up
   ```

4. **Update repository** to use new table

5. **Write tests** for new functionality

6. **Document** in CLAUDE.md if significant

## Best Practices

### Performance
- Use connection pooling (pgxpool)
- Implement caching (Redis)
- Add database indexes
- Use pagination for lists
- Profile before optimizing

### Security
- Validate all inputs
- Use parameterized queries
- Never log secrets
- Check user permissions
- Use HTTPS only

### Reliability
- Handle errors gracefully
- Log operations
- Use transactions
- Implement retries
- Add health checks

### Maintainability
- Write readable code
- Add comments for complex logic
- Follow existing patterns
- Keep functions small
- Write tests

## Debugging

### Logs
- All services log to stdout
- Format: `[SERVICE] timestamp message`
- Levels: debug, info, warn, error
- Use structured logging (JSON)

### Health Checks
- Endpoint: `/health`
- Returns: `{"status":"healthy","service":"service-name"}`
- Use for load balancer checks

### Database
```bash
# Connect to database
psql $DATABASE_URL

# Check tables
\dt

# Check recent data
SELECT * FROM table ORDER BY created_at DESC LIMIT 10;
```

### Redis
```bash
# Connect to Redis
redis-cli -a openprint

# Check keys
KEYS *

# Get value
GET key
```

## Troubleshooting

### Service Won't Start
1. Check logs: `docker logs container-name`
2. Check port: `netstat -tulpn | grep PORT`
3. Check env vars: `docker exec container env`
4. Check health: `curl http://localhost:PORT/health`

### Database Connection Failed
1. Check PostgreSQL is running: `docker ps`
2. Check connection string in `.env`
3. Check network: `docker network ls`
4. Check logs: `docker logs openprint-postgres`

### Redis Connection Failed
1. Check Redis is running: `docker ps`
2. Check password matches
3. Check network connectivity
4. Check logs: `docker logs openprint-redis`

### Tests Failing
1. Check test database: `psql $DATABASE_URL -c "\dt"`
2. Run migrations: `make db-migrate`
3. Check test output: `go test -v ./...`
4. Check test configuration

## Resources

### Documentation
- `CLAUDE.md` - Project overview
- `docs/testing.md` - Testing guide
- `internal/testutil/README.md` - Test utilities
- `web/dashboard/README.md` - Dashboard guide

### Examples
- `services/auth-service/` - Complete service example
- `services/job-service/` - Complex business logic
- `services/api-gateway/` - Gateway pattern

### Tools
- `make` - Build and test commands
- `docker-compose` - Local development
- `migrate` - Database migrations
- `golangci-lint` - Code linting

## Tips

1. **Read existing code** before implementing new features
2. **Follow patterns** established in the codebase
3. **Write tests first** when fixing bugs (TDD)
4. **Keep it simple** - avoid over-engineering
5. **Document complex logic** with comments
6. **Test edge cases** and error paths
7. **Review your code** before committing
8. **Ask questions** if unsure about approach
9. **Check security** implications of changes
10. **Update docs** when changing behavior
