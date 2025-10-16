#!/bin/bash

# SmartTicket gRPC E2E Test Helper Functions
# This file contains common helper functions for E2E testing

# Source the configuration
source "$(dirname "${BASH_SOURCE[0]}")/test_config.sh"

# Global variables for test state
export TEST_ACCESS_TOKEN=""
export TEST_REFRESH_TOKEN=""
export TEST_USER_ID=""
export TEST_TENANT_ID=""
export CURRENT_TICKET_ID=""
export CURRENT_USER_ID=""

# JSON helper functions
create_json_metadata() {
    local tenant_id="$1"
    local user_id="$2"
    local request_id="${3:-$(uuidgen)}"

    cat <<EOF
{
  "tenant_id": "${tenant_id}",
  "user_id": "${user_id}",
  "request_id": "${request_id}"
}
EOF
}

create_json_pagination() {
    local page_size="${1:-20}"
    local page_token="${2:-}"

    cat <<EOF
{
  "page_size": ${page_size},
  "page_token": "${page_token}"
}
EOF
}

# gRPC call helper functions
make_grpc_call() {
    local service="$1"
    local method="$2"
    local data="$3"
    local expected_success="${4:-true}"

    echo "Calling gRPC: ${service}.${method}" >&2
    echo "Proto directory: ${PROTO_FILE}" >&2

    # Find proto files for the service
    local proto_files=""
    local proto_dir="${PROTO_FILE}"

    if [[ -d "${proto_dir}" ]]; then
        # Build grpcurl arguments as an array
        GRPC_ARGS=(-plaintext)
        GRPC_ARGS+=(-import-path "${proto_dir}/../")
        # Add proto files based on service to avoid conflicts
        # User proto contains auth service definitions
        if [[ -f "${proto_dir}/user.proto" ]]; then
            GRPC_ARGS+=(-proto smartticket/user.proto)
        fi
        # Add other proto files as needed, avoiding common.proto/definitions.proto conflicts
        if [[ -f "${proto_dir}/ticket.proto" ]]; then
            GRPC_ARGS+=(-proto smartticket/ticket.proto)
        fi
        if [[ -f "${proto_dir}/knowledge.proto" ]]; then
            GRPC_ARGS+=(-proto smartticket/knowledge.proto)
        fi
        if [[ -f "${proto_dir}/tenant.proto" ]]; then
            GRPC_ARGS+=(-proto smartticket/tenant.proto)
        fi
        if [[ -f "${proto_dir}/role_permission.proto" ]]; then
            GRPC_ARGS+=(-proto smartticket/role_permission.proto)
        fi
    else
        log_warning "Proto directory not found: ${proto_dir}"
    fi

    echo "GRPC args: ${GRPC_ARGS[*]}" >&2

    if [[ -n "${TEST_ACCESS_TOKEN}" ]]; then
        # Call with authentication
        local result
        echo "Making authenticated gRPC call to ${GRPC_SERVER_ADDRESS}" >&2
        result=$(grpcurl "${GRPC_ARGS[@]}" \
            -H "authorization: Bearer ${TEST_ACCESS_TOKEN}" \
            -H "x-tenant-id: ${TEST_TENANT_ID}" \
            -H "x-user-id: ${TEST_USER_ID}" \
            -d "${data}" \
            "${GRPC_SERVER_ADDRESS}" \
            "${service}.${method}" 2>&1)

        local exit_code=$?
    else
        # Call without authentication
        local result
        echo "Making unauthenticated gRPC call to ${GRPC_SERVER_ADDRESS}" >&2
        result=$(grpcurl "${GRPC_ARGS[@]}" \
            -d "${data}" \
            "${GRPC_SERVER_ADDRESS}" \
            "${service}.${method}" 2>&1)

        local exit_code=$?
    fi

    # Extract only JSON from the result, filtering out log messages
    local json_result=$(echo "${result}" | sed -n '/^{/,$p')

    if [[ $exit_code -eq 0 ]]; then
        if [[ "${expected_success}" == "true" ]]; then
            echo "gRPC call successful" >&2
            echo "${json_result}"
            return 0
        else
            echo "gRPC call succeeded but was expected to fail" >&2
            echo "${json_result}"
            return 1
        fi
    else
        if [[ "${expected_success}" == "false" ]]; then
            echo "gRPC call failed as expected" >&2
            echo "${json_result}"
            return 0
        else
            echo "gRPC call failed: ${result}" >&2
            echo "${json_result}"
            return 1
        fi
    fi
}

# Extract field from JSON response
extract_json_field() {
    local json="$1"
    local field="$2"

    echo "${json}" | jq -r ".${field}" 2>/dev/null || echo ""
}

# Authentication helper functions
login_user() {
    local email="$1"
    local password="$2"
    local tenant_domain="$3"

    log_info "Logging in user: ${email}"

    local login_data
    login_data=$(cat <<EOF
{
  "email": "${email}",
  "password": "${password}",
  "tenant_domain": "${tenant_domain}",
  "remember_me": false
}
EOF
)

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "Login" "${login_data}")

    log_info "Login response: ${response}"

    if [[ $? -eq 0 ]]; then
        TEST_ACCESS_TOKEN=$(extract_json_field "${response}" "accessToken")
        TEST_REFRESH_TOKEN=$(extract_json_field "${response}" "refreshToken")
        TEST_USER_ID=$(extract_json_field "${response}" "user.id")
        TEST_TENANT_ID=$(extract_json_field "${response}" "user.tenantId")

        log_success "Login successful - User ID: ${TEST_USER_ID}, Tenant ID: ${TEST_TENANT_ID}"
        return 0
    else
        log_error "Login failed"
        return 1
    fi
}

logout_user() {
    log_info "Logging out user"
    TEST_ACCESS_TOKEN=""
    TEST_REFRESH_TOKEN=""
    TEST_USER_ID=""
    TEST_TENANT_ID=""
    CURRENT_TICKET_ID=""
    CURRENT_USER_ID=""
}

refresh_token() {
    log_info "Refreshing access token"

    local refresh_data
    refresh_data=$(cat <<EOF
{
  "refresh_token": "${TEST_REFRESH_TOKEN}"
}
EOF
)

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "RefreshToken" "${refresh_data}")

    if [[ $? -eq 0 ]]; then
        TEST_ACCESS_TOKEN=$(extract_json_field "${response}" "accessToken")
        TEST_REFRESH_TOKEN=$(extract_json_field "${response}" "refreshToken")
        log_success "Token refreshed successfully"
        return 0
    else
        log_error "Token refresh failed"
        return 1
    fi
}

# Test assertion helpers
assert_equals() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Assertion failed}"

    if [[ "${actual}" == "${expected}" ]]; then
        log_success "${message}"
        return 0
    else
        log_error "${message}: Expected '${expected}', got '${actual}'"
        return 1
    fi
}

assert_not_empty() {
    local value="$1"
    local message="${2:-Value should not be empty}"

    if [[ -n "${value}" ]]; then
        log_success "${message}"
        return 0
    else
        log_error "${message}: Value is empty"
        return 1
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-String should contain substring}"

    if [[ "${haystack}" == *"${needle}"* ]]; then
        log_success "${message}"
        return 0
    else
        log_error "${message}: '${haystack}' does not contain '${needle}'"
        return 1
    fi
}

# Test data cleanup functions
cleanup_test_data() {
    log_info "Cleaning up test data..."

    # Logout current user
    logout_user

    # Additional cleanup can be added here
    log_success "Test data cleanup completed"
}

# Test execution helper
run_test() {
    local test_name="$1"
    local test_function="$2"

    log_info "Running test: ${test_name}"

    if ${test_function}; then
        log_success "✓ ${test_name} - PASSED"
        return 0
    else
        log_error "✗ ${test_name} - FAILED"
        return 1
    fi
}

# Test suite runner
run_test_suite() {
    local suite_name="$1"
    shift
    local tests=("$@")

    log_info "Running test suite: ${suite_name}"
    log_info "=================================="

    local passed=0
    local failed=0

    for test in "${tests[@]}"; do
        if run_test "${test}" "${test}"; then
            ((passed++))
        else
            ((failed++))
        fi
        echo ""
    done

    log_info "=================================="
    log_info "Test suite completed: ${passed} passed, ${failed} failed"

    if [[ $failed -eq 0 ]]; then
        log_success "All tests in suite passed!"
        return 0
    else
        log_error "Some tests in suite failed!"
        return 1
    fi
}

# Setup test environment
setup_test_env() {
    init_test_env

    # Login as test admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as test admin"
        return 1
    fi

    log_success "Test environment setup completed"
    return 0
}

# Teardown test environment
teardown_test_env() {
    cleanup_test_data
    log_success "Test environment teardown completed"
}

# Trap to ensure cleanup on script exit
trap 'teardown_test_env' EXIT