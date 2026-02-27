# OpenPrint Docker Deployment

This directory contains the Docker configuration for running the entire OpenPrint stack.

## Services

| Service | Port | Description |
|---------|------|-------------|
| dashboard | 3000 | React web dashboard |
| auth-service | 8001 | Authentication and authorization |
| registry-service | 8002 | Printer and agent registry |
| job-service | 8003 | Print job management |
| storage-service | 8004 | Document storage |
| notification-service | 8005 | WebSocket notifications |
| postgres | 5432 | PostgreSQL 16 database |
| redis | 6379 | Redis 7 cache/message broker |

## Quick Start

1. Copy the environment file and configure:
```bash
cp .env.example .env
# Edit .env and set a secure JWT_SECRET
```

2. Start all services:
```bash
docker-compose up -d
```

3. Check service status:
```bash
docker-compose ps
docker-compose logs -f
```

## Building Individual Services

Each service has its own Dockerfile that can be built independently:

```bash
# Build auth-service
docker build -f deployments/docker/Dockerfile.auth-service -t openprint/auth-service .

# Build dashboard
docker build -f deployments/docker/Dockerfile.dashboard -t openprint/dashboard .
```

## Volumes

- `openprint-postgres-data` - PostgreSQL database data
- `openprint-redis-data` - Redis persistence
- `openprint-storage-data` - Local file storage (fallback when S3 not configured)

## Health Checks

All services expose a `/health` endpoint for health checks. The docker-compose configuration includes health checks for all services and proper dependency management.

## Production Considerations

1. **Security**: Change default passwords in docker-compose.yml and set a strong JWT_SECRET
2. **Persistence**: All data is stored in named volumes for persistence
3. **Networking**: Services communicate via the `openprint-network` bridge network
4. **Scaling**: Services can be scaled using `docker-compose up --scale`
5. **Monitoring**: Configure JAEGER_ENDPOINT for distributed tracing

## Stopping Services

```bash
docker-compose down
```

To remove volumes as well (WARNING: deletes all data):
```bash
docker-compose down -v
```
