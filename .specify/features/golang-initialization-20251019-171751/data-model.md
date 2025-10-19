# Data Model Design

## Overview

The SmartTicket data model follows a multi-tenant architecture with clear separation of concerns. All entities include tenant isolation, audit trails, and support for enterprise features.

## Core Entities

### 1. Tenant

**Purpose**: Multi-tenant isolation and configuration management.

```go
type Tenant struct {
    ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    Name        string    `gorm:"not null;type:varchar(200)" json:"name"`
    Domain      string    `gorm:"uniqueIndex;type:varchar(200)" json:"domain"`
    Plan        string    `gorm:"not null;type:varchar(50);default:'basic'" json:"plan"`
    Status      string    `gorm:"not null;type:varchar(20);default:'active'" json:"status"`
    Settings    string    `gorm:"type:text" json:"settings"` // JSON configuration
    CreatedAt   time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
    DeletedAt   time.Time `gorm:"index" json:"deleted_at,omitempty"`

    // Relationships
    Users       []User    `gorm:"foreignKey:TenantID" json:"users,omitempty"`
    Tickets     []Ticket  `gorm:"foreignKey:TenantID" json:"tickets,omitempty"`
    KnowledgeArticles []KnowledgeArticle `gorm:"foreignKey:TenantID" json:"knowledge_articles,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `Name`: Required, max 200 characters
- `Domain`: Required, unique, valid domain format
- `Plan`: Must be one of: `basic`, `professional`, `enterprise`
- `Status`: Must be one of: `active`, `inactive`, `suspended`

### 2. User

**Purpose**: User management with role-based access control.

```go
type User struct {
    ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TenantID    string    `gorm:"not null;index;type:varchar(50)" json:"tenant_id"`
    Email       string    `gorm:"not null;uniqueIndex:idx_tenant_email;type:varchar(255)" json:"email"`
    Name        string    `gorm:"not null;type:varchar(200)" json:"name"`
    Role        string    `gorm:"not null;type:varchar(50);default:'customer'" json:"role"`
    Status      string    `gorm:"not null;type:varchar(20);default:'active'" json:"status"`
    Avatar      string    `gorm:"type:varchar(500)" json:"avatar,omitempty"`
    Preferences string    `gorm:"type:text" json:"preferences"` // JSON preferences
    LastLoginAt time.Time `json:"last_login_at,omitempty"`
    CreatedAt   time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
    DeletedAt   time.Time `gorm:"index" json:"deleted_at,omitempty"`

    // Relationships
    Tenant      *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Tickets     []Ticket  `gorm:"foreignKey:AssignedToID" json:"assigned_tickets,omitempty"`
    CreatedTickets []Ticket `gorm:"foreignKey:CreatedBy" json:"created_tickets,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TenantID`: Required, must reference valid tenant
- `Email`: Required, unique within tenant, valid email format
- `Name`: Required, max 200 characters
- `Role`: Must be one of: `admin`, `engineer`, `support`, `customer`, `sales`
- `Status`: Must be one of: `active`, `inactive`, `locked`

### 3. Ticket

**Purpose**: Core ticketing entity with full lifecycle management.

```go
type Ticket struct {
    ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TenantID    string    `gorm:"not null;index;type:varchar(50)" json:"tenant_id"`
    Number      string    `gorm:"not null;uniqueIndex:idx_tenant_number;type:varchar(20)" json:"number"`
    Title       string    `gorm:"not null;type:varchar(500)" json:"title"`
    Description string    `gorm:"type:text" json:"description"`
    Status      string    `gorm:"not null;type:varchar(50);default:'open'" json:"status"`
    Priority    string    `gorm:"not null;type:varchar(20);default:'medium'" json:"priority"`
    Severity    string    `gorm:"not null;type:varchar(20);default:'low'" json:"severity"`
    Category    string    `gorm:"type:varchar(100)" json:"category,omitempty"`
    Tags        string    `gorm:"type:text" json:"tags"` // JSON array
    CreatedBy   string    `gorm:"not null;type:varchar(50)" json:"created_by"`
    AssignedTo  string    `gorm:"type:varchar(50)" json:"assigned_to,omitempty"`
    DueDate     time.Time `json:"due_date,omitempty"`
    ResolvedAt  time.Time `json:"resolved_at,omitempty"`
    ClosedAt    time.Time `json:"closed_at,omitempty"`
    CreatedAt   time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
    DeletedAt   time.Time `gorm:"index" json:"deleted_at,omitempty"`

    // Relationships
    Tenant      *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Creator     *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
    Assignee    *User     `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
    Messages    []Message `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
    Attachments []Attachment `gorm:"foreignKey:TicketID" json:"attachments,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TenantID`: Required, must reference valid tenant
- `Number`: Required, unique within tenant, auto-generated format (TICKET-####)
- `Title`: Required, max 500 characters
- `Status`: Must be one of: `open`, `in_progress`, `pending_customer`, `resolved`, `closed`
- `Priority`: Must be one of: `low`, `medium`, `high`, `critical`
- `Severity`: Must be one of: `low`, `medium`, `high`, `critical`
- `CreatedBy`: Required, must reference valid user

### 4. Message

**Purpose**: Communication within tickets.

```go
type Message struct {
    ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TicketID    string    `gorm:"not null;index;type:varchar(50)" json:"ticket_id"`
    Content     string    `gorm:"not null;type:text" json:"content"`
    Type        string    `gorm:"not null;type:varchar(20);default:'internal'" json:"type"`
    IsInternal  bool      `gorm:"not null;default:false" json:"is_internal"`
    AuthorID    string    `gorm:"not null;type:varchar(50)" json:"author_id"`
    AuthorType  string    `gorm:"not null;type:varchar(20)" json:"author_type"`
    CreatedAt   time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`

    // Relationships
    Ticket      *Ticket   `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
    Author      *User     `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
    Attachments []Attachment `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TicketID`: Required, must reference valid ticket
- `Content`: Required, max 100,000 characters
- `Type`: Must be one of: `internal`, `external`, `system`
- `AuthorType`: Must be one of: `user`, `customer`, `system`

### 5. Knowledge Article

**Purpose**: Knowledge base for documentation and solutions.

```go
type KnowledgeArticle struct {
    ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TenantID    string    `gorm:"not null;index;type:varchar(50)" json:"tenant_id"`
    Title       string    `gorm:"not null;type:varchar(500)" json:"title"`
    Content     string    `gorm:"type:text" json:"content"`
    Summary     string    `gorm:"type:text" json:"summary"`
    Category    string    `gorm:"type:varchar(100)" json:"category,omitempty"`
    Tags        string    `gorm:"type:text" json:"tags"` // JSON array
    Status      string    `gorm:"not null;type:varchar(20);default:'draft'" json:"status"`
    Visibility  string    `gorm:"not null;type:varchar(20);default:'private'" json:"visibility"`
    AuthorID    string    `gorm:"not null;type:varchar(50)" json:"author_id"`
    Version     int       `gorm:"not null;default:1" json:"version"`
    Views       int       `gorm:"not null;default:0" json:"views"`
    Helpful     int       `gorm:"not null;default:0" json:"helpful"`
    NotHelpful  int       `gorm:"not null;default:0" json:"not_helpful"`
    CreatedAt   time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
    DeletedAt   time.Time `gorm:"index" json:"deleted_at,omitempty"`

    // Relationships
    Tenant      *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Author      *User     `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
    Attachments []Attachment `gorm:"foreignKey:KnowledgeArticleID" json:"attachments,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TenantID`: Required, must reference valid tenant
- `Title`: Required, max 500 characters
- `Content`: Required for published articles
- `Status`: Must be one of: `draft`, `review`, `published`, `archived`
- `Visibility`: Must be one of: `private`, `internal`, `public`
- `AuthorID`: Required, must reference valid user

### 6. LLM Provider

**Purpose**: Configuration for AI/LLM service integration.

```go
type LLMProvider struct {
    ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TenantID        string    `gorm:"not null;index;type:varchar(50)" json:"tenant_id"`
    Name            string    `gorm:"not null;type:varchar(100)" json:"name"`
    ProviderType    string    `gorm:"not null;type:varchar(50)" json:"provider_type"`
    APIEndpoint     string    `gorm:"type:varchar(500)" json:"api_endpoint,omitempty"`
    APIKey          string    `gorm:"type:text" json:"api_key,omitempty"` // Encrypted
    Model           string    `gorm:"type:varchar(100)" json:"model,omitempty"`
    MaxTokens       int       `gorm:"default:4096" json:"max_tokens"`
    Temperature     float64   `gorm:"default:0.7" json:"temperature"`
    TaskTypes       string    `gorm:"type:text" json:"task_types"` // JSON array
    IsDefault       bool      `gorm:"not null;default:false" json:"is_default"`
    IsEnabled       bool      `gorm:"not null;default:true" json:"is_enabled"`
    QuotaLimit      int       `json:"quota_limit,omitempty"`
    QuotaUsed       int       `gorm:"default:0" json:"quota_used"`
    CreatedAt       time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt       time.Time `gorm:"not null" json:"updated_at"`
    DeletedAt       time.Time `gorm:"index" json:"deleted_at,omitempty"`

    // Relationships
    Tenant          *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TenantID`: Required, must reference valid tenant
- `Name`: Required, max 100 characters
- `ProviderType`: Must be one of: `openai`, `azure`, `anthropic`, `deepseek`, `ollama`, `local`
- `APIKey`: Required for external providers, encrypted at rest
- `TaskTypes`: Must be subset of: `chat`, `embedding`, `rerank`, `summarization`, `generation`, `classification`

### 7. Import Export Job

**Purpose**: Batch data processing for import/export operations.

```go
type ImportExportJob struct {
    ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TenantID        string    `gorm:"not null;index;type:varchar(50)" json:"tenant_id"`
    Type            string    `gorm:"not null;type:varchar(20)" json:"type"` // import or export
    EntityType      string    `gorm:"not null;type:varchar(50)" json:"entity_type"`
    Status          string    `gorm:"not null;type:varchar(20);default:'pending'" json:"status"`
    Format          string    `gorm:"not null;type:varchar(20)" json:"format"` // csv, json, xml, etc.
    FilePath        string    `gorm:"type:varchar(500)" json:"file_path,omitempty"`
    Progress        float64   `gorm:"default:0" json:"progress"`
    TotalRecords    int       `json:"total_records,omitempty"`
    ProcessedRecords int      `json:"processed_records,omitempty"`
    ErrorRecords    int       `json:"error_records,omitempty"`
    ErrorMessage    string    `gorm:"type:text" json:"error_message,omitempty"`
    Settings        string    `gorm:"type:text" json:"settings"` // JSON configuration
    StartedAt       time.Time `json:"started_at,omitempty"`
    CompletedAt     time.Time `json:"completed_at,omitempty"`
    CreatedBy       string    `gorm:"not null;type:varchar(50)" json:"created_by"`
    CreatedAt       time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt       time.Time `gorm:"not null" json:"updated_at"`

    // Relationships
    Tenant          *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Creator         *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TenantID`: Required, must reference valid tenant
- `Type`: Must be one of: `import`, `export`
- `EntityType`: Must be one of: `tickets`, `users`, `knowledge_articles`, `contracts`
- `Status`: Must be one of: `pending`, `running`, `completed`, `failed`, `cancelled`
- `Format`: Must be one of: `csv`, `json`, `xml`, `markdown`, `sqlite`
- `Progress`: Range 0-100, updated during processing

### 8. Attachment

**Purpose**: File attachments for tickets, messages, and knowledge articles.

```go
type Attachment struct {
    ID                string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
    TenantID          string    `gorm:"not null;index;type:varchar(50)" json:"tenant_id"`
    OriginalName      string    `gorm:"not null;type:varchar(500)" json:"original_name"`
    FileName          string    `gorm:"not null;type:varchar(500)" json:"file_name"`
    FilePath          string    `gorm:"not null;type:varchar(1000)" json:"file_path"`
    FileSize          int64     `gorm:"not null" json:"file_size"`
    MimeType          string    `gorm:"not null;type:varchar(200)" json:"mime_type"`
    Hash              string    `gorm:"not null;type:varchar(64)" json:"hash"` // SHA-256
    TicketID          string    `gorm:"type:varchar(50)" json:"ticket_id,omitempty"`
    MessageID         string    `gorm:"type:varchar(50)" json:"message_id,omitempty"`
    KnowledgeArticleID string   `gorm:"type:varchar(50)" json:"knowledge_article_id,omitempty"`
    UploadedBy        string    `gorm:"not null;type:varchar(50)" json:"uploaded_by"`
    CreatedAt         time.Time `gorm:"not null" json:"created_at"`

    // Relationships
    Tenant            *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Ticket            *Ticket   `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
    Message           *Message  `gorm:"foreignKey:MessageID" json:"message,omitempty"`
    KnowledgeArticle  *KnowledgeArticle `gorm:"foreignKey:KnowledgeArticleID" json:"knowledge_article,omitempty"`
    Uploader          *User     `gorm:"foreignKey:UploadedBy" json:"uploader,omitempty"`
}
```

**Validation Rules**:
- `ID`: Required, unique identifier (UUID format)
- `TenantID`: Required, must reference valid tenant
- `OriginalName`: Required, max 500 characters
- `FileName`: Required, max 500 characters (system-generated)
- `FilePath`: Required, valid file path
- `FileSize`: Required, must be within limits (configurable, default 100MB)
- `MimeType`: Required, must be allowed type (configurable whitelist)
- `Hash`: Required, SHA-256 hash for integrity verification

## Database Indexes

### Performance Indexes

```sql
-- Ticket performance indexes
CREATE INDEX idx_tickets_tenant_status ON tickets(tenant_id, status);
CREATE INDEX idx_tickets_tenant_assigned ON tickets(tenant_id, assigned_to);
CREATE INDEX idx_tickets_created_at ON tickets(created_at);
CREATE INDEX idx_tickets_due_date ON tickets(due_date) WHERE due_date IS NOT NULL;

-- User performance indexes
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX idx_users_tenant_role ON users(tenant_id, role);

-- Message performance indexes
CREATE INDEX idx_messages_ticket_created ON messages(ticket_id, created_at);
CREATE INDEX idx_messages_author ON messages(author_id, created_at);

-- Knowledge article performance indexes
CREATE INDEX idx_knowledge_tenant_status ON knowledge_articles(tenant_id, status);
CREATE INDEX idx_knowledge_fulltext ON knowledge_articles USING fts5(title, content, summary);

-- Attachment performance indexes
CREATE INDEX idx_attachments_tenant ON attachments(tenant_id);
CREATE INDEX idx_attachments_entity ON attachments(ticket_id, message_id, knowledge_article_id);

-- Import export job performance indexes
CREATE INDEX idx_jobs_tenant_status ON import_export_jobs(tenant_id, status);
CREATE INDEX idx_jobs_created_at ON import_export_jobs(created_at);
```

## Data Integrity Constraints

### Foreign Key Constraints

```sql
-- Tenant foreign keys
ALTER TABLE users ADD CONSTRAINT fk_users_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE knowledge_articles ADD CONSTRAINT fk_knowledge_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- User foreign keys
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_creator
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_assignee
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE messages ADD CONSTRAINT fk_messages_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE knowledge_articles ADD CONSTRAINT fk_knowledge_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL;

-- Entity relationship foreign keys
ALTER TABLE messages ADD CONSTRAINT fk_messages_ticket
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE;
ALTER TABLE attachments ADD CONSTRAINT fk_attachments_ticket
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE;
ALTER TABLE attachments ADD CONSTRAINT fk_attachments_message
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE;
ALTER TABLE attachments ADD CONSTRAINT fk_attachments_knowledge
    FOREIGN KEY (knowledge_article_id) REFERENCES knowledge_articles(id) ON DELETE CASCADE;
```

### Check Constraints

```sql
-- Status check constraints
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_status
    CHECK (status IN ('active', 'inactive', 'suspended'));
ALTER TABLE users ADD CONSTRAINT chk_user_status
    CHECK (status IN ('active', 'inactive', 'locked'));
ALTER TABLE tickets ADD CONSTRAINT chk_ticket_status
    CHECK (status IN ('open', 'in_progress', 'pending_customer', 'resolved', 'closed'));
ALTER TABLE tickets ADD CONSTRAINT chk_ticket_priority
    CHECK (priority IN ('low', 'medium', 'high', 'critical'));
ALTER TABLE tickets ADD CONSTRAINT chk_ticket_severity
    CHECK (severity IN ('low', 'medium', 'high', 'critical'));

-- Value range constraints
ALTER TABLE import_export_jobs ADD CONSTRAINT chk_job_progress
    CHECK (progress >= 0 AND progress <= 100);
ALTER TABLE knowledge_articles ADD CONSTRAINT chk_knowledge_version
    CHECK (version >= 1);
ALTER TABLE llm_providers ADD CONSTRAINT chk_llm_temperature
    CHECK (temperature >= 0.0 AND temperature <= 2.0);
```

## Data Migration Strategy

### Initial Schema Migration

```sql
-- Migration 001: Create initial schema
-- Creates all tables, indexes, and constraints defined above

-- Migration 002: Add audit tables
CREATE TABLE audit_logs (
    id VARCHAR(50) PRIMARY KEY,
    tenant_id VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(50) NOT NULL,
    action VARCHAR(20) NOT NULL,
    old_values TEXT,
    new_values TEXT,
    user_id VARCHAR(50),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_audit_tenant_entity ON audit_logs(tenant_id, entity_type, entity_id);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at);

-- Migration 003: Add configuration tables
CREATE TABLE system_configs (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT,
    description TEXT,
    is_encrypted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Data Validation Rules

### Input Validation Patterns

1. **UUID Validation**: All ID fields must be valid UUID v4 format
2. **Email Validation**: RFC 5322 compliant email validation
3. **URL Validation**: URL-safe path validation for file paths
4. **JSON Validation**: All JSON fields must be valid JSON structures
5. **File Validation**: File size, type, and hash validation for attachments

### Business Logic Validation

1. **Ticket Number Generation**: Auto-incrementing ticket numbers per tenant
2. **Permission Validation**: Role-based access control for all operations
3. **Tenant Isolation**: All queries must include tenant_id filtering
4. **Data Retention**: Configurable retention policies for audit logs and soft-deleted data
5. **Quota Enforcement**: Tenant-specific quotas for storage and API usage

## Performance Considerations

### Query Optimization

1. **Tenant Filtering**: All queries must include tenant_id for proper isolation
2. **Pagination**: Large datasets must use cursor-based pagination
3. **Connection Pooling**: Optimize connection pool settings for SQLite
4. **Query Caching**: Cache frequently accessed reference data
5. **Bulk Operations**: Use batch operations for large data imports/exports

### Storage Optimization

1. **File Storage**: Separate file storage from database storage
2. **Compression**: Compress large text fields (content, description)
3. **Archival**: Move old data to archival storage
4. **Backup Strategy**: Incremental backups with point-in-time recovery
5. **Cleanup**: Regular cleanup of temporary files and expired data

This data model provides a solid foundation for the SmartTicket application, ensuring proper multi-tenant isolation, data integrity, and enterprise-grade performance characteristics.