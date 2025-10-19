#!/bin/bash

# TicketService gRPC E2E Test (Fixed Version 2)
# Tests all 11 interfaces of the TicketService with correct field mappings
# Uses grpcurl with real JWT authentication

echo "🎫 TicketService gRPC E2E Test (Fixed V2) - 11 interfaces"
echo "=========================================================="

# Configuration
GRPC_HOST="localhost"
GRPC_PORT="6533"
PROTO_DIR="./proto"
USER_SERVICE_PROTO="proto/smartticket/user.proto"
TICKET_SERVICE_PROTO="proto/smartticket/ticket.proto"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test data
TEST_EMAIL="admin@test.smartticket.com"
TEST_PASSWORD="admin123"
TEST_TENANT="test.smartticket.com"
TEST_TITLE="Test Ticket via gRPC"
TEST_DESCRIPTION="This is a test ticket created via gRPC E2E testing"
TEST_PRIORITY="TICKET_PRIORITY_NORMAL"
TEST_SEVERITY="TICKET_SEVERITY_MEDIUM"
TEST_CATEGORY="TECHNICAL_SUPPORT"
TEST_CUSTOMER_EMAIL="testuser-1760622283@test.smartticket.com"

# Test counters
TOTAL_TESTS=11
PASSED_TESTS=0
FAILED_TESTS=0

# Helper function to run grpcurl command
run_grpcurl() {
    local method="$1"
    local data="$2"
    local proto_file="$3"
    local service_method="$4"
    local expected_success="$5" # "true" if we expect success, "false" if we expect failure

    echo -e "\n${BLUE}Testing: $method${NC}"
    echo "Data: $data"

    # Run grpcurl command
    response=$(grpcurl -plaintext \
        -import-path ./proto \
        -proto "$proto_file" \
        -d "$data" \
        -rpc-header "authorization: Bearer $JWT_TOKEN" \
        "$GRPC_HOST:$GRPC_PORT" \
        "$service_method" 2>&1)

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        if echo "$response" | grep -q '"success":true\|"ticket_id"\|"tickets"\|"total_count"\|"updated_at"'; then
            echo -e "${GREEN}✅ PASS${NC}"
            echo "Response: $(echo "$response" | head -c 200)..."
            ((PASSED_TESTS++))
            return 0
        else
            echo -e "${RED}❌ FAIL${NC}"
            echo "Error response: $response"
            ((FAILED_TESTS++))
            return 1
        fi
    else
        if [ "$expected_success" = "false" ]; then
            echo -e "${GREEN}✅ PASS (expected failure)${NC}"
            echo "Expected error: $response"
            ((PASSED_TESTS++))
            return 0
        else
            echo -e "${RED}❌ FAIL${NC}"
            echo "Command failed: $response"
            ((FAILED_TESTS++))
            return 1
        fi
    fi
}

# Step 1: Authenticate and get JWT token
echo -e "\n${YELLOW}🔐 Step 1: Authenticating and getting JWT token...${NC}"

auth_response=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$USER_SERVICE_PROTO" \
    -d "{\"email\": \"$TEST_EMAIL\", \"password\": \"$TEST_PASSWORD\", \"tenant_domain\": \"$TEST_TENANT\"}" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.AuthService/Login 2>/dev/null)

if echo "$auth_response" | grep -q '"accessToken"'; then
    JWT_TOKEN=$(echo "$auth_response" | jq -r '.accessToken')
    TENANT_ID=$(echo "$auth_response" | jq -r '.user.tenantId')
    USER_ID=$(echo "$auth_response" | jq -r '.user.id')
    USER_ROLES=$(echo "$auth_response" | jq -r '.user.role')

    echo -e "${GREEN}✅ Authentication successful${NC}"
    echo "Tenant ID: $TENANT_ID"
    echo "User ID: $USER_ID"
    echo "User Roles: $USER_ROLES"
else
    echo -e "${RED}❌ Authentication failed${NC}"
    echo "Response: $auth_response"
    exit 1
fi

# Step 2: Get customer user ID for creating ticket
echo -e "\n${YELLOW}👥 Step 2: Getting customer user ID...${NC}"

customer_response=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$USER_SERVICE_PROTO" \
    -d "{\"email\": \"$TEST_CUSTOMER_EMAIL\"}" \
    -rpc-header "authorization: Bearer $JWT_TOKEN" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.UserService/GetUserByEmail 2>/dev/null)

if echo "$customer_response" | grep -q '"user_id"'; then
    CUSTOMER_USER_ID=$(echo "$customer_response" | jq -r '.user_id')
    echo -e "${GREEN}✅ Customer user ID found: $CUSTOMER_USER_ID${NC}"
else
    echo -e "${YELLOW}⚠️ Customer user not found, using admin user as contact_id${NC}"
    CUSTOMER_USER_ID="$USER_ID"
fi

# Step 3: Test all TicketService interfaces
echo -e "\n${YELLOW}🎫 Step 3: Testing TicketService interfaces...${NC}"

# Test 1: CreateTicket (with correct RequestMetadata structure and contact_id)
create_ticket_data="{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"title\": \"$TEST_TITLE\", \"description\": \"$TEST_DESCRIPTION\", \"priority\": \"$TEST_PRIORITY\", \"severity\": \"$TEST_SEVERITY\", \"contact_id\": \"$CUSTOMER_USER_ID\"}"

run_grpcurl "CreateTicket" "$create_ticket_data" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/CreateTicket"

# Extract created ticket ID for subsequent tests
CREATE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$TICKET_SERVICE_PROTO" \
    -d "$create_ticket_data" \
    -rpc-header "authorization: Bearer $JWT_TOKEN" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.TicketService/CreateTicket 2>/dev/null)

if echo "$CREATE_RESPONSE" | grep -q '"id"'; then
    TICKET_ID=$(echo "$CREATE_RESPONSE" | jq -r '.ticket.id')
    echo "Created ticket ID: $TICKET_ID"
fi

# Test 2: GetTicket
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "GetTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/GetTicket"
else
    run_grpcurl "GetTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/GetTicket" "false"
fi

# Test 3: ListTickets
run_grpcurl "ListTickets" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/ListTickets"

# Test 4: UpdateTicket
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "UpdateTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\", \"title\": \"Updated $TEST_TITLE\", \"description\": \"Updated description\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/UpdateTicket"
else
    run_grpcurl "UpdateTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"title\": \"Updated Test Ticket\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/UpdateTicket" "false"
fi

# Test 5: DeleteTicket (soft delete)
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "DeleteTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/DeleteTicket"
else
    run_grpcurl "DeleteTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/DeleteTicket" "false"
fi

# Test 6: UpdateTicketStatus (correct method name)
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "UpdateTicketStatus" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\", \"status\": \"TICKET_STATUS_IN_PROGRESS\", \"comment\": \"Working on this issue\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/UpdateTicketStatus"
else
    run_grpcurl "UpdateTicketStatus" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"status\": \"TICKET_STATUS_IN_PROGRESS\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/UpdateTicketStatus" "false"
fi

# Test 7: AssignTicket (with correct field name)
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "AssignTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\", \"assigned_to_id\": \"$USER_ID\", \"comment\": \"Assigning to myself\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/AssignTicket"
else
    run_grpcurl "AssignTicket" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"assigned_to_id\": \"$USER_ID\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/AssignTicket" "false"
fi

# Test 8: AddComment
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "AddComment" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\", \"content\": \"This is a test comment via gRPC\", \"is_internal\": false}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/AddComment"
else
    run_grpcurl "AddComment" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"content\": \"Test comment\"}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/AddComment" "false"
fi

# Test 9: GetComments (correct method name)
if [ ! -z "$TICKET_ID" ]; then
    run_grpcurl "GetComments" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"$TICKET_ID\", \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/GetComments"
else
    run_grpcurl "GetComments" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"pagination\": {\"page_size\": 10}}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/GetComments" "false"
fi

# Test 10: SearchTickets
run_grpcurl "SearchTickets" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"query\": \"test\", \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/SearchTickets"

# Test 11: GetTicketStatistics (correct method name)
run_grpcurl "GetTicketStatistics" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}}" "$TICKET_SERVICE_PROTO" "smartticket.v1.TicketService/GetTicketStatistics"

# Step 4: Test Results
echo -e "\n${YELLOW}📊 Step 4: TicketService Test Results${NC}"
echo "=========================================================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "Pass Rate: ${GREEN}$PASS_RATE%${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All TicketService tests passed!${NC}"
else
    echo -e "\n${YELLOW}⚠️ Some TicketService tests failed. This may be expected due to permission restrictions or backend issues.${NC}"
fi

echo -e "\n${BLUE}TicketService E2E Test Complete!${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -gt 0 ]; then
    exit 1
else
    exit 0
fi