#!/bin/bash

# SlaService gRPC E2E Test
# Tests all 9 interfaces of the SlaService
# Uses grpcurl with real JWT authentication

echo "⏱️ SlaService gRPC E2E Test (9 interfaces)"
echo "========================================="

# Configuration
GRPC_HOST="localhost"
GRPC_PORT="6533"
PROTO_DIR="./proto"
USER_SERVICE_PROTO="proto/smartticket/user.proto"
SLA_SERVICE_PROTO="proto/smartticket/sla.proto"

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
TEST_SLA_NAME="Standard Support SLA"
TEST_SLA_DESCRIPTION="Standard SLA policy for normal priority tickets"
TEST_PRIORITY="TICKET_PRIORITY_NORMAL"
TEST_SEVERITY="TICKET_SEVERITY_MEDIUM"
TEST_RESPONSE_TIME=60
TEST_RESOLUTION_TIME=480
TEST_TIMEZONE="UTC"
TEST_BREACH_TYPE="response"

# Test counters
TOTAL_TESTS=9
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
        if echo "$response" | grep -q '"success":true\|"policy_id"\|"policies"\|"metrics"\|"breaches"\|"dashboard"'; then
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

# Step 2: Test all SlaService interfaces
echo -e "\n${YELLOW}⏱️ Step 2: Testing SlaService interfaces...${NC}"

# Test 1: CreateSlaPolicy
create_sla_data="{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"name\": \"$TEST_SLA_NAME\", \"description\": \"$TEST_SLA_DESCRIPTION\", \"priority\": \"$TEST_PRIORITY\", \"severity\": \"$TEST_SEVERITY\", \"response_time_minutes\": $TEST_RESPONSE_TIME, \"resolution_time_minutes\": $TEST_RESOLUTION_TIME, \"business_hours_only\": false, \"timezone\": \"$TEST_TIMEZONE\"}"

run_grpcurl "CreateSlaPolicy" "$create_sla_data" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/CreateSlaPolicy"

# Extract created SLA policy ID for subsequent tests
CREATE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$SLA_SERVICE_PROTO" \
    -d "$create_sla_data" \
    -rpc-header "authorization: Bearer $JWT_TOKEN" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.SlaService/CreateSlaPolicy 2>/dev/null)

if echo "$CREATE_RESPONSE" | grep -q '"id"'; then
    SLA_POLICY_ID=$(echo "$CREATE_RESPONSE" | jq -r '.policy.id')
    echo "Created SLA policy ID: $SLA_POLICY_ID"
fi

# Test 2: GetSlaPolicy
if [ ! -z "$SLA_POLICY_ID" ]; then
    run_grpcurl "GetSlaPolicy" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"policy_id\": \"$SLA_POLICY_ID\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/GetSlaPolicy"
else
    run_grpcurl "GetSlaPolicy" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"policy_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/GetSlaPolicy" "false"
fi

# Test 3: UpdateSlaPolicy
if [ ! -z "$SLA_POLICY_ID" ]; then
    run_grpcurl "UpdateSlaPolicy" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"policy_id\": \"$SLA_POLICY_ID\", \"name\": \"Updated $TEST_SLA_NAME\", \"description\": \"Updated description\", \"response_time_minutes\": 30, \"resolution_time_minutes\": 240}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/UpdateSlaPolicy"
else
    run_grpcurl "UpdateSlaPolicy" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"policy_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"name\": \"Updated Test SLA\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/UpdateSlaPolicy" "false"
fi

# Test 4: ListSlaPolicies
run_grpcurl "ListSlaPolicies" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/ListSlaPolicies"

# Test 5: DeleteSlaPolicy
if [ ! -z "$SLA_POLICY_ID" ]; then
    run_grpcurl "DeleteSlaPolicy" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"policy_id\": \"$SLA_POLICY_ID\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/DeleteSlaPolicy"
else
    run_grpcurl "DeleteSlaPolicy" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"policy_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/DeleteSlaPolicy" "false"
fi

# Test 6: GetSlaMetrics
run_grpcurl "GetSlaMetrics" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/GetSlaMetrics"

# Test 7: GetSlaDashboard
run_grpcurl "GetSlaDashboard" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"group_by\": \"day\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/GetSlaDashboard"

# Test 8: GetSlaBreaches
run_grpcurl "GetSlaBreaches" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}, \"breach_type\": \"$TEST_BREACH_TYPE\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/GetSlaBreaches"

# Test 9: UpdateSlaMetrics
run_grpcurl "UpdateSlaMetrics" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"event_type\": \"first_response\"}" "$SLA_SERVICE_PROTO" "smartticket.v1.SlaService/UpdateSlaMetrics"

# Step 3: Test Results
echo -e "\n${YELLOW}📊 Step 3: SlaService Test Results${NC}"
echo "========================================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "Pass Rate: ${GREEN}$PASS_RATE%${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All SlaService tests passed!${NC}"
else
    echo -e "\n${YELLOW}⚠️ Some SlaService tests failed. This may be expected due to permission restrictions or backend issues.${NC}"
fi

echo -e "\n${BLUE}SlaService E2E Test Complete!${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -gt 0 ]; then
    exit 1
else
    exit 0
fi