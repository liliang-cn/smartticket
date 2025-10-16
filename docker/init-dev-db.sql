-- SmartTicket Development Database Initialization
-- This script sets up the database with necessary extensions and basic schema

-- Create necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create custom types
DO $$ BEGIN
    CREATE TYPE ticket_status AS ENUM (
        'open', 'in_progress', 'pending_customer', 'resolved', 'closed', 'reopened'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE ticket_priority AS ENUM (
        'low', 'normal', 'high', 'urgent', 'critical'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE ticket_type AS ENUM (
        'incident', 'service_request', 'bug_report', 'feature_request', 'question'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM (
        'customer', 'agent', 'team_lead', 'admin', 'system_admin'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE subscription_tier AS ENUM (
        'trial', 'standard', 'premium', 'enterprise'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE knowledge_status AS ENUM (
        'draft', 'review', 'published', 'archived'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE knowledge_visibility AS ENUM (
        'public', 'internal', 'restricted'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE comment_type AS ENUM (
        'comment', 'status_change', 'assignment', 'internal_note'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create indexes for common queries
-- These will be created after tables are created by migrations

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

-- Print initialization complete message
\echo 'SmartTicket development database initialized successfully!'