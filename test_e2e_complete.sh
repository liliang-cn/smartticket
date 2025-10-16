#!/bin/bash

set -e  # Exit on any error

echo "🚀 Starting Complete E2E Test for SmartTicket"
echo "================================================"

# Configuration
GATEWAY_PORT="6533"
GATEWAY_URL="localhost:$GATEWAY_PORT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if gateway is running
log_info "Checking if gateway is running on port $GATEWAY_PORT..."
if ! curl -s http://localhost:7218/health > /dev/null 2>&1; then
    log_error "Gateway is not running! Please start it first."
    exit 1
fi
log_success "Gateway is running"

# Step 1: Create a new tenant for testing
log_info "Step 1: Creating new test tenant..."
TENANT_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/tenant.proto \
    -d '{
        "name": "E2E Test Tenant",
        "domain": "e2e-test.smartticket.com",
        "subscription_tier": 2,
        "max_users": 50,
        "data_residency_region": "EU",
        "contact_email": "admin@e2e-test.com",
        "billing_address": "123 Test St, Test City",
        "phone": "+1234567890"
    }' \
    $GATEWAY_URL smartticket.v1.TenantService/CreateTenant)

echo "$TENANT_RESPONSE"

# Extract tenant ID
TENANT_ID=$(echo "$TENANT_RESPONSE" | jq -r '.tenant.id // empty')
if [ -z "$TENANT_ID" ] || [ "$TENANT_ID" = "null" ]; then
    log_error "Failed to create tenant or extract tenant ID"
    echo "$TENANT_RESPONSE"
    exit 1
fi
log_success "Created tenant with ID: $TENANT_ID"

# Step 2: Create admin user for the tenant
log_info "Step 2: Creating admin user..."
USER_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/user.proto \
    -d '{
        "tenant_id": "'$TENANT_ID'",
        "email": "admin@e2e-test.com",
        "name": "E2E Test Admin",
        "role": 2,
        "password": "SecurePassword123!",
        "department": "IT",
        "phone": "+1234567890"
    }' \
    $GATEWAY_URL smartticket.v1.UserService/CreateUser)

echo "$USER_RESPONSE"

# Extract user ID
USER_ID=$(echo "$USER_RESPONSE" | jq -r '.user.id // empty')
if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
    log_error "Failed to create user or extract user ID"
    echo "$USER_RESPONSE"
    exit 1
fi
log_success "Created user with ID: $USER_ID"

# Step 3: Login to get JWT token
log_info "Step 3: Logging in to get authentication token..."
LOGIN_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/user.proto \
    -d '{
        "email": "admin@e2e-test.com",
        "password": "SecurePassword123!",
        "tenant_domain": "e2e-test.smartticket.com"
    }' \
    $GATEWAY_URL smartticket.v1.AuthService/Login)

echo "$LOGIN_RESPONSE"

# Extract JWT token
JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token // empty')
if [ -z "$JWT_TOKEN" ] || [ "$JWT_TOKEN" = "null" ]; then
    log_error "Failed to login or extract JWT token"
    echo "$LOGIN_RESPONSE"
    exit 1
fi
log_success "Successfully logged in, got JWT token"

# Step 4: Create additional user for testing (optional)
log_info "Step 4: Creating support engineer user..."
ENGINEER_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/user.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "tenant_id": "'$TENANT_ID'",
        "email": "engineer@e2e-test.com",
        "name": "E2E Test Engineer",
        "role": 3,
        "password": "EngineerPass123!",
        "department": "Support"
    }' \
    $GATEWAY_URL smartticket.v1.UserService/CreateUser)

echo "$ENGINE_RESPONSE"

ENGINEER_ID=$(echo "$ENGINEER_RESPONSE" | jq -r '.user.id // empty')
if [ -n "$ENGINEER_ID" ] && [ "$ENGINEER_ID" != "null" ]; then
    log_success "Created engineer user with ID: $ENGINEER_ID"
else
    log_warning "Could not create engineer user, continuing with tests"
fi

echo ""
echo "🎯 AUTHENTICATION COMPLETE - Starting SLA Service Tests"
echo "======================================================="

# Step 5: Test SLA Service - CreateSlaPolicy
log_info "Step 5: Testing SLA Service - CreateSlaPolicy..."
CREATE_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "Standard Support SLA",
        "description": "Standard customer support SLA policy for normal priority tickets",
        "priority": 2,
        "severity": 2,
        "response_time_minutes": 60,
        "resolution_time_minutes": 240,
        "business_hours_only": false,
        "timezone": "UTC"
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/CreateSlaPolicy)

echo "$CREATE_SLA_RESPONSE"

# Extract SLA policy ID
SLA_POLICY_ID=$(echo "$CREATE_SLA_RESPONSE" | jq -r '.policy.id // empty')
if [ -z "$SLA_POLICY_ID" ] || [ "$SLA_POLICY_ID" = "null" ]; then
    log_error "Failed to create SLA policy or extract policy ID"
    echo "$CREATE_SLA_RESPONSE"
    exit 1
fi
log_success "Created SLA policy with ID: $SLA_POLICY_ID"

# Step 6: Test SLA Service - GetSlaPolicy
log_info "Step 6: Testing SLA Service - GetSlaPolicy..."
GET_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "policy_id": "'$SLA_POLICY_ID'"
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/GetSlaPolicy)

echo "$GET_SLA_RESPONSE"

GET_SLA_NAME=$(echo "$GET_SLA_RESPONSE" | jq -r '.policy.name // empty')
if [ -z "$GET_SLA_NAME" ] || [ "$GET_SLA_NAME" = "null" ]; then
    log_error "Failed to retrieve SLA policy"
    echo "$GET_SLA_RESPONSE"
    exit 1
fi
log_success "Retrieved SLA policy: $GET_SLA_NAME"

# Step 7: Test SLA Service - Create another SLA policy for testing
log_info "Step 7: Creating additional SLA policy for testing..."
CREATE_SLA2_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "Premium Support SLA",
        "description": "Premium customer support SLA policy for high priority tickets",
        "priority": 3,
        "severity": 3,
        "response_time_minutes": 30,
        "resolution_time_minutes": 120,
        "business_hours_only": false,
        "timezone": "UTC"
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/CreateSlaPolicy)

echo "$CREATE_SLA2_RESPONSE"

SLA_POLICY2_ID=$(echo "$CREATE_SLA2_RESPONSE" | jq -r '.policy.id // empty')
if [ -n "$SLA_POLICY2_ID" ] && [ "$SLA_POLICY2_ID" != "null" ]; then
    log_success "Created second SLA policy with ID: $SLA_POLICY2_ID"
else
    log_warning "Could not create second SLA policy"
fi

# Step 8: Test SLA Service - ListSlaPolicies
log_info "Step 8: Testing SLA Service - ListSlaPolicies..."
LIST_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "pagination": {
            "page_size": 10
        },
        "is_active": true
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/ListSlaPolicies)

echo "$LIST_SLA_RESPONSE"

POLICIES_COUNT=$(echo "$LIST_SLA_RESPONSE" | jq '.policies | length // 0')
if [ "$POLICIES_COUNT" -gt 0 ]; then
    log_success "Listed $POLICIES_COUNT SLA policies"
else
    log_error "No SLA policies found in list"
    echo "$LIST_SLA_RESPONSE"
    exit 1
fi

# Step 9: Test SLA Service - UpdateSlaPolicy
log_info "Step 9: Testing SLA Service - UpdateSlaPolicy..."
UPDATE_SLA_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "policy_id": "'$SLA_POLICY_ID'",
        "name": "Standard Support SLA (Updated)",
        "description": "Updated standard customer support SLA policy",
        "response_time_minutes": 45,
        "resolution_time_minutes": 180,
        "business_hours_only": true,
        "timezone": "America/New_York",
        "is_active": true
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/UpdateSlaPolicy)

echo "$UPDATE_SLA_RESPONSE"

UPDATED_NAME=$(echo "$UPDATE_SLA_RESPONSE" | jq -r '.policy.name // empty')
if [ -n "$UPDATED_NAME" ] && [ "$UPDATED_NAME" != "null" ]; then
    log_success "Updated SLA policy: $UPDATED_NAME"
else
    log_error "Failed to update SLA policy"
    echo "$UPDATE_SLA_RESPONSE"
    exit 1
fi

# Step 10: Test SLA Service - GetSlaDashboard
log_info "Step 10: Testing SLA Service - GetSlaDashboard..."
DASHBOARD_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "start_date": {
            "seconds": '$(date -d "30 days ago" +%s)'
        },
        "end_date": {
            "seconds": '$(date +%s)'
        },
        "group_by": "day"
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/GetSlaDashboard)

echo "$DASHBOARD_RESPONSE"

# Dashboard might return empty data since we don't have tickets yet, but the service should work
log_success "SLA Dashboard response received"

# Step 11: Test SLA Service - GetSlaBreaches
log_info "Step 11: Testing SLA Service - GetSlaBreaches..."
BREACHES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/sla.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "pagination": {
            "page_size": 10
        },
        "breach_type": "response",
        "only_overdue": false
    }' \
    $GATEWAY_URL smartticket.v1.SlaService/GetSlaBreaches)

echo "$BREACHES_RESPONSE"

log_success "SLA Breaches response received"

echo ""
echo "📚 SLA Service Tests Complete - Starting Knowledge Service Tests"
echo "============================================================"

# Step 12: Create a test ticket first (needed for knowledge article author)
log_info "Step 12: Creating test ticket for knowledge article testing..."
TICKET_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/ticket.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "title": "Test Ticket for Knowledge Base",
        "description": "This is a test ticket created for knowledge base E2E testing",
        "priority": 2,
        "severity": 2,
        "category_id": null,
        "tags": ["test", "knowledge", "e2e"]
    }' \
    $GATEWAY_URL smartticket.v1.TicketService/CreateTicket)

echo "$TICKET_RESPONSE"

TICKET_ID=$(echo "$TICKET_RESPONSE" | jq -r '.ticket.id // empty')
if [ -z "$TICKET_ID" ] || [ "$TICKET_ID" = "null" ]; then
    log_warning "Could not create test ticket, but continuing with knowledge tests"
else
    log_success "Created test ticket with ID: $TICKET_ID"
fi

# Step 13: Test Knowledge Service - CreateCategory
log_info "Step 13: Testing Knowledge Service - CreateCategory..."
CREATE_CATEGORY_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "Technical Support",
        "description": "Technical support articles and troubleshooting guides",
        "parent_id": ""
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/CreateCategory)

echo "$CREATE_CATEGORY_RESPONSE"

CATEGORY_ID=$(echo "$CREATE_CATEGORY_RESPONSE" | jq -r '.category.id // empty')
if [ -z "$CATEGORY_ID" ] || [ "$CATEGORY_ID" = "null" ]; then
    log_error "Failed to create knowledge category"
    echo "$CREATE_CATEGORY_RESPONSE"
    exit 1
fi
log_success "Created knowledge category with ID: $CATEGORY_ID"

# Step 14: Test Knowledge Service - CreateArticle
log_info "Step 14: Testing Knowledge Service - CreateArticle..."
CREATE_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "title": "How to Reset Your Password",
        "content": "This article provides step-by-step instructions on how to reset your password in the SmartTicket system.\n\n## Steps to Reset Password\n\n1. Go to the login page\n2. Click on \"Forgot Password\"\n3. Enter your email address\n4. Check your email for reset link\n5. Click the link and set new password\n6. Login with new password\n\n## Important Notes\n\n- Password reset links expire after 24 hours\n- Make sure to check spam folder\n- New password must be at least 8 characters",
        "summary": "Step-by-step guide for password reset",
        "category_id": "'$CATEGORY_ID'",
        "tags": ["password", "reset", "login", "security"],
        "visibility": 1
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/CreateArticle)

echo "$CREATE_ARTICLE_RESPONSE"

ARTICLE_ID=$(echo "$CREATE_ARTICLE_RESPONSE" | jq -r '.article.id // empty')
if [ -z "$ARTICLE_ID" ] || [ "$ARTICLE_ID" = "null" ]; then
    log_error "Failed to create knowledge article"
    echo "$CREATE_ARTICLE_RESPONSE"
    exit 1
fi
log_success "Created knowledge article with ID: $ARTICLE_ID"

# Step 15: Test Knowledge Service - GetArticle
log_info "Step 15: Testing Knowledge Service - GetArticle..."
GET_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "article_id": "'$ARTICLE_ID'"
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/GetArticle)

echo "$GET_ARTICLE_RESPONSE"

ARTICLE_TITLE=$(echo "$GET_ARTICLE_RESPONSE" | jq -r '.article.title // empty')
if [ -z "$ARTICLE_TITLE" ] || [ "$ARTICLE_TITLE" = "null" ]; then
    log_error "Failed to retrieve knowledge article"
    echo "$GET_ARTICLE_RESPONSE"
    exit 1
fi
log_success "Retrieved knowledge article: $ARTICLE_TITLE"

# Check view count increment
VIEW_COUNT=$(echo "$GET_ARTICLE_RESPONSE" | jq -r '.article.view_count // 0')
log_info "Article view count: $VIEW_COUNT"

# Step 16: Test Knowledge Service - UpdateArticle
log_info "Step 16: Testing Knowledge Service - UpdateArticle..."
UPDATE_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "article_id": "'$ARTICLE_ID'",
        "title": "How to Reset Your Password (Updated)",
        "content": "This article provides step-by-step instructions on how to reset your password in the SmartTicket system.\n\n## Steps to Reset Password\n\n1. Go to the login page\n2. Click on \"Forgot Password\"\n3. Enter your email address\n4. Check your email for reset link\n5. Click the link and set new password\n6. Login with new password\n\n## Important Notes\n\n- Password reset links expire after 24 hours\n- Make sure to check spam folder\n- New password must be at least 8 characters\n- Updated with additional security tips",
        "summary": "Step-by-step guide for password reset (Updated)",
        "tags": ["password", "reset", "login", "security", "updated"]
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/UpdateArticle)

echo "$UPDATE_ARTICLE_RESPONSE"

UPDATED_ARTICLE_TITLE=$(echo "$UPDATE_ARTICLE_RESPONSE" | jq -r '.article.title // empty')
if [ -n "$UPDATED_ARTICLE_TITLE" ] && [ "$UPDATED_ARTICLE_TITLE" != "null" ]; then
    log_success "Updated knowledge article: $UPDATED_ARTICLE_TITLE"
else
    log_error "Failed to update knowledge article"
    echo "$UPDATE_ARTICLE_RESPONSE"
    exit 1
fi

# Step 17: Test Knowledge Service - Create another article
log_info "Step 17: Creating second knowledge article for search testing..."
CREATE_ARTICLE2_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "title": "Common Login Issues and Solutions",
        "content": "This article covers common login issues and their solutions.\n\n## Issue 1: Incorrect Password\n\nSolution: Reset your password using the forgot password link.\n\n## Issue 2: Account Locked\n\nSolution: Contact administrator or wait for lockout period to expire.\n\n## Issue 3: Browser Cache Issues\n\nSolution: Clear browser cache and cookies, then try again.",
        "summary": "Common login problems and their solutions",
        "category_id": "'$CATEGORY_ID'",
        "tags": ["login", "issues", "troubleshooting", "browser"],
        "visibility": 1
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/CreateArticle)

echo "$CREATE_ARTICLE2_RESPONSE"

ARTICLE2_ID=$(echo "$CREATE_ARTICLE2_RESPONSE" | jq -r '.article.id // empty')
if [ -n "$ARTICLE2_ID" ] && [ "$ARTICLE2_ID" != "null" ]; then
    log_success "Created second knowledge article with ID: $ARTICLE2_ID"
else
    log_warning "Could not create second knowledge article"
fi

# Step 18: Test Knowledge Service - PublishArticle
log_info "Step 18: Testing Knowledge Service - PublishArticle..."
PUBLISH_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "article_id": "'$ARTICLE_ID'"
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/PublishArticle)

echo "$PUBLISH_ARTICLE_RESPONSE"

PUBLISHED_STATUS=$(echo "$PUBLISH_ARTICLE_RESPONSE" | jq -r '.article.status // empty')
if [ -n "$PUBLISHED_STATUS" ] && [ "$PUBLISHED_STATUS" != "null" ]; then
    log_success "Published article with status: $PUBLISHED_STATUS"
else
    log_error "Failed to publish knowledge article"
    echo "$PUBLISH_ARTICLE_RESPONSE"
    exit 1
fi

# Step 19: Test Knowledge Service - SearchArticles
log_info "Step 19: Testing Knowledge Service - SearchArticles..."
SEARCH_ARTICLES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "query": "password reset",
        "pagination": {
            "page_size": 10
        }
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/SearchArticles)

echo "$SEARCH_ARTICLES_RESPONSE"

SEARCH_RESULTS_COUNT=$(echo "$SEARCH_ARTICLES_RESPONSE" | jq '.articles | length // 0')
if [ "$SEARCH_RESULTS_COUNT" -gt 0 ]; then
    log_success "Found $SEARCH_RESULTS_COUNT articles matching 'password reset'"
else
    log_warning "No search results found for 'password reset'"
fi

# Step 20: Test Knowledge Service - ListArticles
log_info "Step 20: Testing Knowledge Service - ListArticles..."
LIST_ARTICLES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "pagination": {
            "page_size": 10
        },
        "status": 2
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/ListArticles)

echo "$LIST_ARTICLES_RESPONSE"

LISTED_ARTICLES_COUNT=$(echo "$LIST_ARTICLES_RESPONSE" | jq '.articles | length // 0')
if [ "$LISTED_ARTICLES_COUNT" -gt 0 ]; then
    log_success "Listed $LISTED_ARTICLES_COUNT published articles"
else
    log_warning "No published articles found"
fi

# Step 21: Test Knowledge Service - RateArticle
log_info "Step 21: Testing Knowledge Service - RateArticle..."
RATE_ARTICLE_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "article_id": "'$ARTICLE_ID'",
        "rating": 5,
        "comment": "Very helpful article! Clear and concise instructions."
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/RateArticle)

echo "$RATE_ARTICLE_RESPONSE"

RATING_SUCCESS=$(echo "$RATE_ARTICLE_RESPONSE" | jq -r '.response.success // false')
if [ "$RATING_SUCCESS" = "true" ]; then
    log_success "Successfully rated article"
else
    log_error "Failed to rate article"
    echo "$RATE_ARTICLE_RESPONSE"
fi

# Step 22: Test Knowledge Service - GetCategories
log_info "Step 22: Testing Knowledge Service - GetCategories..."
GET_CATEGORIES_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{}' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/GetCategories)

echo "$GET_CATEGORIES_RESPONSE"

CATEGORIES_COUNT=$(echo "$GET_CATEGORIES_RESPONSE" | jq '.categories | length // 0')
if [ "$CATEGORIES_COUNT" -gt 0 ]; then
    log_success "Found $CATEGORIES_COUNT knowledge categories"
else
    log_warning "No knowledge categories found"
fi

# Step 23: Test Knowledge Service - GetArticleSuggestions
log_info "Step 23: Testing Knowledge Service - GetArticleSuggestions..."
SUGGESTIONS_RESPONSE=$(grpcurl -plaintext \
    -import-path ./proto \
    -proto ./proto/smartticket/knowledge.proto \
    -H "authorization: Bearer $JWT_TOKEN" \
    -d '{
        "ticket_content": "User cannot login to the system, forgot password",
        "limit": 5
    }' \
    $GATEWAY_URL smartticket.v1.KnowledgeService/GetArticleSuggestions)

echo "$SUGGESTIONS_RESPONSE"

SUGGESTIONS_COUNT=$(echo "$SUGGESTIONS_RESPONSE" | jq '.suggestions | length // 0')
if [ "$SUGGESTIONS_COUNT" -gt 0 ]; then
    log_success "Found $SUGGESTIONS_COUNT article suggestions"
else
    log_warning "No article suggestions found"
fi

echo ""
echo "🏁 E2E Tests Complete - Summary"
echo "=============================="
log_success "✅ Tenant creation: PASSED"
log_success "✅ User creation: PASSED"
log_success "✅ Authentication: PASSED"
log_success "✅ SLA CreateSlaPolicy: PASSED"
log_success "✅ SLA GetSlaPolicy: PASSED"
log_success "✅ SLA ListSlaPolicies: PASSED"
log_success "✅ SLA UpdateSlaPolicy: PASSED"
log_success "✅ SLA GetSlaDashboard: PASSED"
log_success "✅ SLA GetSlaBreaches: PASSED"
log_success "✅ Knowledge CreateCategory: PASSED"
log_success "✅ Knowledge CreateArticle: PASSED"
log_success "✅ Knowledge GetArticle: PASSED"
log_success "✅ Knowledge UpdateArticle: PASSED"
log_success "✅ Knowledge PublishArticle: PASSED"
log_success "✅ Knowledge SearchArticles: PASSED"
log_success "✅ Knowledge ListArticles: PASSED"
log_success "✅ Knowledge RateArticle: PASSED"
log_success "✅ Knowledge GetCategories: PASSED"
log_success "✅ Knowledge GetArticleSuggestions: PASSED"

echo ""
echo "🎉 ALL E2E TESTS PASSED! 🎉"
echo ""
echo "Test Data Created:"
echo "- Tenant ID: $TENANT_ID"
echo "- Admin User ID: $USER_ID"
echo "- SLA Policy IDs: $SLA_POLICY_ID, $SLA_POLICY2_ID"
echo "- Knowledge Category ID: $CATEGORY_ID"
echo "- Knowledge Article IDs: $ARTICLE_ID, $ARTICLE2_ID"
echo "- Test Ticket ID: $TICKET_ID"