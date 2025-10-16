#!/bin/bash

set -e

echo "🚀 Simple E2E Test for SmartTicket"
echo "================================="

GATEWAY_URL="localhost:6533"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check gateway
if ! curl -s http://localhost:7218/health > /dev/null; then
    log_error "Gateway not running"
    exit 1
fi
log_success "Gateway is running"

# Login
log_info "Logging in..."
LOGIN_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/user.proto \
    -d '{"email": "admin@test.smartticket.com", "password": "admin123", "tenant_domain": "test.smartticket.com"}' \
    $GATEWAY_URL smartticket.v1.AuthService/Login)

JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.accessToken')
TENANT_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user.tenantId')
USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user.id')

if [ -z "$JWT_TOKEN" ] || [ "$JWT_TOKEN" = "null" ]; then
    log_error "Login failed"
    exit 1
fi
log_success "Login successful"

# Test 1: Create SLA Policy
log_info "Test 1: Creating SLA Policy..."
CREATE_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -H "x-tenant-id: $TENANT_ID" \
    -H "x-user-id: $USER_ID" \
    -d '{
        "metadata": {"request_id": "test-001", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "name": "Test SLA Policy",
        "description": "Test SLA policy",
        "priority": 2,
        "severity": 2,
        "response_time_minutes": 60,
        "resolution_time_minutes": 240,
        "business_hours_only": false,
        "timezone": "UTC"
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/CreateSlaPolicy)

echo "$CREATE_SLA_RESPONSE"
SLA_ID=$(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.id // empty')

if [ -n "$SLA_ID" ] && [ "$SLA_ID" != "null" ]; then
    log_success "✅ SLA Create: PASSED (ID: $SLA_ID)"
else
    log_error "❌ SLA Create: FAILED"
    exit 1
fi

# Test 2: Get SLA Policy
log_info "Test 2: Getting SLA Policy..."
GET_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-002", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "policy_id": "'$SLA_ID'"
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/GetSlaPolicy)

SLA_NAME=$(echo "$GET_SLA_RESPONSE" | jq -r '.policy.name // empty')
if [ -n "$SLA_NAME" ] && [ "$SLA_NAME" != "null" ]; then
    log_success "✅ SLA Get: PASSED ($SLA_NAME)"
else
    log_error "❌ SLA Get: FAILED"
    exit 1
fi

# Test 3: List SLA Policies
log_info "Test 3: Listing SLA Policies..."
LIST_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-003", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "pagination": {"page_size": 10},
        "is_active": true
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/ListSlaPolicies)

POLICIES_COUNT=$(echo "$LIST_SLA_RESPONSE" | jq '.policies | length // 0')
if [ "$POLICIES_COUNT" -gt 0 ]; then
    log_success "✅ SLA List: PASSED ($POLICIES_COUNT policies)"
else
    log_error "❌ SLA List: FAILED"
    exit 1
fi

# Test 4: Create Knowledge Category
log_info "Test 4: Creating Knowledge Category..."
CREATE_CATEGORY_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-004", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "name": "Test Category",
        "description": "Test category",
        "parent_id": ""
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/CreateCategory)

CATEGORY_ID=$(echo "$CREATE_CATEGORY_RESPONSE" | jq -r '.category.id // empty')
if [ -n "$CATEGORY_ID" ] && [ "$CATEGORY_ID" != "null" ]; then
    log_success "✅ Knowledge Create Category: PASSED (ID: $CATEGORY_ID)"
else
    log_error "❌ Knowledge Create Category: FAILED"
    exit 1
fi

# Test 5: Create Knowledge Article
log_info "Test 5: Creating Knowledge Article..."
CREATE_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-005", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "title": "Test Article",
        "content": "This is a test article for E2E testing.\n\n## Section 1\n\nTest content here.\n\n## Section 2\n\nMore test content.",
        "summary": "Test article summary",
        "category_id": "'$CATEGORY_ID'",
        "tags": ["test", "e2e"],
        "visibility": 1
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/CreateArticle)

ARTICLE_ID=$(echo "$CREATE_ARTICLE_RESPONSE" | jq -r '.article.id // empty')
if [ -n "$ARTICLE_ID" ] && [ "$ARTICLE_ID" != "null" ]; then
    log_success "✅ Knowledge Create Article: PASSED (ID: $ARTICLE_ID)"
else
    log_error "❌ Knowledge Create Article: FAILED"
    exit 1
fi

# Test 6: Get Knowledge Article
log_info "Test 6: Getting Knowledge Article..."
GET_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-006", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "article_id": "'$ARTICLE_ID'"
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/GetArticle)

ARTICLE_TITLE=$(echo "$GET_ARTICLE_RESPONSE" | jq -r '.article.title // empty')
if [ -n "$ARTICLE_TITLE" ] && [ "$ARTICLE_TITLE" != "null" ]; then
    log_success "✅ Knowledge Get Article: PASSED ($ARTICLE_TITLE)"
else
    log_error "❌ Knowledge Get Article: FAILED"
    exit 1
fi

# Test 7: Publish Knowledge Article
log_info "Test 7: Publishing Knowledge Article..."
PUBLISH_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-007", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "article_id": "'$ARTICLE_ID'"
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/PublishArticle)

PUBLISH_STATUS=$(echo "$PUBLISH_ARTICLE_RESPONSE" | jq -r '.article.status // empty')
if [ -n "$PUBLISH_STATUS" ] && [ "$PUBLISH_STATUS" != "null" ]; then
    log_success "✅ Knowledge Publish Article: PASSED (Status: $PUBLISH_STATUS)"
else
    log_error "❌ Knowledge Publish Article: FAILED"
    exit 1
fi

# Test 8: Search Knowledge Articles
log_info "Test 8: Searching Knowledge Articles..."
SEARCH_ARTICLES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-008", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "query": "test",
        "pagination": {"page_size": 10}
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/SearchArticles)

SEARCH_COUNT=$(echo "$SEARCH_ARTICLES_RESPONSE" | jq '.articles | length // 0')
if [ "$SEARCH_COUNT" -gt 0 ]; then
    log_success "✅ Knowledge Search: PASSED ($SEARCH_COUNT results)"
else
    log_error "❌ Knowledge Search: FAILED"
    exit 1
fi

# Test 9: List Knowledge Articles
log_info "Test 9: Listing Knowledge Articles..."
LIST_ARTICLES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-009", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"},
        "pagination": {"page_size": 10},
        "status": 2
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/ListArticles)

LIST_COUNT=$(echo "$LIST_ARTICLES_RESPONSE" | jq '.articles | length // 0')
if [ "$LIST_COUNT" -gt 0 ]; then
    log_success "✅ Knowledge List: PASSED ($LIST_COUNT articles)"
else
    log_error "❌ Knowledge List: FAILED"
    exit 1
fi

# Test 10: Get Knowledge Categories
log_info "Test 10: Getting Knowledge Categories..."
GET_CATEGORIES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "metadata": {"request_id": "test-010", "user_id": "'$USER_ID'", "tenant_id": "'$TENANT_ID'"}
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/GetCategories)

CATEGORIES_COUNT=$(echo "$GET_CATEGORIES_RESPONSE" | jq '.categories | length // 0')
if [ "$CATEGORIES_COUNT" -gt 0 ]; then
    log_success "✅ Knowledge Get Categories: PASSED ($CATEGORIES_COUNT categories)"
else
    log_error "❌ Knowledge Get Categories: FAILED"
    exit 1
fi

echo ""
echo "🎉 ALL E2E TESTS PASSED! 🎉"
echo ""
echo "✅ Authentication: WORKING"
echo "✅ SLA Management: WORKING"
echo "✅ Knowledge Base: WORKING"
echo "✅ Search: WORKING"
echo "✅ Content Management: WORKING"
echo ""
echo "Test Data Created:"
echo "- SLA Policy ID: $SLA_ID"
echo "- Knowledge Category ID: $CATEGORY_ID"
echo "- Knowledge Article ID: $ARTICLE_ID"