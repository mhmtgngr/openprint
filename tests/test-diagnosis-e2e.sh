#!/bin/bash
# test-diagnosis-e2e.sh - E2E tests for dev.sh diagnosis functionality
#
# End-to-end tests for the diagnose_project function and diagnosis.json output:
# - Accurate TSX/TS file enumeration
# - Correct feature breakdown
# - Proper JSON structure
# - Integration with full dev.sh pipeline
#
# Usage:
#   ./tests/test-diagnosis-e2e.sh              # Run all E2E tests
#   ./tests/test-diagnosis-e2e.sh --verbose    # Run with detailed output
#   ./tests/test-diagnosis-e2e.sh --help       # Show help

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
VERBOSE=false

# Project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ARTIFACTS_DIR="$REPO_DIR/.team/artifacts"
DIAGNOSIS_FILE="$ARTIFACTS_DIR/diagnosis.json"

# Expected file counts (based on actual codebase)
EXPECTED_TSX_COUNT=80
EXPECTED_TS_COUNT=33
EXPECTED_TOTAL_COUNT=113

# Expected features
EXPECTED_FEATURES="dashboard documents printers auth agents settings jobs devices"

# Functions

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Extract JSON value from diagnosis file
get_diagnosis_value() {
    local key="$1"
    if [ -f "$DIAGNOSIS_FILE" ]; then
        python3 -c "import json,sys; d=json.load(open('$DIAGNOSIS_FILE')); print(d.get('project',{}).get('$key',''))" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Get nested value from diagnosis
get_diagnosis_nested() {
    local path="$1"  # e.g., "compile_errors"
    if [ -f "$DIAGNOSIS_FILE" ]; then
        python3 -c "import json,sys; d=json.load(open('$DIAGNOSIS_FILE')); print(d.get('$path',''))" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Assert functions
assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="${3:-Expected $expected, got $actual}"

    TESTS_RUN=$((TESTS_RUN + 1))

    if [ "$expected" = "$actual" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "$message"
        [ "$VERBOSE" = true ] && echo "  Expected: $expected, Actual: $actual"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "$message"
        echo "  Expected: $expected"
        echo "  Actual: $actual"
        return 1
    fi
}

assert_not_zero() {
    local value="$1"
    local message="${2:-Value should not be zero}"

    TESTS_RUN=$((TESTS_RUN + 1))

    if [ "$value" -gt 0 ] 2>/dev/null; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "$message"
        [ "$VERBOSE" = true ] && echo "  Value: $value"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "$message"
        echo "  Value: $value"
        return 1
    fi
}

assert_file_exists() {
    local file="$1"
    local message="${2:-File should exist: $file}"

    TESTS_RUN=$((TESTS_RUN + 1))

    if [ -f "$file" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "$message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "$message"
        return 1
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Expected '$needle' in result}"

    TESTS_RUN=$((TESTS_RUN + 1))

    if echo "$haystack" | grep -q "$needle"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "$message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "$message"
        return 1
    fi
}

assert_not_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Should not contain '$needle'}"

    TESTS_RUN=$((TESTS_RUN + 1))

    if ! echo "$haystack" | grep -q "$needle"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "$message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "$message"
        return 1
    fi
}

assert_valid_json() {
    local file="$1"
    local message="${2:-File should be valid JSON}"

    TESTS_RUN=$((TESTS_RUN + 1))

    if python3 -m json.tool "$file" >/dev/null 2>&1; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "$message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "$message"
        echo "  File: $file"
        return 1
    fi
}

# Test: Run diagnose_project and verify output
test_diagnosis_runs_successfully() {
    local test_name="diagnosis_runs_successfully"
    log_info "Running: $test_name"

    # Run the diagnose command from dev.sh
    cd "$REPO_DIR"
    bash -c '. dev.sh && diagnose_project' >/dev/null 2>&1

    assert_file_exists "$DIAGNOSIS_FILE" "diagnosis.json should be created"
}

# Test: Verify diagnosis.json is valid JSON
test_diagnosis_is_valid_json() {
    local test_name="diagnosis_is_valid_json"
    log_info "Running: $test_name"

    assert_file_exists "$DIAGNOSIS_FILE" "diagnosis.json should exist"
    assert_valid_json "$DIAGNOSIS_FILE" "diagnosis.json should be valid JSON"
}

# Test: Verify tsx_files count
test_tsx_files_count() {
    local test_name="tsx_files_count"
    log_info "Running: $test_name"

    local tsx_files
    tsx_files=$(get_diagnosis_value "tsx_files")

    assert_equals "$EXPECTED_TSX_COUNT" "$tsx_files" "tsx_files should be $EXPECTED_TSX_COUNT"
}

# Test: Verify ts_files count
test_ts_files_count() {
    local test_name="ts_files_count"
    log_info "Running: $test_name"

    local ts_files
    ts_files=$(get_diagnosis_value "ts_files")

    assert_equals "$EXPECTED_TS_COUNT" "$ts_files" "ts_files should be $EXPECTED_TS_COUNT"
}

# Test: Verify frontend_total count
test_frontend_total_count() {
    local test_name="frontend_total_count"
    log_info "Running: $test_name"

    local frontend_total
    frontend_total=$(get_diagnosis_value "frontend_total")

    assert_equals "$EXPECTED_TOTAL_COUNT" "$frontend_total" "frontend_total should be $EXPECTED_TOTAL_COUNT"
}

# Test: Verify frontend_total equals tsx_files + ts_files
test_frontend_total_calculation() {
    local test_name="frontend_total_calculation"
    log_info "Running: $test_name"

    local tsx_files ts_files frontend_total expected_total
    tsx_files=$(get_diagnosis_value "tsx_files")
    ts_files=$(get_diagnosis_value "ts_files")
    frontend_total=$(get_diagnosis_value "frontend_total")
    expected_total=$((tsx_files + ts_files))

    assert_equals "$expected_total" "$frontend_total" "frontend_total should equal tsx_files + ts_files"
}

# Test: Verify go_files count is positive
test_go_files_count() {
    local test_name="go_files_count"
    log_info "Running: $test_name"

    local go_files
    go_files=$(get_diagnosis_value "go_files")

    assert_not_zero "$go_files" "go_files should be greater than zero"
}

# Test: Verify test_files count is positive
test_test_files_count() {
    local test_name="test_test_files_count"
    log_info "Running: $test_name"

    local test_files
    test_files=$(get_diagnosis_value "test_files")

    assert_not_zero "$test_files" "test_files should be greater than zero"
}

# Test: Verify features breakdown exists
test_features_breakdown_exists() {
    local test_name="features_breakdown_exists"
    log_info "Running: $test_name"

    local features
    features=$(get_diagnosis_value "features")

    # Check that features is not empty
    TESTS_RUN=$((TESTS_RUN + 1))
    if [ -n "$features" ] && [ "$features" != "{}" ] && [ "$features" != "" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "features breakdown exists and is not empty"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "features breakdown is missing or empty"
    fi
}

# Test: Verify all expected features are present
test_expected_features_present() {
    local test_name="expected_features_present"
    log_info "Running: $test_name"

    local features
    features=$(get_diagnosis_value "features")

    for feature in $EXPECTED_FEATURES; do
        assert_contains "$features" "\"$feature\"" "Feature '$feature' should be in breakdown"
    done
}

# Test: Verify diagnosis contains all required fields
test_diagnosis_required_fields() {
    local test_name="diagnosis_required_fields"
    log_info "Running: $test_name"

    local required_fields="go_files test_files tsx_files ts_files frontend_total todo_count build tests test_count test_passed dockerfiles compose frontend"

    for field in $required_fields; do
        local value
        value=$(get_diagnosis_value "$field")
        TESTS_RUN=$((TESTS_RUN + 1))
        if [ -n "$value" ]; then
            TESTS_PASSED=$((TESTS_PASSED + 1))
            log_success "Field '$field' exists in diagnosis"
        else
            TESTS_FAILED=$((TESTS_FAILED + 1))
            log_error "Field '$field' is missing from diagnosis"
        fi
    done
}

# Test: Verify actual file counts match filesystem
test_file_counts_match_filesystem() {
    local test_name="file_counts_match_filesystem"
    log_info "Running: $test_name"

    # Count actual files on filesystem
    local actual_tsx actual_ts
    actual_tsx=$(find "$REPO_DIR/web/dashboard/src" -type f -name "*.tsx" ! -path "*/node_modules/*" ! -path "*/dist/*" ! -path "*/build/*" 2>/dev/null | wc -l)
    actual_ts=$(find "$REPO_DIR/web/dashboard/src" -type f -name "*.ts" ! -name "*.tsx" ! -path "*/node_modules/*" ! -path "*/dist/*" ! -path "*/build/*" 2>/dev/null | wc -l)

    # Clean up whitespace
    actual_tsx="${actual_tsx//[^0-9]/}"
    actual_ts="${actual_ts//[^0-9]/}"

    local diagnosed_tsx diagnosed_ts
    diagnosed_tsx=$(get_diagnosis_value "tsx_files")
    diagnosed_ts=$(get_diagnosis_value "ts_files")

    assert_equals "$actual_tsx" "$diagnosed_tsx" "Diagnosed TSX count should match filesystem"
    assert_equals "$actual_ts" "$diagnosed_ts" "Diagnosed TS count should match filesystem"
}

# Test: Verify no node_modules files counted
test_no_node_modules_counted() {
    local test_name="no_node_modules_counted"
    log_info "Running: $test_name"

    # Verify features JSON doesn't contain node_modules paths
    local features
    features=$(get_diagnosis_value "features")

    assert_not_contains "$features" "node_modules" "features should not contain node_modules references"
}

# Test: Verify dockerfiles count
test_dockerfiles_count() {
    local test_name="dockerfiles_count"
    log_info "Running: $test_name"

    local actual_dockerfiles
    actual_dockerfiles=$(find "$REPO_DIR/deployments/docker" -maxdepth 1 -name "Dockerfile.*" 2>/dev/null | wc -l)
    actual_dockerfiles="${actual_dockerfiles//[^0-9]/}"

    local diagnosed_dockerfiles
    diagnosed_dockerfiles=$(get_diagnosis_value "dockerfiles")

    assert_equals "$actual_dockerfiles" "$diagnosed_dockerfiles" "Dockerfiles count should match filesystem"
}

# Test: Verify scan_frontend_files function output matches diagnosis
test_scan_function_matches_diagnosis() {
    local test_name="scan_function_matches_diagnosis"
    log_info "Running: $test_name"

    # Extract and run scan_frontend_files function
    local scan_output
    scan_output=$(bash -c '
        REPO_DIR="'"$REPO_DIR"'"
        FRONTEND_DIR="web/dashboard"
        # Extract function
        start_line=$(grep -n "^scan_frontend_files()" "$REPO_DIR/dev.sh" | cut -d: -f1)
        end_line=$(tail -n "+$start_line" "$REPO_DIR/dev.sh" | grep -n "^}" | head -1 | cut -d: -f1)
        end_line=$((start_line + end_line - 1))
        func_body=$(sed -n "${start_line},${end_line}p" "$REPO_DIR/dev.sh")
        eval "$func_body"
        scan_frontend_files "$REPO_DIR/web/dashboard/src"
    ' 2>/dev/null)

    local scan_tsx scan_ts
    scan_tsx=$(echo "$scan_output" | python3 -c "import json,sys; print(json.load(sys.stdin).get('tsx_count',0))" 2>/dev/null || echo "0")
    scan_ts=$(echo "$scan_output" | python3 -c "import json,sys; print(json.load(sys.stdin).get('ts_count',0))" 2>/dev/null || echo "0")

    local diag_tsx diag_ts
    diag_tsx=$(get_diagnosis_value "tsx_files")
    diag_ts=$(get_diagnosis_value "ts_files")

    assert_equals "$scan_tsx" "$diag_tsx" "scan_frontend_files tsx_count should match diagnosis tsx_files"
    assert_equals "$scan_ts" "$diag_ts" "scan_frontend_files ts_count should match diagnosis ts_files"
}

# Print usage
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

End-to-end tests for dev.sh diagnosis functionality

OPTIONS:
    -v, --verbose    Show detailed output for each test
    -h, --help       Show this help message

TEST CASES:
    diagnosis_runs_successfully         Ensures diagnose_project creates diagnosis.json
    diagnosis_is_valid_json             Ensures diagnosis.json is valid JSON
    tsx_files_count                     Verifies TSX file count (80)
    ts_files_count                      Verifies TS file count (33)
    frontend_total_count                Verifies total frontend count (113)
    frontend_total_calculation          Verifies total = tsx + ts
    go_files_count                      Verifies Go files counted
    test_test_files_count               Verifies test files counted
    features_breakdown_exists           Ensures feature breakdown present
    expected_features_present           Ensures all 8 features listed
    diagnosis_required_fields           Ensures all required fields present
    file_counts_match_filesystem        Verifies counts match actual files
    no_node_modules_counted             Ensures node_modules excluded
    dockerfiles_count                   Verifies Dockerfiles counted
    scan_function_matches_diagnosis     Ensures function matches output

EOF
}

# Main test runner

main() {
    local start_time
    start_time=$(date +%s)

    # Parse arguments
    while [ $# -gt 0 ]; do
        case "$1" in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    echo "=========================================="
    echo "Diagnosis E2E Tests"
    echo "=========================================="
    echo "Repository: $REPO_DIR"
    echo "Artifacts: $ARTIFACTS_DIR"
    echo "=========================================="
    echo ""

    # Ensure artifacts directory exists
    mkdir -p "$ARTIFACTS_DIR"

    # Run all tests
    test_diagnosis_runs_successfully
    test_diagnosis_is_valid_json
    test_tsx_files_count
    test_ts_files_count
    test_frontend_total_count
    test_frontend_total_calculation
    test_go_files_count
    test_test_files_count
    test_features_breakdown_exists
    test_expected_features_present
    test_diagnosis_required_fields
    test_file_counts_match_filesystem
    test_no_node_modules_counted
    test_dockerfiles_count
    test_scan_function_matches_diagnosis

    # Print summary
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    echo "  Total:   $TESTS_RUN"
    echo -e "  ${GREEN}Passed:  $TESTS_PASSED${NC}"
    [ "$TESTS_FAILED" -gt 0 ] && echo -e "  ${RED}Failed:  $TESTS_FAILED${NC}"
    echo "  Duration: ${duration}s"
    echo "=========================================="

    if [ "$TESTS_FAILED" -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Run main if script is executed
main "$@"
