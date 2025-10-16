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