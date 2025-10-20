-- Migration: 001_create_initial_tables
-- Description: Create initial database tables for SmartTicket system
-- Created: 2025-01-15

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;

-- Create system_settings table
CREATE TABLE IF NOT EXISTS system_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    key TEXT(255) NOT NULL UNIQUE,
    value TEXT,
    type TEXT(50) DEFAULT 'string',
    description TEXT,
    is_public BOOLEAN DEFAULT 0
);

-- Create unique index for system_settings key
CREATE UNIQUE INDEX IF NOT EXISTS idx_system_settings_key ON system_settings(key);

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    name TEXT(255) NOT NULL,
    slug TEXT(100) NOT NULL UNIQUE,
    domain TEXT(255),
    settings TEXT,
    plan TEXT(50) DEFAULT 'basic',
    max_users INTEGER DEFAULT 100,
    is_active BOOLEAN DEFAULT 1,
    expired_at DATETIME
);

-- Create unique index for tenants slug
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);

-- Create index for tenants active status
CREATE INDEX IF NOT EXISTS idx_tenants_is_active ON tenants(is_active);

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    email TEXT(255) NOT NULL,
    username TEXT(100) NOT NULL,
    password_hash TEXT(255) NOT NULL,
    first_name TEXT(100),
    last_name TEXT(100),
    role TEXT(50) DEFAULT 'customer',
    is_active BOOLEAN DEFAULT 1,
    last_login_at DATETIME,
    preferences TEXT,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(tenant_id, email),
    UNIQUE(tenant_id, username)
);

-- Create index for users tenant_id
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);

-- Create index for users role
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Create index for users active status
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Create tickets table
CREATE TABLE IF NOT EXISTS tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    ticket_number TEXT(50) NOT NULL,
    title TEXT(255) NOT NULL,
    description TEXT,
    status TEXT(50) DEFAULT 'open',
    priority TEXT(20) DEFAULT 'medium',
    severity TEXT(20) DEFAULT 'minor',
    category TEXT(100),
    type TEXT(50),
    assigned_to INTEGER,
    requester_name TEXT(255),
    requester_email TEXT(255),
    tags TEXT,
    custom_fields TEXT,
    is_deleted BOOLEAN DEFAULT 0,
    resolution_time DATETIME,
    resolved_at DATETIME,
    due_date DATETIME,
    sla_status TEXT(20) DEFAULT 'within',
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(tenant_id, ticket_number)
);

-- Create index for tickets tenant_id
CREATE INDEX IF NOT EXISTS idx_tickets_tenant_id ON tickets(tenant_id);

-- Create index for tickets status
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);

-- Create index for tickets priority
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);

-- Create index for tickets assigned_to
CREATE INDEX IF NOT EXISTS idx_tickets_assigned_to ON tickets(assigned_to);

-- Create index for tickets created_by
CREATE INDEX IF NOT EXISTS idx_tickets_created_by ON tickets(created_by);

-- Create index for tickets deleted status
CREATE INDEX IF NOT EXISTS idx_tickets_is_deleted ON tickets(is_deleted);

-- Create index for tickets due_date
CREATE INDEX IF NOT EXISTS idx_tickets_due_date ON tickets(due_date);

-- Create messages table
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    ticket_id INTEGER NOT NULL,
    user_id INTEGER,
    content TEXT NOT NULL,
    content_type TEXT(50) DEFAULT 'text',
    is_internal BOOLEAN DEFAULT 0,
    is_from_ai BOOLEAN DEFAULT 0,
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create index for messages ticket_id
CREATE INDEX IF NOT EXISTS idx_messages_ticket_id ON messages(ticket_id);

-- Create index for messages user_id
CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);

-- Create index for messages created_at
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- Create attachments table
CREATE TABLE IF NOT EXISTS attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    ticket_id INTEGER NOT NULL,
    message_id INTEGER,
    knowledge_article_id INTEGER,
    file_name TEXT(255) NOT NULL,
    original_name TEXT(255) NOT NULL,
    file_path TEXT(500) NOT NULL,
    file_size INTEGER NOT NULL,
    content_type TEXT(100),
    hash TEXT(64),
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE SET NULL,
    FOREIGN KEY (knowledge_article_id) REFERENCES knowledge_articles(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create index for attachments ticket_id
CREATE INDEX IF NOT EXISTS idx_attachments_ticket_id ON attachments(ticket_id);

-- Create index for attachments message_id
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

-- Create index for attachments knowledge_article_id
CREATE INDEX IF NOT EXISTS idx_attachments_knowledge_article_id ON attachments(knowledge_article_id);

-- Create index for attachments hash
CREATE INDEX IF NOT EXISTS idx_attachments_hash ON attachments(hash);

-- Create knowledge_articles table
CREATE TABLE IF NOT EXISTS knowledge_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    title TEXT(255) NOT NULL,
    slug TEXT(255) NOT NULL,
    content TEXT,
    content_type TEXT(50) DEFAULT 'markdown',
    summary TEXT,
    author_id INTEGER,
    status TEXT(50) DEFAULT 'draft',
    visibility TEXT(50) DEFAULT 'public',
    access_level TEXT(50) DEFAULT 'all',
    category TEXT(100),
    tags TEXT,
    views INTEGER DEFAULT 0,
    helpful_votes INTEGER DEFAULT 0,
    version INTEGER DEFAULT 1,
    parent_id INTEGER,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_id) REFERENCES knowledge_articles(id) ON DELETE SET NULL
);

-- Create index for knowledge_articles tenant_id
CREATE INDEX IF NOT EXISTS idx_knowledge_articles_tenant_id ON knowledge_articles(tenant_id);

-- Create index for knowledge_articles slug
CREATE INDEX IF NOT EXISTS idx_knowledge_articles_slug ON knowledge_articles(slug);

-- Create index for knowledge_articles status
CREATE INDEX IF NOT EXISTS idx_knowledge_articles_status ON knowledge_articles(status);

-- Create index for knowledge_articles visibility
CREATE INDEX IF NOT EXISTS idx_knowledge_articles_visibility ON knowledge_articles(visibility);

-- Create index for knowledge_articles parent_id
CREATE INDEX IF NOT EXISTS idx_knowledge_articles_parent_id ON knowledge_articles(parent_id);

-- Create llm_providers table
CREATE TABLE IF NOT EXISTS llm_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    name TEXT(255) NOT NULL,
    provider_type TEXT(50) NOT NULL,
    api_endpoint TEXT(500),
    api_key TEXT(500),
    model TEXT(100),
    max_tokens INTEGER DEFAULT 4096,
    temperature REAL DEFAULT 0.7,
    task_types TEXT,
    is_default BOOLEAN DEFAULT 0,
    is_enabled BOOLEAN DEFAULT 1,
    quota_limit INTEGER DEFAULT 1000,
    quota_used INTEGER DEFAULT 0,
    configuration TEXT,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create index for llm_providers tenant_id
CREATE INDEX IF NOT EXISTS idx_llm_providers_tenant_id ON llm_providers(tenant_id);

-- Create index for llm_providers is_enabled
CREATE INDEX IF NOT EXISTS idx_llm_providers_is_enabled ON llm_providers(is_enabled);

-- Create import_export_jobs table
CREATE TABLE IF NOT EXISTS import_export_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    type TEXT(20) NOT NULL,
    status TEXT(50) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    total_records INTEGER DEFAULT 0,
    processed_records INTEGER DEFAULT 0,
    failed_records INTEGER DEFAULT 0,
    source_format TEXT(50),
    target_format TEXT(50),
    file_path TEXT(500),
    configuration TEXT,
    error TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    started_by INTEGER,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (started_by) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create index for import_export_jobs tenant_id
CREATE INDEX IF NOT EXISTS idx_import_export_jobs_tenant_id ON import_export_jobs(tenant_id);

-- Create index for import_export_jobs status
CREATE INDEX IF NOT EXISTS idx_import_export_jobs_status ON import_export_jobs(status);

-- Create index for import_export_jobs type
CREATE INDEX IF NOT EXISTS idx_import_export_jobs_type ON import_export_jobs(type);

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    user_id INTEGER,
    action TEXT(100) NOT NULL,
    resource_type TEXT(100) NOT NULL,
    resource_id INTEGER,
    resource_name TEXT(255),
    ip_address TEXT(45),
    user_agent TEXT(500),
    changes TEXT,
    old_values TEXT,
    new_values TEXT,
    request_id TEXT(100),
    hash TEXT(64),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create index for audit_logs tenant_id
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs(tenant_id);

-- Create index for audit_logs user_id
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);

-- Create index for audit_logs action
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

-- Create index for audit_logs resource_type
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);

-- Create index for audit_logs created_at
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Create index for audit_logs hash
CREATE INDEX IF NOT EXISTS idx_audit_logs_hash ON audit_logs(hash);

-- Create api_keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT(255),
    updated_by TEXT(255),
    tenant_id INTEGER NOT NULL,
    name TEXT(255) NOT NULL,
    key_hash TEXT(255) NOT NULL UNIQUE,
    key_prefix TEXT(20) NOT NULL,
    permissions TEXT,
    is_active BOOLEAN DEFAULT 1,
    expires_at DATETIME,
    last_used_at DATETIME,
    usage_count INTEGER DEFAULT 0,
    creator_id INTEGER,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create index for api_keys tenant_id
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);

-- Create index for api_keys key_hash
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);

-- Create index for api_keys is_active
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);

-- Insert default system settings
INSERT OR IGNORE INTO system_settings (key, value, type, description, is_public) VALUES
('app_name', 'SmartTicket', 'string', 'Application name', true),
('app_version', '1.0.0', 'string', 'Application version', true),
('max_file_size', '104857600', 'int', 'Maximum file size in bytes (100MB)', false),
('allowed_file_types', '["jpg","jpeg","png","gif","pdf","doc","docx","xls","xlsx","txt","csv"]', 'json', 'Allowed file types for uploads', false),
('session_timeout', '86400', 'int', 'Session timeout in seconds (24 hours)', false),
('default_language', 'en', 'string', 'Default application language', true),
('maintenance_mode', 'false', 'bool', 'Enable maintenance mode', true),
('registration_enabled', 'true', 'bool', 'Enable user registration', true);