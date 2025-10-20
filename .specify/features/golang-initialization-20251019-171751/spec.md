# Go Backend Infrastructure Initialization

## Overview

**Context**: SmartTicket is a self-hosted multi-tenant ticketing and knowledge collaboration platform that needs a robust Go backend to support enterprise deployment requirements. The project currently has design documentation but lacks the actual Go implementation foundation.

**Problem**: Without the Go backend infrastructure in place, the project cannot proceed with core functionality development, testing, or deployment. The architecture is defined but needs to be translated into working code.

**Solution**: Initialize the complete Go backend project structure with all foundational components including project layout, dependencies, configuration management, database setup, basic API framework, and development tooling.

## Clarifications

### Session 2025-10-19

- Q: How should departments be structured in relation to products and SLA policies? → A: Department model only for organizational hierarchy, not tied to products or SLAs
- Q: What permission system architecture should be implemented? → A: Hybrid RBAC + Resource-Based Permissions - combines role inheritance with resource-level permissions
- Q: What level of permission granularity is needed? → A: Resource + Action Level - fine-grained permissions like "tickets:read", "tickets:write"
- Q: How should permissions be stored and managed? → A: Separate Permissions Table + Role + User Assignments - independent permissions table with many-to-many relationships
- Q: What default permission sets should be provided? → A: Predefined Permission Templates - default role templates (Admin, Manager, Agent, Customer) that can be customized
- Q: How should permissions be evaluated for performance? → A: Database Lookup per Request - query permissions each time for maximum consistency and real-time updates

## User Scenarios & Testing

### Primary User Scenarios

1. **Developer Sets Up Development Environment**
   - **Given**: A developer wants to start working on the SmartTicket backend
   - **When**: They clone the repository and run the setup commands
   - **Then**: All dependencies are installed, database is initialized, and the development server starts successfully

2. **Developer Runs Initial Tests**
   - **Given**: The Go project structure is in place
   - **When**: Developer runs `go test ./... -v`
   - **Then**: All tests pass with 100% success rate and clean linting results

3. **Developer Starts Development Server**
   - **Given**: Dependencies are installed and database is initialized
   - **When**: Developer runs the main application with serve command
   - **Then**: Server starts on configured port (6533) and responds to health check endpoints

4. **Developer Builds Production Binary**
   - **Given**: Application code is ready for deployment
   - **When**: Developer runs build command
   - **Then**: Single binary is created that can run without external dependencies

### Edge Cases

1. **Database Initialization Failure**
   - **Given**: SQLite database file is corrupted or missing
   - **When**: Application tries to start
   - **Then**: Application logs appropriate error message and exits gracefully

2. **Missing Configuration File**
   - **Given**: Configuration file is not present
   - **When**: Application starts
   - **Then**: Application uses sensible defaults and logs warnings about missing config

3. **Port Already in Use**
   - **Given**: Configured port (6533) is already occupied
   - **When**: Application tries to bind to port
   - **Then**: Application fails with clear error message indicating port conflict

## Functional Requirements

### Core Functionality

1. **GOL-001 - Project Structure Initialization**
   - **Description**: Create complete Go project directory structure following standard conventions
   - **Acceptance Criteria**:
     - All required directories (cmd/, internal/, pkg/, tests/, configs/) are created
     - Go module is initialized with appropriate module name
     - Basic main.go entry point is created with serve and migrate commands
     - Project follows Clean Architecture principles as defined in CLAUDE.md

2. **GOL-002 - Dependency Management**
   - **Description**: Set up all required third-party dependencies and Go modules
   - **Acceptance Criteria**:
     - go.mod contains all required dependencies (GIN, GORM, JWT, Viper, etc.)
     - Dependencies are locked with compatible versions
     - go.sum is generated and up to date
     - All dependencies can be downloaded successfully with `go mod download`

3. **GOL-003 - Configuration Management**
   - **Description**: Implement flexible configuration system using Viper
   - **Acceptance Criteria**:
     - Support for environment variables and config files (YAML)
     - Default configuration values for development environment
     - Configuration validation for required fields
     - Secure handling of sensitive configuration (API keys, database credentials)

4. **GOL-004 - Database Setup**
   - **Description**: Initialize SQLite database with proper schema and migrations
   - **Acceptance Criteria**:
     - SQLite database files are created in appropriate directories (dev/test/prod)
     - Database connection is established with proper configuration
     - Basic migration system is in place for schema updates
     - Database health check endpoint is functional

5. **GOL-005 - Basic Web Server**
   - **Description**: Set up GIN web framework with basic routing and middleware
   - **Acceptance Criteria**:
     - Server starts on non-standard port 6533
     - Basic health check endpoint (/api/v1/health) returns success status
     - Request logging middleware is configured
     - CORS middleware is properly configured
     - Structured error handling is implemented

### Development Tooling

6. **GOL-006 - Testing Infrastructure**
   - **Description**: Set up comprehensive testing framework and utilities
   - **Acceptance Criteria**:
     - Test database setup with isolated test environment
     - Basic test utilities and helpers are available
     - Test coverage reporting is configured
     - Sample unit tests are provided for core components

7. **GOL-007 - Build and Development Scripts**
   - **Description**: Create Makefile and scripts for common development tasks
   - **Acceptance Criteria**:
     - Makefile with targets for dev, test, build, and clean
     - Development server can be started with `make dev`
     - Production binary can be built with `make build`
     - All tests can be run with `make test`

## Non-Functional Requirements

### Performance

- Application startup time must be under 5 seconds on typical development machine
- Health check endpoint response time under 50ms
- Memory usage under 100MB in idle state

### Security

- Configuration with sensitive data must not be logged
- Default configuration must not expose development secrets in production
- Database connection must use appropriate security settings

### Reliability

- Application must handle graceful shutdown on SIGTERM/SIGINT
- Database connections must be properly managed and closed
- Error logging must capture sufficient context for debugging

## Success Criteria

### Primary Success Metrics

1. **Development Environment Setup Time**: New developers can set up and run the application in under 10 minutes
2. **Test Success Rate**: 100% of initial tests pass without failures
3. **Build Success**: Production binary builds successfully on first attempt
4. **Server Startup**: Development server starts and responds to health checks consistently

### User Experience Goals

- Developer can run `make dev` and have a working development server immediately
- Clear error messages when setup steps fail
- Comprehensive documentation for setup and development procedures
- Consistent project structure that follows Go best practices

## Scope & Boundaries

### In Scope

- Go project initialization and structure
- Basic web server with health checks
- Database setup and configuration
- Configuration management system
- Development tooling and testing framework
- Basic API structure and middleware

### Out of Scope

- Complete business logic implementation
- Full API endpoints for tickets, users, etc.
- Frontend integration
- Production deployment configurations
- Database migrations for specific business entities
- Department-product relationship modeling (departments are organizational only)
- SLA policy assignment to departments (SLAs are configured independently)

## Assumptions & Dependencies

### Assumptions

- Developer has Go 1.21+ installed
- SQLite is available on the development machine
- Git is initialized in the project directory
- Developer has basic familiarity with Go development

### Dependencies

- Go 1.21+ toolchain
- SQLite 3.41+ for database functionality
- Network access for downloading Go modules
- File system permissions for creating directories and files

## Key Entities

1. **Application Server**
   - **Description**: Main HTTP server handling API requests
   - **Key Attributes**: Port configuration, middleware stack, route handlers
   - **Relationships**: Depends on database connection, configuration, and logging

2. **Database Manager**
   - **Description**: Handles SQLite database connections and operations
   - **Key Attributes**: Connection string, connection pool settings, health status
   - **Relationships**: Used by application server, provides data access to services

3. **Configuration Manager**
   - **Description**: Loads and validates application configuration
   - **Key Attributes**: Config file paths, environment variables, default values
   - **Relationships**: Provides configuration to all application components

4. **Department Structure**
   - **Description**: Organizational hierarchy for user management and role assignment
   - **Key Attributes**: Department name, department type (sales, pre-sales, post-sales, support, engineering), parent department relationships
   - **Relationships**: Departments contain users with specific roles, separate from product ownership and SLA policy assignments

5. **Permission System**
   - **Description**: Flexible role-based access control with resource-action permissions
   - **Key Attributes**: Permission code (resource:action), description, category, default role assignments
   - **Relationships**: Permissions assigned to roles, roles assigned to users, users can have direct permissions

6. **Role Management**
   - **Description**: Role definitions with associated permission sets
   - **Key Attributes**: Role name, description, role type (system/custom), permission assignments
   - **Relationships**: Roles have many permissions, users belong to many roles, tenant-scoped

## Risks & Mitigations

### High Risk Items

1. **Dependency Version Conflicts**
   - **Impact**: Application fails to build or run due to incompatible dependency versions
   - **Probability**: Medium - Go ecosystem changes frequently
   - **Mitigation**: Pin specific versions in go.mod, test with clean module cache

2. **Database Setup Issues**
   - **Impact**: Application cannot start due to database initialization failures
   - **Probability**: Low - SQLite is relatively simple, but file permissions can cause issues
   - **Mitigation**: Clear error messages, automatic directory creation, fallback configurations

3. **Port Configuration Conflicts**
   - **Impact**: Development server fails to start due to port conflicts
   - **Probability**: Medium - Port 6533 is uncommon but not guaranteed to be available
   - **Mitigation**: Detect port conflicts, suggest alternative ports, configurable port settings