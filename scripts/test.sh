#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_COMPOSE_FILE="$PROJECT_ROOT/docker/docker-compose.test.yml"
POSTGRES_TEST_PORT=5433
REDIS_TEST_PORT=6380

echo -e "${BLUE}🧪 SmartTicket Test Runner${NC}"
echo "========================================"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo -e "${YELLOW}📋 Checking prerequisites...${NC}"

if ! command_exists docker; then
    echo -e "${RED}❌ Docker is not installed or not in PATH${NC}"
    exit 1
fi

if ! command_exists docker-compose; then
    echo -e "${RED}❌ Docker Compose is not installed or not in PATH${NC}"
    exit 1
fi

if ! command_exists cargo; then
    echo -e "${RED}❌ Cargo is not installed or not in PATH${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All prerequisites found${NC}"

# Parse command line arguments
COMMAND=${1:-"all"}
SCOPE=${2:-"unit"}

echo -e "${YELLOW}🎯 Running: $COMMAND for scope: $SCOPE${NC}"

# Function to start test infrastructure
start_test_infrastructure() {
    echo -e "${YELLOW}🚀 Starting test infrastructure...${NC}"

    cd "$PROJECT_ROOT"

    # Stop any existing test containers
    docker-compose -f "$TEST_COMPOSE_FILE" down -v 2>/dev/null || true

    # Start test containers
    docker-compose -f "$TEST_COMPOSE_FILE" up -d

    # Wait for services to be ready
    echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"

    # Wait for PostgreSQL
    local postgres_ready=false
    for i in {1..30}; do
        if docker-compose -f "$TEST_COMPOSE_FILE" exec -T postgres-test pg_isready -U postgres -d smartticket_test >/dev/null 2>&1; then
            postgres_ready=true
            break
        fi
        sleep 1
    done

    if [ "$postgres_ready" = false ]; then
        echo -e "${RED}❌ PostgreSQL test service is not ready${NC}"
        docker-compose -f "$TEST_COMPOSE_FILE" logs postgres-test
        exit 1
    fi

    # Wait for Redis
    local redis_ready=false
    for i in {1..30}; do
        if docker-compose -f "$TEST_COMPOSE_FILE" exec -T redis-test redis-cli ping >/dev/null 2>&1; then
            redis_ready=true
            break
        fi
        sleep 1
    done

    if [ "$redis_ready" = false ]; then
        echo -e "${RED}❌ Redis test service is not ready${NC}"
        docker-compose -f "$TEST_COMPOSE_FILE" logs redis-test
        exit 1
    fi

    echo -e "${GREEN}✅ Test infrastructure is ready${NC}"
}

# Function to stop test infrastructure
stop_test_infrastructure() {
    echo -e "${YELLOW}🛑 Stopping test infrastructure...${NC}"
    cd "$PROJECT_ROOT"
    docker-compose -f "$TEST_COMPOSE_FILE" down
    echo -e "${GREEN}✅ Test infrastructure stopped${NC}"
}

# Function to run unit tests
run_unit_tests() {
    echo -e "${YELLOW}🧪 Running unit tests...${NC}"
    cd "$PROJECT_ROOT"

    # Set test environment variables
    export TEST_DB_HOST=localhost
    export TEST_DB_PORT=$POSTGRES_TEST_PORT
    export TEST_DB_NAME=smartticket_test
    export TEST_DB_USER=postgres
    export TEST_DB_PASSWORD=postgres
    export TEST_REDIS_HOST=localhost
    export TEST_REDIS_PORT=$REDIS_TEST_PORT
    export RUST_LOG=debug
    export RUST_BACKTRACE=1

    # Run unit tests for shared modules
    echo -e "${BLUE}📦 Testing shared modules...${NC}"

    local failed_tests=()

    # Test shared config
    echo -e "${BLUE}  - Testing shared config...${NC}"
    if ! cargo test --package smartticket-shared-config; then
        failed_tests+=("shared-config")
    fi

    # Test shared error
    echo -e "${BLUE}  - Testing shared error...${NC}"
    if ! cargo test --package smartticket-shared-error; then
        failed_tests+=("shared-error")
    fi

    # Test shared auth
    echo -e "${BLUE}  - Testing shared auth...${NC}"
    if ! cargo test --package smartticket-shared-auth; then
        failed_tests+=("shared-auth")
    fi

    # Test shared database (requires test infrastructure)
    if [ "$SCOPE" = "unit" ] || [ "$SCOPE" = "all" ]; then
        echo -e "${BLUE}  - Testing shared database...${NC}"
        if ! cargo test --package smartticket-shared-database; then
            failed_tests+=("shared-database")
        fi
    fi

    # Report results
    if [ ${#failed_tests[@]} -eq 0 ]; then
        echo -e "${GREEN}✅ All unit tests passed!${NC}"
    else
        echo -e "${RED}❌ Some unit tests failed: ${failed_tests[*]}${NC}"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    echo -e "${YELLOW}🔗 Running integration tests...${NC}"
    cd "$PROJECT_ROOT"

    # Set test environment variables
    export TEST_DB_HOST=localhost
    export TEST_DB_PORT=$POSTGRES_TEST_PORT
    export TEST_DB_NAME=smartticket_test
    export TEST_DB_USER=postgres
    export TEST_DB_PASSWORD=postgres
    export TEST_REDIS_HOST=localhost
    export TEST_REDIS_PORT=$REDIS_TEST_PORT
    export RUST_LOG=debug
    export RUST_BACKTRACE=1

    # Run integration tests
    echo -e "${BLUE}🔗 Running integration tests...${NC}"

    if cargo test --test integration; then
        echo -e "${GREEN}✅ Integration tests passed!${NC}"
    else
        echo -e "${RED}❌ Integration tests failed${NC}"
        return 1
    fi
}

# Function to run performance tests
run_performance_tests() {
    echo -e "${YELLOW}⚡ Running performance tests...${NC}"
    cd "$PROJECT_ROOT"

    # Set test environment variables
    export TEST_DB_HOST=localhost
    export TEST_DB_PORT=$POSTGRES_TEST_PORT
    export TEST_DB_NAME=smartticket_test
    export TEST_DB_USER=postgres
    export TEST_DB_PASSWORD=postgres
    export TEST_REDIS_HOST=localhost
    export TEST_REDIS_PORT=$REDIS_TEST_PORT
    export RUST_LOG=info

    # Run performance tests
    echo -e "${BLUE}⚡ Running performance tests...${NC}"

    if cargo test --test performance; then
        echo -e "${GREEN}✅ Performance tests passed!${NC}"
    else
        echo -e "${RED}❌ Performance tests failed${NC}"
        return 1
    fi
}

# Function to run all tests
run_all_tests() {
    echo -e "${YELLOW}🧪 Running all tests...${NC}"

    local failed_tests=()

    # Run unit tests
    if ! run_unit_tests; then
        failed_tests+=("unit")
    fi

    # Run integration tests
    if ! run_integration_tests; then
        failed_tests+=("integration")
    fi

    # Run performance tests
    if ! run_performance_tests; then
        failed_tests+=("performance")
    fi

    # Report results
    if [ ${#failed_tests[@]} -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}❌ Some tests failed: ${failed_tests[*]}${NC}"
        return 1
    fi
}

# Function to run code coverage
run_coverage() {
    echo -e "${YELLOW}📊 Running test coverage...${NC}"
    cd "$PROJECT_ROOT"

    if ! command_exists cargo-tarpaulin; then
        echo -e "${YELLOW}📦 Installing cargo-tarpaulin...${NC}"
        cargo install cargo-tarpaulin
    fi

    # Set test environment variables
    export TEST_DB_HOST=localhost
    export TEST_DB_PORT=$POSTGRES_TEST_PORT
    export TEST_DB_NAME=smartticket_test
    export TEST_DB_USER=postgres
    export TEST_DB_PASSWORD=postgres
    export TEST_REDIS_HOST=localhost
    export TEST_REDIS_PORT=$REDIS_TEST_PORT

    # Run coverage
    cargo tarpaulin --out Html --output-dir target/coverage --workspace --exclude-files '*/tests/*' --skip-clean

    echo -e "${GREEN}📊 Coverage report generated in target/coverage/tarpaulin-report.html${NC}"
}

# Function to clean test artifacts
clean() {
    echo -e "${YELLOW}🧹 Cleaning test artifacts...${NC}"
    cd "$PROJECT_ROOT"

    # Stop test containers
    docker-compose -f "$TEST_COMPOSE_FILE" down -v 2>/dev/null || true

    # Clean cargo artifacts
    cargo clean

    # Remove test data directory
    rm -rf ./test_data

    echo -e "${GREEN}✅ Test artifacts cleaned${NC}"
}

# Main execution logic
case "$COMMAND" in
    "start")
        start_test_infrastructure
        ;;
    "stop")
        stop_test_infrastructure
        ;;
    "unit")
        start_test_infrastructure
        run_unit_tests
        ;;
    "integration")
        start_test_infrastructure
        run_integration_tests
        ;;
    "performance")
        start_test_infrastructure
        run_performance_tests
        ;;
    "all")
        start_test_infrastructure
        if run_all_tests; then
            echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
        else
            echo -e "${RED}❌ Some tests failed${NC}"
            exit 1
        fi
        ;;
    "coverage")
        start_test_infrastructure
        run_coverage
        ;;
    "clean")
        clean
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Test Runner"
        echo ""
        echo "Usage: $0 [COMMAND] [SCOPE]"
        echo ""
        echo "Commands:"
        echo "  start         Start test infrastructure"
        echo "  stop          Stop test infrastructure"
        echo "  unit          Run unit tests"
        echo "  integration   Run integration tests"
        echo "  performance   Run performance tests"
        echo "  all           Run all tests (default)"
        echo "  coverage      Run test coverage analysis"
        echo "  clean         Clean test artifacts"
        echo "  help          Show this help message"
        echo ""
        echo "Scopes:"
        echo "  unit          Unit tests only"
        echo "  integration   Integration tests only"
        echo "  all           All tests (default)"
        echo ""
        echo "Examples:"
        echo "  $0                    # Run all tests"
        echo "  $0 unit               # Run unit tests only"
        echo "  $0 start              # Start test infrastructure"
        echo "  $0 integration        # Run integration tests"
        echo "  $0 coverage           # Generate coverage report"
        echo "  $0 clean              # Clean everything"
        ;;
    *)
        echo -e "${RED}❌ Unknown command: $COMMAND${NC}"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac

# Cleanup on exit
trap 'stop_test_infrastructure' EXIT