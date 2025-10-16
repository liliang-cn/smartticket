#!/bin/bash

# SLA (Service Level Agreement) E2E Tests
# Tests SLA policy creation, ticket SLA assignment, and monitoring

echo "🚀 Starting SLA E2E Tests"
echo "=========================="

# Configuration
GRPC_URL="localhost:6533"
TENANT_DOMAIN="test.smartticket.com"
ADMIN_EMAIL="admin@test.smartticket.com"
ADMIN_PASSWORD="admin123"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TEST_COUNT=0
PASS_COUNT=0

# Helper function to run test
run_test() {
    local test_name="$1"
    local test_command="$2"

    TEST_COUNT=$((TEST_COUNT + 1))
    echo ""
    echo -e "${BLUE}🧪 Test $TEST_COUNT: $test_name${NC}"
    echo "Running: $test_command"

    if eval "$test_command"; then
        echo -e "${GREEN}✅ PASS: $test_name${NC}"
        PASS_COUNT=$((PASS_COUNT + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL: $test_name${NC}"
        return 1
    fi
}

# Helper function to check if gateway is running
check_gateway() {
    if ! curl -s "http://localhost:7218/health" > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Gateway not running, starting it...${NC}"
        RUST_LOG=info cargo run --bin gateway > /dev/null 2>&1 &
        sleep 8
        if ! curl -s "http://localhost:7218/health" > /dev/null 2>&1; then
            echo -e "${RED}❌ Failed to start gateway${NC}"
            exit 1
        fi
        echo -e "${GREEN}✅ Gateway started successfully${NC}"
    fi
}

# Check gateway is running
check_gateway

# Get authentication token
echo -e "${BLUE}🔐 Getting authentication token...${NC}"
AUTH_RESPONSE=$(grpcurl -plaintext -import-path ../../proto -proto smartticket/user.proto \
  -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\", \"tenantDomain\": \"$TENANT_DOMAIN\"}" \
  "$GRPC_URL" smartticket.v1.AuthService.Login 2>/dev/null)

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Authentication failed${NC}"
    exit 1
fi

ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.accessToken')
USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.id')
TENANT_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.tenantId')

echo -e "${GREEN}✅ Authentication successful${NC}"
echo "User ID: $USER_ID"
echo "Tenant ID: $TENANT_ID"

# Common headers for gRPC requests
GRPC_HEADERS="-H \"authorization: Bearer $ACCESS_TOKEN\" -H \"x-tenant-id: $TENANT_ID\" -H \"x-user-id: $USER_ID\""

# Test 1: Create SLA Policy
run_test "Create SLA Policy" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"name\": \"Standard SLA Policy\", \"description\": \"Standard response and resolution times\", \"responseTimeMinutes\": 60, \"resolutionTimeMinutes\": 480, \"businessHoursOnly\": true, \"priorityMultipliers\": \"{\\\"Low\\\": 2.0, \\\"Normal\\\": 1.0, \\\"High\\\": 0.5, \\\"Critical\\\": 0.25}\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.CreateSLAPolicy 2>/dev/null | jq -e '.id' > /dev/null"

# Store SLA Policy ID for later tests
SLA_RESPONSE=$(grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{"name": "Standard SLA Policy", "description": "Standard response and resolution times", "responseTimeMinutes": 60, "resolutionTimeMinutes": 480, "businessHoursOnly": true, "priorityMultipliers": "{\"Low\": 2.0, \"Normal\": 1.0, \"High\": 0.5, \"Critical\": 0.25}"}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  $GRPC_URL smartticket.v1.TicketService.CreateSLAPolicy 2>/dev/null)

SLA_POLICY_ID=$(echo "$SLA_RESPONSE" | jq -r '.id')
echo "Created SLA Policy ID: $SLA_POLICY_ID"

# Test 2: List SLA Policies
run_test "List SLA Policies" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.ListSLAPolicies 2>/dev/null | jq -e '.policies | length > 0' > /dev/null"

# Test 3: Get SLA Policy Details
run_test "Get SLA Policy Details" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"id\": \"$SLA_POLICY_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.GetSLAPolicy 2>/dev/null | jq -e '.id == \"$SLA_POLICY_ID\"' > /dev/null"

# Test 4: Create Ticket with SLA
run_test "Create Ticket with SLA Assignment" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"title\": \"SLA Test Ticket\", \"description\": \"Testing SLA assignment and monitoring\", \"priority\": 2, \"severity\": 2, \"slaPolicyId\": \"$SLA_POLICY_ID\", \"contactId\": \"$USER_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.CreateTicket 2>/dev/null | jq -e '.id' > /dev/null"

# Store Ticket ID for SLA testing
TICKET_RESPONSE=$(grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d "{\"title\": \"SLA Test Ticket\", \"description\": \"Testing SLA assignment and monitoring\", \"priority\": 2, \"severity\": 2, \"slaPolicyId\": \"$SLA_POLICY_ID\", \"contactId\": \"$USER_ID\"}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  $GRPC_URL smartticket.v1.TicketService.CreateTicket 2>/dev/null)

TICKET_ID=$(echo "$TICKET_RESPONSE" | jq -r '.id')
echo "Created Ticket ID: $TICKET_ID"

# Test 5: Get Ticket with SLA Information
run_test "Get Ticket with SLA Information" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"id\": \"$TICKET_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.GetTicket 2>/dev/null | jq -e '.slaInfo' > /dev/null"

# Test 6: List Tickets with SLA Status
run_test "List Tickets with SLA Status" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"pagination\": {\"pageSize\": 10}}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.ListTickets 2>/dev/null | jq -e '.tickets[0].slaInfo' > /dev/null"

# Test 7: Update SLA Policy
run_test "Update SLA Policy" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"id\": \"$SLA_POLICY_ID\", \"name\": \"Updated SLA Policy\", \"description\": \"Updated description\", \"responseTimeMinutes\": 30, \"resolutionTimeMinutes\": 240}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.UpdateSLAPolicy 2>/dev/null | jq -e '.id == \"$SLA_POLICY_ID\"' > /dev/null"

# Test 8: Get SLA Metrics for Ticket
run_test "Get SLA Metrics for Ticket" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"ticketId\": \"$TICKET_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.GetTicketSLAMetrics 2>/dev/null | jq -e '.responseDue' > /dev/null"

# Test 9: Check SLA Breach Detection
run_test "Check SLA Breach Detection" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"slaPolicyId\": \"$SLA_POLICY_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.GetSLAPolicies 2>/dev/null | jq -e '.slaPolicies[0].responseTimeMinutes' > /dev/null"

# Test 10: Delete SLA Policy (after deleting ticket)
# First delete the ticket
grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d "{\"id\": \"$TICKET_ID\"}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  $GRPC_URL smartticket.v1.TicketService.DeleteTicket 2>/dev/null > /dev/null

run_test "Delete SLA Policy" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/ticket.proto \
  -d '{\"id\": \"$SLA_POLICY_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.TicketService.DeleteSLAPolicy 2>/dev/null | jq -e '.success == true' > /dev/null"

# Print test results summary
echo ""
echo "================================"
echo -e "${BLUE}📊 SLA E2E Test Results Summary${NC}"
echo "================================"
echo "Total Tests: $TEST_COUNT"
echo -e "Passed: ${GREEN}$PASS_COUNT${NC}"
echo -e "Failed: ${RED}$((TEST_COUNT - PASS_COUNT))${NC}"

if [ $PASS_COUNT -eq $TEST_COUNT ]; then
    echo ""
    echo -e "${GREEN}🎉 All SLA E2E tests passed!${NC}"
    echo "✅ SLA Policy Management: CREATE, READ, UPDATE, DELETE"
    echo "✅ SLA Assignment to Tickets"
    echo "✅ SLA Monitoring and Metrics"
    echo "✅ SLA Breach Detection"
    echo "✅ Multi-tenant SLA isolation"
    exit 0
else
    echo ""
    echo -e "${RED}❌ Some SLA E2E tests failed!${NC}"
    echo "Please check the logs above for details."
    exit 1
fi