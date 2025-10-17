# Development Tools

This directory contains development and testing utility scripts for SmartTicket.

## Available Tools

### test.sh
Comprehensive test runner for SmartTicket.

**Usage:**
```bash
# Run all tests (default)
./test.sh

# Run specific test types
./test.sh unit              # Unit tests only
./test.sh integration        # Integration tests
./test.sh performance       # Performance tests

# Test infrastructure management
./test.sh start             # Start test infrastructure
./test.sh stop              # Stop test infrastructure
./test.sh coverage          # Generate coverage report
./test.sh clean             # Clean test artifacts
```

**Features:**
- Docker-based test infrastructure
- Unit, integration, and performance tests
- Automatic service startup/teardown
- Test coverage analysis with cargo-tarpaulin
- Comprehensive logging and error reporting

### generate-openapi.sh
Generates OpenAPI documentation from proto files.

**Usage:**
```bash
./generate-openapi.sh
```

**Requirements:**
- protoc
- protoc-gen-go
- protoc-gen-go-grpc
- protoc-gen-openapiv2

**Output:**
- `api/smartticket.v1.openapi.json`
- `api/smartticket.v1.openapi.yaml` (if yq is available)
- Interactive Swagger UI compatible files

### cleanup-temp-files.sh
Cleans up temporary files and ensures professional naming conventions.

**Usage:**
```bash
./cleanup-temp-files.sh
```

**Features:**
- Removes temporary directories
- Cleans up unprofessional file names
- Verifies API documentation structure
- Ensures consistent naming conventions

## Prerequisites

### Docker & Docker Compose
Required for running the test infrastructure:
```bash
# Install Docker
# macOS: Download from docker.com
# Linux: sudo apt-get install docker.io docker-compose
# Windows: Download from docker.com
```

### Development Tools
For OpenAPI generation:
```bash
# Install protocol buffers
go install google.golang.org/protobuf/cmd/protoc@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

### Test Coverage (Optional)
```bash
# Install cargo-tarpaulin for coverage analysis
cargo install cargo-tarpaulin
```

## Configuration

### Test Environment
The test script uses the following environment variables:
- `TEST_DB_HOST` - PostgreSQL test host (default: localhost)
- `TEST_DB_PORT` - PostgreSQL test port (default: 5433)
- `TEST_DB_NAME` - Test database name (default: smartticket_test)
- `TEST_REDIS_HOST` - Redis test host (default: localhost)
- `TEST_REDIS_PORT` - Redis test port (default: 6380)

### Docker Compose Files
- `docker/docker-compose.test.yml` - Test infrastructure configuration

## Usage Examples

### Full Test Suite
```bash
# Start test infrastructure and run all tests
./test.sh all

# Run tests with coverage
./test.sh coverage

# Clean up after testing
./test.sh clean
```

### Development Workflow
```bash
# 1. Generate API documentation
./generate-openapi.sh

# 2. Clean up any temporary files
./cleanup-temp-files.sh

# 3. Run unit tests
./test.sh unit

# 4. Stop test infrastructure
./test.sh stop
```

## Notes

- These tools are intended for development and testing environments only
- Do not use in production environments
- Ensure Docker and required tools are installed before running scripts
- Test scripts will automatically start/stop required services