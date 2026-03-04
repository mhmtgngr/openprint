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
| `auth-service` | 8001 | User authentication, sessions, OIDC/SAML identity providers |
| `registry-service` | 8002 | Printer and agent registration, heartbeat monitoring |
| `job-service` | 8003 | Print job queuing, routing, and processing |
| `storage-service` | 8004 | Document storage (S3 or local), encryption support |
| `notification-service` | 8005 | WebSocket notifications for real-time updates |

### Internal Packages

```
internal/
├── auth/              # Authentication utilities
│   ├── jwt/          # JWT token generation/validation
│   ├── oidc/         # OpenID Connect provider support
│   ├── saml/         # SAML SSO support
│   └── password/     # Password hashing (bcrypt)
└── shared/           # Shared service components
    ├── errors/       # Common error types
    ├── middleware/   # HTTP middleware (auth, CORS, logging, recovery)
    └── telemetry/    # OpenTelemetry tracing
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

# Build all services
go build ./...

# Build specific service
go build ./services/auth-service
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

```bash
JWT_SECRET=your-secret-key-min-32-chars
S3_ENDPOINT=          # Optional, falls back to local storage
S3_BUCKET=openprint-documents
ENCRYPTION_KEY=       # Optional AES-256 key
JAEGER_ENDPOINT=      # Optional tracing
```

### Service Ports (Docker)

| Service | Internal | External |
|---------|----------|----------|
| PostgreSQL | 5432 | 15432 |
| Redis | 6379 | 16379 |
| auth-service | 8001 | 18001 |
| registry-service | 8002 | 8002 |
| job-service | 8003 | 8003 |
| storage-service | 8004 | 8004 |
| notification-service | 8005 | 18005 |
| dashboard | 3000 | 3000 |

### Building Individual Service Images

```bash
docker build -f deployments/docker/Dockerfile.auth-service -t openprint/auth-service .
docker build -f deployments/docker/Dockerfile.dashboard -t openprint/dashboard .
```

## Database Migrations

```bash
# Run migrations (requires golang-migrate or similar)
migrate -path migrations -database "postgres://openprint:openprint@localhost:5432/openprint?sslmode=disable" up
```

## Development Workflow

1. **Adding a new feature**: Update the relevant service in `services/*/` and add tests
2. **Adding shared functionality**: Add to `internal/shared/`
3. **Updating schema**: Add migration files to `migrations/`
4. **After changes**: Run `go test ./...` and `cd web/dashboard && npm run type-check`

## Service Communication

- Services communicate via HTTP/REST
- JWT tokens used for inter-service authentication
- Redis used for async job queue and pub/sub
- PostgreSQL as single source of truth

## Health Checks

Each service exposes a `/health` endpoint:

```bash
curl http://localhost:8001/health  # auth-service
curl http://localhost:8002/health  # registry-service
curl http://localhost:8003/health  # job-service
curl http://localhost:8004/health  # storage-service
curl http://localhost:8005/health  # notification-service
```
