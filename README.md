# SmartTicket

A self-hosted single-tenant ticketing and knowledge collaboration platform designed for enterprise deployment.

## Overview

SmartTicket provides enterprise-grade ticketing and knowledge management with:
- **Self-Hosting**: Complete control over data and infrastructure
- **Single-Tenant**: Simple deployment model - one instance per organization
- **Single Binary**: Zero external dependencies for core functionality
- **Custom AI Integration**: Support for any LLM provider or local models
- **Data Sovereignty**: Complete data export and portability features

## Technology Stack

- **Backend**: Go 1.21+ with Clean Architecture
- **Database**: SQLite with GORM ORM
- **Web Framework**: GIN for REST APIs
- **Authentication**: JWT-based authentication
- **Configuration**: Viper configuration management
- **Testing**: Go standard library + Testify

## Project Structure

```
smartticket/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── api/                        # API layer (handlers, middleware, routes)
│   ├── application/                # Application services
│   ├── domain/                     # Domain entities and business rules
│   ├── infrastructure/             # External implementations (database, APIs)
│   ├── config/                     # Configuration management
│   └── utils/                      # Utility functions
├── pkg/                            # Public libraries (logger, cache, validator)
├── migrations/                     # Database migrations
├── configs/                        # Configuration files
├── tests/                          # Test suites
├── data/                           # Database files
├── logs/                           # Application logs
└── scripts/                        # Build and deployment scripts
```

## Quick Start

### Prerequisites

- Go 1.25+ installed
- SQLite 3.41+ (usually included with Go)
- Docker & Docker Compose (optional, for containerized deployment)
- Make (optional, for convenience commands)

### Installation

#### Option 1: Direct Go Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/company/smartticket.git
   cd smartticket
   ```

2. **Install dependencies**:
   ```bash
   make deps
   # or: go mod download && go mod verify
   ```

3. **Set up environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Set up database**:
   ```bash
   make db-setup
   ```

5. **Start development server**:
   ```bash
   make dev
   # or: go run cmd/server/main.go serve --config configs/config.dev.yaml
   ```

The server will start on port 6533 by default.

#### Option 2: Docker Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/company/smartticket.git
   cd smartticket
   ```

2. **Set up environment**:
   ```bash
   cp .env.example .env
   ```

3. **Start development environment**:
   ```bash
   docker-compose -f deployments/docker-compose.dev.yml up -d
   ```

4. **View logs**:
   ```bash
   docker-compose -f deployments/docker-compose.dev.yml logs -f smartticket-dev
   ```

The application will be available at http://localhost:6533 with hot reload enabled.

### Health Check

Verify the server is running:

```bash
curl http://localhost:6533/api/v1/health
```

Expected response:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": 1703123456,
    "version": "1.0.0"
  }
}
```

### Configuration

Create a configuration file at `configs/config.dev.yaml`:

```yaml
server:
  port: 6533
  read_timeout: 30
  write_timeout: 30

database:
  connection_url: "./data/smartticket_dev.db"
  max_connections: 25
  log_level: debug

logger:
  level: debug
  format: json
  output: stdout
```

### Health Check

Verify the server is running:

```bash
curl http://localhost:6533/health
```

## Development

### Build and Run

```bash
# Build production binary
go build -o smartticket cmd/server/main.go

# Run with configuration
./smartticket serve --config configs/config.prod.yaml
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Development Commands

#### Using Make (Recommended)

```bash
# Environment setup
make env-setup      # Set up development environment
make env-check      # Check required tools

# Development
make dev            # Start development server
make dev-debug      # Start with debug logging
make run            # Run without building
make watch          # Watch for changes and rebuild

# Building
make build          # Build production binary (Linux)
make build-local    # Build for local platform
make build-all      # Build for all platforms

# Testing
make test           # Run all tests
make test-short     # Run short tests only
make test-race      # Run with race detector
make test-cover     # Generate coverage report
make test-integration # Run integration tests
make test-e2e       # Run end-to-end tests

# Code Quality
make fmt            # Format code
make lint           # Run linter
make lint-fix       # Fix lint issues
make vet            # Run go vet
make check          # Run all checks
make pre-commit     # Run pre-commit checks

# Database
make db-setup       # Set up development database
make db-reset       # Reset development database
make migrate        # Run migrations

# Docker
make docker-build   # Build Docker image
make docker-run     # Run with Docker Compose
make docker-stop    # Stop Docker Compose
make docker-clean   # Clean Docker resources

# Utilities
make clean          # Clean build artifacts
make clean-all      # Clean everything
make help           # Show all available commands
make version        # Show version info
make info           # Show project info

# Quick Start
make quickstart     # Full setup and start for new developers
```

#### Using Go Directly

```bash
# Development
go run cmd/server/main.go serve --config configs/config.dev.yaml

# Building
go build -o smartticket cmd/server/main.go

# Testing
go test ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Dependencies
go mod download
go mod tidy
go mod verify

# Formatting
gofmt -s -w .
go vet ./...
```

## Architecture

SmartTicket follows Clean Architecture principles with clear separation of concerns:

- **Domain Layer**: Business entities and rules
- **Application Layer**: Use cases and application services
- **Infrastructure Layer**: External implementations (database, APIs)
- **Interface Layer**: API handlers and middleware

## Features

### Ticket Management
- Full ticket lifecycle management
- Priority and severity tracking
- Assignment and status management
- Message threading and communication
- File attachments and metadata

### Knowledge Base
- Article creation and versioning
- Full-text search capabilities
- Category and tag management
- Access control and visibility settings

### Multi-Tenancy
- Complete data isolation between tenants
- Tenant-specific configurations
- Role-based access control per tenant
- Resource usage monitoring and limits

### AI Integration
- Support for multiple LLM providers
- Custom API endpoint configuration
- Task-specific model mapping
- Cost monitoring and quota management

### Data Management
- Complete data export (CSV, JSON, XML, Markdown)
- Third-party system migration (Zendesk, Jira)
- Automated backup and recovery
- Point-in-time recovery capabilities

## API Documentation

API documentation is available at:
- **OpenAPI Specification**: `contracts/openapi.yaml`
- **Interactive Docs**: Available when server is running

## Development Environment

### Environment Variables

Key environment variables for development:

```bash
# Application
PORT=6533                      # Server port (non-standard to avoid conflicts)
ENVIRONMENT=development        # Environment name
LOG_LEVEL=debug               # Logging level
GIN_MODE=debug               # Gin framework mode

# Database
DB_PATH=./data/smartticket_dev.db  # SQLite database path
DB_TYPE=sqlite               # Database type
DB_MAX_CONNECTIONS=10        # Max connections

# Authentication
JWT_SECRET=your-secret-key   # JWT signing secret (change in production!)
JWT_EXPIRATION_TIME=1h       # Token expiration
JWT_ISSUER=smartticket       # JWT issuer

# CORS (for frontend development)
FRONTEND_URL=http://localhost:3000

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_SECOND=100
RATE_LIMIT_BURST=200
```

### Port Configuration

SmartTicket uses non-standard ports to avoid conflicts:

- **Main API**: 6533 (not 8080, 3000, or 8000)
- **Development Tools**:
  - Adminer (DB admin): 8080
  - MailHog (email testing): 8025
  - Redis: 6379

### Project Structure Details

```
smartticket/
├── cmd/server/main.go          # Application entry point
├── internal/
│   ├── api/                    # API layer
│   │   ├── handlers/           # HTTP request handlers
│   │   ├── middleware/         # HTTP middleware
│   │   └── routes/             # Route definitions
│   ├── application/            # Application services
│   ├── domain/                 # Business entities
│   ├── infrastructure/         # External integrations
│   ├── config/                 # Configuration management
│   ├── database/               # Database setup and migrations
│   ├── logger/                 # Logging utilities
│   ├── errors/                 # Error handling
│   └── utils/                  # Utility functions
├── pkg/                        # Public packages
├── tests/
│   ├── testutils/              # Test utilities and fixtures
│   ├── integration/            # Integration tests
│   └── e2e/                    # End-to-end tests
├── deployments/                # Deployment configurations
│   ├── docker/                 # Docker files
│   ├── k8s/                    # Kubernetes manifests
│   └── nginx/                  # Nginx configuration
├── docs/                       # Documentation
├── configs/                    # Configuration files
├── migrations/                 # Database migrations
├── data/                       # Database files (gitignored)
├── logs/                       # Log files (gitignored)
└── build/                      # Build output (gitignored)
```

### Hot Reload with Docker

The development Docker environment includes hot reload using `air`:
- Changes to Go files automatically rebuild the application
- Configuration changes require manual restart
- Logs are streamed in real-time

### Testing Strategy

The project includes multiple test levels:

1. **Unit Tests**: `internal/*/..._test.go`
2. **Integration Tests**: `tests/integration/`
3. **End-to-End Tests**: `tests/e2e/`
4. **Test Infrastructure**: `tests/testutils/`

Run tests with isolation:
```bash
make test              # All tests
make test-integration  # Integration tests only
make test-e2e         # E2E tests only
```

## Troubleshooting

### Common Issues

1. **Port 6533 already in use**:
   ```bash
   # Find process using port
   lsof -i :6533
   # Kill process
   kill -9 <PID>
   ```

2. **Database permission errors**:
   ```bash
   # Ensure data directory exists and is writable
   mkdir -p data
   chmod 755 data
   ```

3. **Go version mismatch**:
   ```bash
   # Check Go version
   go version
   # Update to Go 1.25+ if needed
   ```

4. **Docker build failures**:
   ```bash
   # Check Docker daemon
   docker info
   # Clean Docker cache
   docker system prune -f
   ```

5. **Test failures**:
   ```bash
   # Clean test artifacts
   make clean-temp
   # Reset test database
   make db-reset
   # Run tests with verbose output
   go test -v ./...
   ```

### Debug Mode

Enable debug logging:
```bash
# Using Make
make dev-debug

# Using environment variables
LOG_LEVEL=debug go run cmd/server/main.go serve

# In Docker
LOG_LEVEL=debug docker-compose -f deployments/docker-compose.dev.yml up smartticket-dev
```

### Getting Help

- Check logs: `make logs` or `docker-compose logs`
- Run health check: `curl http://localhost:6533/api/v1/health`
- Review configuration: Check `.env` and `configs/config.dev.yaml`
- Check dependencies: `make env-check`

## Contributing

1. **Fork the repository**
2. **Set up development environment**:
   ```bash
   git clone <your-fork>
   cd smartticket
   make env-setup
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/amazing-feature
   ```
4. **Make your changes**:
   - Write tests for new functionality
   - Ensure all tests pass: `make test`
   - Run linting: `make lint`
   - Format code: `make fmt`
5. **Commit your changes**:
   ```bash
   git commit -m 'feat: add amazing feature'
   ```
6. **Push to the branch**:
   ```bash
   git push origin feature/amazing-feature
   ```
7. **Open a Pull Request**
   - Include tests for new features
   - Update documentation if needed
   - Ensure CI checks pass

### Code Style

- Follow Go formatting standards (`gofmt`)
- Write meaningful commit messages (Conventional Commits)
- Add tests for all new functionality
- Keep functions small and focused
- Use meaningful variable and function names

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the documentation in the `docs/` directory
- Review the troubleshooting guide

## Roadmap

See the project management section for current roadmap and release planning.