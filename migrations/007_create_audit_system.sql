-- Create audit logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for audit logs
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);

-- Composite index for common queries
CREATE INDEX idx_audit_logs_tenant_resource ON audit_logs(tenant_id, resource_type, resource_id);

-- Enable RLS
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Super admins can see all audit logs
CREATE POLICY superadmin_audit_logs ON audit_logs
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.current_user_id', true)::uuid
            AND role = 'system_admin'
        )
    );

-- Function to create audit trigger
CREATE OR REPLACE FUNCTION create_audit_trigger(table_name TEXT)
RETURNS VOID AS $$
DECLARE
    trigger_name TEXT;
    trigger_function_name TEXT;
BEGIN
    trigger_name := 'audit_' || table_name || '_trigger';
    trigger_function_name := 'audit_' || table_name || '_function';

    -- Create trigger function
    EXECUTE format('
        CREATE OR REPLACE FUNCTION %I()
        RETURNS TRIGGER AS $audit_trigger$
        BEGIN
            IF TG_OP = ''INSERT'' THEN
                INSERT INTO audit_logs (
                    tenant_id,
                    user_id,
                    action,
                    resource_type,
                    resource_id,
                    new_values,
                    ip_address,
                    user_agent,
                    request_id
                ) VALUES (
                    NEW.tenant_id,
                    current_setting(''app.current_user_id'', true)::uuid,
                    ''CREATE'',
                    ''%s'',
                    NEW.id::TEXT,
                    row_to_json(NEW),
                    current_setting(''app.client_ip_address'', true)::inet,
                    current_setting(''app.user_agent'', true),
                    current_setting(''app.request_id'', true)
                );
                RETURN NEW;
            ELSIF TG_OP = ''UPDATE'' THEN
                -- Only log if tenant_id hasn't changed (which would indicate a bug)
                IF OLD.tenant_id = NEW.tenant_id THEN
                    INSERT INTO audit_logs (
                        tenant_id,
                        user_id,
                        action,
                        resource_type,
                        resource_id,
                        old_values,
                        new_values,
                        ip_address,
                        user_agent,
                        request_id
                    ) VALUES (
                        NEW.tenant_id,
                        current_setting(''app.current_user_id'', true)::uuid,
                        ''UPDATE'',
                        ''%s'',
                        NEW.id::TEXT,
                        row_to_json(OLD),
                        row_to_json(NEW),
                        current_setting(''app.client_ip_address'', true)::inet,
                        current_setting(''app.user_agent'', true),
                        current_setting(''app.request_id'', true)
                    );
                END IF;
                RETURN NEW;
            ELSIF TG_OP = ''DELETE'' THEN
                INSERT INTO audit_logs (
                    tenant_id,
                    user_id,
                    action,
                    resource_type,
                    resource_id,
                    old_values,
                    ip_address,
                    user_agent,
                    request_id
                ) VALUES (
                    OLD.tenant_id,
                    current_setting(''app.current_user_id'', true)::uuid,
                    ''DELETE'',
                    ''%s'',
                    OLD.id::TEXT,
                    row_to_json(OLD),
                    current_setting(''app.client_ip_address'', true)::inet,
                    current_setting(''app.user_agent'', true),
                    current_setting(''app.request_id'', true)
                );
                RETURN OLD;
            END IF;
            RETURN NULL;
        END;
        $audit_trigger$ LANGUAGE plpgsql SECURITY DEFINER;',
        trigger_function_name, table_name, table_name, table_name);

    -- Create trigger
    EXECUTE format('
        DROP TRIGGER IF EXISTS %I ON %I;
        CREATE TRIGGER %I
            AFTER INSERT OR UPDATE OR DELETE ON %I
            FOR EACH ROW
            EXECUTE FUNCTION %I();',
        trigger_name, table_name, trigger_name, table_name, trigger_function_name);

END;
$$ LANGUAGE plpgsql;

-- Create audit triggers for important tables
SELECT create_audit_trigger('users');
SELECT create_audit_trigger('tickets');
SELECT create_audit_trigger('knowledge_articles');
SELECT create_audit_trigger('sla_policies');

-- Function to set application context for RLS and auditing
CREATE OR REPLACE FUNCTION set_app_context(
    p_tenant_id UUID,
    p_user_id UUID,
    p_client_ip_address TEXT DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_request_id TEXT DEFAULT NULL
)
RETURNS VOID AS $$
BEGIN
    -- Set session variables for RLS and auditing
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, true);
    PERFORM set_config('app.current_user_id', p_user_id::TEXT, true);

    IF p_client_ip_address IS NOT NULL THEN
        PERFORM set_config('app.client_ip_address', p_client_ip_address, true);
    END IF;

    IF p_user_agent IS NOT NULL THEN
        PERFORM set_config('app.user_agent', p_user_agent, true);
    END IF;

    IF p_request_id IS NOT NULL THEN
        PERFORM set_config('app.request_id', p_request_id, true);
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to clear application context
CREATE OR REPLACE FUNCTION clear_app_context()
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', '', true);
    PERFORM set_config('app.current_user_id', '', true);
    PERFORM set_config('app.client_ip_address', '', true);
    PERFORM set_config('app.user_agent', '', true);
    PERFORM set_config('app.request_id', '', true);
END;
$$ LANGUAGE plpgsql;