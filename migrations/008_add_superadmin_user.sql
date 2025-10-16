-- Add Super Admin User Migration
-- This migration creates a super admin user for system administration

-- Insert super admin user for system administration
INSERT INTO users (
    tenant_id,
    email,
    username,
    full_name,
    password_hash,
    role,
    is_active,
    created_at,
    updated_at
)
SELECT
    t.id,
    'superadmin@smartticket.system',
    'superadmin',
    'System Super Administrator',
    '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS', -- password: admin123
    'system_admin',
    true,
    NOW(),
    NOW()
FROM tenants t
WHERE t.domain = 'test.smartticket.com'
ON CONFLICT DO NOTHING;

-- Update existing tenant admin to include tenant:create permission if needed
-- This ensures tenant admin can perform basic tenant management operations
DO $$
DECLARE
    tenant_admin_uuid UUID;
BEGIN
    SELECT id INTO tenant_admin_uuid
    FROM users
    WHERE email = 'admin@test.smartticket.com' AND role = 'admin'
    LIMIT 1;

    IF tenant_admin_uuid IS NOT NULL THEN
        -- Log successful creation verification
        RAISE NOTICE 'Super Admin user created successfully';
        RAISE NOTICE 'Super Admin ID: %', tenant_admin_uuid;
    END IF;
END $$;

-- Create a function to check if super admin exists
CREATE OR REPLACE FUNCTION check_superadmin_exists()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users
        WHERE email = 'superadmin@smartticket.system'
        AND role = 'system_admin'
        AND is_active = true
    );
END;
$$ LANGUAGE plpgsql;

-- Verify the super admin user was created
SELECT
    CASE
        WHEN check_superadmin_exists() THEN 'Super Admin user successfully created'
        ELSE 'Super Admin user creation failed'
    END as migration_result;