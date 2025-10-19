#!/bin/bash

# Update Agent Context Script for Speckit Framework
set -euo pipefail

# Parse arguments
AGENT_TYPE="claude"

while [[ $# -gt 0 ]]; do
    case $1 in
        claude|openai|anthropic)
            AGENT_TYPE="$1"
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Determine context file based on agent type
case "$AGENT_TYPE" in
    claude)
        CONTEXT_FILE=".claude/CLAUDE.md"
        ;;
    openai)
        CONTEXT_FILE=".openai/context.md"
        ;;
    anthropic)
        CONTEXT_FILE=".anthropic/context.md"
        ;;
    *)
        echo "Error: Unsupported agent type: $AGENT_TYPE" >&2
        exit 1
        ;;
esac

# Create context file if it doesn't exist
mkdir -p "$(dirname "$CONTEXT_FILE")"
if [[ ! -f "$CONTEXT_FILE" ]]; then
    cat > "$CONTEXT_FILE" << 'EOF'
# AI Agent Context

This file contains context information for AI agents working on this project.

## Project Overview

SmartTicket is a self-hosted multi-tenant ticketing and knowledge collaboration platform.

## Technology Stack

- **Backend**: Go 1.21+ with GIN framework
- **Database**: SQLite with GORM ORM
- **Authentication**: JWT-based authentication
- **Configuration**: Viper configuration management
- **Testing**: Go standard library + Testify

## Architecture

- Clean Architecture pattern with clear separation of concerns
- Multi-tenant data isolation
- Single binary deployment
- Comprehensive testing strategy

## Development Guidelines

- Follow Go best practices and idiomatic patterns
- Maintain 100% test coverage
- Use structured logging throughout the application
- Implement proper error handling with custom error types
- Follow the established project structure

## Recent Changes

(Updates will be added here)

## Manual Additions

(Manual additions will be preserved here)

EOF
fi

# Read current context
CURRENT_CONTEXT=$(cat "$CONTEXT_FILE")

# Extract manual additions section
MANUAL_ADDITIONS=""
if echo "$CURRENT_CONTEXT" | grep -q "## Manual Additions"; then
    MANUAL_ADDITIONS=$(echo "$CURRENT_CONTEXT" | sed -n '/## Manual Additions/,$p')
fi

# Create new context with recent updates
NEW_CONTEXT="## Recent Changes

### Go Backend Infrastructure Initialization (2024-01-15)

**Architecture Decisions**:
- Implemented Clean Architecture with standard Go project layout
- SQLite with WAL mode for better concurrency
- GIN framework with enterprise middleware stack
- Viper configuration management with environment variable support
- Comprehensive testing strategy with isolated test databases

**Project Structure**:
\`\`\`
smartticket/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── api/                        # Interface layer
│   ├── application/                # Application services
│   ├── domain/                     # Business entities
│   ├── infrastructure/             # External implementations
│   └── config/                     # Configuration management
├── pkg/                            # Public libraries
├── migrations/                     # Database migrations
├── configs/                        # Configuration files
└── tests/                          # Test suites
\`\`\`

**Key Components**:
- **Database**: SQLite with GORM, connection pooling, WAL mode optimization
- **Web Server**: GIN with CORS, rate limiting, structured logging middleware
- **Configuration**: Viper with YAML files and environment variables
- **Authentication**: JWT-based auth with role-based access control
- **Testing**: Unit, integration, and E2E tests with isolated databases

**Data Models**:
- Multi-tenant entities with tenant isolation
- Core entities: Tenant, User, Ticket, Message, KnowledgeArticle, LLMProvider, ImportExportJob, Attachment
- Comprehensive validation rules and database constraints
- Performance indexes and query optimization

**API Design**:
- RESTful JSON APIs with consistent response format
- OpenAPI 3.0 specification
- Comprehensive error handling with structured error responses
- Health check endpoints for monitoring

**Development Tools**:
- Makefile with common development commands
- Comprehensive testing framework
- Docker support for containerized deployment
- Production-ready build optimization

**Configuration Management**:
- Environment-specific configurations (dev/test/prod)
- Sensitive data encryption for API keys
- Comprehensive configuration validation
- Graceful configuration loading with defaults

**Quality Assurance**:
- 100% test coverage requirement
- Comprehensive linting with golangci-lint
- Security scanning with gosec
- Performance monitoring and health checks

$MANUAL_ADDITIONS"

# Write updated context
echo "$NEW_CONTEXT" > "$CONTEXT_FILE"

echo "Context updated for $AGENT_TYPE agent: $CONTEXT_FILE"