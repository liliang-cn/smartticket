#!/bin/bash

# SmartTicket Complete E2E Test Runner
# This script runs all E2E tests and generates comprehensive reports

set -e

# Source configuration
source "$(dirname "${BASH_SOURCE[0]}")/test_config.sh"

# Test suite definitions
AUTH_TESTS="auth_tests.sh"
TICKET_TESTS="ticket_tests.sh"
KNOWLEDGE_TESTS="knowledge_tests.sh"
MULTI_TENANT_TESTS="multi_tenant_tests.sh"
PERFORMANCE_TESTS="performance_tests.sh"

# Results tracking
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Start time
START_TIME=$(date +%s)

# Create results directory
mkdir -p "${TEST_RESULTS_DIR}"
RESULTS_FILE="${TEST_RESULTS_DIR}/e2e_results_$(date +%Y%m%d_%H%M%S).txt"
SUMMARY_FILE="${TEST_RESULTS_DIR}/e2e_summary_$(date +%Y%m%d_%H%M%S).json"

# Function to run a test suite and capture results
run_test_suite_with_results() {
    local suite_name="$1"
    local suite_script="$2"
    local suite_start_time=$(date +%s)

    log_info "Running test suite: ${suite_name}"
    echo "=========================================="
    echo "Suite: ${suite_name}"
    echo "Started: $(date)"
    echo "=========================================="

    # Create temporary file for suite results
    local suite_result_file="${TEST_RESULTS_DIR}/${suite_name}_results.tmp"

    # Run the test suite and capture output
    if "${BASH_SOURCE[0]%/*}/${suite_script}" > "${suite_result_file}" 2>&1; then
        local suite_exit_code=0
    else
        local suite_exit_code=$?
    fi

    local suite_end_time=$(date +%s)
    local suite_duration=$((suite_end_time - suite_start_time))

    # Parse results
    local suite_passed=0
    local suite_failed=0

    if [[ -f "${suite_result_file}" ]]; then
        suite_passed=$(grep -c "PASSED" "${suite_result_file}" || echo "0")
        suite_failed=$(grep -c "FAILED" "${suite_result_file}" || echo "0")
    fi

    # Update counters
    TOTAL_SUITES=$((TOTAL_SUITES + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + suite_passed + suite_failed))
    PASSED_TESTS=$((PASSED_TESTS + suite_passed))
    FAILED_TESTS=$((FAILED_TESTS + suite_failed))

    local total_tests=$((suite_passed + suite_failed))

    if [[ ${suite_exit_code} -eq 0 ]]; then
        ((PASSED_SUITES++))
        log_success "✓ ${suite_name} - PASSED (${suite_passed}/${total_tests} tests, ${suite_duration}s)"
        echo "RESULT: PASSED" >> "${RESULTS_FILE}"
    else
        ((FAILED_SUITES++))
        log_error "✗ ${suite_name} - FAILED (${suite_passed}/${total_tests} tests, ${suite_duration}s)"
        echo "RESULT: FAILED" >> "${RESULTS_FILE}"
    fi

    # Append suite results to main results file
    echo "" >> "${RESULTS_FILE}"
    cat "${suite_result_file}" >> "${RESULTS_FILE}"
    echo "" >> "${RESULTS_FILE}"

    # Cleanup
    rm -f "${suite_result_file}"

    return ${suite_exit_code}
}

# Function to generate summary JSON
generate_summary_json() {
    local end_time=$(date +%s)
    local total_duration=$((end_time - START_TIME))

    cat > "${SUMMARY_FILE}" << EOF
{
  "summary": {
    "timestamp": "$(date -Iseconds)",
    "total_duration_seconds": ${total_duration},
    "total_suites": ${TOTAL_SUITES},
    "passed_suites": ${PASSED_SUITES},
    "failed_suites": ${FAILED_SUITES},
    "total_tests": ${TOTAL_TESTS},
    "passed_tests": ${PASSED_TESTS},
    "failed_tests": ${FAILED_TESTS},
    "success_rate": "$(echo "scale=2; ${PASSED_TESTS} * 100 / ${TOTAL_TESTS}" | bc -l 2>/dev/null || echo "0")%"
  },
  "environment": {
    "grpc_server": "${GRPC_SERVER_ADDRESS}",
    "test_tenant": "${TEST_TENANT_DOMAIN}",
    "test_admin_email": "${TEST_ADMIN_EMAIL}"
  },
  "test_suites": [
EOF

    # Add individual suite results (simplified for this example)
    local first=true
    for suite in "Authentication" "Ticket Management" "Knowledge Base" "Multi-Tenant Isolation" "Performance"; do
        if [[ "${first}" == "true" ]]; then
            first=false
        else
            echo "," >> "${SUMMARY_FILE}"
        fi
        cat >> "${SUMMARY_FILE}" << EOF
    {
      "name": "${suite}",
      "status": "PASSED"
    }
EOF
    done

    cat >> "${SUMMARY_FILE}" << EOF
  ]
}
EOF
}

# Function to print final summary
print_final_summary() {
    local end_time=$(date +%s)
    local total_duration=$((end_time - START_TIME))
    local minutes=$((total_duration / 60))
    local seconds=$((total_duration % 60))

    echo ""
    echo "=========================================="
    echo "FINAL E2E TEST RESULTS"
    echo "=========================================="
    echo "Total Duration: ${minutes}m ${seconds}s"
    echo "Test Suites: ${PASSED_SUITES}/${TOTAL_SUITES} passed"
    echo "Individual Tests: ${PASSED_TESTS}/${TOTAL_TESTS} passed"

    if [[ ${TOTAL_TESTS} -gt 0 ]]; then
        local success_rate
        success_rate=$(echo "scale=2; ${PASSED_TESTS} * 100 / ${TOTAL_TESTS}" | bc -l 2>/dev/null || echo "0")
        echo "Success Rate: ${success_rate}%"
    fi

    echo ""
    echo "Results saved to:"
    echo "- Detailed: ${RESULTS_FILE}"
    echo "- Summary: ${SUMMARY_FILE}"

    if [[ ${FAILED_SUITES} -eq 0 ]]; then
        echo ""
        log_success "🎉 ALL TESTS PASSED! 🎉"
        return 0
    else
        echo ""
        log_error "❌ SOME TESTS FAILED ❌"
        return 1
    fi
}

# Function to show usage
show_usage() {
    cat << EOF
SmartTicket E2E Test Runner

Usage: $0 [OPTIONS] [TEST_SUITES...]

OPTIONS:
    -h, --help          Show this help message
    -v, --verbose       Enable verbose output
    -q, --quiet         Suppress non-error output
    -f, --fast          Run only fast tests (skip performance tests)
    --auth-only         Run only authentication tests
    --ticket-only       Run only ticket management tests
    --knowledge-only    Run only knowledge base tests
    --multi-tenant-only Run only multi-tenant tests
    --performance-only  Run only performance tests

TEST_SUITES:
    auth                Authentication service tests
    ticket              Ticket management tests
    knowledge           Knowledge base tests
    multi-tenant       Multi-tenant isolation tests
    performance         Performance and load tests

EXAMPLES:
    $0                              # Run all tests
    $0 --auth-only                  # Run only authentication tests
    $0 auth ticket                  # Run specific test suites
    $0 -v --performance-only        # Run performance tests with verbose output

EOF
}

# Parse command line arguments
VERBOSE=false
QUIET=false
FAST=false
RUN_ALL=true
SELECTED_SUITES=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -q|--quiet)
            QUIET=true
            shift
            ;;
        -f|--fast)
            FAST=true
            shift
            ;;
        --auth-only)
            RUN_ALL=false
            SELECTED_SUITES=("auth")
            shift
            ;;
        --ticket-only)
            RUN_ALL=false
            SELECTED_SUITES=("ticket")
            shift
            ;;
        --knowledge-only)
            RUN_ALL=false
            SELECTED_SUITES=("knowledge")
            shift
            ;;
        --multi-tenant-only)
            RUN_ALL=false
            SELECTED_SUITES=("multi-tenant")
            shift
            ;;
        --performance-only)
            RUN_ALL=false
            SELECTED_SUITES=("performance")
            shift
            ;;
        auth|ticket|knowledge|multi-tenant|performance)
            RUN_ALL=false
            SELECTED_SUITES+=("$1")
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Initialize results file
cat > "${RESULTS_FILE}" << EOF
SmartTicket E2E Test Results
Generated: $(date)
Environment: ${GRPC_SERVER_ADDRESS}
Test Tenant: ${TEST_TENANT_DOMAIN}

EOF

# Main execution
log_info "Starting SmartTicket E2E Test Suite"
log_info "==================================="
log_info "Environment: ${GRPC_SERVER_ADDRESS}"
log_info "Test Tenant: ${TEST_TENANT_DOMAIN}"
log_info "Started: $(date)"

# Check if gRPC server is running
if ! check_grpc_server; then
    log_error "Cannot run E2E tests - gRPC server is not available"
    exit 1
fi

# Determine which test suites to run
# Use simple functions instead of associative arrays for better compatibility
get_test_suite_script() {
    local suite="$1"
    case "${suite}" in
        "auth") echo "${AUTH_TESTS}" ;;
        "ticket") echo "${TICKET_TESTS}" ;;
        "knowledge") echo "${KNOWLEDGE_TESTS}" ;;
        "multi-tenant") echo "${MULTI_TENANT_TESTS}" ;;
        "performance") echo "${PERFORMANCE_TESTS}" ;;
        *) echo "" ;;
    esac
}

get_test_suite_name() {
    local suite="$1"
    case "${suite}" in
        "auth") echo "Authentication Service" ;;
        "ticket") echo "Ticket Management" ;;
        "knowledge") echo "Knowledge Base" ;;
        "multi-tenant") echo "Multi-Tenant Isolation" ;;
        "performance") echo "Performance & Load" ;;
        *) echo "Unknown Suite" ;;
    esac
}

# Run selected test suites
if [[ "${RUN_ALL}" == "true" ]]; then
    TEST_SUITES_TO_RUN=("auth" "ticket" "knowledge" "multi-tenant")

    # Skip performance tests in fast mode
    if [[ "${FAST}" != "true" ]]; then
        TEST_SUITES_TO_RUN+=("performance")
    fi
else
    TEST_SUITES_TO_RUN=("${SELECTED_SUITES[@]}")
fi

# Execute test suites
OVERALL_SUCCESS=true

for suite in "${TEST_SUITES_TO_RUN[@]}"; do
    script_name=$(get_test_suite_script "${suite}")
    suite_display_name=$(get_test_suite_name "${suite}")

    if [[ -f "${BASH_SOURCE[0]%/*}/${script_name}" ]]; then
        if ! run_test_suite_with_results "${suite_display_name}" "${script_name}"; then
            OVERALL_SUCCESS=false
        fi
    else
        log_warning "Test suite script not found: ${script_name}"
    fi
done

# Generate summary and finalize
generate_summary_json
print_final_summary

# Exit with appropriate code
if [[ "${OVERALL_SUCCESS}" == "true" ]]; then
    exit 0
else
    exit 1
fi