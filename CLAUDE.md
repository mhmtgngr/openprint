# OpenPrint Cloud

OpenPrint is a cloud-based print management platform consisting of microservices for authentication, print job management, device registry, document storage, and real-time notifications.

## Project Overview

- **Module**: `github.com/openprint/openprint`
- **Go Version**: 1.24.0
- **Architecture**: Microservices with shared internal packages
- **Frontend**: React + TypeScript + Vite dashboard

## Architecture

### Services

| Service | Port | Description |
|---------|------|-------------|
| api-gateway | 8000 | Unified API entry point, reverse proxy, rate limiting, developer portal |
| gateway | 8000 | Legacy API gateway (routes to ports 8001-8005) |
| auth-service | 8001 | User authentication, sessions, OIDC/SAML identity providers |
| registry-service | 8002 | Printer and agent registration, heartbeat monitoring |
| job-service | 8003 | Print job queuing, routing, and processing |
| storage-service | 8004 | Document storage (S3 or local), encryption support |
| notification-service | 8005 | WebSocket notifications for real-time updates |
| analytics-service | 8006 | Usage analytics, reporting, and data aggregation |
| compliance-service | 8008 | FedRAMP, HIPAA, GDPR, SOC2 compliance tracking |
| organization-service | 8009 | Organization management, permissions, members |
| policy-service | 8010 | Print policy engine, rule evaluation and enforcement |
| m365-integration-service | 8011 | Microsoft 365 integration for print management |

> **Note:** Port conflicts exist in codebase (see Known Issues). Recommended assignments shown above.

### Internal Packages

```
internal/
├── agent/              # Agent utilities and types
├── auth/               # Authentication utilities
│   ├── jwt/           # JWT token generation/validation
│   ├── oidc/          # OpenID Connect provider support
│   ├── password/      # Password hashing (bcrypt)
│   ├── roles/         # Role-based access control
│   └── saml/          # SAML SSO support
├── middleware/         # HTTP middleware components
│   ├── cookie_auth.go # Cookie-based authentication
│   ├── ratelimit.go   # Rate limiting middleware
│   └── validation.go  # Request validation
├── multi-tenant/       # Multi-tenancy support
│   ├── audit.go       # Audit logging
│   ├── context.go     # Tenant context management
│   ├── middleware.go  # Tenant isolation middleware
│   ├── quota.go       # Quota management
│   └── repository.go  # Tenant-aware data access
├── shared/             # Shared service components
│   ├── context/       # Request context utilities
│   ├── errors/        # Common error types
│   ├── middleware/    # HTTP middleware (auth, CORS, logging, recovery)
│   ├── ratelimit/     # Rate limiting utilities
│   └── telemetry/     # OpenTelemetry tracing
└── testutil/           # Testing utilities (see internal/testutil/README.md)
    ├── db.go          # Database test helpers
    ├── fixtures.go    # Test data fixtures
    ├── http.go        # HTTP test helpers
    └── migrate.go     # Migration test utilities
```

### Service Dependencies

```
┌─────────────────┐
│   Dashboard      │ (React, port 3000)
│   web/dashboard  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  API Gateway    │ (port 8000)
│  api-gateway    │
└────────┬────────┘
         │
    ┌────┴────┬────────┬────────┬────────┬────────┐
    ▼         ▼        ▼        ▼        ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│ auth  │ │registry│ │  job  │ │storage│ │notify │ │analytics│
│ :8001 │ │ :8002 │ │ :8003 │ │ :8004 │ │ :8005 │ │ :8006 │
└───┬───┘ └───┬───┘ └───┬───┘ └───────┘ └───────┘ └───────┘
    │         │         │
    └─────────┴─────────┴──────┐
                              │
                    ┌─────────┴──────────┐
                    │  PostgreSQL 16      │ (port 5432/15432)
                    │  Redis 7            │ (port 6379/16379)
                    └────────────────────┘
```

### Infrastructure

- **PostgreSQL 16**: Primary database (ports 5432/15432)
- **Redis 7**: Job queue and session storage (ports 6379/16379)
- **S3/MinIO**: Optional document storage backend

## Development Commands

### Go Services

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./services/auth-service/...

# Build all services (core 5)
make build

# Build specific service
go build ./services/auth-service
go build ./services/api-gateway
go build ./services/analytics-service

# Run linter
make lint
```

### Dashboard (TypeScript/React)

```bash
cd web/dashboard

# Install dependencies
npm install

# Development server
npm run dev

# Type checking
npm run type-check    # or: npx tsc --noEmit

# Run tests
npm run test

# Build for production
npm run build

# Lint
npm run lint
```

## Docker Deployment

### Quick Start

```bash
cd deployments/docker
docker-compose up -d
```

### Environment Variables

Create a `.env` file (see `.env.example`):

#### Required Variables
```bash
JWT_SECRET=your-secret-key-min-32-chars    # Required for all services
DATABASE_URL=postgres://openprint:openprint@localhost:5432/openprint
```

#### Service URLs (for api-gateway)
```bash
AUTH_SERVICE_URL=http://localhost:8001
REGISTRY_SERVICE_URL=http://localhost:8002
JOB_SERVICE_URL=http://localhost:8003
STORAGE_SERVICE_URL=http://localhost:8004
NOTIFICATION_SERVICE_URL=http://localhost:8005
ANALYTICS_SERVICE_URL=http://localhost:8006
ORGANIZATION_SERVICE_URL=http://localhost:8007
```

#### Optional Variables
```bash
SERVER_ADDR=:8001                    # Override default port
S3_ENDPOINT=                         # Optional, falls back to local storage
S3_BUCKET=openprint-documents
ENCRYPTION_KEY=                      # Optional AES-256 key
JAEGER_ENDPOINT=                     # Optional tracing endpoint
M365_STORAGE_PATH=/var/lib/openprint/m365  # M365 integration storage
LOG_LEVEL=info                       # debug, info, warn, error
TEST_MODE=true                       # Enable test mode
```

### Service Ports (Docker)

| Service | Internal | External | Notes |
|---------|----------|----------|-------|
| PostgreSQL | 5432 | 15432 | Database |
| Redis | 6379 | 16379 | Cache/Queue |
| api-gateway | 8000 | 8000 | Main entry point |
| auth-service | 8001 | 18001 | Authentication |
| registry-service | 8002 | 8002 | Device registry |
| job-service | 8003 | 8003 | Print jobs |
| storage-service | 8004 | 8004 | Document storage |
| notification-service | 8005 | 18005 | WebSockets |
| analytics-service | 8006 | 8006 | Analytics |
| organization-service | 8007 | 8007 | Organizations |
| compliance-service | 8008 | 8008 | Compliance |
| policy-service | 8009 | 8009 | Policy engine |
| m365-integration-service | 8010 | 8010 | M365 integration |
| dashboard | 3000 | 3000 | Web UI |

### Building Individual Service Images

```bash
# Core services (have Dockerfiles)
docker build -f deployments/docker/Dockerfile.auth-service -t openprint/auth-service .
docker build -f deployments/docker/Dockerfile.registry-service -t openprint/registry-service .
docker build -f deployments/docker/Dockerfile.job-service -t openprint/job-service .
docker build -f deployments/docker/Dockerfile.storage-service -t openprint/storage-service .
docker build -f deployments/docker/Dockerfile.notification-service -t openprint/notification-service .
docker build -f deployments/docker/Dockerfile.dashboard -t openprint/dashboard .

# Additional services (build from source)
go build -o bin/analytics-service ./services/analytics-service
go build -o bin/api-gateway ./services/api-gateway
go build -o bin/compliance-service ./services/compliance-service
go build -o bin/organization-service ./services/organization-service
go build -o bin/policy-service ./services/policy-service
go build -o bin/m365-integration-service ./services/m365-integration-service
```

## API Gateway Routes

The api-gateway (port 8000) routes requests to backend services:

| Route Pattern | Target Service | Auth Required |
|--------------|----------------|---------------|
| `/auth/*` | auth-service (:8001) | No (login/register) |
| `/api/v1/auth/*` | auth-service (:8001) | Varies |
| `/api/v1/jobs/*` | job-service (:8003) | Yes |
| `/api/v1/quota/*` | job-service (:8003) | Yes |
| `/api/v1/cost/*` | job-service (:8003) | Yes |
| `/api/v1/reports/*` | job-service (:8003) | Yes |
| `/api/v1/printers/*` | registry-service (:8002) | Yes |
| `/api/v1/agents/*` | registry-service (:8002) | Yes |
| `/api/v1/devices/*` | registry-service (:8002) | Yes |
| `/api/v1/documents/*` | storage-service (:8004) | Yes |
| `/api/v1/watermarks/*` | storage-service (:8004) | Yes |
| `/api/v1/notifications/*` | notification-service (:8005) | Yes |
| `/api/v1/analytics/*` | analytics-service (:8006) | Yes |
| `/api/v1/organizations/*` | organization-service (:8007) | Yes |
| `/api/v1/developer/*` | api-gateway (local) | Yes |

## Database Migrations

```bash
# Run migrations (requires golang-migrate or similar)
migrate -path migrations -database "postgres://openprint:openprint@localhost:5432/openprint?sslmode=disable" up

# Rollback last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Reset database (WARNING: deletes all data)
migrate -path migrations -database "$DATABASE_URL" down -all
migrate -path migrations -database "$DATABASE_URL" up
```

## Development Workflow

1. **Adding a new feature**: Update the relevant service in `services/*/` and add tests
2. **Adding shared functionality**: Add to `internal/shared/` or appropriate internal package
3. **Updating schema**: Add migration files to `migrations/`
4. **After changes**: Run `go test ./...` and `cd web/dashboard && npm run type-check`
5. **Adding a new service**: Create in `services/`, add to Makefile, create Dockerfile

## Service Communication

- Services communicate via HTTP/REST
- API Gateway routes external requests to appropriate services
- JWT tokens used for inter-service authentication
- Redis used for async job queue and pub/sub
- PostgreSQL as single source of truth

## Health Checks

Each service exposes a `/health` endpoint:

```bash
# Core services
curl http://localhost:8000/health  # api-gateway
curl http://localhost:8001/health  # auth-service
curl http://localhost:8002/health  # registry-service
curl http://localhost:8003/health  # job-service
curl http://localhost:8004/health  # storage-service
curl http://localhost:8005/health  # notification-service

# Additional services
curl http://localhost:8006/health  # analytics-service
curl http://localhost:8007/health  # organization-service
curl http://localhost:8008/health  # compliance-service
curl http://localhost:8009/health  # policy-service
curl http://localhost:8010/health  # m365-integration-service
```

## Testing

See `docs/testing.md` and `internal/testutil/README.md` for detailed testing documentation.

### Unit Tests
```bash
make test-unit
# or
go test -short ./...
```

### Integration Tests
```bash
make test-integration
# or
go test -tags=integration ./tests/integration/...
```

### Dashboard Tests
```bash
cd web/dashboard
npm run test        # Unit tests
npm run test:e2e    # E2E tests
```

## Known Issues

### Port Conflicts
The following services have conflicting default ports in their code:
- `analytics-service` and `compliance-service` both default to :8006
- `organization-service` and `policy-service` both default to :8007
- `api-gateway` and `gateway` both use port 8000

**Resolution:** Use `SERVER_ADDR` environment variable to assign unique ports.

### Services Not in Makefile
The following services are not included in the `make build` target:
- api-gateway
- analytics-service
- compliance-service
- organization-service
- policy-service
- m365-integration-service

Build individually with: `go build ./services/<service-name>`

### Services Without Dockerfiles
The following services do not have Dockerfiles in `deployments/docker/`:
- api-gateway, analytics-service, compliance-service
- organization-service, policy-service, m365-integration-service, gateway

## Additional Documentation

- **Testing**: `docs/testing.md`
- **Docker Deployment**: `deployments/docker/README.md`
- **Test Utilities**: `internal/testutil/README.md`
- **Dashboard**: `web/dashboard/README.md`
- **Load Testing**: `tests/load/README.md`
- **Integration Tests**: `tests/integration/README.md`
