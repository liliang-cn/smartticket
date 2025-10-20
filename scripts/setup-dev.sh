#!/bin/bash

# SmartTicket Development Environment Setup Script
# This script sets up a complete development environment for SmartTicket

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GO_VERSION_REQUIRED="1.21"

# Helper functions
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

check_command() {
    if command -v "$1" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing_prereqs=()

    # Check Go
    if check_command go; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Found Go version: $GO_VERSION"

        # Compare Go versions (simple check)
        if [[ "$GO_VERSION" < "go1.21" ]] && [[ "$GO_VERSION" != "go1.25.1" ]]; then
            log_error "Go 1.21+ is required. Found: $GO_VERSION"
            missing_prereqs+=("go")
        else
            log_success "Go version is compatible"
        fi
    else
        log_error "Go is not installed"
        missing_prereqs+=("go")
    fi

    # Check Git
    if check_command git; then
        log_success "Git is available"
    else
        log_error "Git is required"
        missing_prereqs+=("git")
    fi

    # Check SQLite
    if check_command sqlite3; then
        log_success "SQLite is available"
    else
        log_warning "SQLite is not installed. It's recommended for development."
    fi

    # Check Docker (optional)
    if check_command docker; then
        log_success "Docker is available"
    else
        log_warning "Docker is not installed. It's optional but recommended."
    fi

    # Check Make
    if check_command make; then
        log_success "Make is available"
    else
        log_error "Make is required"
        missing_prereqs+=("make")
    fi

    if [[ ${#missing_prereqs[@]} -gt 0 ]]; then
        log_error "Missing prerequisites: ${missing_prereqs[*]}"
        log_info "Please install the missing prerequisites and run this script again."
        echo
        log_info "macOS installation commands:"
        echo "  brew install go git sqlite3 make"
        echo
        log_info "Ubuntu/Debian installation commands:"
        echo "  sudo apt-get update"
        echo "  sudo apt-get install golang git sqlite3 make"
        echo
        exit 1
    fi

    log_success "All prerequisites are satisfied"
}

setup_project_structure() {
    log_info "Setting up project structure..."

    cd "$PROJECT_ROOT"

    # Create necessary directories
    local directories=(
        "data"
        "data/backups"
        "data/uploads"
        "logs"
        "temp"
        "build"
        "coverage"
        "dist"
    )

    for dir in "${directories[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log_info "Created directory: $dir"
        fi
    done

    # Set proper permissions
    chmod 755 data data/backups data/uploads logs temp

    log_success "Project structure created"
}

install_dependencies() {
    log_info "Installing Go dependencies..."

    cd "$PROJECT_ROOT"

    # Download dependencies
    if ! go mod download; then
        log_error "Failed to download dependencies"
        exit 1
    fi

    # Verify dependencies
    if ! go mod verify; then
        log_error "Failed to verify dependencies"
        exit 1
    fi

    # Tidy dependencies
    if ! go mod tidy; then
        log_error "Failed to tidy dependencies"
        exit 1
    fi

    log_success "Dependencies installed"
}

setup_git_hooks() {
    log_info "Setting up Git hooks..."

    cd "$PROJECT_ROOT"

    local hooks_dir=".git/hooks"
    local scripts_hooks_dir="scripts/git-hooks"

    if [[ ! -d ".git" ]]; then
        log_warning "Not a Git repository. Skipping Git hooks setup."
        return 0
    fi

    if [[ -d "$scripts_hooks_dir" ]]; then
        # Install available hooks
        for hook in pre-commit pre-push commit-msg; do
            if [[ -f "$scripts_hooks_dir/$hook" ]]; then
                ln -sf "../../scripts/git-hooks/$hook" "$hooks_dir/$hook"
                chmod +x "$hooks_dir/$hook"
                log_info "Installed Git hook: $hook"
            fi
        done
    else
        log_warning "Git hooks directory not found. Skipping hook installation."
    fi

    log_success "Git hooks setup completed"
}

build_application() {
    log_info "Building SmartTicket application..."

    cd "$PROJECT_ROOT"

    # Build the main application
    if ! make build-local; then
        log_error "Failed to build application"
        exit 1
    fi

    # Build seed data tool
    if ! make seed-build; then
        log_error "Failed to build seed data tool"
        exit 1
    fi

    log_success "Application built successfully"
}

setup_database() {
    log_info "Setting up development database..."

    cd "$PROJECT_ROOT"

    # Run database migrations
    if [[ -f "build/smartticket" ]]; then
        if ! ./build/smartticket migrate --config configs/config.dev.yaml; then
            log_error "Failed to run database migrations"
            exit 1
        fi
        log_success "Database migrations completed"
    else
        log_warning "Built application not found. Skipping database migrations."
    fi

    # Seed development data
    if [[ -f "scripts/seed/seed" ]]; then
        if ! ./scripts/seed/seed -config configs/config.dev.yaml -force; then
            log_warning "Failed to seed development data. You can run it manually later."
        else
            log_success "Development database seeded"
        fi
    else
        log_warning "Seed tool not found. You can run 'make seed' manually later."
    fi
}

run_tests() {
    log_info "Running tests to verify setup..."

    cd "$PROJECT_ROOT"

    # Run unit tests
    if ! go test ./... -v -short; then
        log_error "Some tests failed"
        return 1
    fi

    log_success "All tests passed"
}

setup_development_config() {
    log_info "Setting up development configuration..."

    cd "$PROJECT_ROOT"

    # Create development environment file if it doesn't exist
    if [[ ! -f ".env.dev" ]]; then
        cat > .env.dev << EOF
# SmartTicket Development Environment Configuration
# Copy this file to .env.local and modify as needed

# Application
APP_ENV=development
APP_PORT=6533
APP_HOST=localhost

# Database
DB_PATH=./data/smartticket_dev.db
DB_TYPE=sqlite
DB_LOG_LEVEL=debug

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json

# Security
JWT_SECRET=dev-secret-key-change-in-production
JWT_EXPIRY=24h

# API Settings
API_RATE_LIMIT=100
API_CORS_ORIGINS=http://localhost:3000,http://localhost:6533

# File Uploads
UPLOAD_MAX_SIZE=10485760  # 10MB
UPLOAD_ALLOWED_TYPES=jpg,jpeg,png,gif,pdf,doc,docx,txt

# Development Settings
DEV_MODE=true
DEV_RELOAD=true
DEV_PROFILING=true
EOF
        log_info "Created .env.dev configuration file"
    else
        log_info ".env.dev already exists"
    fi

    # Create local environment file template
    if [[ ! -f ".env.local" ]]; then
        cp .env.dev .env.local.template
        log_info "Created .env.local.template"
        log_info "Copy .env.local.template to .env.local and customize as needed"
    fi

    log_success "Development configuration setup completed"
}

install_development_tools() {
    log_info "Installing development tools..."

    cd "$PROJECT_ROOT"

    # Install golangci-lint if not present
    if ! check_command golangci-lint; then
        log_info "Installing golangci-lint..."
        if check_command curl; then
            curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v1.54.2
            log_success "golangci-lint installed"
        else
            log_warning "curl not available. Please install golangci-lint manually."
        fi
    else
        log_success "golangci-lint is already available"
    fi

    # Install gosec if not present
    if ! check_command gosec; then
        log_info "Installing gosec..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        log_success "gosec installed"
    else
        log_success "gosec is already available"
    fi

    # Install mockgen if not present
    if ! check_command mockgen; then
        log_info "Installing mockgen..."
        go install github.com/golang/mock/mockgen@latest
        log_success "mockgen installed"
    else
        log_success "mockgen is already available"
    fi
}

setup_vscode_config() {
    log_info "Setting up VS Code configuration..."

    cd "$PROJECT_ROOT"

    local vscode_dir=".vscode"

    if [[ ! -d "$vscode_dir" ]]; then
        mkdir -p "$vscode_dir"
    fi

    # Create VS Code settings
    if [[ ! -f "$vscode_dir/settings.json" ]]; then
        cat > "$vscode_dir/settings.json" << EOF
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.gopath": "",
    "go.goroot": "",
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "workspace",
    "go.testOnSave": false,
    "go.coverOnSave": false,
    "go.coverageDecorator": {
        "type": "gutter",
        "coveredHighlightColor": "rgba(64,128,64,0.5)",
        "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
    },
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.organizeImports": true
    },
    "files.exclude": {
        "**/build": true,
        "**/dist": true,
        "**/coverage": true,
        "**/.git": true,
        "**/node_modules": true
    }
}
EOF
        log_info "Created VS Code settings"
    fi

    # Create VS Code launch configuration
    if [[ ! -f "$vscode_dir/launch.json" ]]; then
        cat > "$vscode_dir/launch.json" << EOF
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch SmartTicket",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "\${workspaceFolder}/cmd/server/main.go",
            "args": ["serve", "--config", "configs/config.dev.yaml"],
            "env": {
                "SMARTTICKET_ENV": "development"
            },
            "console": "integratedTerminal"
        },
        {
            "name": "Launch Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "\${workspaceFolder}",
            "env": {
                "SMARTTICKET_ENV": "test"
            },
            "console": "integratedTerminal"
        }
    ]
}
EOF
        log_info "Created VS Code launch configuration"
    fi

    # Create VS Code tasks
    if [[ ! -f "$vscode_dir/tasks.json" ]]; then
        cat > "$vscode_dir/tasks.json" << EOF
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Build",
            "type": "shell",
            "command": "make",
            "args": ["build-local"],
            "group": {
                "kind": "build",
                "isDefault": true
            },
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": ["$go"]
        },
        {
            "label": "Test",
            "type": "shell",
            "command": "make",
            "args": ["test"],
            "group": {
                "kind": "test",
                "isDefault": true
            },
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": ["$go"]
        },
        {
            "label": "Lint",
            "type": "shell",
            "command": "make",
            "args": ["lint"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Seed Database",
            "type": "shell",
            "command": "make",
            "args": ["seed"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
EOF
        log_info "Created VS Code tasks"
    fi

    log_success "VS Code configuration completed"
}

display_next_steps() {
    echo
    log_success "🎉 SmartTicket development environment setup completed!"
    echo
    echo "📋 Next steps:"
    echo "  1. Copy .env.dev to .env.local and customize as needed"
    echo "  2. Start the development server:"
    echo "     make dev"
    echo "     # or"
    echo "     ./build/smartticket serve --config configs/config.dev.yaml"
    echo "  3. Open your browser and navigate to http://localhost:6533"
    echo "  4. Check the health endpoint: http://localhost:6533/api/v1/health"
    echo
    echo "🛠️  Development commands:"
    echo "  make dev          - Start development server"
    echo "  make test         - Run tests"
    echo "  make lint         - Run linter"
    echo "  make seed         - Seed development database"
    echo "  make clean        - Clean build artifacts"
    echo "  make help         - Show all available commands"
    echo
    echo "📚 Useful resources:"
    echo "  - API Documentation: http://localhost:6533/docs (when running)"
    echo "  - Project README: README.md"
    echo "  - Configuration: configs/config.dev.yaml"
    echo "  - Makefile targets: make help"
    echo
    if [[ -d ".git" ]]; then
        echo "🔧 Git setup:"
        echo "  - The development environment is ready for Git operations"
        echo "  - Pre-commit hooks are installed for code quality"
        echo "  - Consider creating a feature branch for your work"
        echo
    fi
    echo "Happy coding! 🚀"
}

main() {
    echo "🚀 SmartTicket Development Environment Setup"
    echo "=========================================="
    echo

    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]]; then
        log_error "This script must be run from the project root directory (where go.mod is located)"
        exit 1
    fi

    # Run setup steps
    check_prerequisites
    setup_project_structure
    install_dependencies
    setup_git_hooks
    setup_development_config
    install_development_tools
    setup_vscode_config
    build_application
    setup_database

    # Run tests (optional, continue if they fail)
    if ! run_tests; then
        log_warning "Some tests failed, but setup continues. Please check the failing tests."
    fi

    display_next_steps
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi