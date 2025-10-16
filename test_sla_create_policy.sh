#!/bin/bash

echo "🔐 SLA Test 1: Authenticating..."
LOGIN_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/user.proto \
  -d '{"email": "admin@test.smartticket.com", "password": "admin123", "tenant_domain": "test.smartticket.com"}' \
  localhost:6533 smartticket.v1.AuthService.Login 2>/dev/null)

if [[ $? -ne 0 ]]; then
  echo "❌ FAILED: Authentication"
  exit 1
fi

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.accessToken')
USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user.id')
TENANT_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user.tenantId')

echo "✅ SUCCESS: Authentication working"
echo "   👤 User ID: $USER_ID"
echo "   🏢 Tenant ID: $TENANT_ID"

echo ""
echo "🎯 SLA Test 2: CreateSlaPolicy Interface"

# Test CreateSlaPolicy
CREATE_SLA_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/sla.proto \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  -d '{
    "metadata": {
      "tenant_id": "'$TENANT_ID'",
      "user_id": "'$USER_ID'",
      "request_id": "test-sla-create-001"
    },
    "name": "Standard Support SLA",
    "description": "Standard SLA for regular support tickets",
    "priority": "TICKET_PRIORITY_NORMAL",
    "severity": "TICKET_SEVERITY_MEDIUM",
    "response_time_minutes": 240,
    "resolution_time_minutes": 1440,
    "business_hours_only": true,
    "timezone": "UTC"
  }' \
  localhost:6533 smartticket.v1.SlaService.CreateSlaPolicy 2>/dev/null)

if [[ $? -eq 0 ]]; then
  SLA_POLICY_ID=$(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.id')
  echo "✅ SUCCESS: CreateSlaPolicy interface working"
  echo "   📋 SLA Policy ID: $SLA_POLICY_ID"
  echo "   📝 Policy Name: $(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.name')"
  echo "   ⏱️ Response Time: $(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.response_time_minutes') minutes"
  echo "   🎯 Resolution Time: $(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.resolution_time_minutes') minutes"
else
  echo "❌ FAILED: CreateSlaPolicy interface"
  echo "Error details:"
  grpcurl -plaintext -import-path proto -proto smartticket/sla.proto \
    -H "authorization: Bearer $ACCESS_TOKEN" \
    -H "x-tenant-id: $TENANT_ID" \
    -H "x-user-id: $USER_ID" \
    -d '{
      "metadata": {
        "tenant_id": "'$TENANT_ID'",
        "user_id": "'$USER_ID'",
        "request_id": "test-sla-create-001"
      },
      "name": "Standard Support SLA",
      "description": "Standard SLA for regular support tickets",
      "priority": "TICKET_PRIORITY_NORMAL",
      "severity": "TICKET_SEVERITY_MEDIUM",
      "response_time_minutes": 240,
      "resolution_time_minutes": 1440,
      "business_hours_only": true,
      "timezone": "UTC"
    }' \
    localhost:6533 smartticket.v1.SlaService.CreateSlaPolicy
  exit 1
fi