#!/bin/bash

# SmartTicket gRPC测试用例定义
# 包含所有gRPC服务的测试用例

# 测试数据生成函数
generate_tenant_id() {
    echo "tenant_$(date +%s)_$RANDOM"
}

generate_user_id() {
    echo "user_$(date +%s)_$RANDOM"
}

generate_ticket_id() {
    echo "ticket_$(date +%s)_$RANDOM"
}

generate_email() {
    echo "test${RANDOM}@example.com"
}

# 生成当前时间戳（ISO格式）
timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%S.%3NZ"
}

# 生成RequestMetadata
generate_metadata() {
    local tenant_id="${1:-$(generate_tenant_id)}"
    local user_id="${2:-$(generate_user_id)}"

    cat <<EOF
{
  "metadata": {
    "tenant_id": "$tenant_id",
    "user_id": "$user_id",
    "request_id": "req_$(date +%s)_$RANDOM",
    "client_ip_address": "127.0.0.1",
    "user_agent": "grpcurl-test"
  }
}
EOF
}

# ========================================
# TenantService 测试用例
# ========================================
test_tenant_service() {
    log_info "=== Testing TenantService ==="

    local tenant_id=$(generate_tenant_id)
    local metadata_json=$(generate_metadata "$tenant_id")

    # 1. CreateTenant
    execute_grpc_call "smartticket.v1.TenantService/CreateTenant" \
        "$(cat <<EOF
$metadata_json,
  "name": "Test Company Inc.",
  "domain": "test-company.example.com",
  "subscription_tier": 1,
  "max_users": 50,
  "data_residency_region": "EU",
  "contact_email": "admin@test-company.example.com",
  "billing_address": "123 Test St, Test City, TC 12345",
  "phone": "+1-555-0123",
  "is_trial": true
EOF
)" \
        "true" \
        "TenantService.CreateTenant - Create new tenant"

    # 2. GetTenant
    execute_grpc_call "smartticket.v1.TenantService/GetTenant" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id"
EOF
)" \
        "true" \
        "TenantService.GetTenant - Get tenant by ID"

    # 3. ListTenants
    execute_grpc_call "smartticket.v1.TenantService/ListTenants" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  }
EOF
)" \
        "true" \
        "TenantService.ListTenants - List tenants with pagination"

    # 4. UpdateTenant
    execute_grpc_call "smartticket.v1.TenantService/UpdateTenant" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id",
  "name": "Test Company Inc. (Updated)",
  "contact_email": "updated@test-company.example.com"
EOF
)" \
        "true" \
        "TenantService.UpdateTenant - Update tenant information"

    # 5. UpdateTenantStatus
    execute_grpc_call "smartticket.v1.TenantService/UpdateTenantStatus" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id",
  "is_active": true,
  "reason": "Test activation"
EOF
)" \
        "true" \
        "TenantService.UpdateTenantStatus - Activate tenant"

    # 6. GetTenantUsage
    execute_grpc_call "smartticket.v1.TenantService/GetTenantUsage" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id",
  "period_start": "$(date -d '30 days ago' -Iseconds)",
  "period_end": "$(date -Iseconds)",
  "include_detailed_metrics": true
EOF
)" \
        "true" \
        "TenantService.GetTenantUsage - Get tenant usage statistics"

    # 7. GetTenantBilling
    execute_grpc_call "smartticket.v1.TenantService/GetTenantBilling" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id",
  "include_line_items": true
EOF
)" \
        "true" \
        "TenantService.GetTenantBilling - Get tenant billing information"

    # 8. UpdateSubscription
    execute_grpc_call "smartticket.v1.TenantService/UpdateSubscription" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id",
  "new_tier": 2,
  "new_max_users": 100,
  "effective_date": "$(date -Iseconds)",
  "prorate": true,
  "billing_change_reason": "Upgrade to Premium"
EOF
)" \
        "true" \
        "TenantService.UpdateSubscription - Update tenant subscription"

    # 9. GetCurrentTenant
    execute_grpc_call "smartticket.v1.TenantService/GetCurrentTenant" \
        "$(cat <<EOF
$metadata_json
EOF
)" \
        "true" \
        "TenantService.GetCurrentTenant - Get current tenant"

    # 10. DeleteTenant
    execute_grpc_call "smartticket.v1.TenantService/DeleteTenant" \
        "$(cat <<EOF
$metadata_json,
  "tenant_id": "$tenant_id"
EOF
)" \
        "true" \
        "TenantService.DeleteTenant - Delete tenant"
}

# ========================================
# UserService 测试用例
# ========================================
test_user_service() {
    log_info "=== Testing UserService ==="

    local tenant_id=$(generate_tenant_id)
    local user_id=$(generate_user_id)
    local metadata_json=$(generate_metadata "$tenant_id" "$user_id")
    local test_email=$(generate_email)

    # 1. CreateUser
    execute_grpc_call "smartticket.v1.UserService/CreateUser" \
        "$(cat <<EOF
$metadata_json,
  "email": "$test_email",
  "username": "testuser_$RANDOM",
  "full_name": "Test User",
  "password": "SecurePass123!",
  "role": 3,
  "phone": "+1-555-0123",
  "timezone": "UTC",
  "language": "en"
EOF
)" \
        "true" \
        "UserService.CreateUser - Create new user"

    # 2. GetUser
    execute_grpc_call "smartticket.v1.UserService/GetUser" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id"
EOF
)" \
        "true" \
        "UserService.GetUser - Get user by ID"

    # 3. GetCurrentUser
    execute_grpc_call "smartticket.v1.UserService/GetCurrentUser" \
        "$(cat <<EOF
$metadata_json
EOF
)" \
        "true" \
        "UserService.GetCurrentUser - Get current user profile"

    # 4. UpdateUser
    execute_grpc_call "smartticket.v1.UserService/UpdateUser" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id",
  "email": "$test_email",
  "full_name": "Test User (Updated)",
  "phone": "+1-555-0456"
EOF
)" \
        "true" \
        "UserService.UpdateUser - Update user profile"

    # 5. UpdateCurrentUser
    execute_grpc_call "smartticket.v1.UserService/UpdateCurrentUser" \
        "$(cat <<EOF
$metadata_json,
  "full_name": "Test User (Self Updated)",
  "phone": "+1-555-0789"
EOF
)" \
        "true" \
        "UserService.UpdateCurrentUser - Update current user profile"

    # 6. ListUsers
    execute_grpc_call "smartticket.v1.UserService/ListUsers" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  }
EOF
)" \
        "true" \
        "UserService.ListUsers - List users with pagination"

    # 7. UpdateUserStatus
    execute_grpc_call "smartticket.v1.UserService/UpdateUserStatus" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id",
  "is_active": true,
  "reason": "Test activation"
EOF
)" \
        "true" \
        "UserService.UpdateUserStatus - Activate user"

    # 8. ChangePassword
    execute_grpc_call "smartticket.v1.UserService/ChangePassword" \
        "$(cat <<EOF
$metadata_json,
  "current_password": "SecurePass123!",
  "new_password": "NewSecurePass456!"
EOF
)" \
        "true" \
        "UserService.ChangePassword - Change user password"

    # 9. ResetPassword
    execute_grpc_call "smartticket.v1.UserService/ResetPassword" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id",
  "temporary_password": "TempPass789!",
  "send_email": true
EOF
)" \
        "true" \
        "UserService.ResetPassword - Reset user password"

    # 10. GetUserPermissions
    execute_grpc_call "smartticket.v1.UserService/GetUserPermissions" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id"
EOF
)" \
        "true" \
        "UserService.GetUserPermissions - Get user permissions"

    # 11. DeleteUser
    execute_grpc_call "smartticket.v1.UserService/DeleteUser" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id"
EOF
)" \
        "true" \
        "UserService.DeleteUser - Delete user"
}

# ========================================
# AuthService 测试用例
# ========================================
test_auth_service() {
    log_info "=== Testing AuthService ==="

    local test_email=$(generate_email)

    # 1. Login
    execute_grpc_call "smartticket.v1.AuthService/Login" \
        "$(cat <<EOF
{
  "email": "$test_email",
  "password": "TestPass123!",
  "tenant_domain": "test.example.com",
  "remember_me": false
}
EOF
)" \
        "true" \
        "AuthService.Login - User login"

    # 2. RefreshToken
    execute_grpc_call "smartticket.v1.AuthService/RefreshToken" \
        "$(cat <<EOF
{
  "refresh_token": "test_refresh_token_$(date +%s)"
}
EOF
)" \
        "false" \
        "AuthService.RefreshToken - Refresh access token (expected to fail with invalid token)"
}

# ========================================
# TicketService 测试用例
# ========================================
test_ticket_service() {
    log_info "=== Testing TicketService ==="

    local tenant_id=$(generate_tenant_id)
    local user_id=$(generate_user_id)
    local ticket_id=$(generate_ticket_id)
    local metadata_json=$(generate_metadata "$tenant_id" "$user_id")

    # 1. CreateTicket
    execute_grpc_call "smartticket.v1.TicketService/CreateTicket" \
        "$(cat <<EOF
$metadata_json,
  "title": "Test Ticket - Login Issue",
  "description": "User cannot login to the system. Getting authentication error.",
  "priority": 2,
  "severity": 2,
  "category_id": "cat_auth_001",
  "contact_id": "$user_id",
  "tags": ["login", "authentication", "urgent"]
EOF
)" \
        "true" \
        "TicketService.CreateTicket - Create new ticket"

    # 2. GetTicket
    execute_grpc_call "smartticket.v1.TicketService/GetTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "include_comments": true
EOF
)" \
        "true" \
        "TicketService.GetTicket - Get ticket by ID"

    # 3. UpdateTicket
    execute_grpc_call "smartticket.v1.TicketService/UpdateTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "title": "Test Ticket - Login Issue (Updated)",
  "description": "Updated: User cannot login to the system. Getting authentication error. Additional details provided.",
  "priority": 3,
  "tags": ["login", "authentication", "urgent", "updated"]
EOF
)" \
        "true" \
        "TicketService.UpdateTicket - Update ticket"

    # 4. ListTickets
    execute_grpc_call "smartticket.v1.TicketService/ListTickets" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  },
  "statuses": ["1", "2"],
  "priorities": ["2", "3"]
EOF
)" \
        "true" \
        "TicketService.ListTickets - List tickets with filters"

    # 5. UpdateTicketStatus
    execute_grpc_call "smartticket.v1.TicketService/UpdateTicketStatus" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "status": 3,
  "comment": "Started investigating the login issue"
EOF
)" \
        "true" \
        "TicketService.UpdateTicketStatus - Update ticket status"

    # 6. AssignTicket
    execute_grpc_call "smartticket.v1.TicketService/AssignTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "assigned_to_id": "$user_id",
  "comment": "Assigning to senior support engineer"
EOF
)" \
        "true" \
        "TicketService.AssignTicket - Assign ticket to user"

    # 7. AddComment
    execute_grpc_call "smartticket.v1.TicketService/AddComment" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "content": "I've checked the logs and found the authentication service is responding normally. The issue might be with user credentials.",
  "is_internal": false
EOF
)" \
        "true" \
        "TicketService.AddComment - Add comment to ticket"

    # 8. GetComments
    execute_grpc_call "smartticket.v1.TicketService/GetComments" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "pagination": {
    "page_size": 20
  }
EOF
)" \
        "true" \
        "TicketService.GetComments - Get ticket comments"

    # 9. SearchTickets
    execute_grpc_call "smartticket.v1.TicketService/SearchTickets" \
        "$(cat <<EOF
$metadata_json,
  "query": "login authentication",
  "pagination": {
    "page_size": 10
  }
EOF
)" \
        "true" \
        "TicketService.SearchTickets - Search tickets"

    # 10. GetTicketStatistics
    execute_grpc_call "smartticket.v1.TicketService/GetTicketStatistics" \
        "$(cat <<EOF
$metadata_json,
  "date_from": "$(date -d '30 days ago' -Iseconds | cut -d'T' -f1)",
  "date_to": "$(date -Iseconds | cut -d'T' -f1)"
EOF
)" \
        "true" \
        "TicketService.GetTicketStatistics - Get ticket statistics"

    # 11. DeleteTicket
    execute_grpc_call "smartticket.v1.TicketService/DeleteTicket" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id"
EOF
)" \
        "true" \
        "TicketService.DeleteTicket - Delete ticket"
}

# ========================================
# SlaService 测试用例
# ========================================
test_sla_service() {
    log_info "=== Testing SlaService ==="

    local tenant_id=$(generate_tenant_id)
    local ticket_id=$(generate_ticket_id)
    local metadata_json=$(generate_metadata "$tenant_id")

    # 1. CreateSlaPolicy
    execute_grpc_call "smartticket.v1.SlaService/CreateSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "name": "Standard Support SLA",
  "description": "Standard SLA for regular support tickets",
  "priority": 2,
  "severity": 2,
  "response_time_minutes": 240,
  "resolution_time_minutes": 1440,
  "business_hours_only": true,
  "timezone": "UTC"
EOF
)" \
        "true" \
        "SlaService.CreateSlaPolicy - Create SLA policy"

    # 2. ListSlaPolicies
    execute_grpc_call "smartticket.v1.SlaService/ListSlaPolicies" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  },
  "is_active": true
EOF
)" \
        "true" \
        "SlaService.ListSlaPolicies - List SLA policies"

    # 3. GetSlaPolicy
    execute_grpc_call "smartticket.v1.SlaService/GetSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "policy_id": "sla_policy_$(date +%s)"
EOF
)" \
        "true" \
        "SlaService.GetSlaPolicy - Get SLA policy by ID"

    # 4. UpdateSlaPolicy
    execute_grpc_call "smartticket.v1.SlaService/UpdateSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "policy_id": "sla_policy_$(date +%s)",
  "name": "Standard Support SLA (Updated)",
  "description": "Updated SLA for regular support tickets",
  "response_time_minutes": 180,
  "resolution_time_minutes": 1200,
  "is_active": true
EOF
)" \
        "true" \
        "SlaService.UpdateSlaPolicy - Update SLA policy"

    # 5. GetSlaMetrics
    execute_grpc_call "smartticket.v1.SlaService/GetSlaMetrics" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id"
EOF
)" \
        "true" \
        "SlaService.GetSlaMetrics - Get SLA metrics for ticket"

    # 6. GetSlaDashboard
    execute_grpc_call "smartticket.v1.SlaService/GetSlaDashboard" \
        "$(cat <<EOF
$metadata_json,
  "start_date": "$(date -d '30 days ago' -Iseconds)",
  "end_date": "$(date -Iseconds)",
  "group_by": "day"
EOF
)" \
        "true" \
        "SlaService.GetSlaDashboard - Get SLA dashboard data"

    # 7. GetSlaBreaches
    execute_grpc_call "smartticket.v1.SlaService/GetSlaBreaches" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  },
  "breach_type": "response",
  "only_overdue": true,
  "start_date": "$(date -d '7 days ago' -Iseconds)",
  "end_date": "$(date -Iseconds)"
EOF
)" \
        "true" \
        "SlaService.GetSlaBreaches - Get SLA breach alerts"

    # 8. UpdateSlaMetrics
    execute_grpc_call "smartticket.v1.SlaService/UpdateSlaMetrics" \
        "$(cat <<EOF
$metadata_json,
  "ticket_id": "$ticket_id",
  "event_type": "first_response",
  "event_time": "$(date -Iseconds)"
EOF
)" \
        "true" \
        "SlaService.UpdateSlaMetrics - Update SLA metrics"

    # 9. DeleteSlaPolicy
    execute_grpc_call "smartticket.v1.SlaService/DeleteSlaPolicy" \
        "$(cat <<EOF
$metadata_json,
  "policy_id": "sla_policy_$(date +%s)"
EOF
)" \
        "true" \
        "SlaService.DeleteSlaPolicy - Delete SLA policy"
}

# ========================================
# KnowledgeService 测试用例
# ========================================
test_knowledge_service() {
    log_info "=== Testing KnowledgeService ==="

    local tenant_id=$(generate_tenant_id)
    local user_id=$(generate_user_id)
    local metadata_json=$(generate_metadata "$tenant_id" "$user_id")

    # 1. CreateArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/CreateArticle" \
        "$(cat <<EOF
$metadata_json,
  "title": "How to Fix Common Login Issues",
  "content": "This article explains common login issues and their solutions...",
  "summary": "Step-by-step guide to resolve login problems",
  "category_id": "cat_troubleshooting_001",
  "visibility": 1,
  "language": "en",
  "tags": ["login", "troubleshooting", "authentication"]
EOF
)" \
        "true" \
        "KnowledgeService.CreateArticle - Create knowledge article"

    # 2. ListArticles
    execute_grpc_call "smartticket.v1.KnowledgeService/ListArticles" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  },
  "statuses": [3],
  "visibilities": [1]
EOF
)" \
        "true" \
        "KnowledgeService.ListArticles - List published articles"

    # 3. GetArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/GetArticle" \
        "$(cat <<EOF
$metadata_json,
  "article_id": "article_$(date +%s)",
  "increment_view_count": true
EOF
)" \
        "true" \
        "KnowledgeService.GetArticle - Get article by ID"

    # 4. UpdateArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/UpdateArticle" \
        "$(cat <<EOF
$metadata_json,
  "article_id": "article_$(date +%s)",
  "title": "How to Fix Common Login Issues (Updated)",
  "content": "This article explains common login issues and their solutions. Updated with additional troubleshooting steps...",
  "summary": "Comprehensive step-by-step guide to resolve login problems",
  "comment": "Added new troubleshooting steps"
EOF
)" \
        "true" \
        "KnowledgeService.UpdateArticle - Update knowledge article"

    # 5. PublishArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/PublishArticle" \
        "$(cat <<EOF
$metadata_json,
  "article_id": "article_$(date +%s)",
  "comment": "Article ready for publication"
EOF
)" \
        "true" \
        "KnowledgeService.PublishArticle - Publish article"

    # 6. SearchArticles
    execute_grpc_call "smartticket.v1.KnowledgeService/SearchArticles" \
        "$(cat <<EOF
$metadata_json,
  "query": "login troubleshooting",
  "pagination": {
    "page_size": 10
  },
  "only_published": true
EOF
)" \
        "true" \
        "KnowledgeService.SearchArticles - Search knowledge articles"

    # 7. GetArticleSuggestions
    execute_grpc_call "smartticket.v1.KnowledgeService/GetArticleSuggestions" \
        "$(cat <<EOF
$metadata_json,
  "ticket_title": "Cannot login to system",
  "ticket_description": "User is unable to login with correct credentials",
  "ticket_tags": ["login", "authentication"],
  "limit": 5
EOF
)" \
        "true" \
        "KnowledgeService.GetArticleSuggestions - Get article suggestions for ticket"

    # 8. RateArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/RateArticle" \
        "$(cat <<EOF
$metadata_json,
  "article_id": "article_$(date +%s)",
  "is_helpful": true,
  "comment": "This article helped me resolve the issue"
EOF
)" \
        "true" \
        "KnowledgeService.RateArticle - Rate article helpfulness"

    # 9. GetCategories
    execute_grpc_call "smartticket.v1.KnowledgeService/GetCategories" \
        "$(cat <<EOF
$metadata_json
EOF
)" \
        "true" \
        "KnowledgeService.GetCategories - Get article categories"

    # 10. CreateCategory
    execute_grpc_call "smartticket.v1.KnowledgeService/CreateCategory" \
        "$(cat <<EOF
$metadata_json,
  "name": "Troubleshooting",
  "description": "Articles related to troubleshooting common issues",
  "icon": "wrench"
EOF
)" \
        "true" \
        "KnowledgeService.CreateCategory - Create article category"

    # 11. ArchiveArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/ArchiveArticle" \
        "$(cat <<EOF
$metadata_json,
  "article_id": "article_$(date +%s)",
  "reason": "Article outdated, replaced by newer version"
EOF
)" \
        "true" \
        "KnowledgeService.ArchiveArticle - Archive article"

    # 12. DeleteArticle
    execute_grpc_call "smartticket.v1.KnowledgeService/DeleteArticle" \
        "$(cat <<EOF
$metadata_json,
  "article_id": "article_$(date +%s)"
EOF
)" \
        "true" \
        "KnowledgeService.DeleteArticle - Delete article"
}

# ========================================
# RolePermissionService 测试用例
# ========================================
test_role_permission_service() {
    log_info "=== Testing RolePermissionService ==="

    local tenant_id=$(generate_tenant_id)
    local user_id=$(generate_user_id)
    local metadata_json=$(generate_metadata "$tenant_id" "$user_id")

    # 1. CreateRole
    execute_grpc_call "smartticket.v1.RolePermissionService/CreateRole" \
        "$(cat <<EOF
$metadata_json,
  "name": "Junior Support Engineer",
  "description": "Entry-level support engineer with limited permissions",
  "permission_ids": ["perm_ticket_view", "perm_ticket_create", "perm_comment_create"],
  "is_active": true
EOF
)" \
        "true" \
        "RolePermissionService.CreateRole - Create new role"

    # 2. ListRoles
    execute_grpc_call "smartticket.v1.RolePermissionService/ListRoles" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 10
  },
  "include_system_roles": true,
  "include_inactive": false
EOF
)" \
        "true" \
        "RolePermissionService.ListRoles - List roles"

    # 3. GetRole
    execute_grpc_call "smartticket.v1.RolePermissionService/GetRole" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)"
EOF
)" \
        "true" \
        "RolePermissionService.GetRole - Get role by ID"

    # 4. UpdateRole
    execute_grpc_call "smartticket.v1.RolePermissionService/UpdateRole" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)",
  "name": "Junior Support Engineer (Updated)",
  "description": "Entry-level support engineer with additional permissions",
  "is_active": true
EOF
)" \
        "true" \
        "RolePermissionService.UpdateRole - Update role"

    # 5. ListPermissions
    execute_grpc_call "smartticket.v1.RolePermissionService/ListPermissions" \
        "$(cat <<EOF
$metadata_json,
  "pagination": {
    "page_size": 50
  },
  "include_system_permissions": true
EOF
)" \
        "true" \
        "RolePermissionService.ListPermissions - List available permissions"

    # 6. GetRolePermissions
    execute_grpc_call "smartticket.v1.RolePermissionService/GetRolePermissions" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)"
EOF
)" \
        "true" \
        "RolePermissionService.GetRolePermissions - Get role permissions"

    # 7. AssignPermissionsToRole
    execute_grpc_call "smartticket.v1.RolePermissionService/AssignPermissionsToRole" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)",
  "permission_ids": ["perm_ticket_update", "perm_user_view"],
  "reason": "Adding ticket update and user view permissions"
EOF
)" \
        "true" \
        "RolePermissionService.AssignPermissionsToRole - Assign permissions to role"

    # 8. RemovePermissionsFromRole
    execute_grpc_call "smartticket.v1.RolePermissionService/RemovePermissionsFromRole" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)",
  "permission_ids": ["perm_ticket_update"],
  "reason": "Removing ticket update permissions temporarily"
EOF
)" \
        "true" \
        "RolePermissionService.RemovePermissionsFromRole - Remove permissions from role"

    # 9. AssignRoleToUser
    execute_grpc_call "smartticket.v1.RolePermissionService/AssignRoleToUser" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id",
  "role_id": "role_$(date +%s)",
  "expires_at": "$(date -d '+90 days' -Iseconds)",
  "reason": "Assigning junior support role"
EOF
)" \
        "true" \
        "RolePermissionService.AssignRoleToUser - Assign role to user"

    # 10. GetUserRoles
    execute_grpc_call "smartticket.v1.RolePermissionService/GetUserRoles" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id",
  "include_expired": false,
  "include_inactive": false
EOF
)" \
        "true" \
        "RolePermissionService.GetUserRoles - Get user roles"

    # 11. GetUsersWithRole
    execute_grpc_call "smartticket.v1.RolePermissionService/GetUsersWithRole" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)",
  "pagination": {
    "page_size": 10
  }
EOF
)" \
        "true" \
        "RolePermissionService.GetUsersWithRole - Get users with specific role"

    # 12. RemoveRoleFromUser
    execute_grpc_call "smartticket.v1.RolePermissionService/RemoveRoleFromUser" \
        "$(cat <<EOF
$metadata_json,
  "user_id": "$user_id",
  "role_id": "role_$(date +%s)",
  "reason": "Reassigning to different role"
EOF
)" \
        "true" \
        "RolePermissionService.RemoveRoleFromUser - Remove role from user"

    # 13. DeleteRole
    execute_grpc_call "smartticket.v1.RolePermissionService/DeleteRole" \
        "$(cat <<EOF
$metadata_json,
  "role_id": "role_$(date +%s)",
  "force_delete": false
EOF
)" \
        "true" \
        "RolePermissionService.DeleteRole - Delete role"
}

# ========================================
# 运行所有测试的主函数
# ========================================
run_all_grpc_tests() {
    log_info "Starting all gRPC service tests..."

    # 按服务分组执行测试
    test_tenant_service
    test_user_service
    test_auth_service
    test_ticket_service
    test_sla_service
    test_knowledge_service
    test_role_permission_service

    log_info "Completed all gRPC service tests"
}

# 如果需要单独测试某个服务，可以取消注释以下内容
# run_all_grpc_tests() {
#     test_tenant_service  # 只测试租户服务
# }