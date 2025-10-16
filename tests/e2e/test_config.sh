#!/bin/bash

# SmartTicket gRPC E2E Test Configuration
# This file contains environment variables and configuration for E2E testing

# Service Configuration
export GRPC_SERVER_HOST="${GRPC_SERVER_HOST:-localhost}"
export GRPC_SERVER_PORT="${GRPC_SERVER_PORT:-6533}"
export GRPC_SERVER_ADDRESS="${GRPC_SERVER_HOST}:${GRPC_SERVER_PORT}"

# Test Configuration
export TEST_TENANT_DOMAIN="${TEST_TENANT_DOMAIN:-test.smartticket.com}"
export TEST_ADMIN_EMAIL="${TEST_ADMIN_EMAIL:-admin@test.smartticket.com}"
export TEST_ADMIN_PASSWORD="${TEST_ADMIN_PASSWORD:-admin123}"
export TEST_USER_EMAIL="${TEST_USER_EMAIL:-user@test.smartticket.com}"
export TEST_USER_PASSWORD="${TEST_USER_PASSWORD:-testpass123}"

# Test Data Configuration
export TEST_TIMEOUT="${TEST_TIMEOUT:-30}"
export TEST_RETRY_COUNT="${TEST_RETRY_COUNT:-3}"
export TEST_RETRY_DELAY="${TEST_RETRY_DELAY:-1}"

# File paths
export TEST_DATA_DIR="${TEST_DATA_DIR:-./test_data}"
export TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-./test_results}"
export PROTO_FILE="${PROTO_FILE:-$(dirname "${BASH_SOURCE[0]}")/../../proto/smartticket}"
export PROTO_DIR="${PROTO_FILE}"

# Colors for output
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export BLUE='\033[0;34m'
export NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Ensure required directories exist
setup_test_directories() {
    mkdir -p "${TEST_DATA_DIR}"
    mkdir -p "${TEST_RESULTS_DIR}"
}

# Check if grpc server is running
check_grpc_server() {
    log_info "Checking if gRPC server is running at ${GRPC_SERVER_ADDRESS}..."

    # Try to connect to the server (even if reflection is disabled)
    if timeout 5 bash -c "</dev/tcp/${GRPC_SERVER_HOST}/${GRPC_SERVER_PORT}" 2>/dev/null; then
        log_success "gRPC server is running and accessible on port ${GRPC_SERVER_PORT}"
        log_info "Note: Server may not have reflection API enabled - using proto-based testing"
        return 0
    else
        log_error "gRPC server is not running or not accessible at ${GRPC_SERVER_ADDRESS}"
        return 1
    fi
}

# List available services
list_services() {
    log_info "Available gRPC services:"
    grpcurl -plaintext "${GRPC_SERVER_ADDRESS}" list
}

# Describe a service
describe_service() {
    local service_name="$1"
    log_info "Describing service: ${service_name}"
    grpcurl -plaintext "${GRPC_SERVER_ADDRESS}" describe "${service_name}"
}

# Initialize test environment
init_test_env() {
    setup_test_directories

    if ! check_grpc_server; then
        log_error "Cannot proceed with tests - gRPC server is not available"
        exit 1
    fi

    log_success "Test environment initialized successfully"
}

# Call init_test_env if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    init_test_env
    list_services
fi