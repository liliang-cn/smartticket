#!/bin/bash

# Get authentication info
ADMIN_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/user.proto \
  -d '{"email": "admin@test.smartticket.com", "password": "admin123", "tenantDomain": "test.smartticket.com"}' \
  localhost:6533 smartticket.v1.AuthService.Login 2>/dev/null)

ACCESS_TOKEN=$(echo "$ADMIN_RESPONSE" | jq -r '.accessToken')
USER_ID=$(echo "$ADMIN_RESPONSE" | jq -r '.user.id')
TENANT_ID=$(echo "$ADMIN_RESPONSE" | jq -r '.user.tenantId')

echo "Access Token: $ACCESS_TOKEN"
echo "User ID: $USER_ID"
echo "Tenant ID: $TENANT_ID"

# Create ticket with contact ID
TICKET_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
  -d "{\"title\": \"Test Ticket E2E\", \"description\": \"This is a test ticket for E2E validation\", \"priority\": 2, \"categoryId\": \"b67bb054-5ef4-4f47-adad-c074f64a8d0e\", \"tags\": [\"test\", \"e2e\"], \"contactId\": \"$USER_ID\"}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.TicketService.CreateTicket 2>/dev/null)

echo "Ticket Creation Response:"
echo "$TICKET_RESPONSE"

# Check if ticket was created in database
echo ""
echo "Checking tickets in database:"
docker exec smartticket-postgres-dev psql -U postgres -d smartticket -c "SELECT id, title, status, priority FROM tickets WHERE tenant_id = '$TENANT_ID' ORDER BY created_at DESC LIMIT 3;"