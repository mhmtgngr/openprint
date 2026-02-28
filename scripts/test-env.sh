#!/bin/bash
# test-env.sh - Bootstrap script to check Docker availability and setup test environment
#
# This script verifies that the test environment is properly configured:
# - Docker is installed and running
# - Required Docker images are available
# - Ports are available for test containers
# - Environment variables are set
#
# Usage:
#   source scripts/test-env.sh
#   ./scripts/test-env.sh check
#   ./scripts/test-env.sh setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
POSTGRES_IMAGE="postgres:16-alpine"
REDIS_IMAGE="redis:7-alpine"
MINIO_IMAGE="minio/minio:latest"
GO_VERSION="1.24.0"

# Functions

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_docker() {
    info "Checking Docker installation..."

    if ! command -v docker &> /dev/null; then
        error "Docker is not installed. Please install Docker to run tests with containers."
        info "Visit https://docs.docker.com/get-docker/ for installation instructions."
        return 1
    fi

    success "Docker is installed: $(docker --version)"

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        error "Docker daemon is not running. Please start Docker."
        return 1
    fi

    success "Docker daemon is running"

    return 0
}

check_docker_compose() {
    info "Checking Docker Compose..."

    if docker compose version &> /dev/null; then
        success "Docker Compose is available: $(docker compose version)"
        return 0
    fi

    if command -v docker-compose &> /dev/null; then
        success "Docker Compose is available: $(docker-compose --version)"
        return 0
    fi

    warn "Docker Compose not found. It is optional but recommended."
    return 0
}

check_go() {
    info "Checking Go installation..."

    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install Go $GO_VERSION or later."
        info "Visit https://golang.org/dl/ for installation instructions."
        return 1
    fi

    success "Go is installed: $(go version)"

    # Check Go version
    GO_CURRENT=$(go version | awk '{print $3}' | sed 's/go//')
    info "Current Go version: $GO_CURRENT"

    return 0
}

check_ports() {
    info "Checking if default test ports are available..."

    local ports=(5432 6379 9000)
    local available=true

    for port in "${ports[@]}"; do
        if lsof -i :"$port" &> /dev/null || netstat -an 2>/dev/null | grep ":$port " &> /dev/null; then
            warn "Port $port is already in use. Tests will use random ports."
            available=false
        fi
    done

    if $available; then
        success "All default test ports are available"
    fi

    return 0
}

pull_test_images() {
    info "Pulling required Docker images for testing..."

    info "Pulling PostgreSQL image..."
    docker pull "$POSTGRES_IMAGE" || {
        warn "Failed to pull PostgreSQL image. It will be downloaded during tests."
    }

    info "Pulling Redis image..."
    docker pull "$REDIS_IMAGE" || {
        warn "Failed to pull Redis image. It will be downloaded during tests."
    }

    info "Pulling MinIO image..."
    docker pull "$MINIO_IMAGE" || {
        warn "Failed to pull MinIO image. It will be downloaded during tests."
    }

    success "Docker images are ready"
}

setup_env_vars() {
    info "Setting up test environment variables..."

    export TEST_MODE=true
    export LOG_LEVEL=debug

    # Set test-specific environment variables
    export DATABASE_URL="postgres://testuser:testpass@localhost:5432/openprint_test?sslmode=disable"
    export REDIS_URL="redis://localhost:6379/0"
    export S3_ENDPOINT="http://localhost:9000"
    export S3_BUCKET="test-bucket"
    export AWS_ACCESS_KEY_ID="minioadmin"
    export AWS_SECRET_ACCESS_KEY="minioadmin"
    export AWS_REGION="us-east-1"
    export JWT_SECRET="test-secret-key-min-32-chars-for-testing"

    success "Environment variables set"
    env | grep -E "(TEST_MODE|DATABASE_URL|REDIS_URL|S3_|AWS_|JWT_)" | sed 's/^/  /'
}

run_health_checks() {
    info "Running test environment health checks..."

    local all_good=true

    # Check if we can reach Docker
    if ! docker ps &> /dev/null; then
        error "Cannot communicate with Docker daemon"
        all_good=false
    fi

    # Check Go modules
    if [ -f "go.mod" ]; then
        info "Checking Go modules..."
        if ! go mod download; then
            warn "Failed to download Go modules"
            all_good=false
        else
            success "Go modules are downloaded"
        fi
    fi

    if $all_good; then
        success "All health checks passed"
    else
        warn "Some health checks failed. Tests may still work but could be slower."
    fi

    return 0
}

cleanup_test_containers() {
    info "Cleaning up any orphaned test containers..."

    # Remove testcontainers containers
    docker ps -a --filter "name=testcontainers" --format "{{.Names}}" | while read -r container; do
        info "Removing container: $container"
        docker rm -f "$container" 2>/dev/null || true
    done

    # Remove Ryuk containers (testcontainers cleanup)
    docker ps -a --filter "name=ryuk" --format "{{.Names}}" | while read -r container; do
        info "Removing Ryuk container: $container"
        docker rm -f "$container" 2>/dev/null || true
    done

    success "Cleanup complete"
}

print_usage() {
    cat << EOF
Usage: $0 <command>

Commands:
  check       Check if the test environment is properly configured
  setup       Set up the test environment (pull images, set env vars)
  cleanup     Clean up orphaned test containers
  env         Print environment variables for tests
  help        Show this help message

Examples:
  $0 check
  $0 setup
  source scripts/test-env.sh

EOF
}

print_summary() {
    cat << EOF

${GREEN}Test Environment Summary${NC}
================================

Docker:        $(docker --version 2>/dev/null || echo "Not installed")
Go:            $(go version 2>/dev/null | awk '{print $0}' || echo "Not installed")
Test Mode:     ${TEST_MODE:-false}
Database:      ${DATABASE_URL:-not set}
Redis:         ${REDIS_URL:-not set}
S3 Endpoint:   ${S3_ENDPOINT:-not set}

For more information, run: $0 help

EOF
}

# Main command dispatcher

main() {
    local command="${1:-check}"

    case "$command" in
        check)
            check_docker || exit 1
            check_docker_compose
            check_go || exit 1
            check_ports
            run_health_checks
            print_summary
            ;;
        setup)
            check_docker || exit 1
            check_go || exit 1
            pull_test_images
            setup_env_vars
            print_summary
            ;;
        cleanup)
            check_docker || exit 1
            cleanup_test_containers
            ;;
        env)
            setup_env_vars
            ;;
        help|--help|-h)
            print_usage
            ;;
        *)
            error "Unknown command: $command"
            print_usage
            exit 1
            ;;
    esac
}

# Run main if script is executed (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    # Script is being sourced - set up environment
    setup_env_vars
fi
