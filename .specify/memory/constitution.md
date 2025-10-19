# Project Constitution

## Core Principles

### 1. Port Policy Compliance
- **MUST**: Use non-standard ports to avoid conflicts (forbidden: 3000, 8000, 8080, 9000, 9001, 5173, 4200, 7000, 5000)
- **MUST**: Make ports configurable via environment variables
- **SHOULD**: Document port choices in configuration

### 2. Testing Standards
- **MUST**: Maintain 100% test coverage for all code
- **MUST**: Use isolated test databases
- **MUST**: No test skipping or bypassing
- **SHOULD**: Use standard Go testing framework

### 3. Data Sovereignty & Self-Hosting
- **MUST**: Support complete data export capabilities
- **MUST**: Use embedded SQLite for self-hosting
- **MUST**: No external data dependencies for core functionality
- **SHOULD**: Enable offline deployment

### 4. Code Quality Standards
- **MUST**: No hardcoded data responses in production code
- **MUST**: Implement structured error handling
- **MUST**: Follow Clean Architecture principles
- **SHOULD**: Use structured logging

### 5. Performance Requirements
- **MUST**: API response time P95 < 200ms
- **MUST**: Memory usage < 512MB for deployment
- **SHOULD**: Startup time < 5 seconds

### 6. Security Requirements
- **MUST**: Never log sensitive configuration data
- **MUST**: Use secure defaults for production
- **SHOULD**: Implement proper access controls

### 7. Development Workflow
- **MUST**: Single binary deployment capability
- **MUST**: Comprehensive documentation
- **SHOULD**: Automated build and test processes