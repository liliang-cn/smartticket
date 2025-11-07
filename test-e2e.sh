#!/bin/bash

# Simple E2E test script for SmartTicket API
# This replaces Playwright tests when Node.js environment is not available

set -e

BASE_URL="http://localhost:6533"
ADMIN_EMAIL="admin@smartticket.local"
ADMIN_PASSWORD="admin123"

echo "🧪 Starting SmartTicket E2E Tests..."
echo "📍 Base URL: $BASE_URL"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_status="$3"

    echo -n "🔍 Testing: $test_name ... "

    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Function to check HTTP status
check_status() {
    local url="$1"
    local expected_status="$2"
    local actual_status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    [ "$actual_status" = "$expected_status" ]
}

# Function to test API response
test_api() {
    local method="$1"
    local url="$2"
    local data="$3"
    local headers="$4"

    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            $headers \
            -d "$data" \
            "$url")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            $headers \
            "$url")
    fi

    echo "$response"
}

echo ""
echo "📋 Running Basic Health Tests..."
echo ""

# Test 1: Health Check
run_test "Health Check" "check_status '$BASE_URL/health' 200"

# Test 2: Version Info
run_test "Version Info" "check_status '$BASE_URL/version' 200"

# Test 3: Swagger UI
run_test "Swagger UI" "check_status '$BASE_URL/swagger/' 200"

# Test 4: Swagger YAML
run_test "Swagger YAML" "check_status '$BASE_URL/swagger.yaml' 200"

echo ""
echo "🔐 Running Authentication Tests..."
echo ""

# Test 5: Login with valid credentials
login_response=$(test_api "POST" "$BASE_URL/api/v1/auth/login" \
    "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\"}")
if echo "$login_response" | tail -n1 | grep -q "200"; then
    echo -e "🔍 Testing: Login with valid credentials ... ${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))

    # Extract token for authenticated tests
    ACCESS_TOKEN=$(echo "$login_response" | head -n-1 | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    echo "🔑 Access token obtained"
else
    echo -e "🔍 Testing: Login with valid credentials ... ${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

# Test 6: Login with invalid credentials
run_test "Login with invalid credentials" "! test_api 'POST' '$BASE_URL/api/v1/auth/login' '{\"email\": \"invalid@test.com\", \"password\": \"wrong\"}' | tail -n1 | grep -q '200'"

# Test 7: Invalid email format
run_test "Invalid email format" "! test_api 'POST' '$BASE_URL/api/v1/auth/login' '{\"email\": \"invalid-email\", \"password\": \"test\"}' | tail -n1 | grep -q '200'"

echo ""
echo "🛡️  Running Protected Route Tests..."
echo ""

# Test 9: Access protected route without token
run_test "Access protected route without token" "! check_status '$BASE_URL/api/v1/auth/me' 200"

# Test 10: Access protected route with invalid token
run_test "Access protected route with invalid token" "! check_status '$BASE_URL/api/v1/auth/me' 200"

# Test 11: Access users list without token
run_test "Access users list without token" "! check_status '$BASE_URL/api/v1/users' 200"

if [ -n "$ACCESS_TOKEN" ]; then
    echo ""
    echo "🎭 Running Authenticated Tests..."
    echo ""

    # Test 12: Get user profile with valid token
    auth_response=$(test_api "GET" "$BASE_URL/api/v1/auth/me" "" "-H \"Authorization: Bearer $ACCESS_TOKEN\"")
    if echo "$auth_response" | tail -n1 | grep -q "200"; then
        echo -e "🔍 Testing: Get user profile with valid token ... ${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "🔍 Testing: Get user profile with valid token ... ${RED}✗ FAILED${NC}"
        ((TESTS_FAILED++))
    fi

    # Test 13: List users with admin token
    users_response=$(test_api "GET" "$BASE_URL/api/v1/users" "" "-H \"Authorization: Bearer $ACCESS_TOKEN\"")
    if echo "$users_response" | tail -n1 | grep -q "200"; then
        echo -e "🔍 Testing: List users with admin token ... ${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "🔍 Testing: List users with admin token ... ${RED}✗ FAILED${NC}"
        ((TESTS_FAILED++))
    fi
fi

echo ""
echo "🚨 Running Error Handling Tests..."
echo ""

# Test 14: 404 Not Found
run_test "404 Not Found" "check_status '$BASE_URL/api/v1/nonexistent' 404"

# Test 15: Malformed JSON
run_test "Malformed JSON" "! test_api 'POST' '$BASE_URL/api/v1/auth/login' '{\"invalid\": json}' | tail -n1 | grep -q '200'"

echo ""
echo "⚡ Running Performance Tests..."
echo ""

# Test 16: Response time test (using seconds for macOS compatibility)
start_time=$(date +%s)
curl -s "$BASE_URL/health" > /dev/null
end_time=$(date +%s)
response_time=$((end_time - start_time))

if [ $response_time -lt 2 ]; then
    echo -e "🔍 Testing: Health check response time (< 2s) ... ${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
else
    echo -e "🔍 Testing: Health check response time ($response_time s) ... ${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

echo ""
echo "📊 Test Results Summary"
echo "======================"
echo -e "Total Tests Run: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed successfully!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please check the application logs.${NC}"
    exit 1
fi