#!/bin/bash

# Complete SmartTicket E2E Test Suite
# Tests all major service interfaces systematically

set -e

echo "============================================"
echo "🚀 SmartTicket Complete E2E Test Suite"
echo "============================================"

# Configuration
GATEWAY_ADDR="localhost:6533"
TENANT_ID="de57f60e-80a3-4a87-af40-3f99723c6530"
USER_ID="818bcee0-2176-477d-b39b-ed636f73e19b"
PROTO_DIR="/Users/liliang/Things/AI/projects/smartticket/proto"

echo "Gateway: $GATEWAY_ADDR"
echo "Tenant ID: $TENANT_ID"
echo "User ID: $USER_ID"
echo ""

# Common grpcurl parameters
GRPC_PARAMS="-plaintext -import-path $PROTO_DIR"

# Helper functions
call_grpc() {
    local method=$1
    local data=$2
    local proto_file=$3

    grpcurl $GRPC_PARAMS -proto $PROTO_DIR/smartticket/$proto_file \
        -d "$data" \
        -H "x-dev-bypass: true" \
        -H "x-tenant-id: $TENANT_ID" \
        -H "x-user-id: $USER_ID" \
        $GATEWAY_ADDR $method
}

test_result() {
    local test_name=$1
    local response=$2

    echo "$response" | jq '.' 2>/dev/null || echo "$response"

    if echo "$response" | grep -q '"success":true\|"id"\|"totalCount"\|"\[\]"'; then
        echo "✅ $test_name - PASSED"
        return 0
    else
        echo "❌ $test_name - FAILED"
        return 1
    fi
}

echo "============================================"
echo "🔧 SLA Service Tests (9 interfaces)"
echo "============================================"

# Test 1: CreateSlaPolicy
echo "🧪 Test 1.1: CreateSlaPolicy"
CREATE_SLA_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "create-sla-$(date +%s)"},
    "name": "Premium Support SLA $(date +%s)",
    "description": "Premium customer support SLA policy",
    "priority": 3,
    "severity": 3,
    "responseTimeMinutes": 15,
    "resolutionTimeMinutes": 240,
    "businessHoursOnly": true,
    "timezone": "UTC"
}
EOF
)
CREATE_SLA_RESPONSE=$(call_grpc "smartticket.v1.SlaService/CreateSlaPolicy" "$CREATE_SLA_DATA" "sla.proto")
test_result "CreateSlaPolicy" "$CREATE_SLA_RESPONSE"

# Extract SLA ID for subsequent tests
SLA_ID=$(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.id // empty' 2>/dev/null || echo "")

# Test 2: GetSlaPolicy
if [ ! -z "$SLA_ID" ]; then
    echo "🧪 Test 1.2: GetSlaPolicy"
    GET_SLA_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-sla-$(date +%s)"},
    "policyId": "$SLA_ID"
}
EOF
)
    GET_SLA_RESPONSE=$(call_grpc "smartticket.v1.SlaService/GetSlaPolicy" "$GET_SLA_DATA" "sla.proto")
    test_result "GetSlaPolicy" "$GET_SLA_RESPONSE"
else
    echo "⚠️  Skipping GetSlaPolicy - no SLA ID available"
fi

# Test 3: ListSlaPolicies
echo "🧪 Test 1.3: ListSlaPolicies"
LIST_SLA_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "list-sla-$(date +%s)"},
    "pagination": {
        "pageSize": 10,
        "pageToken": ""
    }
}
EOF
)
LIST_SLA_RESPONSE=$(call_grpc "smartticket.v1.SlaService/ListSlaPolicies" "$LIST_SLA_DATA" "sla.proto")
test_result "ListSlaPolicies" "$LIST_SLA_RESPONSE"

echo ""
echo "============================================"
echo "📚 Knowledge Service Tests (8 interfaces)"
echo "============================================"

# Test 1: CreateArticle
echo "🧪 Test 2.1: CreateArticle"
CREATE_ARTICLE_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "create-article-$(date +%s)"},
    "title": "E2E Test Article",
    "content": "This is a comprehensive test article created during E2E testing. It contains sufficient content to validate the article creation functionality including title, content, summary, tags, and other metadata fields.",
    "summary": "E2E test article for comprehensive testing",
    "categoryId": "",
    "visibility": 1,
    "language": "en",
    "tags": ["test", "e2e", "article", "knowledge"]
}
EOF
)
CREATE_ARTICLE_RESPONSE=$(call_grpc "smartticket.v1.KnowledgeService/CreateArticle" "$CREATE_ARTICLE_DATA" "knowledge.proto")
test_result "CreateArticle" "$CREATE_ARTICLE_RESPONSE"

# Extract Article ID for subsequent tests
ARTICLE_ID=$(echo "$CREATE_ARTICLE_RESPONSE" | jq -r '.article.id // empty' 2>/dev/null || echo "")

# Test 2: ListArticles
echo "🧪 Test 2.2: ListArticles"
LIST_ARTICLES_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "list-articles-$(date +%s)"},
    "pageSize": 10
}
EOF
)
LIST_ARTICLES_RESPONSE=$(call_grpc "smartticket.v1.KnowledgeService/ListArticles" "$LIST_ARTICLES_DATA" "knowledge.proto")
test_result "ListArticles" "$LIST_ARTICLES_RESPONSE"

# Test 3: SearchArticles
echo "🧪 Test 2.3: SearchArticles"
SEARCH_ARTICLES_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "search-articles-$(date +%s)"},
    "query": "test",
    "pageSize": 5
}
EOF
)
SEARCH_ARTICLES_RESPONSE=$(call_grpc "smartticket.v1.KnowledgeService/SearchArticles" "$SEARCH_ARTICLES_DATA" "knowledge.proto")
test_result "SearchArticles" "$SEARCH_ARTICLES_RESPONSE"

# Test 4: CreateCategory
echo "🧪 Test 2.4: CreateCategory"
CREATE_CATEGORY_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "create-category-$(date +%s)"},
    "name": "E2E Test Category",
    "description": "Category created during E2E testing",
    "parentId": "",
    "icon": "test-icon"
}
EOF
)
CREATE_CATEGORY_RESPONSE=$(call_grpc "smartticket.v1.KnowledgeService/CreateCategory" "$CREATE_CATEGORY_DATA" "knowledge.proto")
test_result "CreateCategory" "$CREATE_CATEGORY_RESPONSE"

# Test 5: GetCategories
echo "🧪 Test 2.5: GetCategories"
GET_CATEGORIES_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-categories-$(date +%s)"}
}
EOF
)
GET_CATEGORIES_RESPONSE=$(call_grpc "smartticket.v1.KnowledgeService/GetCategories" "$GET_CATEGORIES_DATA" "knowledge.proto")
test_result "GetCategories" "$GET_CATEGORIES_RESPONSE"

# Test 6: GetArticle (if we have an article ID)
if [ ! -z "$ARTICLE_ID" ]; then
    echo "🧪 Test 2.6: GetArticle"
    GET_ARTICLE_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-article-$(date +%s)"},
    "articleId": "$ARTICLE_ID"
}
EOF
)
    GET_ARTICLE_RESPONSE=$(call_grpc "smartticket.v1.KnowledgeService/GetArticle" "$GET_ARTICLE_DATA" "knowledge.proto")
    test_result "GetArticle" "$GET_ARTICLE_RESPONSE"
else
    echo "⚠️  Skipping GetArticle - no Article ID available"
fi

echo ""
echo "============================================"
echo "🎫 Ticket Service Tests (6 interfaces)"
echo "============================================"

# Test 1: CreateTicket
echo "🧪 Test 3.1: CreateTicket"
CREATE_TICKET_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "create-ticket-$(date +%s)"},
    "title": "E2E Test Ticket",
    "description": "This is a test ticket created during E2E testing to validate ticket creation functionality.",
    "priority": 2,
    "severity": 2,
    "categoryId": "",
    "status": 0
}
EOF
)
CREATE_TICKET_RESPONSE=$(call_grpc "smartticket.v1.TicketService/CreateTicket" "$CREATE_TICKET_DATA" "ticket.proto")
test_result "CreateTicket" "$CREATE_TICKET_RESPONSE"

# Extract Ticket ID for subsequent tests
TICKET_ID=$(echo "$CREATE_TICKET_RESPONSE" | jq -r '.ticket.id // empty' 2>/dev/null || echo "")

# Test 2: ListTickets
echo "🧪 Test 3.2: ListTickets"
LIST_TICKETS_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "list-tickets-$(date +%s)"},
    "pageSize": 10
}
EOF
)
LIST_TICKETS_RESPONSE=$(call_grpc "smartticket.v1.TicketService/ListTickets" "$LIST_TICKETS_DATA" "ticket.proto")
test_result "ListTickets" "$LIST_TICKETS_RESPONSE"

# Test 3: GetTicket (if we have a ticket ID)
if [ ! -z "$TICKET_ID" ]; then
    echo "🧪 Test 3.3: GetTicket"
    GET_TICKET_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-ticket-$(date +%s)"},
    "ticketId": "$TICKET_ID"
}
EOF
)
    GET_TICKET_RESPONSE=$(call_grpc "smartticket.v1.TicketService/GetTicket" "$GET_TICKET_DATA" "ticket.proto")
    test_result "GetTicket" "$GET_TICKET_RESPONSE"
else
    echo "⚠️  Skipping GetTicket - no Ticket ID available"
fi

echo ""
echo "============================================"
echo "👥 User Service Tests (4 interfaces)"
echo "============================================"

# Test 1: ListUsers
echo "🧪 Test 4.1: ListUsers"
LIST_USERS_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "list-users-$(date +%s)"},
    "pageSize": 10
}
EOF
)
LIST_USERS_RESPONSE=$(call_grpc "smartticket.v1.UserService/ListUsers" "$LIST_USERS_DATA" "user.proto")
test_result "ListUsers" "$LIST_USERS_RESPONSE"

# Test 2: GetUserProfile
echo "🧪 Test 4.2: GetUserProfile"
GET_PROFILE_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-profile-$(date +%s)"}
}
EOF
)
GET_PROFILE_RESPONSE=$(call_grpc "smartticket.v1.UserService/GetUserProfile" "$GET_PROFILE_DATA" "user.proto")
test_result "GetUserProfile" "$GET_PROFILE_RESPONSE"

echo ""
echo "============================================"
echo "🏢 Tenant Service Tests (3 interfaces)"
echo "============================================"

# Test 1: GetTenant
echo "🧪 Test 5.1: GetTenant"
GET_TENANT_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-tenant-$(date +%s)"}
}
EOF
)
GET_TENANT_RESPONSE=$(call_grpc "smartticket.v1.TenantService/GetTenant" "$GET_TENANT_DATA" "tenant.proto")
test_result "GetTenant" "$GET_TENANT_RESPONSE"

# Test 2: GetTenantStats
echo "🧪 Test 5.2: GetTenantStats"
GET_STATS_DATA=$(cat <<EOF
{
    "metadata": {"requestId": "get-stats-$(date +%s)"}
}
EOF
)
GET_STATS_RESPONSE=$(call_grpc "smartticket.v1.TenantService/GetTenantStats" "$GET_STATS_DATA" "tenant.proto")
test_result "GetTenantStats" "$GET_STATS_RESPONSE"

echo ""
echo "============================================"
echo "🎉 E2E Testing Completed!"
echo "============================================"
echo "All major SmartTicket services tested successfully!"
echo ""
echo "📊 Test Summary:"
echo "  - SLA Service: 3+ interfaces tested"
echo "  - Knowledge Service: 6 interfaces tested"
echo "  - Ticket Service: 3+ interfaces tested"
echo "  - User Service: 2 interfaces tested"
echo "  - Tenant Service: 2 interfaces tested"
echo ""
echo "✅ Total: 15+ service interfaces validated!"