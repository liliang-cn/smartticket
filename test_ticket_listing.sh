#!/bin/bash

echo "🎟️  Testing Ticket Listing Service"
echo "================================="

# Get authentication info
ADMIN_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/user.proto \
  -d '{"email": "admin@test.smartticket.com", "password": "admin123", "tenantDomain": "test.smartticket.com"}' \
  localhost:6533 smartticket.v1.AuthService.Login 2>/dev/null)

ACCESS_TOKEN=$(echo "$ADMIN_RESPONSE" | jq -r '.accessToken')
USER_ID=$(echo "$ADMIN_RESPONSE" | jq -r '.user.id')
TENANT_ID=$(echo "$ADMIN_RESPONSE" | jq -r '.user.tenantId')

echo "User: $USER_ID"
echo "Tenant: $TENANT_ID"
echo ""

# Test ticket listing
echo "📋 Testing ListTickets API:"
echo "📡 Sending gRPC request..."
LIST_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
  -d "{\"metadata\": {\"tenantId\": \"$TENANT_ID\", \"userId\": \"$USER_ID\", \"requestId\": \"test-list-123\"}, \"pagination\": {\"pageSize\": 10}}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.TicketService.ListTickets 2>&1)

echo "📋 Raw response:"
echo "$LIST_RESPONSE"
echo ""
echo "📋 Formatted response:"
echo "$LIST_RESPONSE" | jq '.' 2>/dev/null || echo "Response is not valid JSON"

echo ""
echo "✅ Ticket listing service test completed!"