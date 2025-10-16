# SmartTicket E2E Test Suite

This directory contains comprehensive end-to-end tests for the SmartTicket system using `grpcurl` for direct gRPC service testing.

## Overview

The E2E test suite validates the complete functionality of the SmartTicket platform including:

- **Authentication Service**: User login, token management, and authorization
- **Ticket Management**: CRUD operations, status management, assignments, and comments
- **Knowledge Base**: Article creation, publishing, search, and categorization
- **Multi-Tenant Isolation**: Data separation between tenants and cross-tenant access prevention
- **Performance & Load**: System responsiveness under various load conditions

## Prerequisites

### Required Tools

- **grpcurl**: gRPC command-line client
- **jq**: JSON processor for parsing responses
- **bash**: Shell environment (version 4.0+)

### Installation

```bash
# Install grpcurl
brew install grpcurl

# Install jq
brew install jq

# Or download from GitHub releases
# https://github.com/fullstorydev/grpcurl/releases
```

### Service Requirements

- SmartTicket gRPC Gateway service running on configured port (default: 50052)
- PostgreSQL database with test data
- Redis for caching (if configured)

## Configuration

### Environment Variables

Create a `.env` file in the project root or set environment variables:

```bash
# Service Configuration
export GRPC_SERVER_HOST=localhost
export GRPC_SERVER_PORT=50052

# Test Configuration
export TEST_TENANT_DOMAIN=testcompany.example.com
export TEST_ADMIN_EMAIL=admin@testcompany.example.com
export TEST_ADMIN_PASSWORD=testpass123
export TEST_USER_EMAIL=user@testcompany.example.com
export TEST_USER_PASSWORD=testpass123

# Performance Test Configuration
export LOAD_TEST_USERS=10
export LOAD_TEST_DURATION=30
export RESPONSE_TIME_THRESHOLD=2000  # milliseconds
```

### Test Configuration Files

- `config/test.yaml`: Test environment configuration
- `test_config.sh`: Test configuration and helper functions
- `test_helpers.sh`: Common test utilities and gRPC wrappers

## Test Structure

```
tests/e2e/
├── README.md                    # This file
├── test_config.sh              # Configuration and utilities
├── test_helpers.sh             # Helper functions and assertions
├── run_all_tests.sh            # Main test runner
├── auth_tests.sh               # Authentication service tests
├── ticket_tests.sh             # Ticket management tests
├── knowledge_tests.sh          # Knowledge base tests
├── multi_tenant_tests.sh       # Multi-tenant isolation tests
├── performance_tests.sh        # Performance and load tests
├── test_data/                  # Test data directory (created automatically)
└── test_results/               # Test results directory (created automatically)
```

## Running Tests

### Quick Start

```bash
# Run all tests
./run_all_tests.sh

# Run specific test suite
./run_all_tests.sh --auth-only

# Run with verbose output
./run_all_tests.sh -v

# Run only fast tests (skip performance tests)
./run_all_tests.sh --fast
```

### Test Suite Options

#### Individual Test Suites

```bash
# Authentication tests
./auth_tests.sh

# Ticket management tests
./ticket_tests.sh

# Knowledge base tests
./knowledge_tests.sh

# Multi-tenant isolation tests
./multi_tenant_tests.sh

# Performance and load tests
./performance_tests.sh
```

#### Main Test Runner Options

```bash
# Show help
./run_all_tests.sh --help

# Run specific suites
./run_all_tests.sh auth ticket knowledge

# Run only performance tests
./run_all_tests.sh --performance-only

# Run with custom configuration
GRPC_SERVER_PORT=50052 ./run_all_tests.sh
```

### Performance Testing

Performance tests can be customized with environment variables:

```bash
# Number of concurrent users for load testing
export LOAD_TEST_USERS=20

# Duration of sustained load test (seconds)
export LOAD_TEST_DURATION=60

# Response time threshold (milliseconds)
export RESPONSE_TIME_THRESHOLD=1000

# Maximum concurrent operations
export CONCURRENT_OPS_THRESHOLD=100

# Run performance tests
./performance_tests.sh
```

## Test Coverage

### Authentication Service Tests

- ✅ User login with valid credentials
- ✅ User login with invalid password (failure case)
- ✅ User login with non-existent user (failure case)
- ✅ User login with invalid tenant domain (failure case)
- ✅ Token refresh functionality
- ✅ Token refresh with invalid token (failure case)
- ✅ User creation and management
- ✅ Current user profile operations
- ✅ Password change functionality
- ✅ User permissions retrieval
- ✅ Access without authentication token (failure case)

### Ticket Management Tests

- ✅ Ticket creation with all fields
- ✅ Ticket retrieval by ID
- ✅ Ticket update operations
- ✅ Ticket status management
- ✅ Ticket assignment operations
- ✅ Ticket comment management
- ✅ Ticket listing with pagination and filtering
- ✅ Ticket search functionality
- ✅ Ticket deletion (soft delete)
- ✅ Complete ticket lifecycle testing

### Knowledge Base Tests

- ✅ Knowledge category creation and management
- ✅ Knowledge article creation with all fields
- ✅ Article retrieval with view counting
- ✅ Article update operations
- ✅ Article publication workflow
- ✅ Article search functionality
- ✅ Article listing with filtering
- ✅ Article rating system
- ✅ Article suggestions for tickets
- ✅ Article archival and deletion
- ✅ Complete knowledge article lifecycle

### Multi-Tenant Isolation Tests

- ✅ Tenant-specific authentication
- ✅ Cross-tenant ticket access prevention
- ✅ Cross-tenant knowledge base access prevention
- ✅ User management isolation
- ✅ Data scoping in list operations
- ✅ Cross-tenant token validation

### Performance and Load Tests

- ✅ Authentication performance under load
- ✅ Ticket CRUD performance metrics
- ✅ Concurrent user load testing
- ✅ Sustained load testing
- ✅ Memory and resource usage monitoring

## Test Data and Cleanup

### Test Data Management

- Test data is created dynamically during test execution
- Each test uses timestamps to ensure data uniqueness
- Test results are stored in `test_results/` directory
- Performance metrics are saved as JSON reports

### Cleanup Procedures

- Automatic cleanup is performed after each test suite
- Test users, tickets, and articles are created with specific patterns
- Temporary files are removed automatically
- Failed tests may leave residual data for debugging

## Results and Reporting

### Test Output

Tests provide detailed output including:
- Individual test results (PASSED/FAILED)
- Response time measurements
- Error messages and stack traces
- Performance metrics and statistics

### Report Files

- `test_results/e2e_results_YYYYMMDD_HHMMSS.txt`: Detailed test results
- `test_results/e2e_summary_YYYYMMDD_HHMMSS.json`: Summary in JSON format
- `test_results/performance_results_YYYYMMDD_HHMMSS.json`: Performance metrics

### Performance Metrics

Performance tests generate comprehensive metrics:
- Response times (average, min, max)
- Throughput measurements
- Success rates
- Memory usage patterns
- Concurrent operation results

## Troubleshooting

### Common Issues

1. **gRPC Server Not Running**
   ```bash
   # Check if server is running
   grpcurl -plaintext localhost:50052 list

   # Start the server
   cargo run -p smartticket-gateway
   ```

2. **Authentication Failures**
   - Verify test user credentials in environment variables
   - Check tenant domain configuration
   - Ensure database has test data

3. **Permission Errors**
   ```bash
   # Make scripts executable
   chmod +x tests/e2e/*.sh
   ```

4. **Missing Dependencies**
   ```bash
   # Install required tools
   which grpcurl || echo "grpcurl not found"
   which jq || echo "jq not found"
   ```

### Debug Mode

Enable debug output for troubleshooting:

```bash
# Enable verbose output
./run_all_tests.sh -v

# Set debug log level
export RUST_LOG=debug
cargo run -p smartticket-gateway
```

### Test Data Issues

If tests fail due to missing data:

1. Ensure test database is properly initialized
2. Check migration status: `cargo run --bin migrate`
3. Verify test tenant and user creation

## Best Practices

### Test Development

1. **Use Descriptive Names**: Test functions should clearly indicate what they test
2. **Include Assertions**: Always verify expected outcomes
3. **Test Failure Cases**: Ensure error conditions are properly handled
4. **Cleanup Resources**: Remove test data after execution
5. **Avoid Hardcoded Values**: Use environment variables for configuration

### Performance Testing

1. **Baseline Measurements**: Establish performance baselines
2. **Incremental Load**: Start with low load and increase gradually
3. **Monitor Resources**: Track memory and CPU usage
4. **Consistent Environment**: Use consistent test environment
5. **Multiple Runs**: Run tests multiple times for statistical significance

### CI/CD Integration

```bash
# Example CI script
#!/bin/bash
set -e

# Start services
docker-compose -f docker-compose.test.yml up -d

# Wait for services
sleep 30

# Run E2E tests
./tests/e2e/run_all_tests.sh --fast

# Capture exit code
TEST_RESULT=$?

# Cleanup
docker-compose -f docker-compose.test.yml down

exit $TEST_RESULT
```

## Contributing

When adding new tests:

1. Follow existing naming conventions
2. Use the helper functions in `test_helpers.sh`
3. Include both positive and negative test cases
4. Add performance metrics where applicable
5. Update this README with new test coverage

## License

This test suite is part of the SmartTicket project and follows the same license terms.