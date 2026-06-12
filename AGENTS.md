# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Project Overview

SmartTicket is a self-hosted single-tenant ticketing and knowledge collaboration platform designed for individual enterprise deployment. Organizations deploy their own instance to serve their customers, providing end-to-end issue handling, knowledge collaboration, and AI-powered assistance (custom RAG/LLM integration) while maintaining complete data sovereignty.

## Core Features

### 🏢 **Enterprise Self-Deployment**
- **Single Binary Deployment**: One executable file, zero external dependencies
- **Zero-Dependency Installation**: Built-in SQLite database, 5-minute deployment
- **Private Network Support**: Complete offline deployment capability
- **Low Resource Usage**: Memory usage < 512MB, suitable for SME deployment

### 📊 **Data Sovereignty & Control**
- **Complete Data Export**: Full export of tickets, knowledge base, users, configurations
- **Multi-Format Support**: CSV, JSON, XML, Markdown, SQLite formats
- **Intelligent Import**: Third-party system migration (Zendesk, Jira, etc.)
- **Automated Backup**: Scheduled backups with point-in-time recovery

### 🤖 **Custom LLM Provider Integration**
- **Multi-Provider Support**: OpenAI, Azure OpenAI, Anthropic Codex, DeepSeek, local models
- **Flexible Configuration**: Custom API endpoints, authentication, model parameters
- **Task-Model Mapping**: Configure specific models for different AI tasks
- **Cost Control**: Quota limits, usage monitoring, expense optimization

## Technology Stack

### Backend Architecture
- **Language**: Golang 1.21+
- **Web Framework**: GIN v1.9+ (REST API)
- **ORM**: GORM v1.25+
- **Database**: SQLite 3.41+ (embedded database)
- **Authentication**: JWT (golang-jwt/jwt)
- **Configuration**: Viper
- **Logging**: Logrus/Zap
- **Testing**: Go standard library + Testify

### Frontend (Optional)
- **Framework**: React + TypeScript (Next.js/SPA)
- **UI Library**: Ant Design or MUI
- **Build**: Vite or Create React App

### Deployment
- **Container**: Docker + Docker Compose
- **Process**: Systemd service
- **Reverse Proxy**: Nginx (optional)
- **Monitoring**: Prometheus + Grafana (optional)

## System Architecture

### Monolithic Design
```
┌─────────────────────────────────────────────────────────────┐
│                    SmartTicket Backend                        │
│                   (Single Binary Executable)                     │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │   API Layer      │ │  Middleware      │ │  Business Logic  │  │
│  │                │ │                │ │                │  │
│  │ • REST API     │ │ • JWT Auth       │ │ • Ticket Mgmt    │  │
│  │ • Validation    │ │ • RBAC Control   │ │ • Knowledge Base │  │
│  │ • Error Handling│ │ • Rate Limiting  │ │ • SLA Engine     │  │
│  │ • Response Format│ │ • Logging        │ │ • AI Integration  │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │  Data Access    │ │   Services       │ │   Utilities      │  │
│  │                │ │                │ │                │  │
│  │ • GORM Models  │ │ • Import/Export  │ │ • Crypto Tools   │  │
│  │ • Database Ops  │ │ • Backup/Recovery│ │ • Validation     │  │
│  │ • Transaction   │ │ • Notifications  │ │ • File Processing│  │
│  │ • Connection Pool│ │ • LLM Providers  │ │ • Cache Manager  │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │  SQLite DB      │ │  File Storage    │ │ External APIs    │  │
│  │                │ │                │ │                │  │
│  │ • Primary DB    │ │ • Attachments    │ │ • Email Service  │  │
│  │ • Vector Data   │ │ • Export Files   │ │ • LLM APIs       │  │
│  │ • Config Data   │ │ • Backup Files   │ │ • Webhooks       │  │
│  │ • Audit Logs    │ │ • Temp Files     │ │ • 3rd Party Sys  │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Project Structure

```
smartticket/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/                     # API layer
│   │   ├── handlers/            # HTTP handlers
│   │   ├── middleware/          # Middlewares
│   │   ├── routes/              # Route definitions
│   │   └── validators/          # Request validation
│   ├── models/                  # Data models (GORM)
│   ├── services/                # Business logic
│   ├── repositories/            # Data access layer
│   ├── database/                # Database configuration
│   ├── config/                  # Configuration management
│   ├── utils/                   # Utility functions
│   └── errors/                  # Error definitions
├── pkg/                         # Public libraries
│   ├── logger/
│   ├── cache/
│   └── validator/
├── api/                         # API specifications
│   └── openapi/
├── docs/                        # Documentation
├── tests/                       # Tests
├── configs/                     # Configuration files
├── deployments/                 # Deployment configs
├── scripts/                     # Build/deployment scripts
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Core Data Models

### Key Entities
- **User**: Role-based access (admin/agent/customer)
- **Ticket**: Full lifecycle with SLA tracking and priority/severity enums
- **Message**: Ticket conversations with AI support
- **KnowledgeArticle**: Versioned content with visibility controls
- **Product/Service**: Service catalog and support scope
- **LlmProvider**: Custom LLM provider configurations
- **ImportExportJob**: Batch data processing with progress tracking

### Important Constraints
- Single-tenant deployment model (no tenant_id needed)
- Tickets use soft deletes (`is_deleted` flag)
- Audit logs are immutable with hash-based integrity
- All timestamp fields use Unix time format

## Development Phases

1. **Phase 0**: Infrastructure & authentication foundation (1-2 weeks)
2. **Phase 1**: Core ticketing & SLA functionality (3-4 weeks)
3. **Phase 2**: Data management & backup systems (2-3 weeks)
4. **Phase 3**: AI service integration & custom LLM providers (4-5 weeks)
5. **Phase 4**: System optimization & production readiness (2-3 weeks)

## Security & Compliance

### Multi-Layer Security
- Field-level and record-level access control
- Encryption at rest and in transit
- API key encryption with key rotation
- Immutable audit logs with hash chaining

### Privacy & Compliance
- Data export and backup capabilities
- PII detection and masking
- Audit logging for compliance
- Secure data deletion capabilities

### Import/Export Security
- Role-based permissions for data access
- PII detection and anonymization
- Comprehensive audit trails
- Encrypted backup storage

## Custom LLM Provider System

### Supported Providers
- **Public Cloud**: OpenAI, Azure OpenAI, Anthropic Codex, Google Gemini, DeepSeek
- **Private Deployment**: Ollama, vLLM, Text Generation Inference, LocalAI, FastChat
- **Enterprise Models**: Fine-tuned models, industry-specific models, local deployments

### Configuration Management
- Multi-provider support with flexible configuration
- Task-to-model mapping for different AI tasks
- Cost monitoring and quota management
- Encrypted credential storage with rotation

### AI Task Types
- **Chat**: Conversational AI and Q&A
- **Embedding**: Text vectorization
- **Rerank**: Result re-ranking
- **Summarization**: Text summarization
- **Generation**: Content generation
- **Classification**: Text classification

## Data Import/Export

### Supported Formats
- **Tickets**: CSV, JSON, XML, Markdown
- **Knowledge Articles**: CSV, JSON, Markdown with front matter
- **Users**: CSV, JSON
- **Contracts**: CSV, JSON
- **Complete Export**: SQLite database file

### Batch Operations
- Maximum 10,000 records per file (configurable)
- 100MB file size limit (configurable)
- Concurrent job limit of 2
- Comprehensive validation and error handling
- Progress tracking and real-time status updates

### Third-Party Integrations
- **Zendesk**: Full data migration support
- **Jira Service Management**: Ticket and project data sync
- **Freshdesk**: Customer support data import
- **Custom Sources**: Generic CSV/JSON import with field mapping

## Port Configuration

### Application Port
- **Main API Service**: Port 6533 (non-standard port to avoid conflicts)
- **Health Check**: Same port as main service
- **Metrics**: Optional, configurable port

### Database Strategy
- **Development**: SQLite file at `./data/smartticket_dev.db`
- **Testing**: SQLite file at `./data/smartticket_test.db`
- **Production**: SQLite file at `./data/smartticket.db`
- **Backups**: SQLite files in `./data/backups/`

### Avoid Common Ports
Do not use: 3000, 8000, 8080, 9000, 9001, 5173, 4200, 7000, 5000

## Development Environment Setup

### Prerequisites
```bash
# Install Go 1.21+
go version

# Install SQLite
# macOS
brew install sqlite

# Ubuntu/Debian
sudo apt-get install sqlite3 libsqlite3-dev

# Install Docker (optional)
docker --version
```

### Quick Start
```bash
# 1. Clone repository
git clone <repository-url>
cd smartticket

# 2. Install dependencies
go mod download

# 3. Run database migrations
go run cmd/server/main.go migrate

# 4. Start development server
go run cmd/server/main.go serve

# Or use Makefile
make dev
```

### Development Commands
```bash
# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Build binary
go build -o smartticket cmd/server/main.go

# Run with configuration
./smartticket serve --config configs/config.dev.yaml
```

## Testing Infrastructure

### Test Structure
- **Unit Tests**: `internal/*/..._test.go`
- **Integration Tests**: `tests/integration/`
- **End-to-End Tests**: `tests/e2e/`
- **Test Fixtures**: `tests/fixtures/`

### Test Database
- **Isolated Test Database**: Separate SQLite file for testing
- **Clean State**: Each test run uses clean data
- **Test Data**: Predefined test users and organizations
- **Mock Services**: Mock external API calls for testing

### Testing Commands
```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e

# Generate coverage report
make coverage
```

## API Design

### RESTful API Structure
```
/api/v1/
├── auth/          # Authentication endpoints
├── tickets/       # Ticket management
├── knowledge/     # Knowledge base
├── users/         # User management
├── data/          # Import/export operations
├── llm/           # LLM provider management
├── admin/         # Administrative functions
└── health/        # Health checks
```

### Authentication
- **JWT Tokens**: Bearer token authentication
- **Multi-tenant**: `x-tenant-id` header required
- **Role-based**: Permissions checked per endpoint
- **Session Management**: Token refresh and expiration

### Response Format
```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "meta": {
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

## Performance Optimization

### Database Optimization
- **Indexing Strategy**: Optimized indexes for common queries
- **Query Optimization**: Efficient GORM queries with proper joins
- **Connection Pooling**: SQLite connection management
- **WAL Mode**: Write-Ahead Logging for better concurrency

### Caching Strategy
- **In-Memory Cache**: Hot data caching in application memory
- **Query Result Cache**: Cache frequently accessed data
- **Static Asset Cache**: Cache knowledge articles and templates

### Performance Targets
- **API Response Time**: P95 < 200ms
- **Database Query**: P95 < 100ms
- **AI/LLM Response**: P95 < 2s
- **Concurrent Users**: 100+ simultaneous users
- **Throughput**: 1000+ QPS

## Deployment

### Single Binary Deployment
```bash
# Build production binary
go build -ldflags="-s -w" -o smartticket cmd/server/main.go

# Run with configuration
./smartticket serve --config /path/to/config.yaml
```

### Docker Deployment
```bash
# Build Docker image
docker build -t smartticket:latest .

# Run with Docker Compose
docker-compose up -d
```

### System Service Deployment
```bash
# Install as systemd service
sudo cp deployments/smartticket.service /etc/systemd/system/
sudo systemctl enable smartticket
sudo systemctl start smartticket
```

## Monitoring & Observability

### Health Checks
- **Application Health**: `/api/v1/health` endpoint
- **Database Health**: SQLite connection status
- **External API Health**: LLM provider connectivity
- **System Health**: Memory usage, disk space, etc.

### Logging
- **Structured Logging**: JSON format with correlation IDs
- **Log Levels**: Debug, Info, Warn, Error
- **Request Tracing**: Request ID tracking across components
- **Audit Logging**: All data mutations with user context

### Metrics (Optional)
- **HTTP Metrics**: Request count, response time, error rate
- **Business Metrics**: Tickets created, SLA compliance, user activity
- **System Metrics**: Memory usage, CPU usage, disk I/O
- **AI Metrics**: LLM usage, cost tracking, response times

## Development Guidelines

### Code Standards
- **Go Formatting**: Use `gofmt` and `golint`
- **Error Handling**: Explicit error handling with proper error types
- **Testing**: Minimum 75% test coverage
- **Documentation**: Public functions must have documentation
- **Security**: Follow secure coding practices

### Git Workflow
- **Main Branch**: Production-ready code
- **Develop Branch**: Integration branch
- **Feature Branches**: `feature/description`
- **Hotfix Branches**: `hotfix/description`

### Commit Messages
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
- **Examples**:
  - `feat(api): add ticket export endpoint`
  - `fix(db): resolve connection pooling issue`
  - `docs(readme): update deployment instructions`

## Current Project State

The project is designed for single-tenant self-deployment with a focus on data sovereignty and AI integration flexibility. Each organization deploys their own instance to serve their customers. The architecture emphasizes simplicity, security, and maintainability while providing enterprise-grade ticketing and knowledge management features.

### Key Differentiators
1. **Self-Hosted**: Complete control over data and infrastructure
2. **Single-Tenant**: Simple deployment model - one instance per organization
3. **Data Export**: Comprehensive data portability features
4. **AI Flexibility**: Support for any LLM provider or local models
5. **Simple Deployment**: Single binary with minimal dependencies
6. **Enterprise Features**: RBAC, audit logs, SLA management, compliance tools

The technology stack prioritizes reliability and ease of deployment while maintaining the flexibility needed for enterprise integration and customization.