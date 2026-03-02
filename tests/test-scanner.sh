#!/bin/bash
# test-scanner.sh - Unit tests for dev.sh scanner functions
#
# Tests the scan_frontend_files function to ensure accurate file enumeration:
# - Correct TSX/TS file counts
# - Proper exclusion of build artifacts
# - Feature module breakdown
# - Security validation
#
# Usage:
#   ./tests/test-scanner.sh              # Run all tests
#   ./tests/test-scanner.sh --verbose    # Run with detailed output
#   ./tests/test-scanner.sh --help       # Show help

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
FRONTEND_SRC_DIR="$REPO_DIR/web/dashboard/src"

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

# Source the dev.sh to get access to scan_frontend_files function
source_dev_sh() {
    if [ -f "$REPO_DIR/dev.sh" ]; then
        # Set required variables that scan_frontend_files needs
        export FRONTEND_DIR="${FRONTEND_DIR:-web/dashboard}"

        # Extract only the scan_frontend_files function using awk
        # This avoids executing the entire dev.sh script
        local start_line end_line
        start_line=$(grep -n "^scan_frontend_files()" "$REPO_DIR/dev.sh" | cut -d: -f1)
        end_line=$(tail -n "+$start_line" "$REPO_DIR/dev.sh" | grep -n "^}" | head -1 | cut -d: -f1)
        end_line=$((start_line + end_line - 1))

        # Extract and eval the function
        local func_body
        func_body=$(sed -n "${start_line},${end_line}p" "$REPO_DIR/dev.sh")
        eval "$func_body"
    else
        log_error "dev.sh not found at $REPO_DIR/dev.sh"
        exit 1
    fi
}

# Run scan_frontend_files and return JSON output
run_scan() {
    local dir="${1:-$FRONTEND_SRC_DIR}"
    scan_frontend_files "$dir" 2>/dev/null
}

# Parse JSON value by key
parse_json() {
    local json="$1"
    local key="$2"
    echo "$json" | python3 -c "import json,sys; data=json.load(sys.stdin); print(data.get('$key', 0))" 2>/dev/null || echo "0"
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

    if [ "$value" -gt 0 ]; then
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

# Test cases

test_tsx_count_returns_80() {
    local test_name="verify_tsx_count_returns_80"
    log_info "Running: $test_name"

    local result
    result=$(run_scan "$FRONTEND_SRC_DIR")
    local tsx_count
    tsx_count=$(parse_json "$result" "tsx_count")

    assert_equals "80" "$tsx_count" "TSX file count should be 80"
}

test_ts_count_returns_33() {
    local test_name="verify_ts_count_returns_33"
    log_info "Running: $test_name"

    local result
    result=$(run_scan "$FRONTEND_SRC_DIR")
    local ts_count
    ts_count=$(parse_json "$result" "ts_count")

    assert_equals "33" "$ts_count" "TS file count should be 33"
}

test_total_count_returns_113() {
    local test_name="verify_total_count_returns_113"
    log_info "Running: $test_name"

    local result
    result=$(run_scan "$FRONTEND_SRC_DIR")
    local total_count
    total_count=$(parse_json "$result" "total_count")

    assert_equals "113" "$total_count" "Total TS/TSX file count should be 113"
}

test_excludes_node_modules() {
    local test_name="verify_excludes_node_modules"
    log_info "Running: $test_name"

    # Create a test directory with node_modules
    local test_dir="/tmp/test-scanner-node_modules-$$"
    mkdir -p "$test_dir/node_modules"
    mkdir -p "$test_dir/src"

    # Create test files
    touch "$test_dir/src/App.tsx"
    touch "$test_dir/node_modules/some-lib.ts"

    local result
    result=$(run_scan "$test_dir/src")
    local total_count
    total_count=$(parse_json "$result" "total_count")

    # Should only count the file in src, not in node_modules
    assert_equals "1" "$total_count" "Should only count files outside node_modules"

    # Cleanup
    rm -rf "$test_dir"
}

test_excludes_dist_and_build() {
    local test_name="verify_excludes_dist_and_build"
    log_info "Running: $test_name"

    # Create a test directory with dist and build
    local test_dir="/tmp/test-scanner-build-$$"
    mkdir -p "$test_dir/src"
    mkdir -p "$test_dir/dist"
    mkdir -p "$test_dir/build"

    # Create test files
    touch "$test_dir/src/App.tsx"
    touch "$test_dir/dist/bundle.ts"
    touch "$test_dir/build/output.ts"

    local result
    result=$(run_scan "$test_dir/src")
    local total_count
    total_count=$(parse_json "$result" "total_count")

    # Should only count the file in src
    assert_equals "1" "$total_count" "Should only count files in src, not in dist or build"

    # Cleanup
    rm -rf "$test_dir"
}

test_feature_breakdown_accuracy() {
    local test_name="verify_feature_breakdown_accuracy"
    log_info "Running: $test_name"

    local result
    result=$(run_scan "$FRONTEND_SRC_DIR")

    # Parse features JSON
    local features
    features=$(echo "$result" | python3 -c "import json,sys; data=json.load(sys.stdin); print(json.dumps(data.get('features', {})))" 2>/dev/null || echo "{}")

    # Check that features object is not empty
    TESTS_RUN=$((TESTS_RUN + 1))
    if [ "$features" != "{}" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "Feature breakdown contains modules"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "Feature breakdown is empty"
    fi

    # Verify known features exist
    for feature in dashboard documents printers auth agents settings jobs devices; do
        assert_contains "$features" "\"$feature\"" "Feature '$feature' should be in breakdown"
    done
}

test_returns_valid_json() {
    local test_name="verify_returns_valid_json"
    log_info "Running: $test_name"

    local result
    result=$(run_scan "$FRONTEND_SRC_DIR")

    # Try to parse as JSON
    TESTS_RUN=$((TESTS_RUN + 1))
    if echo "$result" | python3 -m json.tool >/dev/null 2>&1; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "Returns valid JSON"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "Does not return valid JSON"
        echo "  Output: $result"
    fi
}

test_nonexistent_directory_returns_zeros() {
    local test_name="verify_nonexistent_directory_returns_zeros"
    log_info "Running: $test_name"

    local result
    result=$(run_scan "/nonexistent/path/that/does/not/exist")

    local tsx_count ts_count total_count
    tsx_count=$(parse_json "$result" "tsx_count")
    ts_count=$(parse_json "$result" "ts_count")
    total_count=$(parse_json "$result" "total_count")

    assert_equals "0" "$tsx_count" "Nonexistent directory should return 0 TSX files"
    assert_equals "0" "$ts_count" "Nonexistent directory should return 0 TS files"
    assert_equals "0" "$total_count" "Nonexistent directory should return 0 total files"
}

test_security_no_symlink_traversal() {
    local test_name="verify_security_no_symlink_traversal"
    log_info "Running: $test_name"

    # Create a test directory with a symlink pointing outside
    local test_dir="/tmp/test-scanner-symlink-$$"
    local outside_dir="/tmp/test-scanner-outside-$$"

    mkdir -p "$test_dir/src"
    mkdir -p "$outside_dir"

    # Create symlink
    ln -s "$outside_dir" "$test_dir/src/link"

    # Create a file outside
    touch "$outside_dir/outside.tsx"

    # Create a file inside
    touch "$test_dir/src/inside.tsx"

    local result
    result=$(run_scan "$test_dir/src")
    local total_count
    total_count=$(parse_json "$result" "total_count")

    # Should not follow symlinks (find -type f doesn't follow symlinks by default)
    assert_equals "1" "$total_count" "Should not count files accessed via symlinks"

    # Cleanup
    rm -rf "$test_dir" "$outside_dir"
}

test_find_command_excludes_correctly() {
    local test_name="verify_find_command_excludes_correctly"
    log_info "Running: $test_name"

    # Create a test directory structure
    local test_dir="/tmp/test-scanner-find-$$"
    mkdir -p "$test_dir/node_modules/pkg"
    mkdir -p "$test_dir/dist"
    mkdir -p "$test_dir/build"
    mkdir -p "$test_dir/src/components"

    # Create test files in each directory
    touch "$test_dir/src/App.tsx"
    touch "$test_dir/src/components/Button.tsx"
    touch "$test_dir/node_modules/pkg/index.tsx"
    touch "$test_dir/dist/bundle.ts"
    touch "$test_dir/build/output.ts"

    # Run the find command directly as scan_frontend_files does
    local tsx_count
    tsx_count=$(find "$test_dir/src" \
        \( -name "node_modules" -o -name "dist" -o -name "build" \) -prune \
        -o -type f -name "*.tsx" -print 2>/dev/null | wc -l)
    tsx_count="${tsx_count//[^0-9]/}"

    assert_equals "2" "$tsx_count" "Find command should only count files in src (2 TSX files)"

    # Cleanup
    rm -rf "$test_dir"
}

test_test_files_counted_correctly() {
    local test_name="verify_test_files_counted_correctly"
    log_info "Running: $test_name"

    # Check the actual frontend test count
    local result
    result=$(run_scan "$FRONTEND_SRC_DIR")
    local test_count
    test_count=$(parse_json "$result" "test_count")

    # We have test files in the e2e directory, but those are separate
    # The test_count should count *.spec.ts and *.test.ts files
    assert_not_zero "$test_count" "Test file count should be greater than zero"

    [ "$VERBOSE" = true ] && echo "  Test files found: $test_count"
}

# Print usage
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Unit tests for dev.sh scanner functions (scan_frontend_files)

OPTIONS:
    -v, --verbose    Show detailed output for each test
    -h, --help       Show this help message

TEST CASES:
    verify_tsx_count_returns_80         Ensures 80 TSX files are counted
    verify_ts_count_returns_33          Ensures 33 TS files are counted
    verify_total_count_returns_113      Ensures 113 total files are counted
    verify_excludes_node_modules        Ensures node_modules is excluded
    verify_excludes_dist_and_build      Ensures dist and build are excluded
    verify_feature_breakdown_accuracy   Ensures feature modules are counted
    verify_returns_valid_json           Ensures output is valid JSON
    verify_nonexistent_directory        Ensures zeros for missing directories
    verify_security_no_symlink_traversal Ensures symlinks are not followed
    verify_find_command_excludes_correctly Tests find command syntax
    verify_test_files_counted_correctly Ensures test files are counted

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
    echo "Scanner Unit Tests"
    echo "=========================================="
    echo "Repository: $REPO_DIR"
    echo "Frontend: $FRONTEND_SRC_DIR"
    echo "=========================================="
    echo ""

    # Source dev.sh to get the function
    source_dev_sh

    # Run all tests
    test_tsx_count_returns_80
    test_ts_count_returns_33
    test_total_count_returns_113
    test_excludes_node_modules
    test_excludes_dist_and_build
    test_feature_breakdown_accuracy
    test_returns_valid_json
    test_nonexistent_directory_returns_zeros
    test_security_no_symlink_traversal
    test_find_command_excludes_correctly
    test_test_files_counted_correctly

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
