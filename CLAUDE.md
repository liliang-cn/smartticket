# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SmartTicket is a B2B multi-tenant ticketing and knowledge collaboration platform designed for a 40-person European software company. The system serves enterprise customers with different support tiers (Platinum/Standard) and provides end-to-end issue handling, knowledge collaboration, and AI-powered assistance (native RAG/LLM) while maintaining GDPR compliance.

## High-Level Architecture

### Service Architecture (Simplified)
- **gRPC Gateway**: Unified entry point with mTLS, JWT/OIDC/SAML session injection
- **Core Service**: Combined Ticket + Knowledge management, SLA engine, intelligent routing
- **AI Service**: RAG/LLM document ingestion, embedding, hybrid search, prompt orchestration
- **Platform Service**: Multi-tenant, SSO, RBAC, audit, and external system integrations
- **Notification Service**: Email/chat/Push templates with throttling and retry

### Data Layer
- **PostgreSQL**: Multi-tenant with RLS (Row Level Security), strong consistency
- **OpenSearch/Elasticsearch**: Full-text search for tickets and knowledge
- **PgVector/Weaviate/Qdrant**: Vector database with multi-tenant namespace isolation
- **Redis**: Caching, sessions, rate limiting, distributed locks
- **Kafka/NATS**: Async event bus
- **S3**: Object storage for attachments and exports

### Key Design Principles
- Multi-tenant data isolation using `tenant_id` and RLS
- Zero-trust security architecture
- EU data residency compliance
- High-performance with P95 < 300ms for ticket search, P95 < 2s for RAG responses

## Technology Stack

- **Backend**: Rust (tonic gRPC + prost + tokio)
- **Frontend**: React + TypeScript (Next.js/SPA), Ant Design or MUI
- **Database**: PostgreSQL + RLS, PgVector for vector indexing
- **Authentication**: Keycloak/Auth0 (SAML/OIDC, SCIM)
- **Monitoring**: OpenTelemetry + Prometheus + Grafana
- **AI**: EU-region LLM inference/API, open-source embedding models

## Core Data Models

### Key Entities
- **Tenant**: Multi-tenant isolation with data residency settings
- **User**: Role-based access (admin/customer/engineer/se/sales)
- **Ticket**: Full lifecycle with SLA tracking and priority/severity enums
- **KnowledgeArticle**: Versioned content with visibility controls
- **EmbeddingChunk**: Vector search with tenant isolation
- **ImportExportJob**: Batch data processing with progress tracking

### Important Constraints
- All tables include `tenant_id` for multi-tenant isolation
- Tickets use soft deletes (`is_deleted` flag)
- Audit logs are immutable with hash-based integrity
- All timestamp fields use Unix time format

## Development Phases

1. **Phase 0**: Infrastructure & multi-tenant foundation (4-6 weeks)
2. **Phase 1**: Core ticketing & SLA functionality (6-8 weeks)
3. **Phase 2**: Knowledge base & basic RAG (8-10 weeks)
4. **Phase 3**: Smart routing & integrations (6-8 weeks)
5. **Phase 4**: Data management & production optimization (6-8 weeks)

## Security & Compliance

### Multi-Layer Security
- Field-level and record-level access control
- Encryption at rest and in transit
- Key management with KMS/Vault integration
- Immutable audit logs with hash chaining

### GDPR Features
- Automated DSR (Data Subject Request) handling
- Data residency controls (EU-first)
- PII detection and masking
- Right to be forgotten implementation

### Import/Export Security
- Role-based permissions for data access
- PII detection and anonymization
- Malware scanning for uploaded files
- Comprehensive audit trails

## RAG/LLM Integration

### Quality Metrics
- Retrieval accuracy (precision@k, recall@k)
- Citation accuracy and source relevance
- Business impact (deflection rate, user satisfaction)
- Hallucination detection mechanisms

### AI Provider Management
- Multi-provider support (OpenAI, Azure OpenAI, DeepSeek, local)
- Encrypted credential storage with key rotation
- Task-to-model mapping with cost optimization
- Rate limiting and quota management

## Performance Optimization

### Caching Strategy
- **L1**: Application memory cache for hot data
- **L2**: Redis distributed cache for query results
- **L3**: Database query cache for complex analytics

### Database Optimization
- Partitioning by tenant and time ranges
- Strategic indexing for high-frequency queries
- Connection pooling configuration
- Materialized views for analytics

## gRPC API Design

### Common Patterns
- All services require `x-tenant-id`, `x-user-id`, `x-roles` metadata
- Unified error model with localized messages
- Pagination with `page_size` and `page_token`
- Request ID and idempotency key support

### Key Services
- **TicketService**: Full ticket lifecycle management
- **KnowledgeService**: Article CRUD, search, and publishing
- **RAGService**: Document ingestion, search, and AI generation
- **DataManagementService**: Import/export operations with progress tracking

## Data Import/Export

### Supported Formats
- **Tickets**: CSV, JSON, XML
- **Knowledge Articles**: CSV, JSON, Markdown
- **Users**: CSV, JSON
- **Contracts**: CSV, JSON

### Batch Operations
- Maximum 50,000 records per file
- 500MB file size limit
- Concurrent job limit of 3
- Comprehensive validation and error handling

## Port Configuration

Avoid common ports (3000, 8000, 8080, 9000, 9001, 5173, 4200, 7000, 5000). Use non-standard ports for development and document them in `.env.example` or `ports.json` for team coordination.

## Development Notes

### Multi-Tenant Development
- Always include `tenant_id` in queries and operations
- Test data isolation thoroughly
- Consider cross-tenant data leakage in all implementations

### SLA Implementation
- Implement calendar-aware timing (business hours only)
- Support different SLA policies by contract tier
- Provide upgrade paths for SLA breaches

### AI Integration
- Implement fallback mechanisms for AI failures
- Rate limit AI calls to control costs
- Provide human review workflows for AI-generated content

### Audit Requirements
- Log all data mutations with user context
- Implement tamper-evident audit trails
- Support audit log export for compliance

## Current Project State

This project is currently in the design phase with comprehensive technical specifications documented in `DESIGN.md`. The architecture emphasizes simplified microservices, strong security, and GDPR compliance while maintaining high performance for a European B2B customer base.