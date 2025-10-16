-- Fix ticket schema to match core models
-- This migration fixes field names and enum values to match the Rust core models

-- Drop existing indexes that reference old field names
DROP INDEX IF EXISTS idx_tickets_contact_id;
DROP INDEX IF EXISTS idx_tickets_assigned_to_id;
DROP INDEX IF EXISTS idx_tickets_created_by_id;
DROP INDEX IF EXISTS idx_tickets_tenant_assignee_status;
DROP INDEX IF EXISTS idx_tickets_tenant_contact_status;

-- Rename columns to match core models
ALTER TABLE tickets RENAME COLUMN contact_id TO customer_id;
ALTER TABLE tickets RENAME COLUMN assigned_to_id TO assigned_agent_id;
ALTER TABLE tickets RENAME COLUMN created_by_id TO created_by;
ALTER TABLE tickets RENAME COLUMN due_at TO due_date;

-- Add missing columns
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS ticket_type ticket_type DEFAULT 'incident';
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES users(id);
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS updated_by VARCHAR(255) NOT NULL DEFAULT 'system';

-- Update default values to match core models
ALTER TABLE tickets ALTER COLUMN status SET DEFAULT 'open';
ALTER TABLE tickets ALTER COLUMN priority SET DEFAULT 'normal';

-- Create new indexes with correct field names
CREATE INDEX idx_tickets_customer_id ON tickets(customer_id);
CREATE INDEX idx_tickets_assigned_agent_id ON tickets(assigned_agent_id);
CREATE INDEX idx_tickets_team_id ON tickets(team_id);
CREATE INDEX idx_tickets_tenant_assignee_status ON tickets(tenant_id, assigned_agent_id, status) WHERE assigned_agent_id IS NOT NULL;
CREATE INDEX idx_tickets_tenant_contact_status ON tickets(tenant_id, customer_id, status);

-- Update RLS policies to use correct field names
DROP POLICY IF EXISTS customer_own_tickets ON tickets;
CREATE POLICY customer_own_tickets ON tickets
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        AND customer_id = current_setting('app.current_user_id', true)::uuid
    );

-- Add default category for testing
INSERT INTO ticket_categories (id, tenant_id, name, description, color, created_at, updated_at)
SELECT
    uuid_generate_v4(),
    id,
    'General',
    'General ticket category',
    '#007bff',
    NOW(),
    NOW()
FROM tenants
WHERE domain = 'test.smartticket.com'
ON CONFLICT DO NOTHING;