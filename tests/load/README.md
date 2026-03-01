# OpenPrint Load Testing Suite

Comprehensive performance testing infrastructure using k6 to validate system behavior under stress, establish performance baselines, and detect regressions across all microservices.

## Quick Start

```bash
cd tests/load

# Install dependencies
npm install

# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run a smoke test
npm run test:smoke

# Run baseline tests
npm run test:baseline
```

## Project Structure

```
tests/load/
├── lib/                      # Shared utilities
│   ├── config.js             # Test configuration
│   ├── helpers.js            # Helper functions
│   ├── api-client.js         # HTTP client wrappers
│   └── metrics.js            # Custom metrics
├── scenarios/                # Test scenarios
│   ├── auth/                 # Auth service tests
│   ├── registry/             # Registry service tests
│   ├── job/                  # Job service tests
│   ├── storage/              # Storage service tests
│   ├── notification/         # Notification service tests
│   └── mixed/                # Mixed workload tests
├── k8s/                      # Kubernetes manifests
│   └── k6-operator/          # K6 Operator resources
├── deployments/              # Docker configs
│   ├── grafana/              # Grafana dashboard
│   └── prometheus/           # Prometheus config
├── scripts/                  # Utility scripts
├── docker-compose.test.yml   # Test environment
└── package.json
```

## Available Tests

### Baseline Tests

Establish performance baselines for each service:

```bash
npm run test:auth:baseline
npm run test:registry:baseline
npm run test:job:baseline
npm run test:storage:baseline
npm run test:notification:baseline
```

### Regression Tests

FR-001 through FR-006 functional tests:

```bash
# Auth: Concurrent user login
npm run test:auth:concurrent

# Registry: High-frequency heartbeat
npm run test:registry:heartbeat

# Job: Parallel job submission
npm run test:job:submit

# Storage: Document upload burst
npm run test:storage:upload

# Notification: WebSocket connection scaling
npm run test:notification:connect
```

### Mixed Workload Tests

End-to-end and stress testing:

```bash
# E2E user journey
npm run test:mixed

# Spike test
npm run test:stress

# Soak test (30 minutes)
npm run test:mixed:soak
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8001` | Auth service URL |
| `REGISTRY_URL` | `http://localhost:8002` | Registry service URL |
| `JOB_URL` | `http://localhost:8003` | Job service URL |
| `STORAGE_URL` | `http://localhost:8004` | Storage service URL |
| `NOTIFICATION_URL` | `http://localhost:8005` | Notification service URL |
| `TEST_USER_EMAIL` | `loadtest@example.com` | Test user email |
| `TEST_USER_PASSWORD` | `TestPassword123!` | Test user password |

### Service Thresholds

Default performance thresholds (configurable per scenario):

| Service | P95 | P99 | Max Error Rate |
|---------|-----|-----|----------------|
| Auth | 300ms | 500ms | 1% |
| Registry | 200ms | 400ms | 1% |
| Job | 500ms | 1000ms | 1% |
| Storage | 2000ms | 5000ms | 1% |
| Notification | 500ms | N/A | 5% |

## Reporting

### Generate HTML Report

```bash
npm run report:html
```

### Compare Against Baseline

```bash
npm run report:compare
```

## CI Integration

### GitHub Actions

Workflows are defined in `.github/workflows/load-test.yml`.

- Runs on push to main/develop
- Runs on pull requests
- Runs daily (schedule)
- Can be triggered manually

### GitLab CI

Pipeline defined in `.gitlab-ci.yml`.

## Kubernetes Deployment

Using k6-operator:

```bash
kubectl apply -f k8s/k6-operator/
```

This creates:
- Namespace: `load-testing`
- ConfigMap with test scripts
- K6 resources for baseline and stress tests

## Docker Test Environment

```bash
# Start all services
docker-compose -f docker-compose.test.yml up -d

# View logs
docker-compose -f docker-compose.test.yml logs -f

# Stop and cleanup
docker-compose -f docker-compose.test.yml down -v
```

Includes:
- PostgreSQL (port 15433)
- Redis (port 16380)
- MinIO (ports 9001, 9002)
- Grafana (port 3001)
- InfluxDB (port 8086)
- Prometheus (port 9091)

## Performance Baselines (FR-007, FR-008)

Baselines are stored in `.baseline/metrics.json` and used for regression detection.

To update baselines after improvements:

```bash
npm run test:baseline
node scripts/generate-baseline.js
git add .baseline/metrics.json
git commit -m "Update performance baseline"
```

## Dashboard (FR-009)

Access Grafana at `http://localhost:3001` (admin/admin) after starting the test environment.

Pre-configured dashboards:
- Response times by service
- Error rates
- Throughput metrics
- VU progression

## Troubleshooting

### Tests fail to connect
- Ensure services are running: `docker-compose -f docker-compose.test.yml ps`
- Check service logs
- Verify environment variables

### Out of memory errors
- Reduce VUs in test configuration
- Increase Docker memory limits
- Run tests on separate machines

### Connection timeouts
- Increase timeout in test scenario
- Check network latency between test runner and services
- Verify service health endpoints
