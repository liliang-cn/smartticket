# SmartTicket 后端开发计划

## 项目概览

SmartTicket 后端基于 Rust + gRPC 微服务架构，为 B2B 多租户工单与知识协作平台提供技术支持。本文档详细规划后端开发的实施路径、技术选型、开发规范和质量标准。

## 技术栈与架构

### 核心技术栈
- **编程语言**: Rust 1.75+
- **微服务框架**: tonic + gRPC
- **Web框架**: Axum (HTTP API 支持)
- **数据库**: PostgreSQL 15+ (多租户 RLS)
- **向量数据库**: PgVector / Qdrant
- **缓存**: Redis 7+
- **消息队列**: Kafka / NATS
- **对象存储**: S3 兼容 (AWS/GCP/MinIO)
- **监控**: Prometheus + Grafana + OpenTelemetry
- **日志**: tracing + Loki

### 微服务架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        gRPC Gateway                        │
│                    (Envoy/Traefik + mTLS)                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
┌───────▼───────┐ ┌───▼────┐ ┌───────▼───────┐
│ Core Service  │ │ AI     │ │ Platform       │
│ (Ticket +     │ │ Service│ │ Service       │
│ Knowledge)    │ │ (RAG)  │ │ (Auth +        │
│               │ │        │ │ Integration)   │
└───────────────┘ └────────┘ └───────────────┘
        │             │             │
        └─────────────┼─────────────┘
                      │
        ┌─────────────▼─────────────┐
        │   Notification Service     │
        │ (Email/Chat/Push)          │
        └───────────────────────────┘
```

## 项目结构

```
smartticket-backend/
├── Cargo.toml                 # 工作空间配置
├── README.md                  # 项目说明
├── .github/                   # GitHub Actions CI/CD
│   └── workflows/
│       ├── ci.yml            # 持续集成
│       ├── security.yml       # 安全扫描
│       └── deploy.yml         # 部署流水线
├── crates/                    # 微服务模块
│   ├── gateway/              # gRPC 网关
│   ├── core/                 # 核心业务逻辑
│   ├── ai/                   # RAG/LLM 服务
│   ├── platform/             # 平台服务
│   ├── notification/         # 通知服务
│   └── shared/               # 共享组件
│       ├── config/           # 配置管理
│       ├── database/         # 数据库模块
│       ├── auth/             # 认证授权
│       ├── metrics/          # 监控指标
│       └── error/            # 错误处理
├── proto/                     # Protocol Buffers 定义
│   ├── smartticket/          # 主 proto 包
│   └── google/               # Google API 扩展
├── migrations/                # 数据库迁移脚本
├── config/                    # 配置文件
│   ├── development.yaml
│   ├── staging.yaml
│   └── production.yaml
├── tests/                     # 测试套件
│   ├── integration/          # 集成测试
│   ├── e2e/                  # 端到端测试
│   └── performance/          # 性能测试
├── docs/                      # 文档
│   ├── api/                  # API 文档
│   ├── deployment/           # 部署文档
│   └── development/          # 开发文档
├── scripts/                   # 开发脚本
│   ├── setup.sh             # 环境设置
│   ├── test.sh              # 测试脚本
│   └── deploy.sh            # 部署脚本
└── docker/                    # Docker 配置
    ├── Dockerfile.dev
    ├── Dockerfile.prod
    └── docker-compose.yml
```

## 开发阶段规划

### 阶段 0: 基础设施搭建 (4-6周)

**目标**: 建立开发环境和基础架构

**主要任务**:
1. **项目初始化**
   - 创建 Cargo 工作空间
   - 配置 proto 文件结构
   - 设置代码格式化和 Lint 规则

2. **数据库设计**
   - PostgreSQL schema 设计
   - 多租户 RLS 策略实现
   - 数据库迁移脚本编写

3. **基础服务框架**
   - gRPC 服务模板
   - 配置管理系统
   - 日志和监控基础设施

4. **认证授权系统**
   - JWT/OIDC 集成
   - RBAC 权限控制
   - 多租户数据隔离

**交付物**:
- [x] 完整的项目结构 (Cargo workspace + 微服务架构)
- [x] 基础的数据库 schema (PostgreSQL + 多租户 RLS)
- [x] 可运行的最小服务框架 (所有共享库编译通过)
- [x] 基础的 CI/CD 流水线 (Docker + 构建脚本)

**当前状态** (2025-01-15):
✅ **基础设施搭建完成** - 所有共享库编译成功，包括：
- ✅ `smartticket-shared-config`: 配置管理系统
- ✅ `smartticket-shared-database`: 数据库连接和迁移
- ✅ `smartticket-shared-auth`: 认证授权系统 (JWT + RBAC)
- ✅ `smartticket-shared-error`: 统一错误处理
- ✅ `smartticket-shared-metrics`: 监控指标收集

🔄 **下一步**: 开始阶段 1 的核心业务逻辑实现

### 阶段 1: 核心业务逻辑 (6-8周)

**目标**: 实现工单和知识管理核心功能

**主要任务**:
1. **工单管理系统**
   - 工单 CRUD 操作
   - 状态机实现
   - SLA 计时引擎
   - 智能路由算法

2. **知识管理系统**
   - 知识库 CRUD
   - 版本控制
   - 权限管理
   - 发布流程

3. **用户和权限管理**
   - 用户管理 API
   - 角色权限系统
   - 租户管理
   - 审计日志

4. **API 文档生成**
   - gRPC to OpenAPI 转换
   - Swagger 文档生成
   - 示例代码生成

**交付物**:
- [x] 完整的工单管理 API (CRUD + 状态机 + SLA)
- [x] 完整的工单状态机系统
- [x] SLA 计时引擎和违规监控
- [ ] 知识库管理系统 ❌
- [ ] 用户权限管理功能 ❌
- [ ] 完整的 API 文档 ❌

**当前状态** (2025-10-15):
⚠️ **阶段1 部分完成** - 只有工单管理和基础认证系统：

### ✅ **已完成功能**

#### 🔧 **技术基础设施**
- ✅ **项目架构**: Rust Cargo workspace + 微服务架构
- ✅ **编译系统**: 全工作空间编译成功 (0 错误)
- ✅ **gRPC 框架**: tonic + prost 完整集成
- ✅ **错误处理**: 统一错误处理机制
- ✅ **配置管理**: 多环境配置支持

#### 🗄️ **数据库系统**
- ✅ **PostgreSQL 15.14**: 多租户 RLS 完整实现
- ✅ **Redis 7**: 缓存系统正常 (PONG)
- ✅ **数据迁移**: 8个核心表完整创建
- ✅ **初始数据**: 1个租户 + 1个管理员用户
- ✅ **数据库连接**: 连接池和查询优化

#### 🎫 **工单管理系统**
- ✅ **gRPC 服务定义**: 完整的 TicketService 接口
- ✅ **数据模型**: Ticket, Comment, SLA, Category 等完整模型
- ✅ **CRUD 操作**: 创建、查询、更新、删除、搜索工单
- ✅ **状态机系统**: 完整的工单状态转换逻辑和验证
- ✅ **SLA 引擎**: 响应时间、解决时间计算和违规监控
- ✅ **评论系统**: 公开/内部评论支持
- ✅ **统计报告**: 工单统计和 SLA 违规报告

#### 🔐 **基础认证系统**
- ✅ **JWT 认证**: 完整的 token 生成和验证
- ✅ **多角色支持**: SuperAdmin, TenantAdmin, SupportEngineer, CustomerUser, Sales
- ✅ **权限控制**: 17个精细化权限点
- ✅ **租户隔离**: 完整的多租户数据隔离
- ✅ **权限中间件**: gRPC 服务权限验证

#### 🧪 **测试与演示**
- ✅ **单元测试**: Gateway 服务 4/4 测试通过
- ✅ **认证演示**: JWT 认证和权限系统完整演示
- ✅ **数据库检查**: 数据库状态和连接完整性验证
- ✅ **Docker 环境**: PostgreSQL + Redis 容器化运行

### ❌ **尚未实现的重要功能**

#### 📚 **知识库管理系统**
- ❌ **proto定义已存在**: KnowledgeService 有完整的接口定义
- ❌ **实际实现缺失**: 没有找到任何 KnowledgeService 的 Rust 实现
- ❌ **知识库 CRUD**: 创建、查询、更新、删除知识文章
- ❌ **分类管理**: 知识分类的创建和管理
- ❌ **搜索功能**: 知识库全文搜索
- ❌ **发布流程**: 文章发布、审核、归档

#### 👥 **用户管理 API**
- ❌ **proto定义已存在**: UserService 有完整的接口定义
- ❌ **实际实现缺失**: 没有找到任何 UserService 的 Rust 实现
- ❌ **用户 CRUD**: 创建、查询、更新、删除用户
- ❌ **用户状态管理**: 激活/停用用户
- ❌ **密码管理**: 修改密码、重置密码
- ❌ **用户资料**: 个人资料管理

#### 🏢 **租户管理功能**
- ❌ **proto定义存在**: 在 user.proto 中有 Tenant message
- ❌ **实际实现缺失**: 没有专门的 TenantService 实现
- ❌ **租户 CRUD**: 创建、查询、更新、删除租户
- ❌ **订阅管理**: 订阅等级、用户数量限制
- ❌ **数据隔离**: 租户级别的数据隔离

#### 📖 **API 文档**
- ❌ **OpenAPI 文档**: 没有 gRPC 到 OpenAPI 的转换
- ❌ **Swagger UI**: 没有 API 文档界面
- ❌ **示例代码**: 没有客户端 SDK 或示例

### 📊 **实际运行状态**
- ✅ **编译状态**: 全工作空间 0 错误，仅有 5 个警告
- ✅ **测试覆盖**: 仅 Gateway 服务测试通过
- ✅ **数据库**: 1租户 + 1用户 + 8表完整结构
- ✅ **缓存**: Redis 连接正常
- ✅ **认证系统**: JWT token 生成验证正常运行

🔄 **下一步**: 继续实现缺失的 KnowledgeService、UserService 等核心服务

### 阶段 2: AI 服务集成 (8-10周)

**目标**: 实现 RAG 和 AI 辅助功能

**主要任务**:
1. **向量数据库集成**
   - PgVector / Qdrant 集成
   - 向量索引策略
   - 多租户向量隔离

2. **文档摄取管道**
   - 文档解析器 (PDF/HTML/MD)
   - 内容预处理和清洗
   - 分片和嵌入生成

3. **RAG 查询引擎**
   - 混合检索 (BM25 + 向量)
   - 重排序算法
   - 上下文构建

4. **LLM 集成**
   - 多 Provider 支持
   - 提示模板管理
   - 质量评估和反馈

**交付物**:
- [ ] 完整的 RAG 系统
- [ ] AI 辅助功能
- [ ] 文档摄取管道
- [ ] 质量评估框架

### 阶段 3: 高级功能 (6-8周)

**目标**: 实现高级功能和系统集成

**主要任务**:
1. **通知系统**
   - 多渠道通知 (邮件/聊天/Push)
   - 模板管理
   - 节流和重试

2. **导入导出系统**
   - 批量数据处理
   - 多格式支持 (CSV/JSON/XML)
   - 异步任务处理

3. **外部系统集成**
   - Jira/GitHub 集成
   - Slack/Teams 机器人
   - CRM 系统对接

4. **性能优化**
   - 缓存策略优化
   - 数据库查询优化
   - 并发处理优化

**交付物**:
- [ ] 通知系统
- [ ] 导入导出功能
- [ ] 外部系统集成
- [ ] 性能优化成果

### 阶段 4: 生产就绪 (4-6周)

**目标**: 确保系统生产就绪

**主要任务**:
1. **安全加固**
   - 安全扫描和修复
   - 渗透测试
   - 合规检查 (GDPR)

2. **监控和运维**
   - 全面的监控仪表盘
   - 告警规则配置
   - 备份恢复策略

3. **文档完善**
   - 部署文档
   - 运维手册
   - 故障排查指南

4. **性能基准测试**
   - 负载测试
   - 容量规划
   - 性能优化

**交付物**:
- [ ] 生产级部署包
- [ ] 完整的监控体系
- [ ] 运维文档
- [ ] 性能基准报告

## 技术规范

### 代码质量标准

**测试覆盖率要求**:
- 单元测试覆盖率 ≥ 75%
- 关键模块覆盖率 ≥ 85%
- 集成测试覆盖所有公共 API
- 端到端测试覆盖核心业务流程

**代码规范**:
```rust
// 使用 rustfmt 格式化代码
// 使用 clippy 进行静态分析
// 使用 cargo-audit 检查依赖安全
// 使用 cargo-deny 进行许可证检查
```

**提交规范**:
```
feat: 新功能
fix: 修复问题
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试相关
chore: 构建/工具相关
```

### API 设计规范

**gRPC 服务定义**:
```protobuf
syntax = "proto3";

package smartticket.v1;

import "google/protobuf/timestamp.proto";
import "google/api/annotations.proto";

service TicketService {
  rpc CreateTicket(CreateTicketRequest) returns (Ticket) {
    option (google.api.http) = {
      post: "/api/v1/tickets"
      body: "*"
    };
  }
}
```

**错误处理**:
```rust
#[derive(Debug, thiserror::Error)]
pub enum TicketError {
    #[error("Ticket not found: {0}")]
    NotFound(String),

    #[error("Permission denied: {0}")]
    PermissionDenied(String),

    #[error("Validation error: {0}")]
    Validation(String),

    #[error("Internal server error")]
    Internal(#[from] sqlx::Error),
}
```

### 数据库设计规范

**表命名规范**:
- 使用复数形式: `tickets`, `users`, `knowledge_articles`
- 租户隔离: 所有表包含 `tenant_id` 字段
- 审计字段: `created_at`, `updated_at`, `created_by`

**索引策略**:
```sql
-- 多租户索引
CREATE INDEX idx_tickets_tenant_status ON tickets(tenant_id, status);

-- 时间序列索引
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);

-- 复合索引
CREATE INDEX idx_tickets_assignee_status
ON tickets(assignee_id, status) WHERE status != 'closed';
```

### 性能要求

**响应时间目标**:
- API 响应时间 P95 < 300ms
- 数据库查询 P95 < 100ms
- RAG 查询 P95 < 2s

**并发处理能力**:
- 支持 1000+ 并发用户
- 10000+ QPS 处理能力
- 99.9% 服务可用性

## 开发环境配置

### 本地开发设置

**Prerequisites**:
```bash
# 安装 Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs/ | sh

# 安装 PostgreSQL
brew install postgresql  # macOS
sudo apt-get install postgresql  # Ubuntu

# 安装 Redis
brew install redis  # macOS
sudo apt-get install redis-server  # Ubuntu

# 安装 Docker
brew install docker  # macOS
sudo apt-get install docker.io  # Ubuntu
```

**开发环境启动**:
```bash
# 1. 启动数据库服务
docker-compose up -d postgres redis

# 2. 运行数据库迁移
sqlx migrate run --database-url postgresql://user:pass@localhost/smartticket

# 3. 启动开发服务器
cargo run --bin gateway
cargo run --bin core
cargo run --bin ai
cargo run --bin platform
cargo run --bin notification
```

### 测试环境配置

**测试命令**:
```bash
# 运行所有测试
cargo test

# 运行特定模块测试
cargo test --package core

# 运行集成测试
cargo test --test integration

# 生成测试覆盖率报告
cargo install cargo-tarpaulin
cargo tarpaulin --out Html
```

## 部署策略

### 容器化部署

**Docker 镜像构建**:
```dockerfile
# 多阶段构建优化
FROM rust:1.75 as builder
# ... 构建阶段

FROM debian:bookworm-slim as runtime
# ... 运行时阶段
```

**Kubernetes 部署**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: smartticket-core
spec:
  replicas: 3
  selector:
    matchLabels:
      app: smartticket-core
  template:
    spec:
      containers:
      - name: core
        image: smartticket/core:latest
        ports:
        - containerPort: 50051
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: smartticket-secrets
              key: database-url
```

### CI/CD 流水线

**GitHub Actions 配置**:
```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/clone@v3
      - uses: actions-rs/toolchain@v1
      - run: cargo test --all-features
      - run: cargo tarpaulin --out Xml

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: cargo audit
      - run: cargo deny check

  build:
    needs: [test, security]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: docker build -t smartticket/core .
      - run: docker push ${{ secrets.REGISTRY_URL }}/smartticket/core
```

## 监控和运维

### 监控指标

**关键指标**:
```rust
use prometheus::{Counter, Histogram, Gauge};

// 业务指标
static TICKETS_CREATED: Counter = register_counter!(
    "smartticket_tickets_created_total",
    "Total number of tickets created"
).unwrap();

static TICKET_PROCESSING_DURATION: Histogram = register_histogram!(
    "smartticket_ticket_processing_duration_seconds",
    "Time spent processing tickets"
).unwrap();

// 系统指标
static ACTIVE_CONNECTIONS: Gauge = register_gauge!(
    "smartticket_active_connections",
    "Number of active database connections"
).unwrap();
```

### 健康检查

**健康检查端点**:
```rust
#[get("/health")]
async fn health_check(
    app_state: web::Data<AppState>,
) -> impl Responder {
    let mut health = HealthStatus::default();

    // 检查数据库连接
    if let Err(e) = check_database_health(&app_state.db).await {
        health.database = "unhealthy".to_string();
        health.database_error = Some(e.to_string());
    }

    // 检查 Redis 连接
    if let Err(e) = check_redis_health(&app_state.redis).await {
        health.redis = "unhealthy".to_string();
        health.redis_error = Some(e.to_string());
    }

    let status = if health.is_healthy() {
        StatusCode::OK
    } else {
        StatusCode::SERVICE_UNAVAILABLE
    };

    HttpResponse::build(status).json(health)
}
```

### 日志管理

**结构化日志配置**:
```rust
use tracing::{info, warn, error, instrument};

#[instrument(skip(self))]
impl TicketService {
    pub async fn create_ticket(&self, request: CreateTicketRequest) -> Result<Ticket> {
        info!(
            tenant_id = %request.tenant_id,
            title = %request.title,
            priority = ?request.priority,
            "Creating new ticket"
        );

        match self.internal_create_ticket(request).await {
            Ok(ticket) => {
                info!(
                    ticket_id = %ticket.id,
                    "Ticket created successfully"
                );
                Ok(ticket)
            }
            Err(e) => {
                error!(
                    error = %e,
                    "Failed to create ticket"
                );
                Err(e)
            }
        }
    }
}
```

## 安全考虑

### 数据安全

**加密策略**:
- 传输加密: TLS 1.3
- 静态加密: 数据库加密 + 备份加密
- API 密钥: KMS 管理，定期轮换
- 敏感数据: 字段级加密存储

**访问控制**:
- JWT 令牌认证
- 基于角色的访问控制 (RBAC)
- 多租户数据隔离
- API 速率限制

### 代码安全

**安全扫描**:
```yaml
# .github/workflows/security.yml
name: Security Scan
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run cargo audit
        run: cargo audit
      - name: Run cargo-deny
        uses: EmbarkStudios/cargo-deny-action@v1
```

**依赖安全**:
```toml
# .cargo/deny.toml
[licenses]
allow = [
    "MIT",
    "Apache-2.0",
    "BSD-3-Clause"
]

[bans]
multiple-versions = "deny"
wildcards = "allow"
```

## 质量保证

### 代码审查

**PR 审查清单**:
- [ ] 测试覆盖率 ≥ 75%
- [ ] 所有测试通过
- [ ] 代码格式化检查通过
- [ ] Clippy 警告已修复
- [ ] 安全扫描通过
- [ ] API 文档已更新
- [ ] 性能测试通过

### 发布流程

**版本管理**:
- 语义化版本控制 (SemVer)
- 变更日志维护
- 版本标签创建
- 回滚策略制定

**发布检查**:
- 集成测试通过
- 性能基准达标
- 安全扫描通过
- 文档更新完成
- 部署脚本验证

## 风险管理

### 技术风险

**风险识别**:
- 数据库性能瓶颈
- 第三方依赖风险
- 安全漏洞风险
- 部署运维复杂度

**缓解策略**:
- 性能测试和监控
- 依赖安全扫描
- 定期安全审计
- 容器化部署和自动化

### 项目风险

**时间风险**:
- 功能范围控制
- 技术债务管理
- 团队协作效率

**质量风险**:
- 测试覆盖率保证
- 代码质量标准
- 性能基准达成

## 团队协作

### 开发流程

**Git 工作流**:
```
main (生产) ← develop (开发) ← feature/* (功能开发)
```

**分支策略**:
- `main`: 生产环境代码
- `develop`: 集成开发代码
- `feature/*`: 功能开发分支
- `hotfix/*`: 紧急修复分支

### 代码规范

**命名约定**:
- 文件名: snake_case
- 变量名: snake_case
- 类型名: PascalCase
- 常量名: SCREAMING_SNAKE_CASE

**注释规范**:
- 公共 API 必须有文档注释
- 复杂逻辑需要解释注释
- TODO 注释必须包含责任人

## 总结

SmartTicket 后端开发计划采用分阶段实施策略，确保每个阶段都有明确的目标和交付物。通过严格的技术规范、完善的测试策略和全面的质量保证体系，确保交付高质量、安全可靠的生产级系统。

关键成功因素：
1. 严格的技术标准和代码质量要求
2. 完善的测试覆盖率和 CI/CD 流水线
3. 全面的监控和运维支持
4. 详细的文档和知识管理
5. 持续的性能优化和安全加固