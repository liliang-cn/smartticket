#!/bin/bash

# SmartTicket E2E Test Configuration
# This file contains configuration variables for all E2E tests

# Service Configuration
export GATEWAY_HOST="${GATEWAY_HOST:-localhost}"
export GATEWAY_PORT="${GATEWAY_PORT:-6533}"
export AUTH_HOST="${AUTH_HOST:-localhost}"
export AUTH_PORT="${AUTH_PORT:-50052}"
export CORE_HOST="${CORE_HOST:-localhost}"
export CORE_PORT="${CORE_PORT:-50053}"
export AI_HOST="${AI_HOST:-localhost}"
export AI_PORT="${AI_PORT:-50054}"
export NOTIFICATION_HOST="${NOTIFICATION_HOST:-localhost}"
export NOTIFICATION_PORT="${NOTIFICATION_PORT:-50055}"
export PLATFORM_HOST="${PLATFORM_HOST:-localhost}"
export PLATFORM_PORT="${PLATFORM_PORT:-50056}"

# Database Configuration
export DATABASE_URL="${DATABASE_URL:-postgresql://postgres:postgres@localhost:5434/smartticket}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

# Test Configuration
export TEST_TIMEOUT="${TEST_TIMEOUT:-30}"
export TEST_RETRIES="${TEST_RETRIES:-3}"
export TEST_PARALLEL="${TEST_PARALLEL:-false}"
export TEST_VERBOSE="${TEST_VERBOSE:-true}"

# Test Data Configuration
export TEST_TENANT_DOMAIN="${TEST_TENANT_DOMAIN:-testcompany.com}"
export TEST_ADMIN_EMAIL="${TEST_ADMIN_EMAIL:-admin@testcompany.com}"
export TEST_ADMIN_PASSWORD="${TEST_ADMIN_PASSWORD:-testpassword123}"
export TEST_USER_EMAIL="${TEST_USER_EMAIL:-user@testcompany.com}"
export TEST_USER_PASSWORD="${TEST_USER_PASSWORD:-testpassword123}"

# E2E Test Paths
export E2E_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export E2E_INTEGRATION_DIR="$E2E_ROOT_DIR/integration"
export E2E_SINGLE_SERVICE_DIR="$E2E_ROOT_DIR/single-service"
export E2E_PERFORMANCE_DIR="$E2E_ROOT_DIR/performance"
export E2E_UTILS_DIR="$E2E_ROOT_DIR/utils"
export E2E_CONFIG_DIR="$E2E_ROOT_DIR/config"
export E2E_DATA_DIR="$E2E_ROOT_DIR/data"
export E2E_REPORTS_DIR="$E2E_ROOT_DIR/reports"

# Report Configuration
export REPORT_DIR="$E2E_REPORTS_DIR/results"
export REPORT_TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
export REPORT_FILE="$REPORT_DIR/e2e_results_$REPORT_TIMESTAMP.txt"
export REPORT_JSON="$REPORT_DIR/e2e_summary_$REPORT_TIMESTAMP.json"

# Colors for output
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export BLUE='\033[0;34m'
export NC='\033[0m' # No Color

# Utility Functions
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

# Create report directory
mkdir -p "$REPORT_DIR"

# Load environment-specific configuration
if [ -f "$E2E_CONFIG_DIR/.env.$ENVIRONMENT" ]; then
    source "$E2E_CONFIG_DIR/.env.$ENVIRONMENT"
elif [ -f "$E2E_CONFIG_DIR/.env" ]; then
    source "$E2E_CONFIG_DIR/.env"
fi

# Source additional test configuration if available
if [ -f "$E2E_CONFIG_DIR/test_config.sh" ]; then
    # Only export environment variable definitions (skip functions)
    eval "$(grep -E '^[A-Z_]+=' "$E2E_CONFIG_DIR/test_config.sh")"
fi