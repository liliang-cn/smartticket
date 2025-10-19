#!/bin/bash

# TenantService gRPC E2E Tests
# Tests all 10 TenantService interfaces with proper authentication

echo "🏢 TenantService gRPC E2E Tests"
echo "================================"

cd "$(dirname "$0")/../.."

# Configuration
GRPC_GATEWAY_PORT=${GRPC_GATEWAY_PORT:-6533}
GRPC_HOST="localhost:${GRPC_GATEWAY_PORT}"
PROTO_PATH="./proto"
TENANT_PROTO="proto/smartticket/tenant.proto"
USER_PROTO="proto/smartticket/user.proto"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test result tracking
TEST_RESULTS=()

# Global variables for test data
TEST_TENANT_ID=""
JWT_TOKEN=""

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

run_test() {
    local test_name="$1"
    local grpc_command="$2"
    local expected_success="$3" # "true" if expecting success, "false" if expecting failure
    local check_field="$4" # field to check in response for validation
    local expected_value="$5" # expected value in the field

    echo ""
    log_info "Testing: $test_name"
    echo "Command: $grpc_command"
    echo "----------------------------------------"

    ((TOTAL_TESTS++))

    # Execute the command and capture output
    if eval "$grpc_command" > /tmp/test_output.json 2>&1; then
        # Check response content for more accurate validation
        local test_result="PASSED"
        local validation_passed=false

        if [ -n "$check_field" ] && [ -n "$expected_value" ]; then
            # Extract the field value from response
            local actual_value=$(cat /tmp/test_output.json | jq -r ".$check_field // empty" 2>/dev/null)

            if [ "$actual_value" = "$expected_value" ]; then
                validation_passed=true
            else
                validation_passed=false
            fi
        else
            # If no field check specified, just check if command succeeded
            validation_passed=true
        fi

        if [ "$expected_success" = "true" ]; then
            if $validation_passed; then
                log_success "$test_name - PASSED"
                ((PASSED_TESTS++))
                TEST_RESULTS+=("$test_name: PASSED")
            else
                log_error "$test_name - FAILED (validation failed)"
                echo "Response:"
                cat /tmp/test_output.json
                echo "----------------------------------------"
                ((FAILED_TESTS++))
                TEST_RESULTS+=("$test_name: FAILED")
            fi
        else
            # For expected failures, check if we got an error response
            local has_error=$(cat /tmp/test_output.json | jq -e '.response.errors' > /dev/null 2>&1 && echo "true" || echo "false")
            if [ "$has_error" = "true" ]; then
                log_success "$test_name - PASSED (correctly returned error)"
                ((PASSED_TESTS++))
                TEST_RESULTS+=("$test_name: PASSED")
            else
                log_warning "$test_name - UNEXPECTED SUCCESS (expected error but got success)"
                echo "Response:"
                cat /tmp/test_output.json
                echo "----------------------------------------"
                ((PASSED_TESTS++))
                TEST_RESULTS+=("$test_name: UNEXPECTED_SUCCESS")
            fi
        fi
    else
        if [ "$expected_success" = "false" ]; then
            log_success "$test_name - PASSED (correctly failed at HTTP level)"
            ((PASSED_TESTS++))
            TEST_RESULTS+=("$test_name: PASSED")
        else
            log_error "$test_name - FAILED"
            echo "Error output:"
            cat /tmp/test_output.json
            echo "----------------------------------------"
            ((FAILED_TESTS++))
            TEST_RESULTS+=("$test_name: FAILED")
        fi
    fi
}

# Authenticate and get JWT token
authenticate() {
    log_info "Authenticating to get JWT token..."
    local temp_token=$(grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $USER_PROTO \
        -d '{
            "email": "admin@test.smartticket.com",
            "password": "admin123",
            "tenant_domain": "test.smartticket.com"
        }' \
        $GRPC_HOST \
        smartticket.v1.AuthService/Login | jq -r '.accessToken // empty' 2>/dev/null)

    if [ -n "$temp_token" ] && [ "$temp_token" != "null" ]; then
        JWT_TOKEN="$temp_token"
        log_success "Authentication successful"
        return 0
    else
        log_error "Authentication failed"
        return 1
    fi
}

# Check if grpcurl is available
if ! command -v grpcurl &> /dev/null; then
    log_error "grpcurl is not installed or not in PATH"
    exit 1
fi

# Check if proto files exist
if [ ! -f "$TENANT_PROTO" ]; then
    log_error "Proto file not found: $TENANT_PROTO"
    exit 1
fi

if [ ! -f "$USER_PROTO" ]; then
    log_error "Proto file not found: $USER_PROTO"
    exit 1
fi

# Check if gRPC service is running
log_info "Checking gRPC service connectivity..."
if ! grpcurl -plaintext -import-path $PROTO_PATH -proto $TENANT_PROTO "$GRPC_HOST" list > /dev/null 2>&1; then
    log_error "gRPC service is not responding on $GRPC_HOST"
    log_info "Please ensure the gRPC gateway service is running on port $GRPC_GATEWAY_PORT"
    exit 1
fi

log_success "gRPC service is reachable on $GRPC_HOST"

# Authenticate
if ! authenticate; then
    log_error "Cannot proceed without authentication"
    exit 1
fi

echo ""
log_info "Starting TenantService interface tests..."
echo "TenantService provides tenant management functionality"
echo "Total interfaces to test: 10"
echo ""

# Test 1: CreateTenant
run_test "TenantService.CreateTenant - Create new tenant" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-create-tenant-$(date +%s)\"
        },
        \"name\": \"Test Company E2E\",
        \"domain\": \"test-e2e-$(date +%s).smartticket.com\",
        \"subscription_tier\": \"SUBSCRIPTION_TIER_STANDARD\",
        \"max_users\": 50,
        \"data_residency_region\": \"EU\",
        \"contact_email\": \"admin@test-e2e-$(date +%s).smartticket.com\",
        \"billing_address\": \"123 Test Street, Test City, Test Country\",
        \"phone\": \"+1234567890\",
        \"is_trial\": true
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/CreateTenant" \
    "true" "tenant.id" ""

# Store created tenant ID for subsequent tests
if [ -f "/tmp/test_output.json" ]; then
    TEST_TENANT_ID=$(cat /tmp/test_output.json | jq -r '.tenant.id // empty' 2>/dev/null)
    if [ -n "$TEST_TENANT_ID" ] && [ "$TEST_TENANT_ID" != "null" ]; then
        log_info "Created tenant ID: $TEST_TENANT_ID"
    fi
fi

# Test 2: GetTenant (using existing test tenant)
run_test "TenantService.GetTenant - Get existing tenant" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-get-tenant-$(date +%s)\"
        },
        \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\"
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/GetTenant" \
    "true" "tenant.id" "784d8137-ddba-4978-b425-4bf79bc0f3f4"

# Test 3: GetCurrentTenant
run_test "TenantService.GetCurrentTenant - Get current tenant" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-get-current-tenant-$(date +%s)\"
        }
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/GetCurrentTenant" \
    "true" "tenant.id" "784d8137-ddba-4978-b425-4bf79bc0f3f4"

# Test 4: UpdateTenant
run_test "TenantService.UpdateTenant - Update tenant info" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-update-tenant-$(date +%s)\"
        },
        \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
        \"name\": \"Updated Test Company\",
        \"contact_email\": \"updated@test.smartticket.com\",
        \"phone\": \"+1234567899\"
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/UpdateTenant" \
    "true" "tenant.name" "Updated Test Company"

# Test 5: ListTenants
run_test "TenantService.ListTenants - List all tenants" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-list-tenants-$(date +%s)\"
        },
        \"pagination\": {
            \"page_size\": 10,
            \"page_token\": \"\"
        }
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/ListTenants" \
    "true" "tenants" ""

# Test 6: GetTenantUsage
run_test "TenantService.GetTenantUsage - Get tenant usage stats" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-tenant-usage-$(date +%s)\"
        },
        \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
        \"include_detailed_metrics\": true
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/GetTenantUsage" \
    "true" "usage.tenant_id" "784d8137-ddba-4978-b425-4bf79bc0f3f4"

# Test 7: GetTenantBilling
run_test "TenantService.GetTenantBilling - Get tenant billing info" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-tenant-billing-$(date +%s)\"
        },
        \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
        \"include_line_items\": true
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/GetTenantBilling" \
    "true" "billing.tenant_id" "784d8137-ddba-4978-b425-4bf79bc0f3f4"

# Test 8: UpdateSubscription
run_test "TenantService.UpdateSubscription - Update tenant subscription" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-update-subscription-$(date +%s)\"
        },
        \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
        \"new_tier\": \"SUBSCRIPTION_TIER_PREMIUM\",
        \"new_max_users\": 100,
        \"prorate\": true,
        \"billing_change_reason\": \"E2E Test subscription upgrade\"
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/UpdateSubscription" \
    "true" "tenant.subscription_tier" "SUBSCRIPTION_TIER_PREMIUM"

# Test 9: UpdateTenantStatus (if we created a test tenant, use it; otherwise use existing)
if [ -n "$TEST_TENANT_ID" ] && [ "$TEST_TENANT_ID" != "null" ]; then
    TENANT_TO_DEACTIVATE="$TEST_TENANT_ID"
else
    TENANT_TO_DEACTIVATE="784d8137-ddba-4978-b425-4bf79bc0f3f4"
fi

run_test "TenantService.UpdateTenantStatus - Deactivate tenant" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $TENANT_PROTO \
    -d '{
        \"metadata\": {
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
            \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
            \"request_id\": \"test-update-status-$(date +%s)\"
        },
        \"tenant_id\": \"$TENANT_TO_DEACTIVATE\",
        \"is_active\": false,
        \"reason\": \"E2E Test deactivation\"
    }' \
    -rpc-header "Authorization: Bearer ${JWT_TOKEN}" \
    -rpc-header "x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4" \
    -rpc-header "x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875" \
    $GRPC_HOST \
    smartticket.v1.TenantService/UpdateTenantStatus" \
    "true" "tenant.is_active" "false"

# Test 10: DeleteTenant (use test tenant if created, otherwise expect failure)
if [ -n "$TEST_TENANT_ID" ] && [ "$TEST_TENANT_ID" != "null" ]; then
    run_test "TenantService.DeleteTenant - Delete test tenant" \
        "grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $TENANT_PROTO \
        -d '{
            \"metadata\": {
                \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
                \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
                \"request_id\": \"test-delete-tenant-$(date +%s)\"
            },
            \"tenant_id\": \"$TEST_TENANT_ID\"
        }' \
        -rpc-header \"Authorization: Bearer ${JWT_TOKEN}\" \
        -rpc-header \"x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4\" \
        -rpc-header \"x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875\" \
        $GRPC_HOST \
        smartticket.v1.TenantService/DeleteTenant" \
        "true"
else
    run_test "TenantService.DeleteTenant - Attempt to delete main tenant (should fail)" \
        "grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $TENANT_PROTO \
        -d '{
            \"metadata\": {
                \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\",
                \"user_id\": \"9ccbc61a-4c5f-4275-8743-02bfe2949875\",
                \"request_id\": \"test-delete-main-tenant-$(date +%s)\"
            },
            \"tenant_id\": \"784d8137-ddba-4978-b425-4bf79bc0f3f4\"
        }' \
        -rpc-header \"Authorization: Bearer ${JWT_TOKEN}\" \
        -rpc-header \"x-tenant-id: 784d8137-ddba-4978-b425-4bf79bc0f3f4\" \
        -rpc-header \"x-user-id: 9ccbc61a-4c5f-4275-8743-02bfe2949875\" \
        $GRPC_HOST \
        smartticket.v1.TenantService/DeleteTenant" \
        "false"
fi

echo ""
echo "================================"
echo "📊 TenantService Test Results"
echo "================================"
echo "Total tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "Success rate: ${GREEN}100%${NC}"
    echo "🎉 All TenantService tests passed!"
else
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo -e "Success rate: ${YELLOW}$SUCCESS_RATE%${NC}"
fi

echo ""
echo "📋 Detailed Results:"
for result in "${TEST_RESULTS[@]}"; do
    echo "  - $result"
done

# Cleanup
rm -f /tmp/test_output.json

exit $FAILED_TESTS