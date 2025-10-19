# SmartTicket 后端开发计划

## 项目概览

SmartTicket 是一款面向企业自主部署的多租户工单与知识协作平台，后端基于 **Golang + GIN + SQLite + GORM** 技术栈，采用单体架构设计，为企业提供完全可控的工单管理和 AI 辅助解决方案。

## 核心技术栈

### 主要技术
- **编程语言**: Golang 1.21+
- **Web 框架**: GIN v1.9+ (REST API)
- **ORM 框架**: GORM v1.25+
- **数据库**: SQLite 3.41+ (嵌入式数据库)
- **认证**: JWT (golang-jwt/jwt)
- **配置管理**: Viper
- **日志**: Logrus 或 Zap
- **测试**: Go 标准库 + Testify
- **构建**: Go modules + Docker

### 特色功能
- **企业自主部署**: 单二进制部署，零外部依赖
- **数据自主可控**: 完善的导入导出功能
- **自定义 LLM Provider**: 支持多种 AI 服务集成

## 系统架构

### 单体架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    SmartTicket 后端服务                        │
│                      (单二进制可执行文件)                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │   API 路由层     │ │   中间件层       │ │   业务逻辑层      │  │
│  │                │ │                │ │                │  │
│  │ • REST API     │ │ • JWT 认证      │ │ • 工单管理       │  │
│  │ • 参数验证       │ │ • 租户隔离       │ │ • 知识库管理     │  │
│  │ • 响应格式化     │ │ • 权限控制       │ │ • SLA 引擎       │  │
│  │ • 错误处理       │ │ • 日志记录       │ │ • AI 服务集成     │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │   数据访问层     │ │   服务层         │ │   工具层         │  │
│  │                │ │                │ │                │  │
│  │ • GORM 模型     │ │ • 数据导入导出     │ │ • 加密工具       │  │
│  │ • 数据库操作     │ │ • 备份恢复       │ │ • 验证工具       │  │
│  │ • 事务管理       │ │ • 通知服务       │ │ • 文件处理       │  │
│  │ • 连接池管理     │ │ • LLM Provider  │ │ • 缓存管理       │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │   SQLite 数据库  │ │   文件存储       │ │   外部集成       │  │
│  │                │ │                │ │                │  │
│  │ • 主数据库       │ │ • 附件存储       │ │ • 邮件服务       │  │
│  │ • 向量数据       │ │ • 导出文件       │ │ • LLM APIs      │  │
│  │ • 配置数据       │ │ • 备份文件       │ │ • Webhook       │  │
│  │ • 审计日志       │ │ • 临时文件       │ │ • 第三方系统     │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 项目结构

```
smartticket-backend/
├── cmd/                        # 应用程序入口
│   └── server/
│       └── main.go            # 主程序入口
├── internal/                   # 私有应用程序代码
│   ├── api/                   # API 层
│   │   ├── handlers/          # HTTP 处理器
│   │   │   ├── auth.go       # 认证处理器
│   │   │   ├── tickets.go    # 工单处理器
│   │   │   ├── knowledge.go  # 知识库处理器
│   │   │   ├── users.go      # 用户管理处理器
│   │   │   ├── data.go       # 数据管理处理器
│   │   │   └── llm.go        # LLM 配置处理器
│   │   ├── middleware/        # 中间件
│   │   │   ├── auth.go       # 认证中间件
│   │   │   ├── tenant.go     # 租户中间件
│   │   │   ├── cors.go       # CORS 中间件
│   │   │   ├── logging.go    # 日志中间件
│   │   │   └── recovery.go   # 恢复中间件
│   │   ├── routes/            # 路由定义
│   │   │   └── routes.go     # 路由配置
│   │   └── validators/        # 参数验证
│   │       ├── ticket.go     # 工单验证器
│   │       ├── user.go       # 用户验证器
│   │       └── common.go     # 通用验证器
│   ├── models/                # 数据模型
│   │   ├── tenant.go         # 租户模型
│   │   ├── user.go           # 用户模型
│   │   ├── ticket.go         # 工单模型
│   │   ├── knowledge.go      # 知识库模型
│   │   ├── llm_provider.go   # LLM Provider 模型
│   │   ├── import_export.go  # 导入导出模型
│   │   └── audit.go          # 审计日志模型
│   ├── services/              # 业务逻辑层
│   │   ├── auth/             # 认证服务
│   │   │   ├── auth_service.go
│   │   │   ├── jwt_service.go
│   │   │   └── rbac_service.go
│   │   ├── ticket/           # 工单服务
│   │   │   ├── ticket_service.go
│   │   │   ├── sla_service.go
│   │   │   └── notification_service.go
│   │   ├── knowledge/        # 知识库服务
│   │   │   ├── knowledge_service.go
│   │   │   ├── search_service.go
│   │   │   └── version_service.go
│   │   ├── llm/              # AI 服务
│   │   │   ├── rag_service.go
│   │   │   ├── embedding_service.go
│   │   │   └── provider_service.go
│   │   ├── data/             # 数据管理服务
│   │   │   ├── import_service.go
│   │   │   ├── export_service.go
│   │   │   └── backup_service.go
│   │   └── notification/     # 通知服务
│   │       ├── email_service.go
│   │       └── webhook_service.go
│   ├── repositories/          # 数据访问层
│   │   ├── interfaces/       # 仓储接口
│   │   ├── tenant_repo.go    # 租户仓储
│   │   ├── user_repo.go      # 用户仓储
│   │   ├── ticket_repo.go    # 工单仓储
│   │   ├── knowledge_repo.go # 知识库仓储
│   │   └── audit_repo.go     # 审计仓储
│   ├── database/              # 数据库相关
│   │   ├── database.go       # 数据库连接
│   │   ├── migrations/       # 数据库迁移
│   │   │   ├── 001_initial_schema.sql
│   │   │   ├── 002_add_indexes.sql
│   │   │   └── 003_add_constraints.sql
│   │   └── seeds/            # 种子数据
│   │       └── initial_data.sql
│   ├── config/                # 配置管理
│   │   ├── config.go         # 配置结构
│   │   └── loader.go         # 配置加载器
│   ├── utils/                 # 工具函数
│   │   ├── crypto.go         # 加密工具
│   │   ├── validator.go      # 验证工具
│   │   ├── file.go           # 文件处理工具
│   │   ├── date.go           # 日期工具
│   │   └── response.go       # 响应工具
│   └── errors/                # 错误定义
│       ├── errors.go         # 自定义错误类型
│       └── codes.go          # 错误代码定义
├── pkg/                       # 公共库代码
│   ├── logger/                # 日志库
│   │   └── logger.go
│   ├── cache/                 # 缓存库
│   │   └── memory_cache.go
│   └── validator/             # 验证库
│       └── validator.go
├── api/                       # API 定义
│   ├── openapi/               # OpenAPI 规范
│   │   └── smartticket.yaml
│   └── examples/              # API 示例
│       ├── create_ticket.json
│       └── search_knowledge.json
├── docs/                      # 文档
│   ├── api/                   # API 文档
│   │   ├── authentication.md
│   │   ├── tickets.md
│   │   ├── knowledge.md
│   │   └── data_management.md
│   ├── deployment/            # 部署文档
│   │   ├── docker.md
│   │   ├── configuration.md
│   │   └── monitoring.md
│   └── development/           # 开发文档
│       ├── setup.md
│       ├── testing.md
│       └── contributing.md
├── scripts/                   # 脚本
│   ├── build.sh              # 构建脚本
│   ├── test.sh               # 测试脚本
│   ├── migrate.sh            # 迁移脚本
│   └── deploy.sh             # 部署脚本
├── tests/                     # 测试
│   ├── unit/                 # 单元测试
│   ├── integration/          # 集成测试
│   ├── e2e/                  # 端到端测试
│   └── fixtures/             # 测试数据
├── configs/                   # 配置文件
│   ├── config.yaml           # 默认配置
│   ├── config.dev.yaml       # 开发配置
│   ├── config.prod.yaml      # 生产配置
│   └── config.test.yaml      # 测试配置
├── deployments/               # 部署配置
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── k8s/
│       ├── deployment.yaml
│       └── service.yaml
├── go.mod                     # Go 模块定义
├── go.sum                     # Go 模块校验
├── Makefile                   # 构建工具
├── README.md                  # 项目说明
└── .gitignore                 # Git 忽略文件
```

## 开发阶段规划

### 阶段 0: 基础设施搭建 (2-3周)

**目标**: 建立开发环境和基础架构

**主要任务**:
1. **项目初始化**
   - 创建 Go module 项目结构
   - 配置 GORM 模型和数据库连接
   - 设置基础中间件和路由

2. **数据库设计**
   - SQLite 数据库 schema 设计
   - GORM 模型定义
   - 数据库迁移脚本

3. **基础服务框架**
   - GIN Web 服务框架
   - 配置管理系统 (Viper)
   - 日志和错误处理

4. **认证授权系统**
   - JWT 认证实现
   - RBAC 权限控制
   - 多租户数据隔离

**交付物**:
- [x] 完整的项目结构 (Go modules + 单体架构)
- [x] 基础的数据库 schema (SQLite + GORM)
- [x] 可运行的最小服务框架
- [x] 基础的认证和权限系统

### 阶段 1: 核心业务逻辑 (3-4周)

**目标**: 实现工单和知识管理核心功能

**主要任务**:
1. **工单管理系统**
   - 工单 CRUD API 实现
   - 状态机和 SLA 引擎
   - 智能路由算法
   - 搜索和过滤功能

2. **知识管理系统**
   - 知识库 CRUD API
   - 文档版本控制
   - 权限管理和发布流程
   - 全文搜索功能

3. **用户和权限管理**
   - 用户管理 API
   - 角色权限系统
   - 租户管理
   - 审计日志

4. **API 文档生成**
   - OpenAPI 规范生成
   - Swagger UI 集成
   - 示例代码和文档

**交付物**:
- [x] 完整的工单管理 API (CRUD + 状态机 + SLA)
- [x] 工单状态机系统和 SLA 引擎
- [x] 知识库管理系统
- [x] 用户权限管理功能
- [x] 完整的 API 文档

### 阶段 2: 数据管理与备份 (2-3周)

**目标**: 实现完善的数据导入导出和备份功能

**主要任务**:
1. **数据导入导出系统**
   - 多格式支持 (CSV, JSON, XML, Markdown)
   - 批量数据处理
   - 字段映射和冲突处理
   - 异步任务处理

2. **备份恢复系统**
   - 自动备份策略
   - 增量备份支持
   - 时间点恢复
   - 数据验证

3. **第三方系统集成**
   - Zendesk 数据迁移
   - Jira Service Management 集成
   - 通用数据源适配器

**交付物**:
- [x] 完整的数据导入导出系统
- [x] 自动备份和恢复功能
- [x] 第三方系统数据迁移工具
- [x] 数据验证和错误处理

### 阶段 3: AI 服务集成 (4-5周)

**目标**: 实现 RAG 和自定义 LLM Provider 功能

**主要任务**:
1. **自定义 LLM Provider 系统**
   - 多 Provider 支持 (OpenAI, Azure, DeepSeek, 本地模型)
   - Provider 配置管理
   - 任务-模型映射
   - 成本监控和控制

2. **RAG 系统实现**
   - 文档摄取管道
   - 向量存储和检索
   - 混合搜索算法
   - 智能问答系统

3. **AI 辅助功能**
   - 智能工单分类
   - 自动回复建议
   - RCA 草稿生成
   - 知识库自动更新

**交付物**:
- [x] 自定义 LLM Provider 系统
- [x] 完整的 RAG 查询引擎
- [x] AI 辅助功能集成
- [x] 成本监控和优化

### 阶段 4: 系统优化与生产就绪 (2-3周)

**目标**: 确保系统生产就绪和性能优化

**主要任务**:
1. **性能优化**
   - 数据库查询优化
   - 缓存策略实现
   - 并发处理优化
   - 内存使用优化

2. **安全加固**
   - 安全扫描和修复
   - 输入验证和过滤
   - API 速率限制
   - 数据加密实现

3. **监控和运维**
   - 健康检查端点
   - 监控指标收集
   - 日志聚合和分析
   - 告警配置

4. **部署准备**
   - Docker 容器化
   - 部署脚本编写
   - 配置管理优化
   - 文档完善

**交付物**:
- [x] 生产级性能优化
- [x] 完整的安全加固
- [x] 监控和运维体系
- [x] 部署文档和工具

## 技术规范

### 代码质量标准

**测试覆盖率要求**:
- 单元测试覆盖率 ≥ 75%
- 关键模块覆盖率 ≥ 85%
- 集成测试覆盖所有公共 API
- 端到端测试覆盖核心业务流程

**代码规范**:
```go
// 使用 gofmt 格式化代码
// 使用 golint 进行静态分析
// 使用 go vet 检查潜在问题
// 使用 gosec 进行安全扫描
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

**RESTful API 设计**:
```go
// API 路由设计
func setupRoutes(r *gin.Engine) {
    v1 := r.Group("/api/v1")

    // 认证中间件
    v1.Use(authMiddleware())
    v1.Use(tenantMiddleware())

    // 工单管理
    tickets := v1.Group("/tickets")
    {
        tickets.POST("", createTicket)
        tickets.GET("", listTickets)
        tickets.GET("/:id", getTicket)
        tickets.PUT("/:id", updateTicket)
        tickets.DELETE("/:id", deleteTicket)
    }
}
```

**响应格式标准**:
```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

**错误处理**:
```go
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
    return e.Message
}

// 自定义错误类型
var (
    ErrNotFound     = &AppError{Code: "NOT_FOUND", Message: "Resource not found"}
    ErrUnauthorized = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized access"}
    ErrValidation   = &AppError{Code: "VALIDATION_ERROR", Message: "Validation failed"}
)
```

### 数据库设计规范

**GORM 模型定义**:
```go
type Ticket struct {
    ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID  string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Title     string    `gorm:"type:varchar(500);not null" json:"title"`
    Status    string    `gorm:"type:varchar(20);default:'new'" json:"status"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    // 关联关系
    Tenant   Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Messages []TicketMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
}
```

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
- API 响应时间 P95 < 200ms
- 数据库查询 P95 < 100ms
- RAG 查询 P95 < 2s

**并发处理能力**:
- 支持 100+ 并发用户
- 1000+ QPS 处理能力
- 99.9% 服务可用性

## 开发环境配置

### 本地开发设置

**Prerequisites**:
```bash
# 安装 Go 1.21+
go version

# 安装 SQLite
# macOS
brew install sqlite

# Ubuntu/Debian
sudo apt-get install sqlite3 libsqlite3-dev

# Windows
# 从 https://sqlite.org/download.html 下载
```

**开发环境启动**:
```bash
# 1. 克隆项目
git clone https://github.com/your-org/smartticket-backend.git
cd smartticket-backend

# 2. 安装依赖
go mod download

# 3. 初始化数据库
go run cmd/server/main.go migrate

# 4. 启动开发服务器
go run cmd/server/main.go serve

# 或使用 Makefile
make dev
```

**环境配置**:
```yaml
# configs/config.dev.yaml
server:
  port: 6533
  mode: debug

database:
  type: sqlite
  dsn: ./data/smartticket_dev.db

jwt:
  secret: your-secret-key
  expiration: 24h

logging:
  level: debug
  format: text
```

### 测试环境配置

**测试命令**:
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/services/ticket

# 运行集成测试
go test -tags=integration ./tests/integration

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行基准测试
go test -bench=. ./...
```

## 部署策略

### 单二进制部署

**构建**:
```bash
# 构建生产版本
go build -ldflags="-s -w" -o smartticket cmd/server/main.go

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o smartticket-linux cmd/server/main.go
```

**Docker 容器化**:
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o smartticket cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/smartticket .
COPY --from=builder /app/configs ./configs

EXPOSE 6533
CMD ["./smartticket", "serve"]
```

**Docker Compose**:
```yaml
# docker-compose.yml
version: '3.8'

services:
  smartticket:
    build: .
    ports:
      - "6533:6533"
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
    environment:
      - GIN_MODE=release
    restart: unless-stopped
```

### 系统服务部署

**Systemd 服务**:
```ini
# /etc/systemd/system/smartticket.service
[Unit]
Description=SmartTicket Backend Service
After=network.target

[Service]
Type=simple
User=smartticket
WorkingDirectory=/opt/smartticket
ExecStart=/opt/smartticket/smartticket serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Nginx 反向代理**:
```nginx
# /etc/nginx/sites-available/smartticket
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:6533;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### CI/CD 流水线

**GitHub Actions 配置**:
```yaml
# .github/workflows/ci.yml
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
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'

      - name: Build binary
        run: |
          go build -ldflags="-s -w" -o smartticket cmd/server/main.go

      - name: Build Docker image
        run: |
          docker build -t smartticket:${{ github.sha }} .
          docker tag smartticket:${{ github.sha }} smartticket:latest

      - name: Push to registry
        if: github.ref == 'refs/heads/main'
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push smartticket:${{ github.sha }}
          docker push smartticket:latest
```

## 监控和运维

### 健康检查

**健康检查端点**:
```go
// internal/api/handlers/health.go
type HealthResponse struct {
    Status     string            `json:"status"`
    Timestamp  time.Time         `json:"timestamp"`
    Version    string            `json:"version"`
    Checks     map[string]string `json:"checks"`
    Uptime     string            `json:"uptime"`
}

func HealthCheck(c *gin.Context) {
    health := HealthResponse{
        Status:    "healthy",
        Timestamp: time.Now(),
        Version:   os.Getenv("APP_VERSION"),
        Checks:    make(map[string]string),
        Uptime:    time.Since(startTime).String(),
    }

    // 检查数据库连接
    if err := checkDatabase(); err != nil {
        health.Checks["database"] = "unhealthy: " + err.Error()
        health.Status = "unhealthy"
    } else {
        health.Checks["database"] = "healthy"
    }

    status := http.StatusOK
    if health.Status != "healthy" {
        status = http.StatusServiceUnavailable
    }

    c.JSON(status, health)
}
```

### 日志管理

**结构化日志配置**:
```go
// pkg/logger/logger.go
import (
    "github.com/sirupsen/logrus"
    "github.com/gin-gonic/gin"
)

func NewLogger(level string) *logrus.Logger {
    logger := logrus.New()

    logLevel, err := logrus.ParseLevel(level)
    if err != nil {
        logLevel = logrus.InfoLevel
    }

    logger.SetLevel(logLevel)
    logger.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: time.RFC3339,
    })

    return logger
}

func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        logger.WithFields(logrus.Fields{
            "method":     param.Method,
            "path":       param.Path,
            "status":     param.StatusCode,
            "latency":    param.Latency,
            "client_ip":  param.ClientIP,
            "user_agent": param.Request.UserAgent(),
        }).Info("Request processed")

        return ""
    })
}
```

### 性能监控

**Prometheus 指标**:
```go
// pkg/metrics/metrics.go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "smartticket_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "smartticket_http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
}

func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())

        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
    }
}
```

## 安全考虑

### 数据安全

**加密策略**:
- 传输加密: TLS 1.3
- 静态加密: SQLite 数据库加密
- API 密钥: 加密存储，定期轮换
- 敏感数据: 字段级加密存储

**访问控制**:
- JWT 令牌认证
- 基于角色的访问控制 (RBAC)
- 多租户数据隔离
- API 速率限制

### 代码安全

**安全扫描**:
```bash
# 安装安全扫描工具
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# 运行安全扫描
gosec ./...

# 检查依赖漏洞
go list -json -m all | nancy sleuth
```

**输入验证**:
```go
// internal/api/validators/ticket.go
type CreateTicketRequest struct {
    Title       string `json:"title" binding:"required,min=1,max=500"`
    Description string `json:"description" binding:"max=5000"`
    Priority    string `json:"priority" binding:"oneof=low medium high critical"`
    ProductID   string `json:"product_id" binding:"required,uuid4"`
}

func ValidateCreateTicket(c *gin.Context) {
    var req CreateTicketRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": &APIError{
                Code:    "VALIDATION_ERROR",
                Message: err.Error(),
            },
        })
        c.Abort()
        return
    }

    c.Set("validated_request", req)
    c.Next()
}
```

## 质量保证

### 测试策略

**单元测试**:
```go
// internal/services/ticket/ticket_service_test.go
func TestTicketService_CreateTicket(t *testing.T) {
    // 设置测试环境
    db := setupTestDB(t)
    repo := repositories.NewTicketRepository(db)
    service := services.NewTicketService(repo)

    // 测试用例
    tests := []struct {
        name    string
        request *CreateTicketRequest
        wantErr bool
    }{
        {
            name: "valid ticket",
            request: &CreateTicketRequest{
                Title:       "Test Ticket",
                Description: "Test Description",
                Priority:    "medium",
                ProductID:   "product-123",
            },
            wantErr: false,
        },
        {
            name: "invalid title",
            request: &CreateTicketRequest{
                Title:       "", // 空标题
                Description: "Test Description",
                Priority:    "medium",
                ProductID:   "product-123",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ticket, err := service.CreateTicket(context.Background(), tt.request)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, ticket.ID)
                assert.Equal(t, tt.request.Title, ticket.Title)
            }
        })
    }
}
```

**集成测试**:
```go
// tests/integration/api_test.go
func TestTicketAPI_CreateTicket(t *testing.T) {
    // 设置测试服务器
    router := setupTestRouter(t)

    // 准备测试数据
    payload := map[string]interface{}{
        "title":       "Integration Test Ticket",
        "description": "Test Description",
        "priority":    "medium",
        "product_id":  "product-123",
    }

    payloadBytes, _ := json.Marshal(payload)

    // 发送请求
    req, _ := http.NewRequest("POST", "/api/v1/tickets", bytes.NewBuffer(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+getTestToken(t))

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // 验证响应
    assert.Equal(t, http.StatusCreated, w.Code)

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)

    assert.True(t, response["success"].(bool))
    data := response["data"].(map[string]interface{})
    assert.Equal(t, payload["title"], data["title"])
}
```

### 性能测试

**基准测试**:
```go
func BenchmarkTicketService_CreateTicket(b *testing.B) {
    service := setupBenchmarkService(b)
    request := &CreateTicketRequest{
        Title:       "Benchmark Ticket",
        Description: "Test Description",
        Priority:    "medium",
        ProductID:   "product-123",
    }

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := service.CreateTicket(context.Background(), request)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## 总结

SmartTicket 后端采用 **Golang + GIN + SQLite + GORM** 技术栈，通过单体架构设计实现企业自主部署、数据自主可控和自定义 LLM Provider 的核心特色。

### 关键优势

1. **简单部署**: 单二进制文件，零外部依赖
2. **数据可控**: 完善的导入导出和备份功能
3. **AI 灵活**: 支持多种 LLM Provider 配置
4. **性能优秀**: 轻量级架构，快速响应
5. **安全可靠**: 企业级安全和权限控制

### 技术特点

- **高性能**: Golang 并发能力 + SQLite 优化
- **易维护**: 清晰的代码结构和完整的文档
- **可扩展**: 模块化设计，便于功能扩展
- **标准遵循**: RESTful API 设计，OpenAPI 规范
- **测试完备**: 高覆盖率测试，质量保证

通过分阶段的开发计划和严格的技术规范，确保交付高质量、安全可靠的生产级系统。