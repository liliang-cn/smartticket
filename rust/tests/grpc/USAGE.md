# gRPC E2E测试使用指南

本文档详细介绍如何使用SmartTicket的gRPC端到端测试系统。

## 🎯 概述

我们为SmartTicket项目创建了完整的gRPC E2E测试框架，使用grpcurl工具对所有gRPC接口进行测试。

**测试覆盖统计：**
- **TenantService**: 10个接口
- **UserService**: 11个接口
- **AuthService**: 2个接口
- **TicketService**: 11个接口
- **SlaService**: 9个接口
- **KnowledgeService**: 12个接口
- **RolePermissionService**: 13个接口

**总计：68个gRPC接口**

## 🚀 快速开始

### 1. 环境准备

#### 安装grpcurl
```bash
# macOS
brew install grpcurl

# 其他系统
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

#### 验证安装
```bash
grpcurl --version
# 应该输出: grpcurl 1.x.x
```

### 2. 启动gRPC服务

在项目根目录启动gRPC Gateway：

```bash
# 启动gRPC网关服务
cargo run --bin gateway

# 或者使用不同的端口
GRPC_GATEWAY_PORT=50052 cargo run --bin gateway
```

### 3. 运行测试

#### 方式1: 运行完整的gRPC测试套件
```bash
bash tests/grpc/grpc_e2e_test.sh
```

#### 方式2: 使用统一测试运行器
```bash
bash tests/run_all_tests.sh
```

#### 方式3: 先验证框架，再运行测试
```bash
# 先验证测试框架
bash tests/grpc/test_grpc_framework.sh

# 如果验证通过，运行完整测试
bash tests/grpc/grpc_e2e_test.sh
```

## 📊 测试结果解读

### 成功输出示例
```
🧪 SmartTicket gRPC E2E Tests
==================================
Total Tests: 68
Passed: 68
Failed: 0
Success Rate: 100%
🎉 All gRPC tests passed!

Results saved to:
  Log: test_results/grpc_e2e_20250117_103045.log
  Summary: test_results/grpc_e2e_summary_20250117_103045.json
```

### 失败处理
如果某些测试失败，查看详细日志：

```bash
# 查看最新测试日志
tail -f test_results/grpc_e2e_$(date +%Y%m%d)_*.log

# 或者查看JSON摘要
cat test_results/grpc_e2e_summary_$(date +%Y%m%d)_*.json | jq .
```

## 🔧 高级配置

### 自定义服务地址
```bash
export GRPC_GATEWAY_HOST=192.168.1.100
export GRPC_GATEWAY_PORT=50051
bash tests/grpc/grpc_e2e_test.sh
```

### 调试单个服务
如果你想只测试特定服务，可以修改测试脚本：

```bash
# 编辑 tests/grpc/grpc_test_cases.sh
# 在 run_all_grpc_tests() 函数中只保留想要测试的服务
```

### 自定义测试数据
所有测试数据都是动态生成的，包括：
- tenant_id: `tenant_<timestamp>_<random>`
- user_id: `user_<timestamp>_<random>`
- email: `test<random>@example.com`

## 📈 持续集成

### CI/CD集成
测试已集成到统一测试运行器，可以轻松集成到CI/CD流水线：

```yaml
# GitHub Actions 示例
- name: Run gRPC E2E Tests
  run: |
    # 启动服务
    cargo run --bin gateway &
    sleep 10

    # 运行测试
    bash tests/grpc/grpc_e2e_test.sh
```

### 性能监控
测试框架会记录每个API调用的响应时间，可以用于性能监控。

## 🐛 故障排除

### 常见问题

#### 1. "Cannot connect to gRPC service"
**解决方案：**
```bash
# 检查服务是否启动
lsof -i :50051

# 启动gRPC Gateway
cargo run --bin gateway
```

#### 2. "No smartticket.v1 services found"
**可能原因：**
- gRPC服务没有正确注册服务
- proto文件路径问题
- 服务版本不匹配

**解决方案：**
```bash
# 检查可用的所有服务
grpcurl -plaintext localhost:50051 list

# 检查特定服务的方法
grpcurl -plaintext localhost:50051 describe smartticket.v1.TenantService
```

#### 3. "Permission denied" 认证错误
**解决方案：**
某些接口需要有效的JWT token。检查认证服务是否正常运行。

#### 4. 测试数据冲突
**解决方案：**
测试框架使用随机数据生成器，但仍有极低概率冲突。如果遇到，重新运行测试即可。

### 调试技巧

#### 1. 查看原始gRPC响应
```bash
# 手动调用单个接口
grpcurl -plaintext -d '{
  "metadata": {
    "tenant_id": "test",
    "user_id": "test"
  },
  "pagination": {"page_size": 1}
}' localhost:50051 smartticket.v1.TenantService/ListTenants
```

#### 2. 启用详细日志
```bash
# 在测试脚本中添加调试输出
export GRPC_DEBUG=true
bash tests/grpc/grpc_e2e_test.sh
```

#### 3. 检查proto定义
```bash
# 查看服务定义
grpcurl -plaintext localhost:50051 describe smartticket.v1.TenantService

# 查看消息定义
grpcurl -plaintext localhost:50051 describe smartticket.v1.CreateTenantRequest
```

## 📝 添加新测试

### 步骤1: 了解proto定义
首先查看相关proto文件了解接口定义：

```bash
# 查看proto文件
cat proto/smartticket/tenant.proto
```

### 步骤2: 编写测试函数
在 `tests/grpc/grpc_test_cases.sh` 中添加测试：

```bash
test_new_feature() {
    log_info "=== Testing NewFeatureService ==="

    local tenant_id=$(generate_tenant_id)
    local metadata_json=$(generate_metadata "$tenant_id")

    execute_grpc_call "smartticket.v1.NewFeatureService/Create" \
        "$(cat <<EOF
$metadata_json,
  "name": "Test Feature",
  "description": "Test description"
EOF
)" \
        "true" \
        "NewFeatureService.Create - Create new feature"
}
```

### 步骤3: 注册测试
在 `run_all_grpc_tests()` 函数中调用新测试：

```bash
run_all_grpc_tests() {
    # ... 现有测试
    test_new_feature
}
```

## 🎯 最佳实践

### 1. 测试数据管理
- 使用唯一ID避免冲突
- 清理测试数据（如果需要）
- 使用合理的数据范围

### 2. 错误处理
- 区分预期失败和意外失败
- 提供有意义的错误信息
- 记录详细的错误上下文

### 3. 性能考虑
- 避免在短时间内发送大量请求
- 使用合理的超时设置
- 监控测试执行时间

### 4. 维护性
- 保持测试代码清晰
- 定期更新测试用例
- 同步proto文件变更

## 📞 支持

如果遇到问题或需要帮助：

1. 查看详细日志文件
2. 检查本故障排除指南
3. 在项目中提交issue
4. 联系开发团队

---

**注意：** 这个gRPC E2E测试系统是SmartTicket项目质量保证的重要组成部分，确保所有gRPC接口的正确性和稳定性。