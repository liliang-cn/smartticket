# SmartTicket 测试框架

这个测试框架为SmartTicket项目提供了全面的测试基础设施，包括单元测试、集成测试和性能测试。

## 🏗️ 测试架构

```
tests/
├── common/                    # 通用测试工具和配置
│   ├── mod.rs                # 测试上下文和核心功能
│   ├── fixtures.rs           # 测试数据夹具
│   └── assertions.rs         # 自定义断言函数
├── integration/              # 集成测试
│   └── multi_tenant_isolation_test.rs
├── performance/              # 性能测试
│   └── concurrent_ticket_operations_test.rs
├── e2e/                     # 端到端测试 (Shell脚本)
│   ├── config/              # E2E配置文件
│   ├── utils/               # E2E工具函数
│   ├── integration/         # 集成测试脚本
│   ├── single-service/      # 单服务测试脚本
│   ├── performance/         # 性能测试脚本
│   └── reports/             # 测试报告
└── README.md                # 本文档
```

## 🚀 快速开始

### 1. 启动测试基础设施

```bash
# 启动PostgreSQL和Redis测试容器
./scripts/test.sh start
```

### 2. 运行所有测试

#### 使用统一测试运行器（推荐）
```bash
# 运行所有测试（单元测试 + 集成测试 + 性能测试）
./tests/run_all_tests.sh

# 包含 E2E 测试
./tests/run_all_tests.sh --include-e2e

# 快速测试（跳过性能测试）
./tests/run_all_tests.sh --fast

# 详细输出
./tests/run_all_tests.sh --verbose

# 仅运行特定类型的测试
./tests/run_all_tests.sh --unit-only
./tests/run_all_tests.sh --integration-only
./tests/run_all_tests.sh --performance-only
./tests/run_all_tests.sh --e2e-only
```

#### 传统方式
```bash
# 运行所有测试（单元测试 + 集成测试 + 性能测试）
./scripts/test.sh all

# 或者只运行特定类型的测试
./scripts/test.sh unit          # 只运行单元测试
./scripts/test.sh integration   # 只运行集成测试
./scripts/test.sh performance   # 只运行性能测试

# 运行 E2E 测试
cd tests/e2e
./run_e2e_master.sh            # 运行所有 E2E 测试
./run_e2e_master.sh --fast     # 运行快速 E2E 测试
```

### 3. 查看测试覆盖率

```bash
# 生成HTML覆盖率报告
./scripts/test.sh coverage

# 报告将生成在: target/coverage/tarpaulin-report.html
```

### 4. 清理测试环境

```bash
# 停止容器并清理测试数据
./scripts/test.sh clean

# 或使用统一的清理工具
./tests/cleanup_tests.sh --all

# 仅清理特定内容
./tests/cleanup_tests.sh --db      # 清理数据库
./tests/cleanup_tests.sh --e2e     # 清理 E2E 结果
./tests/cleanup_tests.sh --cache   # 清理缓存
```

## 🧪 测试组件

### 通用测试工具 (`tests/common/`)

#### TestContext
为所有测试提供统一的测试环境设置：

```rust
use tests::common::TestContext;

#[tokio::test]
async fn my_test() -> Result<()> {
    let mut context = TestContext::new().await?;

    // 使用context.db进行数据库操作
    // 使用context.redis进行Redis操作
    // 使用context.jwt_service进行JWT操作

    // 测试完成后自动清理
    context.cleanup().await?;
    Ok(())
}
```

#### 测试夹具 (Fixtures)
提供预配置的测试数据：

```rust
use tests::common::fixtures;

// 创建测试工单
let ticket = fixtures::create_test_ticket(
    tenant_id,
    contact_id,
    created_by_id,
);

// 创建测试知识库文章
let article = fixtures::create_test_knowledge_article(
    tenant_id,
    author_id,
);
```

#### 断言工具
提供自定义断言函数：

```rust
use tests::common::assertions;

// 验证错误类型
assertions::assert_error_type(&error, "NotFound");

// 验证租户隔离
assertions::assert_tenant_isolation(tenant_id, resource_tenant_id);
```

### 集成测试

#### 多租户隔离测试
验证租户数据隔离的完整性：

- ✅ 租户用户只能访问自己的数据
- ✅ 管理员只能访问自己租户的数据
- ✅ 超级管理员可以跨租户访问
- ✅ JWT令牌包含正确的租户信息
- ✅ 跨租户访问尝试被正确阻止

### 性能测试

#### 并发操作测试
验证系统在高并发下的性能：

- 🎯 并发工单创建：50个并发工作线程，每个创建20个工单
- 🎯 并发读取操作：20个并发读取器，每个读取50个工单
- 🎯 全文搜索性能：多种搜索查询的性能基准测试

## 📊 性能基准

### 当前性能目标

| 操作类型 | 并发数 | 目标吞吐量 | 超时限制 |
|---------|--------|-----------|----------|
| 工单创建 | 50个工作者 | >10 tickets/sec | <30秒 |
| 工单读取 | 20个读取器 | >100 reads/sec | <10秒 |
| 全文搜索 | N/A | <100ms/查询 | N/A |

## 🔧 环境配置

### 测试环境变量

测试框架会自动设置以下环境变量：

```bash
# 数据库配置
TEST_DB_HOST=localhost
TEST_DB_PORT=5433
TEST_DB_NAME=smartticket_test
TEST_DB_USER=postgres
TEST_DB_PASSWORD=postgres

# Redis配置
TEST_REDIS_HOST=localhost
TEST_REDIS_PORT=6380

# 应用配置
ENVIRONMENT=test
RUST_LOG=debug
RUST_BACKTRACE=1
```

### Docker测试容器

测试使用独立的Docker容器：

- **PostgreSQL**: 端口5433，数据库`smartticket_test`
- **Redis**: 端口6380，数据库1

## 🧪 运行单个测试

### 单元测试

```bash
# 运行特定模块的单元测试
cargo test --package smartticket-shared-config
cargo test --package smartticket-shared-auth
cargo test --package smartticket-shared-database

# 运行特定测试函数
cargo test test_jwt_token_generation --package smartticket-shared-auth
```

### 集成测试

```bash
# 运行所有集成测试
cargo test --test integration

# 运行特定集成测试
cargo test multi_tenant_isolation --test integration
```

### 性能测试

```bash
# 运行所有性能测试
cargo test --test performance

# 运行特定性能测试
cargo test concurrent_ticket_creation --test performance
```

## 🔍 调试测试

### 启用详细日志

```bash
# 启用调试日志
RUST_LOG=debug cargo test

# 启用SQL查询日志
RUST_LOG=sqlx=debug cargo test

# 显示完整错误回溯
RUST_BACKTRACE=1 cargo test
```

### 测试数据库访问

```bash
# 连接到测试数据库
docker-compose -f docker/docker-compose.test.yml exec postgres-test psql -U postgres -d smartticket_test

# 连接到测试Redis
docker-compose -f docker/docker-compose.test.yml exec redis-test redis-cli
```

## 📋 测试清单

在提交代码前，确保以下测试都通过：

- [ ] 所有单元测试通过
- [ ] 所有集成测试通过
- [ ] 性能测试满足基准要求
- [ ] 代码覆盖率 > 75%
- [ ] 没有内存泄漏或资源未释放

## 🚨 故障排除

### 常见问题

1. **测试数据库连接失败**
   ```bash
   # 确保测试容器正在运行
   docker-compose -f docker/docker-compose.test.yml ps

   # 查看容器日志
   docker-compose -f docker/docker-compose.test.yml logs postgres-test
   ```

2. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :5433
   lsof -i :6380

   # 停止占用端口的服务
   ./scripts/test.sh clean
   ```

3. **权限错误**
   ```bash
   # 确保测试脚本有执行权限
   chmod +x scripts/test.sh
   ```

4. **测试挂起**
   ```bash
   # 检查数据库和Redis健康状态
   ./scripts/test.sh start

   # 查看实时日志
   docker-compose -f docker/docker-compose.test.yml logs -f
   ```

### 性能测试故障排除

1. **性能测试失败**
   - 检查系统资源使用情况
   - 减少并发工作者数量
   - 查看数据库慢查询日志

2. **测试超时**
   - 增加测试超时时间
   - 检查网络连接
   - 优化测试数据量

## 📈 测试报告

测试完成后，可以生成以下报告：

- **单元测试报告**: `cargo test -- --nocapture`
- **集成测试报告**: `cargo test --test integration -- --nocapture`
- **性能测试报告**: 包含吞吐量、延迟等指标
- **覆盖率报告**: `target/coverage/tarpaulin-report.html`

## 🤝 贡献指南

### 添加新测试

1. **单元测试**: 在相应的crate中添加`#[cfg(test)]`模块
2. **集成测试**: 在`tests/integration/`目录下创建新文件
3. **性能测试**: 在`tests/performance/`目录下创建新文件

### 测试命名规范

- 单元测试: `test_[functionality]_[scenario]`
- 集成测试: `test_[feature]_[integration_scenario]`
- 性能测试: `benchmark_[operation]_[condition]`

### 测试最佳实践

1. 使用`TestContext`确保测试隔离
2. 在测试后清理测试数据
3. 使用有意义的断言消息
4. 测试边界条件和错误情况
5. 保持测试快速和确定性

## 📚 更多资源

- [Rust测试文档](https://doc.rust-lang.org/book/ch11-00-testing.html)
- [SQLx测试指南](https://docs.rs/sqlx/latest/sqlx/test/index.html)
- [Tokio测试模式](https://tokio.rs/tokio/topics/testing)
- [Docker Compose文档](https://docs.docker.com/compose/)