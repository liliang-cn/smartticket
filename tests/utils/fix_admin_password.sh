#!/bin/bash

# Fix admin user password in database
echo "🔧 Fixing admin user password..."

# Connect to PostgreSQL and update the admin user password
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d smartticket -c "
UPDATE users
SET password_hash = '\$2b\$12\$r2IkLwr1orrSp4/kzpAfj.bu7bmqv3Y/KPWldwi8BeC4.KrHjPZfi'
WHERE email = 'admin@test.smartticket.com';

-- Also ensure superadmin user exists and has correct password
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
    '\$2b\$12\$r2IkLwr1orrSp4/kzpAfj.bu7bmqv3Y/KPWldwi8BeC4.KrHjPZfi',
    'system_admin',
    true,
    NOW(),
    NOW()
FROM tenants t
WHERE t.domain = 'test.smartticket.com'
ON CONFLICT (tenant_id, email) DO UPDATE SET
    password_hash = '\$2b\$12\$r2IkLwr1orrSp4/kzpAfj.bu7bmqv3Y/KPWldwi8BeC4.KrHjPZfi',
    is_active = true;

SELECT email, username, role, is_active FROM users
WHERE email IN ('admin@test.smartticket.com', 'superadmin@smartticket.system');
"

echo "✅ Password fix completed"