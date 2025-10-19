# Quickstart Guide

## Overview

This guide provides step-by-step instructions for setting up and running the SmartTicket Go backend infrastructure. This is the initial foundation that enables the full SmartTicket platform functionality.

## Prerequisites

### System Requirements

- **Operating System**: Linux, macOS, or Windows (with WSL2)
- **Go**: Version 1.21 or higher
- **SQLite**: Version 3.41 or higher
- **Git**: For version control
- **Disk Space**: Minimum 500MB for development
- **Memory**: Minimum 4GB RAM (8GB recommended)

### Development Tools (Recommended)

- **IDE/Editor**: VS Code, GoLand, or similar with Go extension
- **Database Tool**: DB Browser for SQLite or similar
- **API Client**: Postman, Insomnia, or curl
- **Docker**: Optional, for containerized development

## Installation Steps

### 1. Clone Repository

```bash
git clone https://github.com/company/smartticket.git
cd smartticket
```

### 2. Install Go Dependencies

```bash
# Download all Go module dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy dependencies (remove unused ones)
go mod tidy
```

### 3. Create Configuration

Create a development configuration file:

```bash
mkdir -p configs
cp configs/config.dev.yaml.example configs/config.dev.yaml
```

Edit `configs/config.dev.yaml` with your development settings:

```yaml
environment: development
server:
  port: 6533
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120
  max_header_bytes: 1048576
  host: "localhost"

database:
  type: sqlite
  connection_url: "./data/smartticket_dev.db"
  max_connections: 25
  max_idle_conns: 5
  conn_max_lifetime: 300
  log_level: debug

logger:
  level: debug
  format: json
  output: stdout

jwt:
  secret: "your-super-secret-jwt-key-change-in-production"
  expiration_time: 24h
  refresh_time: 168h
  issuer: "smartticket"

cors:
  allowed_origins:
    - "http://localhost:3000"
    - "http://localhost:7218"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "OPTIONS"

rate_limit:
  requests_per_second: 100
  burst: 200
```

### 4. Initialize Database

Run database migrations:

```bash
# Create data directory
mkdir -p data

# Run migrations
go run cmd/server/main.go migrate

# Alternative: Use Makefile
make migrate
```

### 5. Start Development Server

```bash
# Start development server
go run cmd/server/main.go serve

# Alternative: Use Makefile
make dev
```

The server should start on port 6533. You should see output similar to:

```
[INFO] Starting SmartTicket server on port 6533
[INFO] Database connection established
[INFO] Server ready to accept connections
```

## Verification

### 1. Health Check

Verify the server is running:

```bash
curl http://localhost:6533/health
```

Expected response:

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0.0",
    "checks": {
      "database": {
        "status": "healthy",
        "message": "Database connection successful"
      },
      "memory": {
        "status": "healthy",
        "message": "Memory usage: 45.23 MB"
      }
    }
  },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req_123456789"
  }
}
```

### 2. API Endpoints Test

Test basic API functionality:

```bash
# Test tenant creation (if endpoint available)
curl -X POST http://localhost:6533/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Tenant",
    "domain": "test.example.com",
    "plan": "basic"
  }'
```

## Development Workflow

### 1. Make Development Commands

Use the provided Makefile for common development tasks:

```bash
# Start development server
make dev

# Run all tests
make test

# Run tests with coverage
make coverage

# Build production binary
make build

# Run linting
make lint

# Clean build artifacts
make clean

# Install dependencies
make deps

# Run database migrations
make migrate
```

### 2. Code Structure

The project follows Clean Architecture principles:

```
smartticket/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── api/                        # HTTP handlers, middleware, routes
│   │   ├── handlers/               # Request handlers
│   │   ├── middleware/             # HTTP middleware
│   │   └── routes/                 # Route definitions
│   ├── application/                # Business logic services
│   │   └── services/               # Application services
│   ├── domain/                     # Business entities and rules
│   │   └── entities/               # Domain entities
│   ├── infrastructure/             # External implementations
│   │   ├── database/               # Database operations
│   │   └── repositories/           # Data access layer
│   └── config/                     # Configuration management
├── pkg/                            # Public libraries
│   ├── logger/                     # Logging utilities
│   └── validator/                  # Validation utilities
├── migrations/                     # Database migrations
├── configs/                        # Configuration files
├── tests/                          # Test files
└── data/                           # Database files
```

### 3. Adding New Features

1. **Define Domain Entity**: Add to `internal/domain/entities/`
2. **Create Repository**: Add to `internal/infrastructure/repositories/`
3. **Implement Service**: Add to `internal/application/services/`
4. **Add HTTP Handler**: Add to `internal/api/handlers/`
5. **Define Routes**: Add to `internal/api/routes/`
6. **Write Tests**: Add comprehensive tests for all layers

### 4. Database Changes

1. **Create Migration**: Add new migration file to `migrations/`
2. **Update Models**: Update Go struct definitions
3. **Run Migration**: Apply migration to development database
4. **Test Changes**: Verify data integrity and API functionality

## Configuration

### Environment Variables

You can override configuration using environment variables:

```bash
# Set server port
export SMARTTICKET_SERVER_PORT=6533

# Set database URL
export SMARTTICKET_DATABASE_CONNECTION_URL="./data/custom.db"

# Set JWT secret
export SMARTTICKET_JWT_SECRET="your-secret-key"

# Set log level
export SMARTTICKET_LOGGER_LEVEL=debug
```

### Configuration Files

The application supports multiple configuration files:

- `configs/config.yaml` - Default configuration
- `configs/config.dev.yaml` - Development overrides
- `configs/config.prod.yaml` - Production overrides
- `configs/config.test.yaml` - Test configuration

Configuration is loaded in this priority order:
1. Environment variables
2. Configuration files (highest priority file wins)
3. Default values

## Testing

### 1. Run Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 2. Test Categories

```bash
# Run unit tests only
go test -tags=unit ./...

# Run integration tests
go test -tags=integration ./...

# Run end-to-end tests
go test -tags=e2e ./...
```

### 3. Test Database

Tests use isolated in-memory SQLite databases. No setup required:

```bash
# Run tests with clean database
go test -v ./internal/infrastructure/repositories/
```

## Production Deployment

### 1. Build Production Binary

```bash
# Build optimized binary
make build-optimized

# Or manually:
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -tags=netgo \
  -installsuffix netgo \
  -o bin/smartticket-linux-amd64 \
  cmd/server/main.go
```

### 2. Configuration

Create production configuration:

```yaml
# configs/config.prod.yaml
environment: production
server:
  port: 6533
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

database:
  type: sqlite
  connection_url: "/var/lib/smartticket/smartticket.db"
  max_connections: 50
  max_idle_conns: 10
  log_level: error

logger:
  level: info
  format: json
  output: file
  file_path: "/var/log/smartticket/app.log"
  max_size_mb: 100
  max_backups: 5
  max_age_days: 30

jwt:
  secret: "${SMARTTICKET_JWT_SECRET}"
  expiration_time: 24h
  refresh_time: 168h
```

### 3. Docker Deployment

```bash
# Build Docker image
docker build -t smartticket:latest .

# Run with Docker Compose
docker-compose up -d
```

### 4. Systemd Service

Create systemd service file:

```ini
# /etc/systemd/system/smartticket.service
[Unit]
Description=SmartTicket Service
After=network.target

[Service]
Type=simple
User=smartticket
Group=smartticket
WorkingDirectory=/opt/smartticket
ExecStart=/opt/smartticket/smartticket serve --config /etc/smartticket/config.yaml
Restart=always
RestartSec=5
Environment=SMARTTICKET_JWT_SECRET=your-production-secret

[Install]
WantedBy=multi-user.target
```

Enable and start service:

```bash
sudo systemctl enable smartticket
sudo systemctl start smartticket
sudo systemctl status smartticket
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check what's using port 6533
   lsof -i :6533

   # Kill the process or change port in config
   ```

2. **Database Connection Failed**
   ```bash
   # Check database file permissions
   ls -la data/smartticket_dev.db

   # Ensure data directory exists
   mkdir -p data
   chmod 755 data
   ```

3. **Go Module Issues**
   ```bash
   # Clean module cache
   go clean -modcache

   # Re-download dependencies
   go mod download
   ```

4. **Build Failures**
   ```bash
   # Ensure Go version is correct
   go version

   # Clean build artifacts
   make clean

   # Rebuild
   make build
   ```

### Debug Mode

Enable debug logging:

```yaml
# configs/config.dev.yaml
logger:
  level: debug
  format: json
  output: stdout
```

### Health Monitoring

Monitor application health:

```bash
# Check health status
curl http://localhost:6533/health

# Check readiness
curl http://localhost:6533/health/ready

# Monitor logs
tail -f logs/app.log
```

## Next Steps

After completing the infrastructure setup:

1. **Review API Documentation**: Check `contracts/openapi.yaml` for available endpoints
2. **Create Test Data**: Use the API or direct database access to create test data
3. **Explore Features**: Implement business logic for tickets, users, and knowledge base
4. **Set Up CI/CD**: Configure automated testing and deployment pipelines
5. **Monitor Performance**: Set up monitoring and alerting for production use

## Support

For additional support:

1. **Documentation**: Check the `docs/` directory for detailed guides
2. **Issues**: Report bugs and feature requests on GitHub
3. **Community**: Join the development community for discussions and updates

## Architecture Notes

The SmartTicket backend follows these architectural principles:

- **Single Binary**: All functionality packaged in one executable
- **Multi-Tenant**: Built-in support for tenant isolation
- **Clean Architecture**: Clear separation of concerns
- **Testability**: Comprehensive testing at all layers
- **Configuration**: Flexible configuration management
- **Observability**: Structured logging and health monitoring
- **Security**: JWT authentication and role-based access control

This infrastructure provides a solid foundation for building enterprise-grade ticketing and knowledge management features.