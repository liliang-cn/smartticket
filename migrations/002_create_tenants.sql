-- Create tenant types (skip if already exists from init script)
DO $$ BEGIN
    CREATE TYPE subscription_tier AS ENUM ('trial', 'standard', 'premium', 'enterprise');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    settings JSONB DEFAULT '{}',
    subscription_tier subscription_tier NOT NULL DEFAULT 'standard',
    max_users INTEGER NOT NULL DEFAULT 10,
    data_residency_region VARCHAR(50) NOT NULL DEFAULT 'EU',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_tenants_domain ON tenants(domain);
CREATE INDEX idx_tenants_active ON tenants(is_active);
CREATE INDEX idx_tenants_subscription ON tenants(subscription_tier);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default tenant for testing
INSERT INTO tenants (name, domain, subscription_tier, max_users, data_residency_region)
VALUES ('Test Company', 'test.smartticket.com', 'enterprise', 1000, 'EU');