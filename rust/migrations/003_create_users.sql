-- Create user roles enum (skip if already exists from init script)
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM (
        'customer',
        'agent',
        'team_lead',
        'admin',
        'system_admin'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    username VARCHAR(100) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'customer',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(is_active);
CREATE UNIQUE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE UNIQUE INDEX idx_users_tenant_username ON users(tenant_id, username);

-- Create trigger for updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- RLS policies for users
CREATE POLICY tenant_isolation_users ON users
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Allow super admins to see all users
CREATE POLICY superadmin_all_users ON users
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.current_user_id', true)::uuid
            AND role = 'system_admin'
        )
    );

-- Users can see their own profile
CREATE POLICY users_own_profile ON users
    FOR SELECT
    USING (id = current_setting('app.current_user_id', true)::uuid);

-- Insert default admin user for test tenant
INSERT INTO users (tenant_id, email, username, full_name, password_hash, role)
SELECT
    id,
    'admin@test.smartticket.com',
    'admin',
    'System Administrator',
    '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS', -- password: admin123
    'admin'
FROM tenants
WHERE domain = 'test.smartticket.com';

-- Insert super admin user for system administration
INSERT INTO users (tenant_id, email, username, full_name, password_hash, role)
SELECT
    id,
    'superadmin@smartticket.system',
    'superadmin',
    'System Super Administrator',
    '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS', -- password: admin123
    'system_admin'
FROM tenants
WHERE domain = 'test.smartticket.com';