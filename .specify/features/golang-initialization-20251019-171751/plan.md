# Implementation Plan: Go Backend Infrastructure Initialization

## Technical Context

### Technology Stack
- **Backend Language**: Golang 1.21+
- **Web Framework**: GIN v1.9+ (REST API)
- **ORM**: GORM v1.25+
- **Database**: SQLite 3.41+ (embedded database)
- **Configuration**: Viper
- **Testing**: Go standard library + Testify

### Architecture Decisions
- **Pattern**: Clean Architecture with clear separation of concerns
- **Project Structure**: Standard Go project layout with cmd/, internal/, pkg/ directories
- **Database Strategy**: SQLite with separate databases for dev/test/prod
- **Configuration**: Environment variables + YAML files with Viper
- **API Design**: RESTful JSON APIs with consistent response format

### Key Dependencies
- GIN web framework for HTTP routing and middleware
- GORM for database operations and migrations
- Viper for configuration management
- golang-jwt/jwt for authentication (future)
- Testify for testing utilities

### Integration Points
- SQLite database for data persistence
- File system for configuration and log files
- HTTP endpoints for API serving
- External dependencies via Go modules

## Constitution Check

### Port Policy Compliance
- ✅ Using non-standard port 6533 (avoids forbidden ports: 3000, 8000, 8080, 9000, 9001)
- ✅ Port configurable via environment variables
- ✅ Documented in specification

### Testing Requirements
- ✅ 100% test coverage requirement specified
- ✅ Clean test database isolation
- ✅ Integration with Go standard testing framework

### Data Sovereignty
- ✅ SQLite embedded database (self-hosted)
- ✅ Complete data export capabilities planned
- ✅ No external data dependencies for core functionality

### Code Quality Standards
- ✅ No hardcoded responses (data from database only)
- ✅ Structured error handling
- ✅ Clean Architecture principles
- ✅ Comprehensive logging

### Performance Requirements
- ✅ API response time targets (< 200ms P95)
- ✅ Memory usage limits (< 512MB)
- ✅ Startup time requirements (< 5 seconds)

## Phase 0: Research

### Research Tasks
1. **Go Project Structure Best Practices**: Research standard Go project layouts and conventions
2. **SQLite Integration Patterns**: Investigate SQLite best practices for Go applications
3. **Configuration Management**: Study Viper configuration patterns for Go services
4. **Testing Framework Setup**: Research Go testing strategies with GORM and SQLite
5. **Build and Deployment**: Study Go binary building and deployment strategies

### Decision Outcomes

✅ **COMPLETED**: All research tasks completed with comprehensive findings

**Key Decisions Made**:
- **Project Structure**: Clean Architecture with standard Go layout
- **Database**: SQLite with GORM, WAL mode, and connection pooling
- **Web Framework**: GIN with enterprise middleware stack
- **Configuration**: Viper with hierarchical configuration management
- **Testing**: Comprehensive testing with isolated databases
- **Build Strategy**: Optimized single binary deployment

**Research Documentation**: See `research.md` for detailed findings and rationales

## Phase 1: Design & Contracts

### Data Model Design

✅ **COMPLETED**: Comprehensive data model with multi-tenant architecture

**Core Entities Designed**:
- **Tenant**: Multi-tenant isolation and configuration
- **User**: User management with role-based access control
- **Ticket**: Core ticketing entity with full lifecycle management
- **Message**: Communication within tickets
- **KnowledgeArticle**: Knowledge base for documentation
- **LLMProvider**: AI/LLM service integration configuration
- **ImportExportJob**: Batch data processing operations
- **Attachment**: File attachments for various entities
- **Department**: Organizational hierarchy for user management (CLARIFIED: Separate from products/SLAs)

**Key Features**:
- Multi-tenant data isolation with tenant_id in all entities
- Comprehensive validation rules and database constraints
- Performance indexes and query optimization
- Audit trails and soft delete support
- JSON fields for flexible configuration
- **Department Structure**: Pure organizational hierarchy, not tied to products or SLAs (per clarification 2025-10-19)

**Documentation**: See `data-model.md` for complete entity definitions

### API Contracts

✅ **COMPLETED**: Full REST API specification with OpenAPI 3.0

**API Endpoints Designed**:
- **Health Check**: `/health`, `/health/ready`
- **Authentication**: `/auth/login`, `/auth/refresh`, `/auth/logout`
- **Tickets**: Full CRUD operations, messages, search
- **Knowledge Base**: Articles, categories, search functionality
- **Data Management**: Import/export operations with job tracking

**API Features**:
- RESTful JSON APIs with consistent response format
- Comprehensive error handling with structured error responses
- Multi-tenant request isolation via X-Tenant-ID header
- JWT-based authentication and authorization
- Pagination, sorting, and filtering support
- OpenAPI 3.0 specification for client generation

**Documentation**: See `contracts/openapi.yaml` for complete API specification

### Agent Context Updates

✅ **COMPLETED**: AI agent context updated with new technology stack

**Updated Context Includes**:
- Complete technology stack overview
- Architecture decisions and project structure
- Development guidelines and best practices
- Recent changes and implementation details
- Quality assurance requirements

**Documentation**: Context updated in `.claude/CLAUDE.md` for Claude agent

## Constitution Check - Post Design Validation

✅ **CONSTITUTION COMPLIANCE VERIFIED**: All design decisions align with project constitution

### Port Policy Compliance
- ✅ **Port 6533**: Non-standard port avoids conflicts with forbidden ports (3000, 8000, 8080, 9000, 9001)
- ✅ **Configurable**: Port configuration via environment variables and config files
- ✅ **Documented**: Port choices documented in quickstart and configuration files

### Testing Requirements
- ✅ **100% Coverage**: Comprehensive testing strategy with unit, integration, and E2E tests
- ✅ **Test Isolation**: Isolated test databases with clean state management
- ✅ **No Skipping**: Full test suite execution requirement enforced
- ✅ **Standard Framework**: Using Go standard testing framework with Testify
- ✅ **Lint Compliance**: golangci-lint configuration updated for version 2.x compatibility

### Data Sovereignty & Self-Hosting
- ✅ **SQLite Embedded**: Complete self-hosting capability with embedded database
- ✅ **Data Export**: Import/export functionality designed for complete data portability
- ✅ **No External Dependencies**: Core functionality operates without external services
- ✅ **Offline Capable**: Full offline deployment and operation support
- ✅ **Permission Data Control**: All permission and role data stored in embedded database

### Code Quality Standards
- ✅ **No Hardcoded Data**: All data sourced from database with proper data access layer
- ✅ **Structured Error Handling**: Comprehensive error handling with custom error types
- ✅ **Clean Architecture**: Clear separation between domain, application, and infrastructure layers
- ✅ **Comprehensive Logging**: Structured logging with correlation IDs and context
- ✅ **Permission System Design**: Flexible RBAC + resource-based permissions without hardcoded roles

### Performance Requirements
- ✅ **API Response Targets**: < 200ms P95 response time targets established
- ✅ **Memory Limits**: < 512MB memory usage limits with monitoring
- ✅ **Startup Time**: < 5 seconds startup requirements with optimization strategies
- ✅ **Permission Lookup Performance**: Database indexes optimized for permission evaluation queries

### Security Requirements
- ✅ **No Sensitive Logging**: Configuration with sensitive data encryption and secure logging
- ✅ **Secure Defaults**: Production-ready default configurations
- ✅ **Access Controls**: JWT-based authentication with role-based authorization
- ✅ **Permission Granularity**: Resource-action level permissions (e.g., "tickets:read") for fine-grained access control
- ✅ **Multi-Tenant Isolation**: Permission checks include tenant validation for data isolation
- ✅ **Admin Flexibility**: Permission system designed for admin configuration without code changes

## Phase 2: Implementation Progress Update

### Current Implementation Status (2025-10-20)

**✅ COMPLETED INFRASTRUCTURE TASKS**:
- GOL-001: Project Structure Initialization - Complete with Clean Architecture layout
- GOL-002: Dependency Management - All required dependencies configured and working
- GOL-003: Configuration Management - Viper-based configuration system implemented
- GOL-004: Database Setup - SQLite with GORM, health checks, and connection management
- GOL-005: Basic Web Server - GIN server with middleware stack on port 6533
- GOL-006: Logging Infrastructure - Structured logging with Zap implemented
- GOL-007: Error Handling System - Comprehensive error types and middleware
- GOL-008: Utility Packages - Complete utility suite (validation, crypto, datetime, etc.)
- GOL-009: Linting Configuration - golangci-lint configured with version 2.x compatibility
- GOL-010: Basic Testing Framework - Test infrastructure with isolated databases

### 🔄 IN PROGRESS
- GOL-011: Permission System Implementation - Hybrid RBAC + Resource-Based Permissions designed
- GOL-012: Authentication Handlers - JWT-based auth service with role-based authorization
- GOL-013: Data Model Implementation - Core entities with permission system relationships

### 📋 REMAINING TASKS
- GOL-014 through GOL-020: Docker, documentation, environment setup
- Phase 2: Core implementation (Domain entities, repositories, services)
- Phase 3: Quality assurance and validation

### Permission System Integration (2025-10-20 Clarification)

**🎯 KEY ARCHITECTURE DECISIONS**:
- **Hybrid RBAC + Resource-Based Permissions**: Combines role inheritance with resource-level permissions
- **Fine-Grained Granularity**: Resource + Action level (e.g., "tickets:read", "tickets:write")
- **Separate Permissions Table**: Independent permissions table with many-to-many relationships
- **Predefined Permission Templates**: Default role templates (Admin, Manager, Agent, Customer) that can be customized
- **Database Lookup per Request**: Query permissions each time for maximum consistency and real-time updates

**📊 ENTITIES DESIGNED**:
1. **Permission**: Granular permissions with resource:action format
2. **Role**: Tenant-scoped role definitions with system/custom types
3. **RolePermission**: Many-to-many relationship between roles and permissions
4. **UserPermission**: Direct permission assignments with expiration support

**✅ INTEGRATION STATUS**:
- Data model updated with permission system entities
- Database indexes designed for performance optimization
- Migration scripts prepared with default permission templates
- Foreign key constraints and check constraints defined
- Comprehensive validation rules implemented

### Next Implementation Priorities

**IMMEDIATE (Week 1-2)**:
1. **Complete GOL-011**: Permission System Implementation
   - Implement Permission, Role, RolePermission, UserPermission models
   - Create permission evaluation middleware
   - Add permission management API endpoints
   - Implement default permission templates seeding

2. **Complete GOL-012**: Authentication Handlers
   - Complete JWT token service implementation
   - Integrate permission checking in auth middleware
   - Add role-based access control to existing endpoints
   - Implement user permission override functionality

3. **Complete GOL-013**: Data Model Implementation
   - Implement all core entities with GORM models
   - Add database migrations for permission system
   - Create seed data for default permissions and roles
   - Implement tenant isolation with proper indexing

**PHASE 1 COMPLETION (Week 3-4)**:
4. **Docker Configuration**: Multi-stage builds for production deployment
5. **Documentation Updates**: API documentation with permission examples
6. **Environment Setup**: Development and production configuration templates

**PHASE 2 START (Week 5+)**:
7. **Domain Entity Implementation**: Business logic with integrated permission checking
8. **Repository Layer**: Data access with tenant isolation and permission filtering
9. **Service Layer**: Business services with permission validation
10. **API Integration**: All endpoints protected with appropriate permission checks

### Validation Strategy
**Ready for Continued Implementation**: Core infrastructure established

- ✅ **Infrastructure Complete**: All foundational components implemented and tested
- ✅ **Clarifications Applied**: Department structure properly modeled
- ✅ **Constitution Aligned**: All implementations comply with project principles
- ✅ **Build Pipeline Working**: Go modules compile and tests pass

**Immediate Next Steps**: Continue with remaining Phase 1 tasks, then proceed to Phase 2 domain implementation

## Success Criteria

### Functional Validation
- [ ] All user scenarios execute successfully
- [ ] All functional requirements met with passing acceptance criteria
- [ ] No hardcoded data responses
- [ ] Clean separation of concerns maintained

### Quality Gates
- [ ] 100% test coverage achieved
- [ ] All linting checks pass
- [ ] Performance targets met
- [ ] Security requirements satisfied

### Documentation
- [ ] API documentation complete
- [ ] Setup instructions clear and tested
- [ ] Architecture decisions documented