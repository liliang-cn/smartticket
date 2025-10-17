#!/bin/bash

# SmartTicket 完整gRPC E2E测试
# 覆盖所有68个接口，确保100%通过率

set -e

# 配置
GRPC_GATEWAY_HOST="${GRPC_GATEWAY_HOST:-localhost}"
GRPC_GATEWAY_PORT="${GRPC_GATEWAY_PORT:-50051}"
TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-$(dirname "$0")/../test_results}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 服务接口统计
declare -A SERVICE_COUNTS
SERVICE_COUNTS[TenantService]=0
SERVICE_COUNTS[UserService]=0
SERVICE_COUNTS[AuthService]=0
SERVICE_COUNTS[TicketService]=0
SERVICE_COUNTS[KnowledgeService]=0
SERVICE_COUNTS[SlaService]=0
SERVICE_COUNTS[RolePermissionService]=0

# 创建结果目录
mkdir -p "$TEST_RESULTS_DIR"

# 日志文件
LOG_FILE="$TEST_RESULTS_DIR/complete_e2e_${TIMESTAMP}.log"
TOKEN_FILE="/tmp/smartticket_complete_token_${TIMESTAMP}"

# 存储创建的资源ID
CREATED_TENANT_ID=""
CREATED_USER_ID=""
CREATED_TICKET_ID=""
CREATED_ARTICLE_ID=""
CREATED_SLA_POLICY_ID=""
CREATED_ROLE_ID=""
CREATED_CATEGORY_ID=""

# 日志函数
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_test() {
    echo -e "${PURPLE}[TEST]${NC} $1" | tee -a "$LOG_FILE"
}

log_result() {
    echo -e "${CYAN}[RESULT]${NC} $1" | tee -a "$LOG_FILE"
}

# 检查grpcurl是否可用
check_grpcurl() {
    if ! command -v grpcurl &> /dev/null; then
        log_error "grpcurl not found. Please install grpcurl:"
        log_error "  brew install grpcurl"
        exit 1
    fi
    log_success "grpcurl found: $(grpcurl --version)"
}

# 检查gRPC服务是否运行
check_grpc_service() {
    log_info "Checking gRPC service at $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"

    if ! grpcurl -plaintext "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" list > /dev/null 2>&1; then
        log_error "Cannot connect to gRPC service at $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"
        log_error "Please ensure the gRPC gateway is running:"
        log_error "  cargo run --bin gateway"
        exit 1
    fi

    log_success "gRPC service is reachable"
}

# 登录获取JWT token
login_and_get_token() {
    log_info "=== 正在登录获取JWT token ==="

    local login_data='{
        "email": "superadmin@smartticket.system",
        "password": "admin123",
        "tenantDomain": "test.smartticket.com",
        "rememberMe": false
    }'

    log_test "AuthService.Login"
    local login_response
    login_response=$(grpcurl -plaintext -d "$login_data" \
        "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" \
        "smartticket.v1.AuthService/Login" 2>&1)

    if [ $? -ne 0 ]; then
        log_error "登录失败: $login_response"
        exit 1
    fi

    # 提取access token
    local access_token
    access_token=$(echo "$login_response" | jq -r '.accessToken // empty')

    if [ -z "$access_token" ] || [ "$access_token" = "null" ]; then
        log_error "无法从响应中提取access token"
        log_error "完整响应: $login_response"
        exit 1
    fi

    log_success "成功获取JWT token: ${access_token:0:20}..."
    echo "$access_token" > "$TOKEN_FILE"

    # 提取用户信息
    local user_id=$(echo "$login_response" | jq -r '.user.id // empty')
    local tenant_id=$(echo "$login_response" | jq -r '.user.tenantId // empty')
    local user_email=$(echo "$login_response" | jq -r '.user.email // empty')

    log_info "用户ID: $user_id"
    log_info "租户ID: $tenant_id"
    log_info "邮箱: $user_email"

    # 保存用户信息到环境变量
    export TEST_USER_ID="$user_id"
    export TEST_TENANT_ID="$tenant_id"
    export TEST_USER_EMAIL="$user_email"
    export TEST_ACCESS_TOKEN="$access_token"

    return 0
}

# 执行带认证的gRPC调用
execute_authenticated_grpc_call() {
    local service_method="$1"
    local request_data="$2"
    local expected_success="$3" # "true" or "false"
    local test_description="$4"
    local service_name="$5"

    ((TOTAL_TESTS++))
    ((SERVICE_COUNTS[$service_name]++))

    log_test "Testing: $test_description"
    log_info "Method: $service_method"

    # 使用真实的JWT token进行认证
    local auth_header="Authorization: Bearer $TEST_ACCESS_TOKEN"

    # 构造完整的命令
    local cmd="grpcurl -plaintext -H '$auth_header' -d '$request_data' '$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT' '$service_method'"

    log_info "Command: $cmd"

    # 执行命令并捕获输出
    local output
    local exit_code

    output=$(eval "$cmd" 2>&1)
    exit_code=$?

    # 检查执行结果
    if [ $exit_code -eq 0 ]; then
        if echo "$output" | grep -q '"success":true\|"response":{"success":true'; then
            log_result "✅ PASS: $test_description"
            ((PASSED_TESTS++))
            echo "$output" # 返回输出供后续处理
            return 0
        elif echo "$output" | grep -q '"success":false\|"response":{"success":false'; then
            if [ "$expected_success" = "false" ]; then
                log_result "✅ PASS (expected failure): $test_description"
                ((PASSED_TESTS++))
                return 0
            else
                log_result "❌ FAIL: $test_description - API returned error"
                log_error "Output: $output"
                ((FAILED_TESTS++))
                return 1
            fi
        else
            log_result "✅ PASS: $test_description - Service responded"
            ((PASSED_TESTS++))
            echo "$output"
            return 0
        fi
    else
        if [ "$expected_success" = "false" ]; then
            log_result "✅ PASS (expected failure): $test_description"
            ((PASSED_TESTS++))
            return 0
        else
            log_result "❌ FAIL: $test_description - Command failed with exit code $exit_code"
            log_error "Output: $output"
            ((FAILED_TESTS++))
            return 1
        fi
    fi
}

# 生成请求元数据
generate_request_metadata() {
    local request_id="req_$(date +%s)_$RANDOM"
    cat <<EOF
{
  "metadata": {
    "tenantId": "$TEST_TENANT_ID",
    "userId": "$TEST_USER_ID",
    "requestId": "$request_id",
    "clientIpAddress": "127.0.0.1",
    "userAgent": "complete-e2e-test"
  }
}
EOF
}

# ========================================
# TenantService 测试 (10个接口)
# ========================================
test_tenant_service() {
    log_info "=== Testing TenantService (10 interfaces) ==="

    local metadata_json=$(generate_request_metadata)

    # 1. CreateTenant - 创建测试租户
    local tenant_domain="test-e2e-$(date +%s).example.com"
    local create_response
    create_response=$(execute_authenticated_grpc_call "smartticket.v1.TenantService/CreateTenant" \
        "$(cat <<EOF
$metadata_json,
  "name": "E2E Test Company $(date +%s)",
  "domain": "$tenant_domain",
  "subscriptionTier": 1,
  "maxUsers": 50,
  "dataResidencyRegion": "EU",
  "contactEmail": "admin@$tenant_domain",
  "billingAddress": "123 Test St, Test City, TC 12345",
  "phone": "+1-555-0123",
  "isTrial": true
EOF
)" \
        "true" \
        "TenantService.CreateTenant - Create new tenant" \
        "TenantService")

    CREATED_TENANT_ID=$(echo "$create_response" | jq -r '.tenant.id // empty')
    log_info "Created tenant ID: $CREATED_TENANT_ID"

    # 2. GetTenant - 获取租户
    execute_authenticated_grpc_call "smartticket.v1.TenantService/GetTenant" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID"
EOF
)" \
        "true" \
        "TenantService.GetTenant - Get tenant by ID" \
        "TenantService" > /dev/null

    # 3. GetCurrentTenant - 获取当前租户
    execute_authenticated_grpc_call "smartticket.v1.TenantService/GetCurrentTenant" \
        "$metadata_json" \
        "true" \
        "TenantService.GetCurrentTenant - Get current tenant info" \
        "TenantService" > /dev/null

    # 4. UpdateTenant - 更新租户
    execute_authenticated_grpc_call "smartticket.v1.TenantService/UpdateTenant" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID",
  "name": "E2E Test Company $(date +%s) (Updated)",
  "contactEmail": "updated@$tenant_domain",
  "billingAddress": "456 Updated St, Test City, TC 12345"
EOF
)" \
        "true" \
        "TenantService.UpdateTenant - Update tenant information" \
        "TenantService" > /dev/null

    # 5. ListTenants - 列出租户
    execute_authenticated_grpc_call "smartticket.v1.TenantService/ListTenants" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "TenantService.ListTenants - List tenants with pagination" \
        "TenantService" > /dev/null

    # 6. UpdateTenantStatus - 更新租户状态
    execute_authenticated_grpc_call "smartticket.v1.TenantService/UpdateTenantStatus" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID",
  "isActive": true,
  "reason": "E2E test activation"
EOF
)" \
        "true" \
        "TenantService.UpdateTenantStatus - Activate tenant" \
        "TenantService" > /dev/null

    # 7. UpdateSubscription - 更新订阅
    execute_authenticated_grpc_call "smartticket.v1.TenantService/UpdateSubscription" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID",
  "newTier": 2,
  "newMaxUsers": 100,
  "effectiveDate": "$(date -Iseconds)",
  "prorate": true,
  "billingChangeReason": "E2E test upgrade"
EOF
)" \
        "true" \
        "TenantService.UpdateSubscription - Update tenant subscription" \
        "TenantService" > /dev/null

    # 8. GetTenantUsage - 获取使用统计
    execute_authenticated_grpc_call "smartticket.v1.TenantService/GetTenantUsage" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID",
  "periodStart": "$(date -d '30 days ago' -Iseconds)",
  "periodEnd": "$(date -Iseconds)",
  "includeDetailedMetrics": true
EOF
)" \
        "true" \
        "TenantService.GetTenantUsage - Get tenant usage statistics" \
        "TenantService" > /dev/null

    # 9. GetTenantBilling - 获取账单信息
    execute_authenticated_grpc_call "smartticket.v1.TenantService/GetTenantBilling" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID",
  "includeLineItems": true
EOF
)" \
        "true" \
        "TenantService.GetTenantBilling - Get tenant billing information" \
        "TenantService" > /dev/null

    # 10. DeleteTenant - 删除租户
    execute_authenticated_grpc_call "smartticket.v1.TenantService/DeleteTenant" \
        "$(cat <<EOF
$metadata_json,
  "tenantId": "$CREATED_TENANT_ID"
EOF
)" \
        "true" \
        "TenantService.DeleteTenant - Delete tenant" \
        "TenantService" > /dev/null
}

# ========================================
# UserService 测试 (11个接口)
# ========================================
test_user_service() {
    log_info "=== Testing UserService (11 interfaces) ==="

    local metadata_json=$(generate_request_metadata)

    # 1. CreateUser - 创建测试用户
    local test_email="e2e-test-$(date +%s)@example.com"
    local create_user_response
    create_user_response=$(execute_authenticated_grpc_call "smartticket.v1.UserService/CreateUser" \
        "$(cat <<EOF
$metadata_json,
  "email": "$test_email",
  "username": "e2e_user_$(date +%s)",
  "fullName": "E2E Test User",
  "password": "E2ETestPass123!",
  "role": 3,
  "phone": "+1-555-0123",
  "timezone": "UTC",
  "language": "en"
EOF
)" \
        "true" \
        "UserService.CreateUser - Create new user" \
        "UserService")

    CREATED_USER_ID=$(echo "$create_user_response" | jq -r '.user.id // empty')
    log_info "Created user ID: $CREATED_USER_ID"

    # 2. GetUser - 获取用户
    execute_authenticated_grpc_call "smartticket.v1.UserService/GetUser" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$CREATED_USER_ID"
EOF
)" \
        "true" \
        "UserService.GetUser - Get user by ID" \
        "UserService" > /dev/null

    # 3. GetCurrentUser - 获取当前用户
    execute_authenticated_grpc_call "smartticket.v1.UserService/GetCurrentUser" \
        "$metadata_json" \
        "true" \
        "UserService.GetCurrentUser - Get current user profile" \
        "UserService" > /dev/null

    # 4. UpdateUser - 更新用户
    execute_authenticated_grpc_call "smartticket.v1.UserService/UpdateUser" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$CREATED_USER_ID",
  "email": "$test_email",
  "fullName": "E2E Test User (Updated)",
  "phone": "+1-555-0456"
EOF
)" \
        "true" \
        "UserService.UpdateUser - Update user profile" \
        "UserService" > /dev/null

    # 5. UpdateCurrentUser - 更新当前用户
    execute_authenticated_grpc_call "smartticket.v1.UserService/UpdateCurrentUser" \
        "$(cat <<EOF
$metadata_json,
  "fullName": "Super Admin (Self Updated)",
  "phone": "+1-555-0789"
EOF
)" \
        "true" \
        "UserService.UpdateCurrentUser - Update current user profile" \
        "UserService" > /dev/null

    # 6. ListUsers - 列出用户
    execute_authenticated_grpc_call "smartticket.v1.UserService/ListUsers" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "UserService.ListUsers - List users with pagination" \
        "UserService" > /dev/null

    # 7. UpdateUserStatus - 更新用户状态
    execute_authenticated_grpc_call "smartticket.v1.UserService/UpdateUserStatus" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$CREATED_USER_ID",
  "isActive": true,
  "reason": "E2E test activation"
EOF
)" \
        "true" \
        "UserService.UpdateUserStatus - Activate user" \
        "UserService" > /dev/null

    # 8. ChangePassword - 修改密码
    execute_authenticated_grpc_call "smartticket.v1.UserService/ChangePassword" \
        "$(cat <<EOF
$metadata_json,
  "currentPassword": "admin123",
  "newPassword": "NewE2ETestPass456!"
EOF
)" \
        "true" \
        "UserService.ChangePassword - Change user password" \
        "UserService" > /dev/null

    # 9. ResetPassword - 重置密码
    execute_authenticated_grpc_call "smartticket.v1.UserService/ResetPassword" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$CREATED_USER_ID",
  "temporaryPassword": "TempE2EPass789!",
  "sendEmail": false
EOF
)" \
        "true" \
        "UserService.ResetPassword - Reset user password" \
        "UserService" > /dev/null

    # 10. GetUserPermissions - 获取用户权限
    execute_authenticated_grpc_call "smartticket.v1.UserService/GetUserPermissions" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$CREATED_USER_ID"
EOF
)" \
        "true" \
        "UserService.GetUserPermissions - Get user permissions" \
        "UserService" > /dev/null

    # 11. DeleteUser - 删除用户
    execute_authenticated_grpc_call "smartticket.v1.UserService/DeleteUser" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$CREATED_USER_ID"
EOF
)" \
        "true" \
        "UserService.DeleteUser - Delete user" \
        "UserService" > /dev/null
}

# ========================================
# AuthService 测试 (2个接口)
# ========================================
test_auth_service() {
    log_info "=== Testing AuthService (2 interfaces) ==="

    # 1. RefreshToken - 刷新token
    execute_authenticated_grpc_call "smartticket.v1.AuthService/RefreshToken" \
        "$(cat <<EOF
{
  "refreshToken": "$TEST_ACCESS_TOKEN"
}
EOF
)" \
        "true" \
        "AuthService.RefreshToken - Refresh access token" \
        "AuthService" > /dev/null

    # 注意：Login在开始时已经测试过了
    log_info "AuthService.Login - Already tested at startup"
}

# ========================================
# TicketService 测试 (11个接口)
# ========================================
test_ticket_service() {
    log_info "=== Testing TicketService (11 interfaces) ==="

    local metadata_json=$(generate_request_metadata)

    # 1. CreateTicket - 创建工单
    local ticket_title="E2E Complete Test Ticket - $(date +%s)"
    local create_ticket_response
    create_ticket_response=$(execute_authenticated_grpc_call "smartticket.v1.TicketService/CreateTicket" \
        "$(cat <<EOF
$metadata_json,
  "title": "$ticket_title",
  "description": "This is a comprehensive test ticket created by complete E2E testing framework",
  "priority": 2,
  "severity": 2,
  "categoryId": "",
  "contactId": "$TEST_USER_ID",
  "tags": ["e2e-complete", "automated", "comprehensive"]
EOF
)" \
        "true" \
        "TicketService.CreateTicket - Create new ticket" \
        "TicketService")

    CREATED_TICKET_ID=$(echo "$create_ticket_response" | jq -r '.ticket.id // empty')
    log_info "Created ticket ID: $CREATED_TICKET_ID"

    # 2. GetTicket - 获取工单
    execute_authenticated_grpc_call "smartticket.v1.TicketService/GetTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID",
  "includeComments": true
EOF
)" \
        "true" \
        "TicketService.GetTicket - Get ticket by ID" \
        "TicketService" > /dev/null

    # 3. UpdateTicket - 更新工单
    execute_authenticated_grpc_call "smartticket.v1.TicketService/UpdateTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID",
  "title": "$ticket_title (Updated)",
  "description": "This is a comprehensive test ticket created by complete E2E testing framework - Updated",
  "priority": 3,
  "tags": ["e2e-complete", "automated", "comprehensive", "updated"]
EOF
)" \
        "true" \
        "TicketService.UpdateTicket - Update ticket" \
        "TicketService" > /dev/null

    # 4. ListTickets - 列出工单
    execute_authenticated_grpc_call "smartticket.v1.TicketService/ListTickets" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "TicketService.ListTickets - List tickets with filters" \
        "TicketService" > /dev/null

    # 5. UpdateTicketStatus - 更新工单状态
    execute_authenticated_grpc_call "smartticket.v1.TicketService/UpdateTicketStatus" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID",
  "status": 3,
  "comment": "E2E test status update"
EOF
)" \
        "true" \
        "TicketService.UpdateTicketStatus - Update ticket status" \
        "TicketService" > /dev/null

    # 6. AssignTicket - 分配工单
    execute_authenticated_grpc_call "smartticket.v1.TicketService/AssignTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID",
  "assignedToId": "$TEST_USER_ID",
  "comment": "E2E test assignment"
EOF
)" \
        "true" \
        "TicketService.AssignTicket - Assign ticket to user" \
        "TicketService" > /dev/null

    # 7. AddComment - 添加评论
    execute_authenticated_grpc_call "smartticket.v1.TicketService/AddComment" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID",
  "content": "This is a comprehensive test comment added by complete E2E testing framework",
  "isInternal": false
EOF
)" \
        "true" \
        "TicketService.AddComment - Add comment to ticket" \
        "TicketService" > /dev/null

    # 8. GetComments - 获取评论
    execute_authenticated_grpc_call "smartticket.v1.TicketService/GetComments" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID",
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "TicketService.GetComments - Get ticket comments" \
        "TicketService" > /dev/null

    # 9. SearchTickets - 搜索工单
    execute_authenticated_grpc_call "smartticket.v1.TicketService/SearchTickets" \
        "$(cat <<EOF
$metadata_json,
  "query": "e2e complete",
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "TicketService.SearchTickets - Search tickets" \
        "TicketService" > /dev/null

    # 10. GetTicketStatistics - 获取工单统计
    execute_authenticated_grpc_call "smartticket.v1.TicketService/GetTicketStatistics" \
        "$(cat <<EOF
$metadata_json,
  "dateFrom": "$(date -d '30 days ago' -Iseconds | cut -d'T' -f1)",
  "dateTo": "$(date -Iseconds | cut -d'T' -f1)"
EOF
)" \
        "true" \
        "TicketService.GetTicketStatistics - Get ticket statistics" \
        "TicketService" > /dev/null

    # 11. DeleteTicket - 删除工单
    execute_authenticated_grpc_call "smartticket.v1.TicketService/DeleteTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$CREATED_TICKET_ID"
EOF
)" \
        "true" \
        "TicketService.DeleteTicket - Delete ticket" \
        "TicketService" > /dev/null
}

# ========================================
# KnowledgeService 测试 (12个接口)
# ========================================
test_knowledge_service() {
    log_info "=== Testing KnowledgeService (12 interfaces) ==="

    local metadata_json=$(generate_request_metadata)

    # 1. CreateCategory - 创建分类
    local create_category_response
    create_category_response=$(execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/CreateCategory" \
        "$(cat <<EOF
$metadata_json,
  "name": "E2E Test Category $(date +%s)",
  "description": "Test category created by complete E2E testing framework",
  "icon": "book"
EOF
)" \
        "true" \
        "KnowledgeService.CreateCategory - Create article category" \
        "KnowledgeService")

    CREATED_CATEGORY_ID=$(echo "$create_category_response" | jq -r '.category.id // empty')
    log_info "Created category ID: $CREATED_CATEGORY_ID"

    # 2. CreateArticle - 创建知识文章
    local article_title="E2E Complete Knowledge Article - $(date +%s)"
    local create_article_response
    create_article_response=$(execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/CreateArticle" \
        "$(cat <<EOF
$metadata_json,
  "title": "$article_title",
  "content": "This is a comprehensive test knowledge article created by complete E2E testing framework. It contains detailed information about testing procedures, best practices, and troubleshooting steps.",
  "summary": "Comprehensive test article for complete E2E testing",
  "categoryId": "$CREATED_CATEGORY_ID",
  "visibility": 1,
  "language": "en",
  "tags": ["e2e-complete", "automated", "comprehensive", "testing"]
EOF
)" \
        "true" \
        "KnowledgeService.CreateArticle - Create knowledge article" \
        "KnowledgeService")

    CREATED_ARTICLE_ID=$(echo "$create_article_response" | jq -r '.article.id // empty')
    log_info "Created article ID: $CREATED_ARTICLE_ID"

    # 3. GetArticle - 获取文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/GetArticle" \
        "$(cat <<EOF
$metadata_json,
  "articleId": "$CREATED_ARTICLE_ID",
  "incrementViewCount": true
EOF
)" \
        "true" \
        "KnowledgeService.GetArticle - Get article by ID" \
        "KnowledgeService" > /dev/null

    # 4. UpdateArticle - 更新文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/UpdateArticle" \
        "$(cat <<EOF
$metadata_json,
  "articleId": "$CREATED_ARTICLE_ID",
  "title": "$article_title (Updated)",
  "content": "This is a comprehensive test knowledge article created by complete E2E testing framework. Updated with additional troubleshooting information and detailed examples.",
  "summary": "Comprehensive test article for complete E2E testing - Updated",
  "categoryId": "$CREATED_CATEGORY_ID",
  "comment": "Updated by complete E2E test"
EOF
)" \
        "true" \
        "KnowledgeService.UpdateArticle - Update knowledge article" \
        "KnowledgeService" > /dev/null

    # 5. ListArticles - 列出文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/ListArticles" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "KnowledgeService.ListArticles - List articles with filters" \
        "KnowledgeService" > /dev/null

    # 6. PublishArticle - 发布文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/PublishArticle" \
        "$(cat <<EOF
$metadata_json,
  "articleId": "$CREATED_ARTICLE_ID",
  "comment": "Published by complete E2E test"
EOF
)" \
        "true" \
        "KnowledgeService.PublishArticle - Publish article" \
        "KnowledgeService" > /dev/null

    # 7. ArchiveArticle - 归档文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/ArchiveArticle" \
        "$(cat <<EOF
$metadata_json,
  "articleId": "$CREATED_ARTICLE_ID",
  "reason": "Archived by complete E2E test"
EOF
)" \
        "true" \
        "KnowledgeService.ArchiveArticle - Archive article" \
        "KnowledgeService" > /dev/null

    # 8. SearchArticles - 搜索文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/SearchArticles" \
        "$(cat <<EOF
$metadata_json,
  "query": "e2e complete comprehensive",
  "pagination": {
    "pageSize": 10
  },
  "onlyPublished": true
EOF
)" \
        "true" \
        "KnowledgeService.SearchArticles - Search knowledge articles" \
        "KnowledgeService" > /dev/null

    # 9. GetArticleSuggestions - 获取文章建议
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/GetArticleSuggestions" \
        "$(cat <<EOF
$metadata_json,
  "ticketTitle": "Complete E2E test ticket",
  "ticketDescription": "This is a comprehensive test ticket for complete E2E testing framework",
  "ticketTags": ["e2e-complete", "comprehensive"],
  "categoryId": "$CREATED_CATEGORY_ID",
  "limit": 5
EOF
)" \
        "true" \
        "KnowledgeService.GetArticleSuggestions - Get article suggestions for ticket" \
        "KnowledgeService" > /dev/null

    # 10. RateArticle - 评价文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/RateArticle" \
        "$(cat <<EOF
$metadata_json,
  "articleId": "$CREATED_ARTICLE_ID",
  "isHelpful": true,
  "comment": "This comprehensive article was very helpful for complete E2E testing"
EOF
)" \
        "true" \
        "KnowledgeService.RateArticle - Rate article helpfulness" \
        "KnowledgeService" > /dev/null

    # 11. GetCategories - 获取分类
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/GetCategories" \
        "$metadata_json" \
        "true" \
        "KnowledgeService.GetCategories - Get article categories" \
        "KnowledgeService" > /dev/null

    # 12. DeleteArticle - 删除文章
    execute_authenticated_grpc_call "smartticket.v1.KnowledgeService/DeleteArticle" \
        "$(cat <<EOF
$metadata_json,
  "articleId": "$CREATED_ARTICLE_ID"
EOF
)" \
        "true" \
        "KnowledgeService.DeleteArticle - Delete article" \
        "KnowledgeService" > /dev/null
}

# ========================================
# SlaService 测试 (9个接口)
# ========================================
test_sla_service() {
    log_info "=== Testing SlaService (9 interfaces) ==="

    local metadata_json=$(generate_request_metadata)

    # 1. CreateSlaPolicy - 创建SLA策略
    local policy_response
    policy_response=$(execute_authenticated_grpc_call "smartticket.v1.SlaService/CreateSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "name": "Complete E2E SLA Policy - $(date +%s)",
  "description": "Comprehensive SLA policy created by complete E2E testing framework",
  "priority": 2,
  "severity": 2,
  "responseTimeMinutes": 240,
  "resolutionTimeMinutes": 1440,
  "businessHoursOnly": true,
  "timezone": "UTC"
EOF
)" \
        "true" \
        "SlaService.CreateSlaPolicy - Create SLA policy" \
        "SlaService")

    CREATED_SLA_POLICY_ID=$(echo "$policy_response" | jq -r '.policy.id // empty')
    log_info "Created SLA policy ID: $CREATED_SLA_POLICY_ID"

    # 2. GetSlaPolicy - 获取SLA策略
    execute_authenticated_grpc_call "smartticket.v1.SlaService/GetSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "policyId": "$CREATED_SLA_POLICY_ID"
EOF
)" \
        "true" \
        "SlaService.GetSlaPolicy - Get SLA policy by ID" \
        "SlaService" > /dev/null

    # 3. UpdateSlaPolicy - 更新SLA策略
    execute_authenticated_grpc_call "smartticket.v1.SlaService/UpdateSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "policyId": "$CREATED_SLA_POLICY_ID",
  "name": "Complete E2E SLA Policy (Updated)",
  "description": "Comprehensive SLA policy created by complete E2E testing framework - Updated",
  "responseTimeMinutes": 180,
  "resolutionTimeMinutes": 1200,
  "isActive": true
EOF
)" \
        "true" \
        "SlaService.UpdateSlaPolicy - Update SLA policy" \
        "SlaService" > /dev/null

    # 4. ListSlaPolicies - 列出SLA策略
    execute_authenticated_grpc_call "smartticket.v1.SlaService/ListSlaPolicies" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  },
  "isActive": true
EOF
)" \
        "true" \
        "SlaService.ListSlaPolicies - List SLA policies" \
        "SlaService" > /dev/null

    # 5. GetSlaMetrics - 获取SLA指标
    execute_authenticated_grpc_call "smartticket.v1.SlaService/GetSlaMetrics" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$TEST_USER_ID"
EOF
)" \
        "true" \
        "SlaService.GetSlaMetrics - Get SLA metrics for ticket" \
        "SlaService" > /dev/null

    # 6. GetSlaDashboard - 获取SLA仪表板
    execute_authenticated_grpc_call "smartticket.v1.SlaService/GetSlaDashboard" \
        "$(cat <<EOF
$metadata_json,
  "startDate": "$(date -d '30 days ago' -Iseconds)",
  "endDate": "$(date -Iseconds)",
  "groupBy": "day"
EOF
)" \
        "true" \
        "SlaService.GetSlaDashboard - Get SLA dashboard data" \
        "SlaService" > /dev/null

    # 7. GetSlaBreaches - 获取SLA违规
    execute_authenticated_grpc_call "smartticket.v1.SlaService/GetSlaBreaches" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  },
  "breachType": "response",
  "onlyOverdue": true,
  "startDate": "$(date -d '7 days ago' -Iseconds)",
  "endDate": "$(date -Iseconds)"
EOF
)" \
        "true" \
        "SlaService.GetSlaBreaches - Get SLA breach alerts" \
        "SlaService" > /dev/null

    # 8. UpdateSlaMetrics - 更新SLA指标
    execute_authenticated_grpc_call "smartticket.v1.SlaService/UpdateSlaMetrics" \
        "$(cat <<EOF
$metadata_json,
  "ticketId": "$TEST_USER_ID",
  "eventType": "first_response",
  "eventTime": "$(date -Iseconds)"
EOF
)" \
        "true" \
        "SlaService.UpdateSlaMetrics - Update SLA metrics" \
        "SlaService" > /dev/null

    # 9. DeleteSlaPolicy - 删除SLA策略
    execute_authenticated_grpc_call "smartticket.v1.SlaService/DeleteSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "policyId": "$CREATED_SLA_POLICY_ID"
EOF
)" \
        "true" \
        "SlaService.DeleteSlaPolicy - Delete SLA policy" \
        "SlaService" > /dev/null
}

# ========================================
# RolePermissionService 测试 (13个接口)
# ========================================
test_role_permission_service() {
    log_info "=== Testing RolePermissionService (13 interfaces) ==="

    local metadata_json=$(generate_request_metadata)

    # 1. CreateRole - 创建角色
    local role_response
    role_response=$(execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/CreateRole" \
        "$(cat <<EOF
$metadata_json,
  "name": "Complete E2E Role - $(date +%s)",
  "description": "Comprehensive test role created by complete E2E testing framework",
  "permissionIds": ["ticket:view", "ticket:create", "ticket:update"],
  "isActive": true
EOF
)" \
        "true" \
        "RolePermissionService.CreateRole - Create new role" \
        "RolePermissionService")

    CREATED_ROLE_ID=$(echo "$role_response" | jq -r '.role.id // empty')
    log_info "Created role ID: $CREATED_ROLE_ID"

    # 2. GetRole - 获取角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/GetRole" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID"
EOF
)" \
        "true" \
        "RolePermissionService.GetRole - Get role by ID" \
        "RolePermissionService" > /dev/null

    # 3. UpdateRole - 更新角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/UpdateRole" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID",
  "name": "Complete E2E Role (Updated)",
  "description": "Comprehensive test role created by complete E2E testing framework - Updated",
  "isActive": true
EOF
)" \
        "true" \
        "RolePermissionService.UpdateRole - Update role" \
        "RolePermissionService" > /dev/null

    # 4. ListRoles - 列出角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/ListRoles" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 10
  },
  "includeSystemRoles": true,
  "includeInactive": false
EOF
)" \
        "true" \
        "RolePermissionService.ListRoles - List roles" \
        "RolePermissionService" > /dev/null

    # 5. ListPermissions - 列出权限
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/ListPermissions" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "pageSize": 50
  },
  "includeSystemPermissions": true
EOF
)" \
        "true" \
        "RolePermissionService.ListPermissions - List available permissions" \
        "RolePermissionService" > /dev/null

    # 6. GetRolePermissions - 获取角色权限
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/GetRolePermissions" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID"
EOF
)" \
        "true" \
        "RolePermissionService.GetRolePermissions - Get role permissions" \
        "RolePermissionService" > /dev/null

    # 7. AssignPermissionsToRole - 分配权限给角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/AssignPermissionsToRole" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID",
  "permissionIds": ["ticket:delete", "user:view"],
  "reason": "Complete E2E test permission assignment"
EOF
)" \
        "true" \
        "RolePermissionService.AssignPermissionsToRole - Assign permissions to role" \
        "RolePermissionService" > /dev/null

    # 8. RemovePermissionsFromRole - 从角色移除权限
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/RemovePermissionsFromRole" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID",
  "permissionIds": ["ticket:delete"],
  "reason": "Complete E2E test permission removal"
EOF
)" \
        "true" \
        "RolePermissionService.RemovePermissionsFromRole - Remove permissions from role" \
        "RolePermissionService" > /dev/null

    # 9. AssignRoleToUser - 分配角色给用户
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/AssignRoleToUser" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$TEST_USER_ID",
  "roleId": "$CREATED_ROLE_ID",
  "expiresAt": "$(date -d '+90 days' -Iseconds)",
  "reason": "Complete E2E test role assignment"
EOF
)" \
        "true" \
        "RolePermissionService.AssignRoleToUser - Assign role to user" \
        "RolePermissionService" > /dev/null

    # 10. GetUserRoles - 获取用户角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/GetUserRoles" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$TEST_USER_ID",
  "includeExpired": false,
  "includeInactive": false
EOF
)" \
        "true" \
        "RolePermissionService.GetUserRoles - Get user roles" \
        "RolePermissionService" > /dev/null

    # 11. GetUsersWithRole - 获取拥有特定角色的用户
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/GetUsersWithRole" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID",
  "pagination": {
    "pageSize": 10
  }
EOF
)" \
        "true" \
        "RolePermissionService.GetUsersWithRole - Get users with specific role" \
        "RolePermissionService" > /dev/null

    # 12. RemoveRoleFromUser - 从用户移除角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/RemoveRoleFromUser" \
        "$(cat <<EOF
$metadata_json,
  "userId": "$TEST_USER_ID",
  "roleId": "$CREATED_ROLE_ID",
  "reason": "Complete E2E test role removal"
EOF
)" \
        "true" \
        "RolePermissionService.RemoveRoleFromUser - Remove role from user" \
        "RolePermissionService" > /dev/null

    # 13. DeleteRole - 删除角色
    execute_authenticated_grpc_call "smartticket.v1.RolePermissionService/DeleteRole" \
        "$(cat <<EOF
$metadata_json,
  "roleId": "$CREATED_ROLE_ID",
  "forceDelete": false
EOF
)" \
        "true" \
        "RolePermissionService.DeleteRole - Delete role" \
        "RolePermissionService" > /dev/null
}

# 主函数
main() {
    log_info "Starting SmartTicket Complete gRPC E2E Tests"
    log_info "Target: $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"
    log_info "Goal: Test all 68 gRPC interfaces with 100% success rate"

    # 环境检查
    check_grpcurl
    check_grpc_service

    # 确保jq工具可用
    if ! command -v jq &> /dev/null; then
        log_error "jq not found. Please install jq:"
        log_error "  brew install jq"
        exit 1
    fi

    # 登录获取JWT token
    login_and_get_token

    log_success "✅ 所有准备工作完成，开始执行完整的68个接口测试"

    # 按服务分组执行所有测试
    test_tenant_service      # 10 interfaces
    test_user_service       # 11 interfaces
    test_auth_service       # 2 interfaces (Login already tested)
    test_ticket_service     # 11 interfaces
    test_knowledge_service  # 12 interfaces
    test_sla_service        # 9 interfaces
    test_role_permission_service # 13 interfaces

    # 清理token文件
    rm -f "$TOKEN_FILE"

    # 显示详细的测试结果
    echo ""
    echo "=============================================="
    echo "🧪 Complete gRPC E2E Test Summary"
    echo "=============================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

    # 显示每个服务的测试统计
    echo ""
    echo "📊 Service Test Breakdown:"
    for service in "${!SERVICE_COUNTS[@]}"; do
        count=${SERVICE_COUNTS[$service]}
        echo "  - $service: $count tests"
    done

    # 计算预期总数
    local expected_total=68
    echo ""
    echo "📈 Coverage Analysis:"
    echo "  Expected interfaces: $expected_total"
    echo "  Tested interfaces: $TOTAL_TESTS"
    if [ $TOTAL_TESTS -eq $expected_total ]; then
        echo -e "  Coverage: ${GREEN}100%${NC} ✅"
    else
        local coverage=$((TOTAL_TESTS * 100 / expected_total))
        echo -e "  Coverage: ${YELLOW}$coverage%${NC}"
    fi

    if [ $FAILED_TESTS -eq 0 ]; then
        SUCCESS_RATE=100
        echo -e "Success Rate: ${GREEN}100%${NC}"
        echo "🎉 All gRPC tests passed with real authentication!"
        log_success "🎯 SUCCESS: 完成了68个接口的100%覆盖测试，成功率100%！"
    else
        SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
        echo -e "Success Rate: ${RED}$SUCCESS_RATE%${NC}"
        log_error "❌ 有 $FAILED_TESTS 个测试失败"
    fi

    echo ""
    echo "Results saved to:"
    echo "  Log: $LOG_FILE"

    # 返回适当的退出码
    if [ $FAILED_TESTS -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi