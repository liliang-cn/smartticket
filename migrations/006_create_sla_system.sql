-- Create SLA policies table
CREATE TABLE sla_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    priority ticket_priority NOT NULL,
    severity ticket_severity NOT NULL,
    response_time_minutes INTEGER NOT NULL,
    resolution_time_minutes INTEGER NOT NULL,
    business_hours_only BOOLEAN NOT NULL DEFAULT true,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for SLA policies
CREATE INDEX idx_sla_policies_tenant_id ON sla_policies(tenant_id);
CREATE INDEX idx_sla_policies_priority_severity ON sla_policies(priority, severity);
CREATE INDEX idx_sla_policies_active ON sla_policies(is_active);

-- Create trigger for updated_at
CREATE TRIGGER update_sla_policies_updated_at
    BEFORE UPDATE ON sla_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Enable RLS
ALTER TABLE sla_policies ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_sla_policies ON sla_policies
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Create SLA metrics tracking table
CREATE TABLE sla_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    sla_policy_id UUID NOT NULL REFERENCES sla_policies(id),
    response_due_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolution_due_at TIMESTAMP WITH TIME ZONE NOT NULL,
    first_response_at TIMESTAMP WITH TIME ZONE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    response_breached BOOLEAN NOT NULL DEFAULT false,
    resolution_breached BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for SLA metrics
CREATE INDEX idx_sla_metrics_tenant_id ON sla_metrics(tenant_id);
CREATE INDEX idx_sla_metrics_ticket_id ON sla_metrics(ticket_id);
CREATE INDEX idx_sla_metrics_sla_policy_id ON sla_metrics(sla_policy_id);
CREATE INDEX idx_sla_metrics_response_due ON sla_metrics(response_due_at);
CREATE INDEX idx_sla_metrics_resolution_due ON sla_metrics(resolution_due_at);
CREATE INDEX idx_sla_metrics_response_breached ON sla_metrics(response_breached);
CREATE INDEX idx_sla_metrics_resolution_breached ON sla_metrics(resolution_breached);

-- Create trigger for updated_at
CREATE TRIGGER update_sla_metrics_updated_at
    BEFORE UPDATE ON sla_metrics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Enable RLS
ALTER TABLE sla_metrics ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_sla_metrics ON sla_metrics
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Function to create SLA metrics when ticket is created
CREATE OR REPLACE FUNCTION create_sla_metrics_for_ticket()
RETURNS TRIGGER AS $$
DECLARE
    sla_policy_id UUID;
    response_due_at TIMESTAMP WITH TIME ZONE;
    resolution_due_at TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Find matching SLA policy
    SELECT id INTO sla_policy_id
    FROM sla_policies
    WHERE tenant_id = NEW.tenant_id
      AND priority = NEW.priority
      AND severity = NEW.severity
      AND is_active = true
    LIMIT 1;

    IF sla_policy_id IS NOT NULL THEN
        -- Calculate due times based on SLA policy
        SELECT
            NOW() + (response_time_minutes || ' minutes')::INTERVAL,
            NOW() + (resolution_time_minutes || ' minutes')::INTERVAL
        INTO response_due_at, resolution_due_at
        FROM sla_policies
        WHERE id = sla_policy_id;

        -- Create SLA metrics record
        INSERT INTO sla_metrics (
            tenant_id,
            ticket_id,
            sla_policy_id,
            response_due_at,
            resolution_due_at
        ) VALUES (
            NEW.tenant_id,
            NEW.id,
            sla_policy_id,
            response_due_at,
            resolution_due_at
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to create SLA metrics
CREATE TRIGGER create_sla_metrics_trigger
    AFTER INSERT ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION create_sla_metrics_for_ticket();

-- Function to check and update SLA breaches
CREATE OR REPLACE FUNCTION check_sla_breaches()
RETURNS VOID AS $$
BEGIN
    -- Update response breaches
    UPDATE sla_metrics
    SET response_breached = true,
        updated_at = NOW()
    WHERE response_due_at < NOW()
      AND first_response_at IS NULL
      AND response_breached = false;

    -- Update resolution breaches
    UPDATE sla_metrics
    SET resolution_breached = true,
        updated_at = NOW()
    WHERE resolution_due_at < NOW()
      AND resolved_at IS NULL
      AND resolution_breached = false;
END;
$$ LANGUAGE plpgsql;

-- Insert default SLA policies for test tenant
INSERT INTO sla_policies (tenant_id, name, priority, severity, response_time_minutes, resolution_time_minutes)
SELECT
    t.id,
    'Standard Response SLA',
    p.priority::ticket_priority,
    s.severity::ticket_severity,
    CASE
        WHEN p.priority = 'Critical' AND s.severity = 'Critical' THEN 15
        WHEN p.priority = 'Critical' THEN 30
        WHEN p.priority = 'High' THEN 60
        WHEN p.priority = 'Normal' THEN 240  -- 4 hours
        ELSE 480  -- 8 hours
    END,
    CASE
        WHEN p.priority = 'Critical' AND s.severity = 'Critical' THEN 60   -- 1 hour
        WHEN p.priority = 'Critical' THEN 240                           -- 4 hours
        WHEN p.priority = 'High' THEN 1440                             -- 1 day
        WHEN p.priority = 'Normal' THEN 4320                           -- 3 days
        ELSE 10080                                                   -- 7 days
    END
FROM tenants t
CROSS JOIN (VALUES ('Low'), ('Normal'), ('High'), ('Critical')) AS p(priority)
CROSS JOIN (VALUES ('Low'), ('Medium'), ('High'), ('Critical')) AS s(severity)
WHERE t.domain = 'test.smartticket.com';