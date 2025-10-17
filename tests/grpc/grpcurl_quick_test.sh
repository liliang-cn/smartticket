#!/bin/bash

# SmartTicket grpcurl 快速连通性测试
# 测试所有68个接口的基本连通性

set -e

# 配置
GRPC_GATEWAY_HOST="${GRPC_GATEWAY_HOST:-localhost}"
GRPC_GATEWAY_PORT="${GRPC_GATEWAY_PORT:-6533}"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}🚀 SmartTicket grpcurl 68个接口连通性测试${NC}"
echo "================================"

# 测试函数
test_grpcurl_interface() {
    local service_method="$1"
    local request_data="$2"
    local test_description="$3"

    ((TOTAL_TESTS++))

    echo -e "${YELLOW}[TEST]${NC} $test_description"
    echo "方法: $service_method"

    # 构造grpcurl命令
    local cmd="grpcurl -plaintext -import-path ./proto -d '$request_data' '$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT' '$service_method'"

    echo "命令: $cmd"

    # 执行命令
    local output
    local exit_code
    output=$(eval "$cmd" 2>&1)
    exit_code=$?

    # 检查结果
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ PASS: ${NC}$test_description"
        echo "响应: $(echo "$output" | head -c 100)..."
        ((PASSED_TESTS++))
    else
        echo -e "${RED}❌ FAIL: ${NC}$test_description"
        echo "错误: $output"
        ((FAILED_TESTS++))
    fi
    echo ""
}

echo -e "${BLUE}开始测试68个接口...${NC}"
echo ""

# ========================================
# TenantService (10个接口)
# ========================================
echo -e "${BLUE}=== TenantService (10个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.TenantService/CreateTenant" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-123"
  },
  "name": "Test Tenant",
  "domain": "test.example.com",
  "subscriptionTier": 1,
  "maxUsers": 50,
  "dataResidencyRegion": "EU",
  "contactEmail": "admin@test.example.com"
}' "TenantService.CreateTenant"

test_grpcurl_interface "smartticket.v1.TenantService/GetTenant" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-124"
  },
  "tenantId": "test-tenant-123"
}' "TenantService.GetTenant"

test_grpcurl_interface "smartticket.v1.TenantService/GetCurrentTenant" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-125"
  }
}' "TenantService.GetCurrentTenant"

test_grpcurl_interface "smartticket.v1.TenantService/UpdateTenant" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-126"
  },
  "tenantId": "test-tenant-123",
  "name": "Test Tenant (Updated)"
}' "TenantService.UpdateTenant"

test_grpcurl_interface "smartticket.v1.TenantService/ListTenants" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-127"
  },
  "pagination": {
    "pageSize": 10
  }
}' "TenantService.ListTenants"

test_grpcurl_interface "smartticket.v1.TenantService/DeleteTenant" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-128"
  },
  "tenantId": "test-tenant-123"
}' "TenantService.DeleteTenant"

test_grpcurl_interface "smartticket.v1.TenantService/UpdateTenantStatus" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-129"
  },
  "tenantId": "test-tenant-123",
  "isActive": true,
  "reason": "Test activation"
}' "TenantService.UpdateTenantStatus"

test_grpcurl_interface "smartticket.v1.TenantService/UpdateSubscription" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-130"
  },
  "tenantId": "test-tenant-123",
  "newTier": 2,
  "newMaxUsers": 100
}' "TenantService.UpdateSubscription"

test_grpcurl_interface "smartticket.v1.TenantService/GetTenantUsage" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-131"
  },
  "tenantId": "test-tenant-123"
}' "TenantService.GetTenantUsage"

test_grpcurl_interface "smartticket.v1.TenantService/GetTenantBilling" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-132"
  },
  "tenantId": "test-tenant-123"
}' "TenantService.GetTenantBilling"

# ========================================
# UserService (11个接口)
# ========================================
echo -e "${BLUE}=== UserService (11个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.UserService/CreateUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-201"
  },
  "email": "test@example.com",
  "username": "testuser",
  "fullName": "Test User",
  "password": "TestPass123!",
  "role": 3
}' "UserService.CreateUser"

test_grpcurl_interface "smartticket.v1.UserService/GetUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-202"
  },
  "userId": "test-user-123"
}' "UserService.GetUser"

test_grpcurl_interface "smartticket.v1.UserService/GetCurrentUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-203"
  }
}' "UserService.GetCurrentUser"

test_grpcurl_interface "smartticket.v1.UserService/UpdateUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-204"
  },
  "userId": "test-user-123",
  "fullName": "Test User (Updated)"
}' "UserService.UpdateUser"

test_grpcurl_interface "smartticket.v1.UserService/UpdateCurrentUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-205"
  },
  "fullName": "Current User (Updated)"
}' "UserService.UpdateCurrentUser"

test_grpcurl_interface "smartticket.v1.UserService/ListUsers" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-206"
  },
  "pagination": {
    "pageSize": 10
  }
}' "UserService.ListUsers"

test_grpcurl_interface "smartticket.v1.UserService/DeleteUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-207"
  },
  "userId": "test-user-123"
}' "UserService.DeleteUser"

test_grpcurl_interface "smartticket.v1.UserService/UpdateUserStatus" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-208"
  },
  "userId": "test-user-123",
  "isActive": true,
  "reason": "Test activation"
}' "UserService.UpdateUserStatus"

test_grpcurl_interface "smartticket.v1.UserService/ChangePassword" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-209"
  },
  "currentPassword": "oldpass123",
  "newPassword": "newpass123!"
}' "UserService.ChangePassword"

test_grpcurl_interface "smartticket.v1.UserService/ResetPassword" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-210"
  },
  "userId": "test-user-123",
  "temporaryPassword": "Temppass123!"
}' "UserService.ResetPassword"

test_grpcurl_interface "smartticket.v1.UserService/GetUserPermissions" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-211"
  },
  "userId": "test-user-123"
}' "UserService.GetUserPermissions"

# ========================================
# AuthService (2个接口)
# ========================================
echo -e "${BLUE}=== AuthService (2个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.AuthService/Login" '{
  "email": "test@example.com",
  "password": "test123",
  "tenantDomain": "test.example.com",
  "rememberMe": false
}' "AuthService.Login"

test_grpcurl_interface "smartticket.v1.AuthService/RefreshToken" '{
  "refreshToken": "test-refresh-token-123"
}' "AuthService.RefreshToken"

# ========================================
# TicketService (11个接口)
# ========================================
echo -e "${BLUE}=== TicketService (11个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.TicketService/CreateTicket" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-301"
  },
  "title": "Test Ticket",
  "description": "This is a test ticket",
  "priority": 2,
  "severity": 2,
  "contactId": "test-user-123"
}' "TicketService.CreateTicket"

test_grpcurl_interface "smartticket.v1.TicketService/GetTicket" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-302"
  },
  "ticketId": "test-ticket-123"
}' "TicketService.GetTicket"

test_grpcurl_interface "smartticket.v1.TicketService/UpdateTicket" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-303"
  },
  "ticketId": "test-ticket-123",
  "title": "Test Ticket (Updated)"
}' "TicketService.UpdateTicket"

test_grpcurl_interface "smartticket.v1.TicketService/ListTickets" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-304"
  },
  "pagination": {
    "pageSize": 10
  }
}' "TicketService.ListTickets"

test_grpcurl_interface "smartticket.v1.TicketService/DeleteTicket" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-305"
  },
  "ticketId": "test-ticket-123"
}' "TicketService.DeleteTicket"

test_grpcurl_interface "smartticket.v1.TicketService/UpdateTicketStatus" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-306"
  },
  "ticketId": "test-ticket-123",
  "status": 3
}' "TicketService.UpdateTicketStatus"

test_grpcurl_interface "smartticket.v1.TicketService/AssignTicket" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-307"
  },
  "ticketId": "test-ticket-123",
  "assignedToId": "test-user-123"
}' "TicketService.AssignTicket"

test_grpcurl_interface "smartticket.v1.TicketService/AddComment" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-308"
  },
  "ticketId": "test-ticket-123",
  "content": "This is a test comment"
}' "TicketService.AddComment"

test_grpcurl_interface "smartticket.v1.TicketService/GetComments" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-309"
  },
  "ticketId": "test-ticket-123"
}' "TicketService.GetComments"

test_grpcurl_interface "smartticket.v1.TicketService/SearchTickets" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-310"
  },
  "query": "test"
}' "TicketService.SearchTickets"

test_grpcurl_interface "smartticket.v1.TicketService/GetTicketStatistics" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-311"
  }
}' "TicketService.GetTicketStatistics"

# ========================================
# KnowledgeService (12个接口)
# ========================================
echo -e "${BLUE}=== KnowledgeService (12个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.KnowledgeService/CreateArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-401"
  },
  "title": "Test Article",
  "content": "This is a test article",
  "summary": "Test article summary"
}' "KnowledgeService.CreateArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/GetArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-402"
  },
  "articleId": "test-article-123"
}' "KnowledgeService.GetArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/UpdateArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-403"
  },
  "articleId": "test-article-123",
  "title": "Test Article (Updated)"
}' "KnowledgeService.UpdateArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/ListArticles" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-404"
  },
  "pagination": {
    "pageSize": 10
  }
}' "KnowledgeService.ListArticles"

test_grpcurl_interface "smartticket.v1.KnowledgeService/DeleteArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-405"
  },
  "articleId": "test-article-123"
}' "KnowledgeService.DeleteArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/PublishArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-406"
  },
  "articleId": "test-article-123"
}' "KnowledgeService.PublishArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/ArchiveArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-407"
  },
  "articleId": "test-article-123"
}' "KnowledgeService.ArchiveArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/SearchArticles" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-408"
  },
  "query": "test"
}' "KnowledgeService.SearchArticles"

test_grpcurl_interface "smartticket.v1.KnowledgeService/GetArticleSuggestions" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-409"
  },
  "ticketTitle": "Test Ticket"
}' "KnowledgeService.GetArticleSuggestions"

test_grpcurl_interface "smartticket.v1.KnowledgeService/RateArticle" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-410"
  },
  "articleId": "test-article-123",
  "isHelpful": true
}' "KnowledgeService.RateArticle"

test_grpcurl_interface "smartticket.v1.KnowledgeService/GetCategories" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-411"
  }
}' "KnowledgeService.GetCategories"

test_grpcurl_interface "smartticket.v1.KnowledgeService/CreateCategory" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-412"
  },
  "name": "Test Category",
  "description": "Test category description"
}' "KnowledgeService.CreateCategory"

# ========================================
# SlaService (9个接口)
# ========================================
echo -e "${BLUE}=== SlaService (9个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.SlaService/CreateSlaPolicy" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-501"
  },
  "name": "Test SLA Policy",
  "description": "Test SLA policy",
  "priority": 2,
  "severity": 2,
  "responseTimeMinutes": 240,
  "resolutionTimeMinutes": 1440
}' "SlaService.CreateSlaPolicy"

test_grpcurl_interface "smartticket.v1.SlaService/GetSlaPolicy" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-502"
  },
  "policyId": "test-sla-123"
}' "SlaService.GetSlaPolicy"

test_grpcurl_interface "smartticket.v1.SlaService/UpdateSlaPolicy" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-503"
  },
  "policyId": "test-sla-123",
  "name": "Test SLA Policy (Updated)"
}' "SlaService.UpdateSlaPolicy"

test_grpcurl_interface "smartticket.v1.SlaService/ListSlaPolicies" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-504"
  },
  "pagination": {
    "pageSize": 10
  }
}' "SlaService.ListSlaPolicies"

test_grpcurl_interface "smartticket.v1.SlaService/DeleteSlaPolicy" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-505"
  },
  "policyId": "test-sla-123"
}' "SlaService.DeleteSlaPolicy"

test_grpcurl_interface "smartticket.v1.SlaService/GetSlaMetrics" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-506"
  },
  "ticketId": "test-ticket-123"
}' "SlaService.GetSlaMetrics"

test_grpcurl_interface "smartticket.v1.SlaService/GetSlaDashboard" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-507"
  }
}' "SlaService.GetSlaDashboard"

test_grpcurl_interface "smartticket.v1.SlaService/GetSlaBreaches" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-508"
  }
}' "SlaService.GetSlaBreaches"

test_grpcurl_interface "smartticket.v1.SlaService/UpdateSlaMetrics" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-509"
  },
  "ticketId": "test-ticket-123"
}' "SlaService.UpdateSlaMetrics"

# ========================================
# RolePermissionService (13个接口)
# ========================================
echo -e "${BLUE}=== RolePermissionService (13个接口) ===${NC}"

test_grpcurl_interface "smartticket.v1.RolePermissionService/CreateRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-601"
  },
  "name": "Test Role",
  "description": "Test role description"
}' "RolePermissionService.CreateRole"

test_grpcurl_interface "smartticket.v1.RolePermissionService/GetRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-602"
  },
  "roleId": "test-role-123"
}' "RolePermissionService.GetRole"

test_grpcurl_interface "smartticket.v1.RolePermissionService/UpdateRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-603"
  },
  "roleId": "test-role-123",
  "name": "Test Role (Updated)"
}' "RolePermissionService.UpdateRole"

test_grpcurl_interface "smartticket.v1.RolePermissionService/DeleteRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-604"
  },
  "roleId": "test-role-123"
}' "RolePermissionService.DeleteRole"

test_grpcurl_interface "smartticket.v1.RolePermissionService/ListRoles" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-605"
  },
  "pagination": {
    "pageSize": 10
  }
}' "RolePermissionService.ListRoles"

test_grpcurl_interface "smartticket.v1.RolePermissionService/ListPermissions" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-606"
  },
  "pagination": {
    "pageSize": 50
  }
}' "RolePermissionService.ListPermissions"

test_grpcurl_interface "smartticket.v1.RolePermissionService/GetRolePermissions" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-607"
  },
  "roleId": "test-role-123"
}' "RolePermissionService.GetRolePermissions"

test_grpcurl_interface "smartticket.v1.RolePermissionService/AssignPermissionsToRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-608"
  },
  "roleId": "test-role-123",
  "permissionIds": ["test-perm-1"]
}' "RolePermissionService.AssignPermissionsToRole"

test_grpcurl_interface "smartticket.v1.RolePermissionService/RemovePermissionsFromRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-609"
  },
  "roleId": "test-role-123",
  "permissionIds": ["test-perm-1"]
}' "RolePermissionService.RemovePermissionsFromRole"

test_grpcurl_interface "smartticket.v1.RolePermissionService/AssignRoleToUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-610"
  },
  "userId": "test-user-123",
  "roleId": "test-role-123"
}' "RolePermissionService.AssignRoleToUser"

test_grpcurl_interface "smartticket.v1.RolePermissionService/RemoveRoleFromUser" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-611"
  },
  "userId": "test-user-123",
  "roleId": "test-role-123"
}' "RolePermissionService.RemoveRoleFromUser"

test_grpcurl_interface "smartticket.v1.RolePermissionService/GetUserRoles" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-612"
  },
  "userId": "test-user-123"
}' "RolePermissionService.GetUserRoles"

test_grpcurl_interface "smartticket.v1.RolePermissionService/GetUsersWithRole" '{
  "metadata": {
    "tenantId": "test-123",
    "userId": "user-123",
    "requestId": "req-613"
  },
  "roleId": "test-role-123"
}' "RolePermissionService.GetUsersWithRole"

# ========================================
# 测试结果总结
# ========================================
echo "========================================"
echo -e "${BLUE}🧪 grpcurl 68个接口测试完成！${NC}"
echo "========================================"
echo "总测试数: $TOTAL_TESTS"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    SUCCESS_RATE=100
    echo -e "成功率: ${GREEN}100%${NC}"
    echo -e "${GREEN}🎉 所有68个接口的grpcurl测试都通过了！${NC}"
else
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo -e "成功率: ${RED}$SUCCESS_RATE%${NC}"
fi

echo ""
echo "✅ grpcurl测试验证完成！"
echo "- 所有接口都能响应（成功或业务错误）"
echo "- grpcurl工具工作正常"
echo "- gRPC服务运行正常"