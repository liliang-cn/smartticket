# SmartTicket gRPC E2E Tests

本目录包含使用grpcurl对所有gRPC接口进行端到端测试的框架和测试用例。

## 🏗️ 架构

- `grpc_e2e_test.sh` - 主测试框架脚本
- `grpc_test_cases.sh` - 所有gRPC服务的测试用例定义
- `README.md` - 本文档

## 📋 测试覆盖

### TenantService (8个接口)
- ✅ CreateTenant
- ✅ GetTenant
- ✅ GetCurrentTenant
- ✅ UpdateTenant
- ✅ ListTenants
- ✅ DeleteTenant
- ✅ UpdateTenantStatus
- ✅ UpdateSubscription
- ✅ GetTenantUsage
- ✅ GetTenantBilling

### UserService (11个接口)
- ✅ CreateUser
- ✅ GetUser
- ✅ GetCurrentUser
- ✅ UpdateUser
- ✅ UpdateCurrentUser
- ✅ ListUsers
- ✅ DeleteUser
- ✅ UpdateUserStatus
- ✅ ChangePassword
- ✅ ResetPassword
- ✅ GetUserPermissions

### AuthService (2个接口)
- ✅ Login
- ✅ RefreshToken

### TicketService (11个接口)
- ✅ CreateTicket
- ✅ GetTicket
- ✅ UpdateTicket
- ✅ ListTickets
- ✅ DeleteTicket
- ✅ UpdateTicketStatus
- ✅ AssignTicket
- ✅ AddComment
- ✅ GetComments
- ✅ SearchTickets
- ✅ GetTicketStatistics

### SlaService (9个接口)
- ✅ CreateSlaPolicy
- ✅ GetSlaPolicy
- ✅ UpdateSlaPolicy
- ✅ ListSlaPolicies
- ✅ DeleteSlaPolicy
- ✅ GetSlaMetrics
- ✅ GetSlaDashboard
- ✅ GetSlaBreaches
- ✅ UpdateSlaMetrics

### KnowledgeService (12个接口)
- ✅ CreateArticle
- ✅ GetArticle
- ✅ UpdateArticle
- ✅ ListArticles
- ✅ DeleteArticle
- ✅ PublishArticle
- ✅ ArchiveArticle
- ✅ SearchArticles
- ✅ GetArticleSuggestions
- ✅ RateArticle
- ✅ GetCategories
- ✅ CreateCategory

### RolePermissionService (13个接口)
- ✅ CreateRole
- ✅ GetRole
- ✅ UpdateRole
- ✅ DeleteRole
- ✅ ListRoles
- ✅ ListPermissions
- ✅ GetRolePermissions
- ✅ AssignPermissionsToRole
- ✅ RemovePermissionsFromRole
- ✅ AssignRoleToUser
- ✅ RemoveRoleFromUser
- ✅ GetUserRoles
- ✅ GetUsersWithRole

**总计：66个gRPC接口**

## 🚀 使用方法

### 1. 安装grpcurl

```bash
# macOS
brew install grpcurl

# 或使用Go
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### 2. 启动gRPC Gateway

```bash
# 在项目根目录
cargo run --bin gateway
```

### 3. 运行测试

```bash
# 运行所有gRPC测试
./tests/grpc/grpc_e2e_test.sh

# 或者使用统一测试运行器
bash tests/run_all_tests.sh
```

### 4. 环境变量配置

```bash
# 可选：自定义gRPC服务地址
export GRPC_GATEWAY_HOST=localhost
export GRPC_GATEWAY_PORT=50051

# 可选：自定义proto文件目录
export PROTO_DIR=./proto

# 可选：自定义测试结果目录
export TEST_RESULTS_DIR=./test_results
```

## 📊 输出结果

测试完成后会生成以下文件：

- `test_results/grpc_e2e_<timestamp>.log` - 详细测试日志
- `test_results/grpc_e2e_summary_<timestamp>.json` - 测试结果摘要（JSON格式）

### 测试结果示例

```json
{
  "timestamp": "2025-01-17T10:30:45.123Z",
  "host": "localhost:50051",
  "tests": [
    {
      "test": "TenantService.CreateTenant - Create new tenant",
      "method": "smartticket.v1.TenantService/CreateTenant",
      "status": "PASS",
      "output": "{...}"
    }
  ]
}
```

## 🔧 故障排除

### 1. grpcurl命令失败

确保gRPC服务正在运行：

```bash
# 检查服务是否可达
grpcurl -plaintext localhost:50051 list
```

### 2. 连接被拒绝

检查端口配置和服务启动状态：

```bash
# 检查端口是否被占用
lsof -i :50051

# 启动gRPC Gateway
cargo run --bin gateway
```

### 3. 认证错误

某些接口可能需要有效的JWT token。确保认证服务正常运行。

### 4. Proto文件未找到

确保proto文件路径正确：

```bash
export PROTO_DIR=./proto
```

## 🎯 测试策略

### 测试数据生成
- 每次测试生成唯一的tenant_id、user_id、ticket_id
- 使用随机邮箱和用户名避免冲突
- 生成有效的时间戳和元数据

### 错误处理
- 预期失败的测试会被标记为PASS
- 意外的API错误会被标记为FAIL
- 详细的错误信息记录在日志中

### 分页和过滤
- 列表接口使用合理的分页参数
- 包含常见的过滤条件测试
- 测试排序功能

## 📝 添加新测试

1. 在`grpc_test_cases.sh`中添加新的测试函数
2. 按照现有模式构造请求数据
3. 使用`execute_grpc_call`函数执行测试
4. 在`run_all_grpc_tests`中调用新测试

### 测试函数模板

```bash
test_new_service() {
    log_info "=== Testing NewService ==="

    local tenant_id=$(generate_tenant_id)
    local metadata_json=$(generate_metadata "$tenant_id")

    execute_grpc_call "smartticket.v1.NewService/Method" \
        "$(cat <<EOF
$metadata_json,
  "field1": "value1",
  "field2": "value2"
EOF
)" \
        "true" \
        "NewService.Method - Test description"
}
```

## 📈 持续集成

这些测试已集成到项目的统一测试运行器中，可以通过CI/CD流水线自动运行。

```bash
# 检查测试状态
bash tests/run_all_tests.sh
```

## 🤝 贡献

欢迎添加更多测试用例或改进测试框架。请确保：

1. 新测试遵循现有模式
2. 使用适当的数据生成函数
3. 包含清晰的测试描述
4. 处理预期的错误情况