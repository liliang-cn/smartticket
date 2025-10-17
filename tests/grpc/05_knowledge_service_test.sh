#!/bin/bash

# KnowledgeService gRPC E2E Test
# Tests all 12 interfaces of the KnowledgeService
# Uses grpcurl with real JWT authentication

echo "📚 KnowledgeService gRPC E2E Test (12 interfaces)"
echo "==============================================="

# Configuration
GRPC_HOST="localhost"
GRPC_PORT="6533"
PROTO_DIR="./proto"
USER_SERVICE_PROTO="proto/smartticket/user.proto"
KNOWLEDGE_SERVICE_PROTO="proto/smartticket/knowledge.proto"

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
TEST_ARTICLE_TITLE="Test Knowledge Article via gRPC"
TEST_ARTICLE_CONTENT="# Test Knowledge Article\n\nThis is a test knowledge article created via gRPC E2E testing.\n\n## Features\n- Real authentication\n- Proper metadata\n- Comprehensive testing"
TEST_ARTICLE_SUMMARY="A test knowledge article for gRPC E2E testing"
TEST_VISIBILITY="KNOWLEDGE_VISIBILITY_INTERNAL"
TEST_LANGUAGE="en"
TEST_TAGS='["gRPC", "Testing", "E2E", "SmartTicket"]'
TEST_CATEGORY_NAME="Test Category"

# Test counters
TOTAL_TESTS=12
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
        if echo "$response" | grep -q '"success":true\|"article_id"\|"articles"\|"total_count"\|"helpful_count"\|"categories"'; then
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

# Step 2: Create a test category first
echo -e "\n${YELLOW}📂 Step 2: Creating test category...${NC}"

category_response=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$KNOWLEDGE_SERVICE_PROTO" \
    -d "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"name\": \"$TEST_CATEGORY_NAME\", \"description\": \"Test category for gRPC E2E testing\"}" \
    -rpc-header "authorization: Bearer $JWT_TOKEN" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.KnowledgeService/CreateCategory 2>/dev/null)

if echo "$category_response" | grep -q '"id"'; then
    CATEGORY_ID=$(echo "$category_response" | jq -r '.category.id')
    echo -e "${GREEN}✅ Category created: $CATEGORY_ID${NC}"
else
    echo -e "${YELLOW}⚠️ Category creation failed, will test without category_id${NC}"
    CATEGORY_ID=""
fi

# Step 3: Test all KnowledgeService interfaces
echo -e "\n${YELLOW}📚 Step 3: Testing KnowledgeService interfaces...${NC}"

# Test 1: CreateArticle
create_article_data="{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"title\": \"$TEST_ARTICLE_TITLE\", \"content\": \"$TEST_ARTICLE_CONTENT\", \"summary\": \"$TEST_ARTICLE_SUMMARY\", \"visibility\": \"$TEST_VISIBILITY\", \"language\": \"$TEST_LANGUAGE\", \"tags\": $TEST_TAGS}"
if [ ! -z "$CATEGORY_ID" ]; then
    create_article_data="{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"title\": \"$TEST_ARTICLE_TITLE\", \"content\": \"$TEST_ARTICLE_CONTENT\", \"summary\": \"$TEST_ARTICLE_SUMMARY\", \"category_id\": \"$CATEGORY_ID\", \"visibility\": \"$TEST_VISIBILITY\", \"language\": \"$TEST_LANGUAGE\", \"tags\": $TEST_TAGS}"
fi

run_grpcurl "CreateArticle" "$create_article_data" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/CreateArticle"

# Extract created article ID for subsequent tests
CREATE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto "$KNOWLEDGE_SERVICE_PROTO" \
    -d "$create_article_data" \
    -rpc-header "authorization: Bearer $JWT_TOKEN" \
    "$GRPC_HOST:$GRPC_PORT" \
    smartticket.v1.KnowledgeService/CreateArticle 2>/dev/null)

if echo "$CREATE_RESPONSE" | grep -q '"id"'; then
    ARTICLE_ID=$(echo "$CREATE_RESPONSE" | jq -r '.article.id')
    echo "Created article ID: $ARTICLE_ID"
fi

# Test 2: GetArticle
if [ ! -z "$ARTICLE_ID" ]; then
    run_grpcurl "GetArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"$ARTICLE_ID\", \"increment_view_count\": true}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/GetArticle"
else
    run_grpcurl "GetArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/GetArticle" "false"
fi

# Test 3: UpdateArticle
if [ ! -z "$ARTICLE_ID" ]; then
    run_grpcurl "UpdateArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"$ARTICLE_ID\", \"title\": \"Updated $TEST_ARTICLE_TITLE\", \"summary\": \"Updated summary for gRPC testing\", \"comment\": \"Updated via gRPC E2E test\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/UpdateArticle"
else
    run_grpcurl "UpdateArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"title\": \"Updated Test Article\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/UpdateArticle" "false"
fi

# Test 4: ListArticles
run_grpcurl "ListArticles" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/ListArticles"

# Test 5: DeleteArticle (soft delete)
if [ ! -z "$ARTICLE_ID" ]; then
    run_grpcurl "DeleteArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"$ARTICLE_ID\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/DeleteArticle"
else
    run_grpcurl "DeleteArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/DeleteArticle" "false"
fi

# Test 6: PublishArticle
if [ ! -z "$ARTICLE_ID" ]; then
    run_grpcurl "PublishArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"$ARTICLE_ID\", \"comment\": \"Publishing via gRPC E2E test\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/PublishArticle"
else
    run_grpcurl "PublishArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/PublishArticle" "false"
fi

# Test 7: ArchiveArticle
if [ ! -z "$ARTICLE_ID" ]; then
    run_grpcurl "ArchiveArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"$ARTICLE_ID\", \"reason\": \"Archived via gRPC E2E test\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/ArchiveArticle"
else
    run_grpcurl "ArchiveArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"550e8400-e29b-41d4-a716-446655440000\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/ArchiveArticle" "false"
fi

# Test 8: SearchArticles
run_grpcurl "SearchArticles" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"query\": \"test\", \"pagination\": {\"page_size\": 10, \"page_token\": \"\"}}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/SearchArticles"

# Test 9: GetArticleSuggestions
run_grpcurl "GetArticleSuggestions" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"ticket_title\": \"Test Ticket\", \"ticket_description\": \"This is a test ticket for suggestions\", \"limit\": 5}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/GetArticleSuggestions"

# Test 10: RateArticle
if [ ! -z "$ARTICLE_ID" ]; then
    run_grpcurl "RateArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"$ARTICLE_ID\", \"is_helpful\": true, \"comment\": \"Helpful article via gRPC test\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/RateArticle"
else
    run_grpcurl "RateArticle" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"article_id\": \"550e8400-e29b-41d4-a716-446655440000\", \"is_helpful\": true}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/RateArticle" "false"
fi

# Test 11: GetCategories
run_grpcurl "GetCategories" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/GetCategories"

# Test 12: CreateCategory (additional test)
run_grpcurl "CreateCategory" "{\"metadata\": {\"tenant_id\": \"$TENANT_ID\", \"user_id\": \"$USER_ID\"}, \"name\": \"Additional Test Category\", \"description\": \"Additional test category via gRPC\", \"icon\": \"test\"}" "$KNOWLEDGE_SERVICE_PROTO" "smartticket.v1.KnowledgeService/CreateCategory"

# Step 4: Test Results
echo -e "\n${YELLOW}📊 Step 4: KnowledgeService Test Results${NC}"
echo "==============================================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "Pass Rate: ${GREEN}$PASS_RATE%${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All KnowledgeService tests passed!${NC}"
else
    echo -e "\n${YELLOW}⚠️ Some KnowledgeService tests failed. This may be expected due to permission restrictions or backend issues.${NC}"
fi

echo -e "\n${BLUE}KnowledgeService E2E Test Complete!${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -gt 0 ]; then
    exit 1
else
    exit 0
fi