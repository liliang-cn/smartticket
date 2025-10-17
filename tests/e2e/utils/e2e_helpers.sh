#!/bin/bash

# SmartTicket E2E Test Helper Functions
# This file contains common helper functions for E2E tests

# Source configuration
source "$(dirname "${BASH_SOURCE[0]}")/../config/e2e_config.sh"

# Service Health Check
check_service_health() {
    local service_name=$1
    local host=$2
    local port=$3
    local max_attempts=${4:-30}
    local attempt=1

    log_info "Checking $service_name health at $host:$port"

    while [ $attempt -le $max_attempts ]; do
        if nc -z "$host" "$port" 2>/dev/null; then
            log_success "$service_name is healthy after $attempt attempts"
            return 0
        fi

        log_warning "$service_name not ready (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done

    log_error "$service_name failed to become healthy after $max_attempts attempts"
    return 1
}

# Database Health Check
check_database_health() {
    log_info "Checking database connectivity"

    if command -v psql >/dev/null 2>&1; then
        if psql "$DATABASE_URL" -c "SELECT 1;" >/dev/null 2>&1; then
            log_success "Database is healthy"
            return 0
        fi
    fi

    log_error "Database health check failed"
    return 1
}

# Redis Health Check
check_redis_health() {
    log_info "Checking Redis connectivity"

    if command -v redis-cli >/dev/null 2>&1; then
        if redis-cli -u "$REDIS_URL" ping >/dev/null 2>&1; then
            log_success "Redis is healthy"
            return 0
        fi
    fi

    log_error "Redis health check failed"
    return 1
}

# gRPC Service Test
test_grpc_service() {
    local service_name=$1
    local host=$2
    local port=$3
    local method=$4

    log_info "Testing gRPC service: $service_name.$method"

    if command -v grpcurl >/dev/null 2>&1; then
        if grpcurl -plaintext "$host:$port" "grpc.health.v1.Health/Check" >/dev/null 2>&1; then
            log_success "$service_name gRPC service is responding"
            return 0
        fi
    else
        # Fallback to simple port check
        if nc -z "$host" "$port" 2>/dev/null; then
            log_success "$service_name service port is open"
            return 0
        fi
    fi

    log_error "$service_name gRPC service test failed"
    return 1
}

# HTTP Service Test
test_http_service() {
    local service_name=$1
    local url=$2
    local expected_status=${3:-200}

    log_info "Testing HTTP service: $url"

    if command -v curl >/dev/null 2>&1; then
        local status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
        if [ "$status" = "$expected_status" ]; then
            log_success "$service_name HTTP service is responding with status $status"
            return 0
        else
            log_error "$service_name HTTP service responded with status $status (expected $expected_status)"
            return 1
        fi
    fi

    log_error "$service_name HTTP service test failed (curl not available)"
    return 1
}

# JWT Token Generation Helper
generate_test_token() {
    local user_id=$1
    local email=$2
    local tenant_id=$3
    local role=${4:-"CustomerUser"}

    # This would typically call the auth service or use a test JWT library
    # For now, return a mock token structure
    echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test_payload.signature"
}

# Test Report Functions
start_test_report() {
    local test_name=$1

    echo "=== E2E Test Report ===" > "$REPORT_FILE"
    echo "Test Name: $test_name" >> "$REPORT_FILE"
    echo "Start Time: $(date)" >> "$REPORT_FILE"
    echo "Environment: ${ENVIRONMENT:-development}" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"

    # Initialize JSON report
    cat > "$REPORT_JSON" << EOF
{
  "test_name": "$test_name",
  "start_time": "$(date -Iseconds)",
  "environment": "${ENVIRONMENT:-development}",
  "tests": [],
  "summary": {
    "total": 0,
    "passed": 0,
    "failed": 0,
    "skipped": 0
  }
}
EOF

    log_info "Test report initialized: $REPORT_FILE"
}

add_test_result() {
    local test_name=$1
    local status=$2
    local duration=$3
    local details=${4:-""}

    echo "Test: $test_name - Status: $status - Duration: ${duration}s" >> "$REPORT_FILE"
    if [ -n "$details" ]; then
        echo "  Details: $details" >> "$REPORT_FILE"
    fi

    # Update JSON report (simplified)
    log_info "Test result added: $test_name ($status)"
}

end_test_report() {
    local total_tests=$1
    local passed_tests=$2
    local failed_tests=$3
    local skipped_tests=${4:-0}

    echo "" >> "$REPORT_FILE"
    echo "=== Test Summary ===" >> "$REPORT_FILE"
    echo "Total Tests: $total_tests" >> "$REPORT_FILE"
    echo "Passed: $passed_tests" >> "$REPORT_FILE"
    echo "Failed: $failed_tests" >> "$REPORT_FILE"
    echo "Skipped: $skipped_tests" >> "$REPORT_FILE"
    echo "Success Rate: $(( passed_tests * 100 / total_tests ))%" >> "$REPORT_FILE"
    echo "End Time: $(date)" >> "$REPORT_FILE"

    # Update JSON summary
    local temp_json=$(mktemp)
    jq --arg total "$total_tests" \
       --arg passed "$passed_tests" \
       --arg failed "$failed_tests" \
       --arg skipped "$skipped_tests" \
       --arg end_time "$(date -Iseconds)" \
       '.summary.total = ($total | tonumber) |
        .summary.passed = ($passed | tonumber) |
        .summary.failed = ($failed | tonumber) |
        .summary.skipped = ($skipped | tonumber) |
        .end_time = $end_time' \
       "$REPORT_JSON" > "$temp_json"
    mv "$temp_json" "$REPORT_JSON"

    log_info "Test report completed: $REPORT_FILE"

    # Print summary
    echo ""
    echo "=== E2E Test Summary ==="
    echo "Total Tests: $total_tests"
    echo -e "Passed: ${GREEN}$passed_tests${NC}"
    echo -e "Failed: ${RED}$failed_tests${NC}"
    echo "Skipped: $skipped_tests"
    echo "Success Rate: $(( passed_tests * 100 / total_tests ))%"
    echo "Report: $REPORT_FILE"
}

# Cleanup Functions
cleanup_test_data() {
    log_info "Cleaning up test data"

    # Cleanup test tenants (implementation depends on your setup)
    # Cleanup test users
    # Cleanup test tickets
    # Cleanup test knowledge articles

    log_info "Test data cleanup completed"
}

# Environment Setup
setup_test_environment() {
    log_info "Setting up test environment"

    # Create necessary directories
    mkdir -p "$REPORT_DIR"
    mkdir -p "$E2E_DATA_DIR"

    # Set environment variables
    export ENVIRONMENT="${ENVIRONMENT:-test}"

    # Verify dependencies
    local missing_deps=()

    command -v nc >/dev/null 2>&1 || missing_deps+=("netcat")
    command -v curl >/dev/null 2>&1 || missing_deps+=("curl")
    command -v jq >/dev/null 2>&1 || missing_deps+=("jq")

    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        return 1
    fi

    log_info "Test environment setup completed"
    return 0
}

# Test Execution Wrapper
run_test_with_timeout() {
    local test_command="$1"
    local timeout=${TEST_TIMEOUT:-30}

    log_info "Running test with ${timeout}s timeout: $test_command"

    if timeout "$timeout" bash -c "$test_command"; then
        log_success "Test completed successfully"
        return 0
    else
        local exit_code=$?
        if [ $exit_code -eq 124 ]; then
            log_error "Test timed out after ${timeout}s"
        else
            log_error "Test failed with exit code $exit_code"
        fi
        return $exit_code
    fi
}

# Parallel Test Execution (if enabled)
run_tests_in_parallel() {
    local test_files=("$@")
    local pids=()

    if [ "$TEST_PARALLEL" = "true" ]; then
        log_info "Running tests in parallel"

        for test_file in "${test_files[@]}"; do
            bash "$test_file" &
            pids+=($!)
        done

        for pid in "${pids[@]}"; do
            wait "$pid"
        done
    else
        log_info "Running tests sequentially"
        for test_file in "${test_files[@]}"; do
            bash "$test_file"
        done
    fi
}

# Export all functions
export -f log_info log_success log_warning log_error
export -f check_service_health check_database_health check_redis_health
export -f test_grpc_service test_http_service
export -f generate_test_token
export -f start_test_report add_test_result end_test_report
export -f cleanup_test_data setup_test_environment
export -f run_test_with_timeout run_tests_in_parallel