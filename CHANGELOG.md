# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Documentation**: Comprehensive project documentation
  - `.env.example` for environment configuration
  - `CONTRIBUTING.md` with development guidelines
  - `SECURITY.md` with security policy
  - `CHANGELOG.md` for version tracking
  - `AGENTS.md` for AI assistant instructions
- **Services**: Extended service architecture
  - API Gateway with reverse proxy and rate limiting
  - Analytics service for usage reporting
  - Compliance service (FedRAMP, HIPAA, GDPR, SOC2)
  - Organization service for multi-tenancy
  - Policy service for print rules engine
  - M365 integration service

### Changed
- **Documentation**: Expanded CLAUDE.md with all 12 services
- **Database**: Fixed reserved keyword 'limit' in rate limit tables migration

## [0.1.0] - 2024-01-15

### Added
- **Core Services**: Initial release with 5 microservices
  - Auth service with JWT authentication and session management
  - Registry service for printer and agent registration
  - Job service for print job processing and routing
  - Storage service for document management (S3 and local)
  - Notification service for WebSocket real-time updates
- **Frontend**: React dashboard with TypeScript and Vite
  - Dashboard with print job monitoring
  - Printer management interface
  - Analytics and reporting views
  - Organization management
- **Database**: PostgreSQL 16 schema with 28 migrations
  - Organizations and users tables
  - Printers, agents, and jobs tables
  - Quotas, policies, and compliance tables
  - Rate limiting and audit tables
- **Infrastructure**: Docker Compose setup
  - PostgreSQL 16 container
  - Redis 7 container
  - Service containers with health checks
  - Volume persistence
- **Internal Packages**: Shared utilities
  - Authentication (JWT, OIDC, SAML, password hashing)
  - Middleware (auth, CORS, logging, recovery)
  - Multi-tenancy support
  - Test utilities with testcontainers
- **Testing**: Comprehensive test suite
  - Unit tests for all core services
  - Integration tests with testcontainers
  - E2E tests for dashboard
  - Load testing with k6
- **Security**: Security features
  - JWT-based authentication
  - Role-based access control
  - Rate limiting
  - Input validation
  - CORS protection
  - Audit logging

### Security
- JWT tokens with 24-hour expiry
- bcrypt password hashing
- TLS 1.3 support
- Rate limiting per IP, user, and API key
- SQL injection protection via parameterized queries

### Infrastructure
- Docker Compose for local development
- Health checks for all services
- Volume persistence for data
- Network isolation
- Environment-based configuration

## Version History

- **0.1.0** (2024-01-15) - Initial release with core 5 services
- **0.2.0** (Unreleased) - Extended architecture with 7 additional services

[Unreleased]: https://github.com/openprint/openprint/compare/v0.2.0...main
