#!/bin/bash

# Test CreateSlaPolicy interface

# First, let's test the gateway health
echo "Testing gateway health..."
curl -s http://localhost:7218/health || echo "Health check failed"

# Get a JWT token by logging in first (create a test tenant and user if needed)
echo -e "\nCreating test tenant..."
TENANT_RESPONSE=$(grpcurl -plaintext -d '{
  "name": "Test SLA Tenant",
  "domain": "sla-test.com",
  "subscription_tier": 2,
  "admin_email": "admin@sla-test.com",
  "admin_name": "SLA Test Admin"
}' localhost:6533 smartticket.v1.TenantService/CreateTenant)

echo "Tenant response: $TENANT_RESPONSE"

# Extract tenant ID
TENANT_ID=$(echo $TENANT_RESPONSE | jq -r '.tenant.id')
echo "Created tenant with ID: $TENANT_ID"

# Create admin user for the tenant
echo -e "\nCreating admin user..."
USER_RESPONSE=$(grpcurl -plaintext -d '{
  "tenant_id": "'$TENANT_ID'",
  "email": "admin@sla-test.com",
  "name": "SLA Test Admin",
  "role": 2,
  "password": "password123"
}' localhost:6533 smartticket.v1.UserService/CreateUser)

echo "User response: $USER_RESPONSE"

# Login to get JWT token
echo -e "\nLogging in..."
LOGIN_RESPONSE=$(grpcurl -plaintext -d '{
  "email": "admin@sla-test.com",
  "password": "password123"
}' localhost:6533 smartticket.v1.AuthService/Login)

echo "Login response: $LOGIN_RESPONSE"

# Extract JWT token
JWT_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
echo "JWT token: $JWT_TOKEN"

# Test CreateSlaPolicy interface
echo -e "\nTesting CreateSlaPolicy interface..."

SLA_RESPONSE=$(grpcurl -plaintext -H "authorization: Bearer $JWT_TOKEN" -d '{
  "name": "Standard Support SLA",
  "description": "Standard customer support SLA policy",
  "priority": 2,
  "severity": 2,
  "response_time_minutes": 60,
  "resolution_time_minutes": 240,
  "business_hours_only": false,
  "timezone": "UTC"
}' localhost:6533 smartticket.v1.SlaService/CreateSlaPolicy)

echo "SLA Policy response: $SLA_RESPONSE"

# Extract policy ID for further tests
POLICY_ID=$(echo $SLA_RESPONSE | jq -r '.policy.id')
echo "Created SLA Policy with ID: $POLICY_ID"

# Test GetSlaPolicy interface
echo -e "\nTesting GetSlaPolicy interface..."
GET_SLA_RESPONSE=$(grpcurl -plaintext -H "authorization: Bearer $JWT_TOKEN" -d '{
  "policy_id": "'$POLICY_ID'"
}' localhost:6533 smartticket.v1.SlaService/GetSlaPolicy)

echo "Get SLA Policy response: $GET_SLA_RESPONSE"

# Test ListSlaPolicies interface
echo -e "\nTesting ListSlaPolicies interface..."
LIST_SLA_RESPONSE=$(grpcurl -plaintext -H "authorization: Bearer $JWT_TOKEN" -d '{
  "pagination": {
    "page_size": 10
  },
  "is_active": true
}' localhost:6533 smartticket.v1.SlaService/ListSlaPolicies)

echo "List SLA Policies response: $LIST_SLA_RESPONSE"

echo -e "\nSLA interface tests completed!"