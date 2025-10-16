#!/bin/bash

echo "🧪 Simple SLA Service Test"
echo "============================"

# Start gateway in background
echo "🚀 Starting gateway service..."
RUST_LOG=info cargo run --bin gateway > gateway_sla_test.log 2>&1 &
GATEWAY_PID=$!
sleep 10

# Check if gateway is running
if curl -s http://localhost:7218/health > /dev/null 2>&1; then
    echo "✅ Gateway started successfully"
else
    echo "❌ Gateway failed to start"
    cat gateway_sla_test.log
    exit 1
fi

# Get auth token
echo "🔐 Getting authentication token..."
AUTH_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/user.proto \
  -d '{"email": "admin@test.smartticket.com", "password": "admin123", "tenantDomain": "test.smartticket.com"}' \
  localhost:6533 smartticket.v1.AuthService.Login 2>/dev/null)

if [ $? -ne 0 ]; then
    echo "❌ Authentication failed"
    cat gateway_sla_test.log
    exit 1
fi

ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.accessToken')
USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.id')
TENANT_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.tenantId')

echo "✅ Authentication successful"

# Test SLA policy creation
echo "📋 Testing SLA policy creation..."
SLA_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
  -d '{"name": "Test SLA Policy", "description": "Test SLA policy for E2E", "responseTimeMinutes": 60, "resolutionTimeMinutes": 480, "businessHoursOnly": true}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.TicketService.CreateSLAPolicy 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ SLA policy creation successful"
    SLA_ID=$(echo "$SLA_RESPONSE" | jq -r '.id')
    echo "SLA Policy ID: $SLA_ID"
else
    echo "❌ SLA policy creation failed"
    echo "Response: $SLA_RESPONSE"
fi

# Test SLA policy listing
echo "📋 Testing SLA policy listing..."
LIST_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
  -d '{}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.TicketService.ListSLAPolicies 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ SLA policy listing successful"
    SLA_COUNT=$(echo "$LIST_RESPONSE" | jq '.policies | length')
    echo "Found $SLA_COUNT SLA policies"
else
    echo "❌ SLA policy listing failed"
fi

# Test ticket creation with SLA
echo "🎫 Testing ticket creation with SLA..."
TICKET_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
  -d "{\"title\": \"SLA Test Ticket\", \"description\": \"Testing SLA assignment\", \"priority\": 2, \"contactId\": \"$USER_ID\", \"slaPolicyId\": \"$SLA_ID\"}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.TicketService.CreateTicket 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ Ticket creation with SLA successful"
    TICKET_ID=$(echo "$TICKET_RESPONSE" | jq -r '.id')
    echo "Ticket ID: $TICKET_ID"
else
    echo "❌ Ticket creation with SLA failed"
    echo "Response: $TICKET_RESPONSE"
fi

# Cleanup
echo "🧹 Cleaning up..."
kill $GATEWAY_PID 2>/dev/null

echo ""
echo "============================"
echo "🎉 SLA Service Test Complete"
echo "============================"