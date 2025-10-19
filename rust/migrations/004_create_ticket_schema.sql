-- Create ticket enums (matching core models)
CREATE TYPE ticket_status AS ENUM (
    'unspecified',
    'open',
    'in_progress',
    'pending_customer',
    'pending_third_party',
    'resolved',
    'closed',
    'reopened'
);

CREATE TYPE ticket_priority AS ENUM (
    'unspecified',
    'low',
    'normal',
    'high',
    'urgent',
    'critical'
);

CREATE TYPE ticket_type AS ENUM (
    'unspecified',
    'incident',
    'service_request',
    'problem',
    'change',
    'question'
);
CREATE TYPE ticket_severity AS ENUM ('Low', 'Medium', 'High', 'Critical');

-- Create ticket categories
CREATE TABLE ticket_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES ticket_categories(id),
    color VARCHAR(7), -- hex color code
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for categories
CREATE INDEX idx_ticket_categories_tenant_id ON ticket_categories(tenant_id);
CREATE INDEX idx_ticket_categories_parent_id ON ticket_categories(parent_id);

-- Enable RLS for categories
ALTER TABLE ticket_categories ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_categories ON ticket_categories
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Create tickets table
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ticket_number VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    status ticket_status NOT NULL DEFAULT 'open',
    priority ticket_priority NOT NULL DEFAULT 'normal',
    ticket_type ticket_type NOT NULL DEFAULT 'incident',
    severity ticket_severity NOT NULL DEFAULT 'low',
    category_id UUID REFERENCES ticket_categories(id),
    customer_id UUID NOT NULL REFERENCES users(id),
    assigned_agent_id UUID REFERENCES users(id),
    team_id UUID REFERENCES users(id),
    created_by VARCHAR(255) NOT NULL,
    updated_by VARCHAR(255) NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    closed_at TIMESTAMP WITH TIME ZONE,
    due_at TIMESTAMP WITH TIME ZONE,
    resolution TEXT,
    tags TEXT[] DEFAULT '{}',
    custom_fields JSONB DEFAULT '{}',
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for tickets
CREATE INDEX idx_tickets_tenant_id ON tickets(tenant_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_severity ON tickets(severity);
CREATE INDEX idx_tickets_category_id ON tickets(category_id);
CREATE INDEX idx_tickets_customer_id ON tickets(customer_id);
CREATE INDEX idx_tickets_assigned_agent_id ON tickets(assigned_agent_id);
CREATE INDEX idx_tickets_team_id ON tickets(team_id);
CREATE INDEX idx_tickets_created_at ON tickets(created_at);
CREATE INDEX idx_tickets_due_at ON tickets(due_at);
CREATE INDEX idx_tickets_ticket_number ON tickets(ticket_number);
CREATE UNIQUE INDEX idx_tickets_tenant_number ON tickets(tenant_id, ticket_number);
CREATE INDEX idx_tickets_tags ON tickets USING GIN(tags);
CREATE INDEX idx_tickets_deleted ON tickets(is_deleted);

-- Composite indexes for common queries
CREATE INDEX idx_tickets_tenant_status ON tickets(tenant_id, status);
CREATE INDEX idx_tickets_tenant_assignee_status ON tickets(tenant_id, assigned_agent_id, status) WHERE assigned_agent_id IS NOT NULL;
CREATE INDEX idx_tickets_tenant_contact_status ON tickets(tenant_id, customer_id, status);

-- Full-text search index
CREATE INDEX idx_tickets_search ON tickets USING GIN(
    to_tsvector('english', title || ' ' || COALESCE(description, '') || ' ' || COALESCE(resolution, ''))
);

-- Create trigger for updated_at
CREATE TRIGGER update_tickets_updated_at
    BEFORE UPDATE ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Enable RLS
ALTER TABLE tickets ENABLE ROW LEVEL SECURITY;

-- RLS policies for tickets
CREATE POLICY tenant_isolation_tickets ON tickets
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid AND NOT is_deleted);

-- Super admins can see all tickets across tenants
CREATE POLICY superadmin_all_tickets ON tickets
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.current_user_id', true)::uuid
            AND role = 'system_admin'
        )
    );

-- Support engineers and tenant admins can see all tickets in their tenant
CREATE POLICY support_all_tenant_tickets ON tickets
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        AND EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.current_user_id', true)::uuid
            AND tenant_id = tickets.tenant_id
            AND role IN ('admin', 'team_lead')
        )
    );

-- Customers can only see their own tickets
CREATE POLICY customer_own_tickets ON tickets
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        AND customer_id = current_setting('app.current_user_id', true)::uuid
    );

-- Function to generate ticket numbers
CREATE OR REPLACE FUNCTION generate_ticket_number()
RETURNS TRIGGER AS $$
DECLARE
    tenant_code TEXT;
    sequence_num BIGINT;
BEGIN
    -- Generate tenant code from tenant name (first 3 letters, uppercase)
    SELECT UPPER(SUBSTRING(REPLACE(name, ' ', ''), 1, 3))
    INTO tenant_code
    FROM tenants
    WHERE id = NEW.tenant_id;

    -- Get next sequence number for this tenant
    SELECT COALESCE(MAX(CAST(SUBSTRING(ticket_number, '\d+$') AS BIGINT)), 0) + 1
    INTO sequence_num
    FROM tickets
    WHERE tenant_id = NEW.tenant_id;

    -- Generate ticket number: TTT-YYYY-NNNNN
    NEW.ticket_number := tenant_code || '-' || EXTRACT(YEAR FROM NOW())::TEXT || '-' || LPAD(sequence_num::TEXT, 5, '0');

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to generate ticket number
CREATE TRIGGER generate_ticket_number_trigger
    BEFORE INSERT ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION generate_ticket_number();