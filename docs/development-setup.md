# Development Setup Guide

This guide provides comprehensive instructions for setting up a SmartTicket development environment.

## Prerequisites

### Required Software

1. **Go 1.25+**
   ```bash
   # Install using Homebrew (macOS)
   brew install go

   # Or download from https://go.dev/dl/

   # Verify installation
   go version
   ```

2. **Git**
   ```bash
   # Install using Homebrew (macOS)
   brew install git

   # Verify installation
   git --version
   ```

3. **SQLite** (usually included with Go)
   ```bash
   # Install using Homebrew (macOS)
   brew install sqlite

   # Verify installation
   sqlite3 --version
   ```

### Optional but Recommended

1. **Docker & Docker Compose**
   ```bash
   # Download and install from https://docs.docker.com/get-docker/

   # Verify installation
   docker --version
   docker-compose --version
   ```

2. **Development Tools**
   ```bash
   # golangci-lint for linting
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

   # air for hot reload (if using Docker)
   go install github.com/cosmtrek/air@latest

   # watchexec for file watching
   go install github.com/watchexec/watchexec@latest
   ```

3. **IDE/Editor Setup**
   - **VS Code**: Install Go extension pack
   - **GoLand**: JetBrains IDE for Go development
   - **Vim/Neovim**: vim-go plugin

## Quick Start (5 minutes)

For experienced developers who want to get started quickly:

```bash
# 1. Clone repository
git clone https://github.com/company/smartticket.git
cd smartticket

# 2. Install dependencies and set up environment
make env-setup

# 3. Start development server
make dev

# Server is now running on http://localhost:6533
```

## Detailed Setup

### Step 1: Clone and Initial Setup

```bash
# Clone the repository
git clone https://github.com/company/smartticket.git
cd smartticket

# Verify you're on the correct branch
git branch -a

# Set up Go workspace
go work init
go work use .
```

### Step 2: Environment Configuration

```bash
# Copy environment template
cp .env.example .env

# Edit environment variables
nano .env  # or use your preferred editor
```

Key environment variables to configure:

```bash
# Application
PORT=6533
ENVIRONMENT=development
LOG_LEVEL=debug

# Database
DB_PATH=./data/smartticket_dev.db

# Security (change these!)
JWT_SECRET=your-super-secret-jwt-key-here

# CORS (for frontend development)
FRONTEND_URL=http://localhost:3000
```

### Step 3: Install Dependencies

```bash
# Download and verify Go modules
make deps

# Check required tools are installed
make env-check

# Set up development environment (installs missing tools)
make env-setup
```

### Step 4: Database Setup

```bash
# Create data directory
make db-setup

# The database will be created automatically on first run
# SQLite file: ./data/smartticket_dev.db
```

### Step 5: Start Development Server

```bash
# Option 1: Using Make (recommended)
make dev

# Option 2: Direct Go command
go run cmd/server/main.go serve --config configs/config.dev.yaml

# Option 3: Docker (with hot reload)
docker-compose -f deployments/docker-compose.dev.yml up -d
```

### Step 6: Verify Setup

```bash
# Check health endpoint
curl http://localhost:6533/api/v1/health

# Expected response:
# {"success": true, "data": {"status": "healthy", ...}}
```

## Development Workflow

### Daily Development

```bash
# 1. Pull latest changes
git pull origin main

# 2. Update dependencies
make deps-update

# 3. Start development server
make dev

# 4. In another terminal, run tests
make test

# 5. Make changes...

# 6. Format and lint code
make fmt
make lint

# 7. Run full test suite
make check

# 8. Commit changes
git add .
git commit -m "feat: add new feature"
git push
```

### Testing

```bash
# Run all tests
make test

# Run specific test types
make test-unit          # Unit tests only
make test-integration   # Integration tests
make test-e2e          # End-to-end tests

# Run tests with coverage
make test-cover

# Run tests with race detector
make test-race

# Watch for changes and run tests
make watch-test
```

### Building

```bash
# Build for local platform
make build-local

# Build for production (Linux)
make build

# Build for all platforms
make build-all

# Cross-platform build example
GOOS=windows GOARCH=amd64 go build -o smartticket.exe cmd/server/main.go
```

## IDE Configuration

### VS Code

Install the following extensions:
- Go (golang.go)
- Docker (ms-azuretools.vscode-docker)
- GitLens (eamodio.gitlens)

Create `.vscode/settings.json`:

```json
{
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "gutter",
        "coveredHighlightColor": "rgba(64,128,64,0.5)",
        "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
    }
}
```

### GoLand

1. Open the project directory
2. Configure Go SDK (Settings → Go → GOROOT)
3. Enable indexing and code completion
4. Configure live templates for common patterns
5. Set up file watchers for go fmt

### Vim/Neovim

Install vim-go:
```bash
git clone https://github.com/fatih/vim-go.git ~/.vim/pack/plugins/start/vim-go
```

Configure in `.vimrc`:
```vim
" Go settings
let g:go_fmt_command = "goimports"
let g:go_lint_command = "golangci-lint"
let g:go_test_show_name = 1
let g:go_test_autosave = 1
```

## Docker Development

### Development with Hot Reload

```bash
# Start development environment
docker-compose -f deployments/docker-compose.dev.yml up -d

# View logs
docker-compose -f deployments/docker-compose.dev.yml logs -f smartticket-dev

# Stop environment
docker-compose -f deployments/docker-compose.dev.yml down
```

### Development Services

The development environment includes:
- **SmartTicket**: Main application (port 6533)
- **Redis**: Cache service (port 6379)
- **Adminer**: Database admin (port 8080)
- **MailHog**: Email testing (port 8025)

### Production Docker

```bash
# Build production image
docker build -t smartticket:latest .

# Run production environment
docker-compose -f deployments/docker-compose.yml up -d

# Scale application
docker-compose -f deployments/docker-compose.yml up -d --scale smartticket=3
```

## Troubleshooting

### Port Issues

If you encounter port conflicts:

```bash
# Check what's using port 6533
lsof -i :6533

# Kill the process
kill -9 <PID>

# Or use a different port in .env
PORT=6534 make dev
```

### Dependency Issues

```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download && go mod verify

# Update dependencies
go get -u ./...
go mod tidy
```

### Test Failures

```bash
# Clean test artifacts
make clean-temp

# Reset database
make db-reset

# Run tests with verbose output
go test -v ./...

# Run specific failing test
go test -v ./internal/config -run TestLoad
```

### Docker Issues

```bash
# Clean Docker system
docker system prune -f

# Rebuild images
docker-compose -f deployments/docker-compose.dev.yml build --no-cache

# Check container logs
docker-compose -f deployments/docker-compose.dev.yml logs smartticket-dev
```

### Performance Issues

```bash
# Profile Go application
go tool pprof http://localhost:6533/debug/pprof/profile

# Check memory usage
go tool pprof http://localhost:6533/debug/pprof/heap

# Monitor goroutines
go tool pprof http://localhost:6533/debug/pprof/goroutine
```

## Best Practices

### Code Organization

- Follow Clean Architecture principles
- Keep business logic in the domain layer
- Use dependency injection
- Write tests before writing code (TDD)
- Keep functions small and focused

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/new-feature

# Commit frequently with meaningful messages
git commit -m "feat: add user authentication"

# Use conventional commit types
# feat: new feature
# fix: bug fix
# docs: documentation
# style: formatting
# refactor: refactoring
# test: tests
# chore: maintenance

# Keep commits small and focused
git add .
git commit -m "feat: add JWT token validation"

# Push and create pull request
git push origin feature/new-feature
```

### Testing Strategy

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **End-to-End Tests**: Test complete user workflows

Target 80%+ code coverage.

### Performance

- Use connection pooling for database
- Implement caching where appropriate
- Profile regularly to identify bottlenecks
- Use appropriate data structures
- Avoid premature optimization

## Getting Help

### Resources

- **Documentation**: `docs/` directory
- **API Documentation**: Available when server is running
- **Issues**: GitHub Issues page
- **Discussions**: GitHub Discussions

### Debug Commands

```bash
# Check application status
make info

# Show all Make commands
make help

# Check environment
env | grep SMARTTICKET

# View logs
tail -f logs/smartticket.log

# Database operations
sqlite3 data/smartticket_dev.db ".tables"
```

### Community

- Join our Slack/Discord channel
- Participate in GitHub discussions
- Follow project updates
- Contribute to documentation