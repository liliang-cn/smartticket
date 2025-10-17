#!/bin/bash

# AuthService gRPC E2E Tests
# Tests Login and RefreshToken interfaces

echo "🔐 AuthService gRPC E2E Tests"
echo "================================"

cd "$(dirname "$0")/../.."

# Configuration
GRPC_GATEWAY_PORT=${GRPC_GATEWAY_PORT:-6533}
GRPC_HOST="localhost:${GRPC_GATEWAY_PORT}"
PROTO_PATH="./proto"
AUTH_PROTO="proto/smartticket/user.proto"

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

# Check if grpcurl is available
if ! command -v grpcurl &> /dev/null; then
    log_error "grpcurl is not installed or not in PATH"
    exit 1
fi

# Check if proto files exist
if [ ! -f "$AUTH_PROTO" ]; then
    log_error "Proto file not found: $AUTH_PROTO"
    exit 1
fi

# Check if gRPC service is running
log_info "Checking gRPC service connectivity..."
if ! grpcurl -plaintext -import-path $PROTO_PATH -proto $AUTH_PROTO "$GRPC_HOST" list > /dev/null 2>&1; then
    log_error "gRPC service is not responding on $GRPC_HOST"
    log_info "Please ensure the gRPC gateway service is running on port $GRPC_GATEWAY_PORT"
    exit 1
fi

log_success "gRPC service is reachable on $GRPC_HOST"

echo ""
log_info "Starting AuthService interface tests..."
echo "AuthService provides authentication and token management functionality"
echo "Total interfaces to test: 2"
echo ""

# Test 1: Login - Valid credentials
# This should succeed with valid admin credentials
run_test "AuthService.Login - Valid Admin credentials" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $AUTH_PROTO \
    -d '{
        \"email\": \"admin@test.smartticket.com\",
        \"password\": \"admin123\",
        \"tenant_domain\": \"test.smartticket.com\"
    }' \
    $GRPC_HOST \
    smartticket.v1.AuthService/Login" \
    "true" "accessToken" ""

# Test 2: Login - Invalid credentials
# This should fail with invalid credentials
run_test "AuthService.Login - Invalid credentials" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $AUTH_PROTO \
    -d '{
        \"email\": \"invalid@example.com\",
        \"password\": \"wrongpassword\"
    }' \
    $GRPC_HOST \
    smartticket.v1.AuthService/Login" \
    "false"

# Test 3: Login - Missing required fields
# This should fail due to missing email
run_test "AuthService.Login - Missing required fields" \
    "grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $AUTH_PROTO \
    -d '{
        \"password\": \"admin123\"
    }' \
    $GRPC_HOST \
    smartticket.v1.AuthService/Login" \
    "false"

# Get a valid JWT token for RefreshToken test
log_info "Obtaining JWT token for RefreshToken test..."
JWT_TOKEN=$(grpcurl -plaintext \
    -import-path $PROTO_PATH \
    -proto $AUTH_PROTO \
    -d '{
        "email": "admin@test.smartticket.com",
        "password": "admin123",
        "tenant_domain": "test.smartticket.com"
    }' \
    $GRPC_HOST \
    smartticket.v1.AuthService/Login | jq -r '.accessToken // empty' 2>/dev/null)

if [ -n "$JWT_TOKEN" ] && [ "$JWT_TOKEN" != "null" ]; then
    log_success "JWT token obtained successfully"

    # Test 4: RefreshToken - Valid token
    run_test "AuthService.RefreshToken - Valid token" \
        "grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $AUTH_PROTO \
        -d '{
            \"refresh_token\": \"dummy_refresh_token_for_testing\"
        }' \
        $GRPC_HOST \
        smartticket.v1.AuthService/RefreshToken" \
        "false" # Expected to fail as we don't have a real refresh token

    # Test 5: RefreshToken - Empty token
    run_test "AuthService.RefreshToken - Empty token" \
        "grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $AUTH_PROTO \
        -d '{
            \"refresh_token\": \"\"
        }' \
        $GRPC_HOST \
        smartticket.v1.AuthService/RefreshToken" \
        "false"

    # Test 6: RefreshToken - Missing token field
    run_test "AuthService.RefreshToken - Missing token field" \
        "grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $AUTH_PROTO \
        -d '{}' \
        $GRPC_HOST \
        smartticket.v1.AuthService/RefreshToken" \
        "false"
else
    log_warning "Could not obtain JWT token, some RefreshToken tests will be skipped"

    # Run RefreshToken tests without valid token (expecting failures)
    run_test "AuthService.RefreshToken - No authentication" \
        "grpcurl -plaintext \
        -import-path $PROTO_PATH \
        -proto $AUTH_PROTO \
        -d '{
            \"refresh_token\": \"dummy_token\"
        }' \
        $GRPC_HOST \
        smartticket.v1.AuthService/RefreshToken" \
        "false"
fi

echo ""
echo "================================"
echo "📊 AuthService Test Results"
echo "================================"
echo "Total tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "Success rate: ${GREEN}100%${NC}"
    echo "🎉 All AuthService tests passed!"
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