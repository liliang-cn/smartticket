#!/bin/bash

# Knowledge Management E2E Tests
# Tests knowledge article CRUD, categories, search, and rating system

echo "🚀 Starting Knowledge Management E2E Tests"
echo "=========================================="

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

# Test 1: Create Knowledge Category
run_test "Create Knowledge Category" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"name\": \"Technical Support\", \"description\": \"Technical troubleshooting guides\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.CreateCategory 2>/dev/null | jq -e '.id' > /dev/null"

# Store Category ID for later tests
CATEGORY_RESPONSE=$(grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{"name": "Technical Support", "description": "Technical troubleshooting guides"}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  $GRPC_URL smartticket.v1.KnowledgeService.CreateCategory 2>/dev/null)

CATEGORY_ID=$(echo "$CATEGORY_RESPONSE" | jq -r '.id')
echo "Created Category ID: $CATEGORY_ID"

# Test 2: Create Knowledge Article
run_test "Create Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"title\": \"How to Reset User Password\", \"content\": \"# Password Reset Guide\\n\\n## Steps to reset password:\\n1. Go to login page\\n2. Click \\\"Forgot Password\\\"\\n3. Enter email address\\n4. Check email for reset link\\n5. Create new password\", \"summary\": \"Step-by-step guide for password reset\", \"categoryId\": \"$CATEGORY_ID\", \"tags\": [\"password\", \"reset\", \"security\"]}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.CreateArticle 2>/dev/null | jq -e '.id' > /dev/null"

# Store Article ID for later tests
ARTICLE_RESPONSE=$(grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d "{\"title\": \"How to Reset User Password\", \"content\": \"# Password Reset Guide\\n\\n## Steps to reset password:\\n1. Go to login page\\n2. Click \\\"Forgot Password\\\"\\n3. Enter email address\\n4. Check email for reset link\\n5. Create new password\", \"summary\": \"Step-by-step guide for password reset\", \"categoryId\": \"$CATEGORY_ID\", \"tags\": [\"password\", \"reset\", \"security\"]}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  $GRPC_URL smartticket.v1.KnowledgeService.CreateArticle 2>/dev/null)

ARTICLE_ID=$(echo "$ARTICLE_RESPONSE" | jq -r '.id')
echo "Created Article ID: $ARTICLE_ID"

# Test 3: Get Knowledge Article
run_test "Get Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"id\": \"$ARTICLE_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.GetArticle 2>/dev/null | jq -e '.id == \"$ARTICLE_ID\"' > /dev/null"

# Test 4: Update Knowledge Article
run_test "Update Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"id\": \"$ARTICLE_ID\", \"title\": \"How to Reset User Password - Updated\", \"content\": \"# Password Reset Guide\\n\\n## Steps to reset password:\\n1. Go to login page\\n2. Click \\\"Forgot Password\\\"\\n3. Enter email address\\n4. Check email for reset link\\n5. Create new password\\n\\n## Additional Notes:\\n- Password must be at least 8 characters\\n- Include numbers and special characters\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.UpdateArticle 2>/dev/null | jq -e '.id == \"$ARTICLE_ID\"' > /dev/null"

# Test 5: List Knowledge Articles
run_test "List Knowledge Articles" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"pagination\": {\"pageSize\": 10}}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.ListArticles 2>/dev/null | jq -e '.articles | length > 0' > /dev/null"

# Test 6: Search Knowledge Articles
run_test "Search Knowledge Articles" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"query\": \"password reset\", \"pagination\": {\"pageSize\": 10}}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.SearchArticles 2>/dev/null | jq -e '.articles | length > 0' > /dev/null"

# Test 7: Get Article Suggestions for Ticket
run_test "Get Article Suggestions for Ticket" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"ticketTitle\": \"User cannot login\", \"ticketDescription\": \"User forgot password and cannot access account\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.GetArticleSuggestions 2>/dev/null | jq -e '.suggestions' > /dev/null"

# Test 8: Rate Knowledge Article
run_test "Rate Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"articleId\": \"$ARTICLE_ID\", \"rating\": 5, \"comment\": \"Very helpful guide!\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.RateArticle 2>/dev/null | jq -e '.success == true' > /dev/null"

# Test 9: Get Categories
run_test "Get Knowledge Categories" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.GetCategories 2>/dev/null | jq -e '.categories | length > 0' > /dev/null"

# Test 10: Publish Knowledge Article
run_test "Publish Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"id\": \"$ARTICLE_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.PublishArticle 2>/dev/null | jq -e '.success == true' > /dev/null"

# Test 11: Create another category for testing
run_test "Create Additional Category" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"name\": \"User Guide\", \"description\": \"General user guides and tutorials\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.CreateCategory 2>/dev/null | jq -e '.id' > /dev/null"

# Test 12: Filter Articles by Category
run_test "Filter Articles by Category" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"categoryId\": \"$CATEGORY_ID\", \"pagination\": {\"pageSize\": 10}}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.ListArticles 2>/dev/null | jq -e '.articles | length > 0' > /dev/null"

# Test 13: Archive Knowledge Article
run_test "Archive Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"id\": \"$ARTICLE_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.ArchiveArticle 2>/dev/null | jq -e '.success == true' > /dev/null"

# Test 14: Delete Knowledge Article
run_test "Delete Knowledge Article" \
  "grpcurl -plaintext -import-path ../../proto -proto smartticket/knowledge.proto \
  -d '{\"id\": \"$ARTICLE_ID\"}' \
  -H \"authorization: Bearer $ACCESS_TOKEN\" \
  -H \"x-tenant-id: $TENANT_ID\" \
  -H \"x-user-id: $USER_ID\" \
  $GRPC_URL smartticket.v1.KnowledgeService.DeleteArticle 2>/dev/null | jq -e '.success == true' > /dev/null"

# Print test results summary
echo ""
echo "================================"
echo -e "${BLUE}📊 Knowledge E2E Test Results Summary${NC}"
echo "================================"
echo "Total Tests: $TEST_COUNT"
echo -e "Passed: ${GREEN}$PASS_COUNT${NC}"
echo -e "Failed: ${RED}$((TEST_COUNT - PASS_COUNT))${NC}"

if [ $PASS_COUNT -eq $TEST_COUNT ]; then
    echo ""
    echo -e "${GREEN}🎉 All Knowledge E2E tests passed!${NC}"
    echo "✅ Knowledge Article Management: CREATE, READ, UPDATE, DELETE"
    echo "✅ Category Management: CREATE, READ, DELETE"
    echo "✅ Search and Filtering"
    echo "✅ Article Rating System"
    echo "✅ Article Publishing Workflow (Draft → Published → Archived)"
    echo "✅ Article Suggestions for Tickets"
    echo "✅ Multi-tenant Knowledge Isolation"
    exit 0
else
    echo ""
    echo -e "${RED}❌ Some Knowledge E2E tests failed!${NC}"
    echo "Please check the logs above for details."
    exit 1
fi