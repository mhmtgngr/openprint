# OpenPrint Cloud - Makefile
#
# This Makefile provides targets for building, testing, and deploying
# the OpenPrint Cloud services.
#
# Prerequisites:
#   - Go 1.24.0 or later
#   - Docker (for integration tests)
#   - Docker Compose (optional, for local development)
#
# Usage:
#   make test              # Run all tests
#   make test-unit         # Run unit tests only
#   make test-integration  # Run integration tests
#   make build             # Build all services
#   make run               # Run all services locally

# Variables
GO := go
GOFLAGS := -v
DOCKER := docker
DOCKER_COMPOSE := docker compose

# Project configuration
MODULE := github.com/openprint/openprint
SERVICES := auth-service registry-service job-service storage-service notification-service analytics-service organization-service policy-service compliance-service m365-integration-service api-gateway

# Test configuration
TEST_TIMEOUT := 10m
TEST_RACE := true
TEST_COVERAGE := true
TEST_VERBOSE := false

# Build configuration
BUILD_DIR := ./build
BINARY_PREFIX := openprint

# Docker configuration
DOCKER_REGISTRY := docker.io
DOCKER_IMAGE_PREFIX := openprint

# Color output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)OpenPrint Cloud - Available Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

# ============================================================================
# Development Targets
# ============================================================================

.PHONY: deps
deps: ## Download Go module dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod verify
	@echo "$(GREEN)Dependencies ready$(NC)"

.PHONY: tidy
tidy: ## Tidy Go module dependencies
	@echo "$(BLUE)Tidying dependencies...$(NC)"
	$(GO) mod tidy

.PHONY: clean
clean: ## Clean build artifacts and cache
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.txt coverage_*.txt
	rm -rf /tmp/openprint-test-*
	$(GO) clean -cache -testcache
	@echo "$(GREEN)Clean complete$(NC)"

.PHONY: format
format: ## Format Go source code
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GO) fmt ./...
	@echo "$(GREEN)Formatting complete$(NC)"

.PHONY: lint
lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not found. Install with:$(NC)";
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin";
	fi

# ============================================================================
# Build Targets
# ============================================================================

.PHONY: build
build: ## Build all services
	@echo "$(BLUE)Building all services...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$$service ./services/$$service; \
	done
	@echo "$(GREEN)Build complete$(NC)"

.PHONY: build-auth-service
build-auth-service: ## Build auth service
	@echo "$(BLUE)Building auth-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/auth-service ./services/auth-service

.PHONY: build-registry-service
build-registry-service: ## Build registry service
	@echo "$(BLUE)Building registry-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/registry-service ./services/registry-service

.PHONY: build-job-service
build-job-service: ## Build job service
	@echo "$(BLUE)Building job-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/job-service ./services/job-service

.PHONY: build-storage-service
build-storage-service: ## Build storage service
	@echo "$(BLUE)Building storage-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/storage-service ./services/storage-service

.PHONY: build-notification-service
build-notification-service: ## Build notification service
	@echo "$(BLUE)Building notification-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/notification-service ./services/notification-service

.PHONY: build-analytics-service
build-analytics-service: ## Build analytics service
	@echo "$(BLUE)Building analytics-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/analytics-service ./services/analytics-service

.PHONY: build-organization-service
build-organization-service: ## Build organization service
	@echo "$(BLUE)Building organization-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/organization-service ./services/organization-service

.PHONY: build-policy-service
build-policy-service: ## Build policy service
	@echo "$(BLUE)Building policy-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/policy-service ./services/policy-service

.PHONY: build-compliance-service
build-compliance-service: ## Build compliance service
	@echo "$(BLUE)Building compliance-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/compliance-service ./services/compliance-service

.PHONY: build-m365-integration-service
build-m365-integration-service: ## Build M365 integration service
	@echo "$(BLUE)Building m365-integration-service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/m365-integration-service ./services/m365-integration-service

.PHONY: build-api-gateway
build-api-gateway: ## Build API gateway
	@echo "$(BLUE)Building api-gateway...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/api-gateway ./services/api-gateway

# ============================================================================
# Test Targets
# ============================================================================

.PHONY: test
test: ## Run all tests
	@echo "$(BLUE)Running all tests...$(NC)"
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@echo "$(GREEN)All tests passed$(NC)"

.PHONY: test-unit
test-unit: ## Run unit tests (without //+build integration tag)
	@echo "$(BLUE)Running unit tests...$(NC)"
	$(GO) test \
		-short \
		-timeout $(TEST_TIMEOUT) \
		./... \
		$(if $(filter true,$(TEST_RACE)),-race,) \
		$(if $(filter true,$(TEST_COVERAGE)),-coverprofile=coverage_unit.txt,) \
		$(if $(filter true,$(TEST_VERBOSE)),-v,)
	@echo "$(GREEN)Unit tests passed$(NC)"

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	@echo "$(BLUE)Running integration tests...$(NC)"
	@if ! docker info >/dev/null 2>&1; then \
		echo "$(RED)Error: Docker is not running. Required for integration tests.$(NC)"; \
		exit 1; \
	fi
	$(GO) test \
		-tags=integration \
		-timeout $(TEST_TIMEOUT) \
		./... \
		$(if $(filter true,$(TEST_RACE)),-race,) \
		$(if $(filter true,$(TEST_COVERAGE)),-coverprofile=coverage_integration.txt,) \
		$(if $(filter true,$(TEST_VERBOSE)),-v,)
	@echo "$(GREEN)Integration tests passed$(NC)"

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	$(GO) test \
		-race \
		-timeout $(TEST_TIMEOUT) \
		./...
	@echo "$(GREEN)Race tests passed$(NC)"

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GO) test \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-timeout $(TEST_TIMEOUT) \
		./...
	@$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: test-cover-func
test-cover-func: ## Show coverage by function
	@echo "$(BLUE)Coverage by function:$(NC)"
	$(GO) test -coverprofile=coverage.txt ./... >/dev/null 2>&1 || true
	$(GO) tool cover -func=coverage.txt | sort -t '%' -k 2 -r | head -20

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "$(BLUE)Running tests (verbose)...$(NC)"
	$(GO) test -v -timeout $(TEST_TIMEOUT) ./...

.PHONY: test-auth-service
test-auth-service: ## Run auth service tests
	@echo "$(BLUE)Running auth-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/auth-service/...

.PHONY: test-registry-service
test-registry-service: ## Run registry service tests
	@echo "$(BLUE)Running registry-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/registry-service/...

.PHONY: test-job-service
test-job-service: ## Run job service tests
	@echo "$(BLUE)Running job-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/job-service/...

.PHONY: test-storage-service
test-storage-service: ## Run storage service tests
	@echo "$(BLUE)Running storage-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/storage-service/...

.PHONY: test-notification-service
test-notification-service: ## Run notification service tests
	@echo "$(BLUE)Running notification-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/notification-service/...

.PHONY: test-analytics-service
test-analytics-service: ## Run analytics service tests
	@echo "$(BLUE)Running analytics-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/analytics-service/...

.PHONY: test-organization-service
test-organization-service: ## Run organization service tests
	@echo "$(BLUE)Running organization-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/organization-service/...

.PHONY: test-policy-service
test-policy-service: ## Run policy service tests
	@echo "$(BLUE)Running policy-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/policy-service/...

.PHONY: test-compliance-service
test-compliance-service: ## Run compliance service tests
	@echo "$(BLUE)Running compliance-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/compliance-service/...

.PHONY: test-m365-integration-service
test-m365-integration-service: ## Run M365 integration service tests
	@echo "$(BLUE)Running m365-integration-service tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/m365-integration-service/...

.PHONY: test-api-gateway
test-api-gateway: ## Run API gateway tests
	@echo "$(BLUE)Running api-gateway tests...$(NC)"
	$(GO) test -timeout $(TEST_TIMEOUT) ./services/api-gateway/...

.PHONY: test-bench
test-bench: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	$(GO) test -bench=. -benchmem -timeout 20m ./...

.PHONY: test-env
test-env: ## Check test environment
	@echo "$(BLUE)Checking test environment...$(NC)"
	@./scripts/test-env.sh check

.PHONY: test-env-setup
test-env-setup: ## Set up test environment
	@echo "$(BLUE)Setting up test environment...$(NC)"
	@./scripts/test-env.sh setup

.PHONY: test-cleanup
test-cleanup: ## Clean up test containers
	@echo "$(BLUE)Cleaning up test containers...$(NC)"
	@./scripts/test-env.sh cleanup

# ============================================================================
# Docker Targets
# ============================================================================

.PHONY: docker-build
docker-build: ## Build all Docker images
	@echo "$(BLUE)Building Docker images...$(NC)"
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		$(DOCKER) build -f deployments/docker/Dockerfile.$$service -t $(DOCKER_IMAGE_PREFIX)/$$service:latest .; \
	done
	@echo "$(GREEN)Docker images built$(NC)"

.PHONY: docker-run
docker-run: ## Run all services in Docker
	@echo "$(BLUE)Starting services in Docker...$(NC)"
	cd deployments/docker && $(DOCKER_COMPOSE) up -d

.PHONY: docker-stop
docker-stop: ## Stop Docker services
	@echo "$(BLUE)Stopping Docker services...$(NC)"
	cd deployments/docker && $(DOCKER_COMPOSE) down

.PHONY: docker-logs
docker-logs: ## Show Docker service logs
	cd deployments/docker && $(DOCKER_COMPOSE) logs -f

.PHONY: docker-ps
docker-ps: ## Show running Docker containers
	cd deployments/docker && $(DOCKER_COMPOSE) ps

# ============================================================================
# Database Targets
# ============================================================================

.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(NC)"
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path migrations -database "$$DATABASE_URL" up; \
	else \
		echo "$(YELLOW)migrate tool not found. Install with:$(NC)"; \
		echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
	fi

.PHONY: db-migrate-down
db-migrate-down: ## Rollback database migrations
	@echo "$(BLUE)Rolling back database migrations...$(NC)"
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path migrations -database "$$DATABASE_URL" down 1; \
	else \
		echo "$(YELLOW)migrate tool not found$(NC)"; \
	fi

.PHONY: db-reset
db-reset: ## Reset database (drop and recreate)
	@echo "$(YELLOW)This will delete all data. Are you sure?$(NC)"
	@read -p "Type 'yes' to confirm: " confirm; \
	[ "$$confirm" = "yes" ] || (echo "Aborted"; exit 1)
	@echo "$(BLUE)Resetting database...$(NC)"
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path migrations -database "$$DATABASE_URL" down -all; \
		migrate -path migrations -database "$$DATABASE_URL" up; \
	fi
	@echo "$(GREEN)Database reset complete$(NC)"

# ============================================================================
# CI/CD Targets
# ============================================================================

.PHONY: ci
ci: ## Run CI pipeline checks
	@echo "$(BLUE)Running CI pipeline...$(NC)"
	@$(MAKE) test-env
	@$(MAKE) format
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) build
	@echo "$(GREEN)CI pipeline passed$(NC)"

.PHONY: ci-test
ci-test: ## Run CI tests (for GitHub Actions)
	@echo "$(BLUE)Running CI tests...$(NC)"
	$(GO) test \
		-race \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-timeout $(TEST_TIMEOUT) \
		./...

# ============================================================================
# Utility Targets
# ============================================================================

.PHONY: run
run: ## Run all services locally
	@echo "$(BLUE)Starting all services...$(NC)"
	@for service in $(SERVICES); do \
		echo "Starting $$service..."; \
		$(BUILD_DIR)/$$service & \
	done
	@echo "$(GREEN)All services started$(NC)"
	@echo "Press Ctrl+C to stop all services"

.PHONY: dev
dev: ## Run development environment with Docker Compose
	@echo "$(BLUE)Starting development environment...$(NC)"
	cd deployments/docker && $(DOCKER_COMPOSE) up

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@echo "Installing golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	@echo "Installing migrate..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Installing mockgen..."
	@go install github.com/golang/mock/mockgen@latest
	@echo "$(GREEN)Development tools installed$(NC)"

.PHONY: check-deps
check-deps: ## Check if required dependencies are installed
	@echo "$(BLUE)Checking dependencies...$(NC)"
	@command -v go >/dev/null 2>&1 || (echo "$(RED)Go is not installed$(NC)"; exit 1)
	@command -v docker >/dev/null 2>&1 || echo "$(YELLOW)Docker is not installed (required for integration tests)$(NC)"
	@command -v git >/dev/null 2>&1 || echo "$(YELLOW)Git is not installed$(NC)"
	@echo "$(GREEN)Dependency check complete$(NC)"

# ============================================================================
# Variables export
# ============================================================================

export TEST_MODE ?= true
export LOG_LEVEL ?= debug

# Include environment file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif
