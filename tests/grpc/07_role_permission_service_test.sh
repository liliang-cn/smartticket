#!/bin/bash

# RolePermissionService gRPC E2E Test
# Tests all 13 interfaces of the RolePermissionService
# Uses grpcurl with real JWT authentication

echo "🔑 RolePermissionService gRPC E2E Test (13 interfaces)"
echo "=================================================="

# Configuration
GRPC_HOST="localhost"
GRPC_PORT="6533"
PROTO_DIR="./proto"
USER_SERVICE_PROTO="proto/smartticket/user.proto"
ROLE_PERMISSION_SERVICE_PROTO="proto/smartticket/role_permission.proto"

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
TEST_ROLE_NAME="Test Role via gRPC"
TEST_ROLE_DESCRIPTION="A test role created via gRPC E2E testing"
TEST_PERMISSION_IDS='["ticket:view", "ticket:create", "knowledge:view"]'
TEST_ASSIGN_REASON="Assigned via gRPC E2E test"

# Test counters
TOTAL_TESTS=13
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
        if echo "$response" | grep -q '"success":true\|"role_id"\|"roles"\|"permissions"\|"assignments"\|"users"'; then
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

# Step 2: Test all RolePermissionService interfaces
echo -e "\n${YELLOW}🔑 Step 2: Testing RolePermissionService interfaces...${NC}"

# Test 1: CreateRole
create_role_data="{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"name\": \"$TEST_ROLE_NAME\", \"description\": \"$TEST_ROLE_DESCRIPTION\", \"permission_ids\": $TEST_PERMISSION_IDS, \"is_active\": true}"

run_grpcurl "CreateRole" "$create_role_data" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/CreateRole"

# Extract created role ID for subsequent tests
CREATE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$ROLE_PERMISSION_SERVICE_PROTO" \
    -d "$create_role_data" \
    -rpc-header "authorization: Bearer $JWT_TOKEN" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.RolePermissionService/CreateRole 2>/dev/null)

if echo "$CREATE_RESPONSE" | grep -q '"id"'; then
    ROLE_ID=$(echo "$CREATE_RESPONSE" | jq -r '.role.id')
    echo "Created role ID: $ROLE_ID"
fi

# Test 2: GetRole
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "GetRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetRole"
else
    run_grpcurl "GetRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetRole" "false"
fi

# Test 3: UpdateRole
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "UpdateRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\", \"name\": \"Updated $TEST_ROLE_NAME\", \"description\": \"Updated description via gRPC test\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/UpdateRole"
else
    run_grpcurl "UpdateRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"name\": \"Updated Test Role\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/UpdateRole" "false"
fi

# Test 4: DeleteRole
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "DeleteRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\", \"force_delete\": false}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/DeleteRole"
else
    run_grpcurl "DeleteRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/DeleteRole" "false"
fi

# Test 5: ListRoles
run_grpcurl "ListRoles" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/ListRoles"

# Test 6: ListPermissions
run_grpcurl "ListPermissions" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/ListPermissions"

# Test 7: GetRolePermissions
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "GetRolePermissions" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetRolePermissions"
else
    run_grpcurl "GetRolePermissions" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetRolePermissions" "false"
fi

# Test 8: AssignPermissionsToRole
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "AssignPermissionsToRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\", \"permission_ids\": [\"user:view\", \"user:create\"], \"reason\": \"Added permissions via gRPC test\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/AssignPermissionsToRole"
else
    run_grpcurl "AssignPermissionsToRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"permission_ids\": [\"user:view\"]}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/AssignPermissionsToRole" "false"
fi

# Test 9: RemovePermissionsFromRole
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "RemovePermissionsFromRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\", \"permission_ids\": [\"user:create\"], \"reason\": \"Removed permissions via gRPC test\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/RemovePermissionsFromRole"
else
    run_grpcurl "RemovePermissionsFromRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"permission_ids\": [\"user:create\"]}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/RemovePermissionsFromRole" "false"
fi

# Test 10: AssignRoleToUser
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "AssignRoleToUser" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"user_id\": \"$USER_ID\", \"role_id\": \"$ROLE_ID\", \"reason\": \"$TEST_ASSIGN_REASON\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/AssignRoleToUser"
else
    run_grpcurl "AssignRoleToUser" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"user_id\": \"$USER_ID\", \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/AssignRoleToUser" "false"
fi

# Test 11: RemoveRoleFromUser
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "RemoveRoleFromUser" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"user_id\": \"$USER_ID\", \"role_id\": \"$ROLE_ID\", \"reason\": \"Removed via gRPC test\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/RemoveRoleFromUser"
else
    run_grpcurl "RemoveRoleFromUser" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"user_id\": \"$USER_ID\", \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/RemoveRoleFromUser" "false"
fi

# Test 12: GetUserRoles
run_grpcurl "GetUserRoles" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"user_id\": \"$USER_ID\"}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetUserRoles"

# Test 13: GetUsersWithRole
if [ ! -z "$ROLE_ID" ]; then
    run_grpcurl "GetUsersWithRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"$ROLE_ID\", \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetUsersWithRole"
else
    run_grpcurl "GetUsersWithRole" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"role_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"pagination\": {\"page_size\": 10}}" "$ROLE_PERMISSION_SERVICE_PROTO" "smartticket.v1.RolePermissionService/GetUsersWithRole" "false"
fi

# Step 3: Test Results
echo -e "\n${YELLOW}📊 Step 3: RolePermissionService Test Results${NC}"
echo "=================================================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "Pass Rate: ${GREEN}$PASS_RATE%${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All RolePermissionService tests passed!${NC}"
else
    echo -e "\n${YELLOW}⚠️ Some RolePermissionService tests failed. This may be expected due to permission restrictions or backend issues.${NC}"
fi

echo -e "\n${BLUE}RolePermissionService E2E Test Complete!${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -gt 0 ]; then
    exit 1
else
    exit 0
fi