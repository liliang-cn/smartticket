SmartTicket 设计文档（草案）

更新时间：2025-10-19

概览

SmartTicket 是一款面向企业自主部署的多租户工单与知识协作平台，核心特色在于**企业完全掌控自己的数据**、**灵活的导入导出功能**以及**可自定义的 LLM Provider**。系统为企业客户提供端到端的问题受理、协同处理、知识沉淀与 AI 辅助（原生 RAG/LLM）能力，同时满足 GDPR 与审计合规要求。

## 核心特色

1. **企业自主部署**：单文件部署，无需外部依赖，支持私有化部署
2. **数据自主可控**：完善的导入导出功能，用户可随时备份和迁移所有数据
3. **自定义 LLM Provider**：支持接入任意 LLM 服务，包括公有云、私有部署或本地模型

目标与非目标

- 目标
  - 降低平均首次响应时间（FRT）与平均解决时间（MTTR），提升 SLA 兑现率
  - 让客户在创建工单前通过 RAG 自助排查，减少无效/重复工单
  - 支持按"产品/版本/支持层级/合同"细粒度的 SLA 策略、升级路径与排班
  - 全角色协同：Admin、客户（Customer）、工程师（Support Engineer）、售前（SE/SA）、销售（Sales）
  - 原生知识沉淀：帖子/解决方案/运行手册/已解决 Ticket 摘要自动入库并可被 RAG 检索
  - 强审计与数据隔离，满足欧洲客户的数据驻留与 GDPR 要求
- 非目标
  - 不做完整 ERP/CRM 替代，仅提供必要的轻量商机/合同联动与外部 CRM 集成
  - 不承载大规模监控采集（可与现有 Observability 平台集成而非自建）

用户与角色画像

- Admin（平台/租户管理员）
  - 配置租户、产品目录、版本、支持计划（完全自定义，如"企业VIP"、"技术支持Plus"等）、SLA 策略（响应/恢复/解决时限）、工作时间与排班
  - 客户管理：客户组织、合同（产品、席位、SLA 层级、有效期、数据驻留要求）、特殊折扣/价目表
  - 权限与合规：RBAC、数据驻留、审计、PII 脱敏策略、跨区域存储选择
  - 集成：SSO（SAML/OIDC）、邮件入单、Webhooks、Jira/GitHub、Slack/Teams、PagerDuty、CRM（Salesforce/HubSpot）
- 客户（Customer End-User/管理员）
  - 提交/查看/更新工单、上传附件、查看 SLA/合同、参与 RAG 自助排查、接收进展通知
  - 客户管理员可管理本组织成员、查看使用与 SLA 报告
- 工程师（Support/DevOps/SRE）
  - 工单队列、分配、合并/拆分、状态流转、内部备注、Runbook 引用、知识回写
  - 与 CI/CD、监控、Jira/GitHub issue 联动；变更窗口与升级流程
- 售前（SE/SA）
  - 跟进 PoC/试用类工单、撰写解决方案帖子、与工程师协作
- 销售（Sales）
  - 查看关键客户工单态势、到期合同与续费风险提醒、生成对外 RCA/摘要材料（受权限控制）

核心场景与痛点

1. 多产品多版本：需要按产品/版本/部署形态（SaaS/On-Prem/Hybrid）建模，便于精准路由与知识匹配。
2. 分层支持：用户自定义支持计划（如"企业VIP"、"技术支持Plus"等）影响 SLA、升级路径、可用渠道（热线/专线/工作时段）。
3. RAG 自助与 AI 辅助：客户在创建工单前先结构化问题，系统基于手册/FAQ/历史 Ticket 摘要检索答案；工程师侧 AI 建议下一步排查、生成回复草稿与 RCA 初稿。
4. 合规：GDPR、审计、数据驻留；PII 与敏感信息处理；客户租户隔离。
5. 协作与可观测：跨团队沟通、内外部备注、变更同步、自动化动作（Auto-triage/Auto-assign）。

功能与权限（RBAC）

- Admin
  - 租户与组织：创建/停用客户组织，设置命名空间与数据驻留区域
  - 产品与服务：产品目录、版本、组件、依赖矩阵；支持计划与 SLA 策略；排班模板（工作日历/正班/值班）
  - 合同与定价：合同（多产品/席位/支持层级）、折扣（客户级/产品级/临时活动）、发票与计费集成
  - 集成：SSO、邮件入单、Webhook、Jira/GitHub、Slack/Teams、PagerDuty、CRM
  - 合规与安全：审计日志、数据导出/删改请求（GDPR DSR）、PII 策略、加密与密钥轮转
  - **数据管理与备份**：
    - 全量数据导出：支持一键导出所有工单、知识库、用户、配置等数据
    - 增量数据导出：按时间范围导出新增或变更的数据
    - 多格式支持：CSV、JSON、XML、Markdown 格式，适配不同系统需求
    - 数据导入：支持从外部系统批量导入历史数据，字段映射与冲突处理
    - 自动备份：定时自动备份，支持本地存储和云存储
    - 数据迁移：完整的数据迁移工具，支持版本升级和数据迁移

  - **自定义 LLM Provider 管理**：
    - 多 Provider 支持：OpenAI、Azure OpenAI、DeepSeek、Anthropic Claude、本地部署模型等
    - 灵活配置：自定义 API 端点、认证方式、模型参数、重试策略
    - 任务模型映射：为不同 AI 任务（检索、重排、生成、函数调用）配置专用模型
    - 成本控制：设置配额限制、费用监控、使用统计
    - 密钥管理：加密存储 API 密钥，支持密钥轮换
    - 私有化支持：支持企业内部部署的 LLM 服务和本地模型
- 客户
  - 工单：创建/查看/回复/评价、附件、授权支持联系人、按合同级别查看 SLA
  - 自助：RAG 检索、智能表单（产品/版本/日志片段/环境）、状态订阅
- 工程师
  - 工作台：队列、筛选、智能路由、群组分配、SLA 计时与警示、宏与模板
  - 协作：内部备注、@提及、与 Jira/GitHub 双向链接、变更记录、知识回写
  - 自动化：相似工单聚类、重复检测、合并/拆分、批量操作
- 售前/销售
  - 视图：关键客户健康度、工单趋势、到期合同提醒
  - 内容：解决方案帖子、FAQ、RCA 对外版（经审核后公开给客户）

核心业务流程

1. 工单创建与预检
   - 客户门户/邮件/接口创建 → 智能表单收集产品、版本、部署形态、影响范围、紧急度
   - RAG 自助：根据输入实时检索知识与历史票据摘要，给出建议步骤与可能解法；用户确认后继续创建或自助关闭
2. 分配与响应
   - 基于产品技能、排班、负载与支持层级的智能路由；未响应 SLA 计时器启动
   - 通知渠道：邮件、Slack/Teams、Webhook；值班（On-Call）联动 PagerDuty
3. 处理与升级
   - 工程师与客户双向沟通，内部备注与 Runbook；必要时升级到高级支持/研发；SLA 阶段性计时（响应/恢复/解决）
4. 解决与复盘
   - 关闭前生成 RCA 草稿与解决摘要，客户满意度反馈（CSAT/NPS）
   - 结构化沉淀：将复盘要点、关键日志特征、命令/步骤写入知识库与向量索引
5. 报告与改进
   - SLA 达成、FRT/MTTR、重复问题 top-N、知识命中率、AI 辅助采用度与质量

系统架构

### 部署架构特色

**企业自主部署模式**：
- **单二进制部署**：整个系统编译为单个可执行文件，无需额外依赖
- **零依赖安装**：内置 SQLite 数据库，无需数据库服务器配置
- **即插即用**：解压即可运行，5分钟完成部署
- **私有化支持**：支持完全离线环境部署，满足安全合规要求
- **资源占用低**：内存占用 < 512MB，适合中小企业部署

### 技术架构

- 前端
  - 客户门户（Web，移动友好）：创建工单、自助 RAG、查看进度、知识库
  - 内部控制台（Web）：工作台、队列、排班、知识编辑、集成配置、审计
  - 数据管理界面：导入导出向导、备份恢复、LLM 配置
- 后端服务（Golang + GIN）- 单体架构
  - Web API 服务：基于 Gin 框架的 REST API，处理 HTTP 请求和响应
  - 认证中间件：JWT/OIDC/SAML 会话注入，租户隔离
  - 业务逻辑层：工单生命周期、知识库管理、SLA 引擎、智能路由
  - **AI 服务集成**：灵活的 LLM Provider 适配器，支持多种 AI 服务
  - **数据管理服务**：导入导出引擎、备份服务、数据转换器
  - 通知服务：邮件/通知模板、节流与重试
- 数据层
  - SQLite 数据库：嵌入式数据库，支持 ACID 事务，数据文件可直接备份
  - 多租户隔离：使用 tenant_id 字段和数据库查询过滤
  - 文件存储：本地文件系统存储附件和文档，支持加密存储
  - 向量存储：简化的内存向量检索（基础版本），可扩展至专业向量数据库
- 可观测与运维
  - 结构化日志：logrus 或 zap
  - 基础指标：运行时指标收集
  - 审计与合规：不可变日志、签名与留存策略（按租户与法规）
  - 备份恢复：SQLite 数据库文件定期备份

数据模型（核心实体）

GORM 模型定义：
```go
// Tenant 租户
type Tenant struct {
    ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    Name         string    `gorm:"type:varchar(255);not null" json:"name"`
    Region       string    `gorm:"type:varchar(100)" json:"region"`
    DataResidency string   `gorm:"type:varchar(100)" json:"data_residency"`
    Settings     string    `gorm:"type:text" json:"settings"` // JSON
    CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// User 用户
type User struct {
    ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID  string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Email     string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
    Role      string    `gorm:"type:varchar(50);not null" json:"role"` // admin/customer/engineer/se/sales
    Profile   string    `gorm:"type:text" json:"profile"` // JSON
    Status    string    `gorm:"type:varchar(20);default:'active'" json:"status"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

// CustomerOrg 客户组织
type CustomerOrg struct {
    ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID     string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Name         string    `gorm:"type:varchar(255);not null" json:"name"`
    Domain       string    `gorm:"type:varchar(255)" json:"domain"`
    BillingInfo  string    `gorm:"type:text" json:"billing_info"` // JSON
    CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

// Product 产品
type Product struct {
    ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID        string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Name            string    `gorm:"type:varchar(255);not null" json:"name"`
    VersioningPolicy string   `gorm:"type:text" json:"versioning_policy"`
    CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Tenant       Tenant         `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Versions     []ProductVersion `gorm:"foreignKey:ProductID" json:"versions,omitempty"`
}

// ProductVersion 产品版本
type ProductVersion struct {
    ID         string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    ProductID  string    `gorm:"type:varchar(36);not null;index" json:"product_id"`
    Version    string    `gorm:"type:varchar(100);not null" json:"version"`
    Status     string    `gorm:"type:varchar(20);default:'GA'" json:"status"` // GA/LTS/EOL
    CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// SupportPlan 支持计划
type SupportPlan struct {
    ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID     string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Name         string    `gorm:"type:varchar(255);not null" json:"name"`
    Description  string    `gorm:"type:text" json:"description"`
    SlaPolicies  string    `gorm:"type:text" json:"sla_policies"` // JSON
    Channels     string    `gorm:"type:text" json:"channels"` // JSON
    Features     string    `gorm:"type:text" json:"features"` // JSON
    PricingModel string    `gorm:"type:varchar(100)" json:"pricing_model"`
    IsActive     bool      `gorm:"default:true" json:"is_active"`
    CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

// Contract 合同
type Contract struct {
    ID             string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID       string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    CustomerOrgID  string    `gorm:"type:varchar(36);not null;index" json:"customer_org_id"`
    Products       string    `gorm:"type:text" json:"products"` // JSON array
    Seats          int       `gorm:"default:10" json:"seats"`
    SupportPlanID  string    `gorm:"type:varchar(36);index" json:"support_plan_id"`
    StartAt        time.Time `gorm:"not null" json:"start_at"`
    EndAt          time.Time `gorm:"not null" json:"end_at"`
    Discounts      string    `gorm:"type:text" json:"discounts"` // JSON
    CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Tenant       Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    CustomerOrg  CustomerOrg `gorm:"foreignKey:CustomerOrgID" json:"customer_org,omitempty"`
    SupportPlan  SupportPlan `gorm:"foreignKey:SupportPlanID" json:"support_plan,omitempty"`
}

// Ticket 工单
type Ticket struct {
    ID                string     `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID          string     `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    CustomerOrgID     string     `gorm:"type:varchar(36);not null;index" json:"customer_org_id"`
    ContractID        string     `gorm:"type:varchar(36);index" json:"contract_id"`
    ProductID         string     `gorm:"type:varchar(36);index" json:"product_id"`
    ProductVersionID  string     `gorm:"type:varchar(36);index" json:"product_version_id"`
    Priority          string     `gorm:"type:varchar(20);default:'medium'" json:"priority"` // low/medium/high/critical
    Severity          string     `gorm:"type:varchar(20);default:'minor'" json:"severity"` // minor/major/critical
    Status            string     `gorm:"type:varchar(20);default:'new'" json:"status"` // new/in_progress/resolved/closed
    AssigneeGroupID   *string    `gorm:"type:varchar(36);index" json:"assignee_group_id"`
    AssigneeID        *string    `gorm:"type:varchar(36);index" json:"assignee_id"`
    CreatedBy         string     `gorm:"type:varchar(36);not null" json:"created_by"`
    CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
    DueAt             *time.Time `json:"due_at"`
    IsDeleted         bool       `gorm:"default:false" json:"is_deleted"`

    Tenant           Tenant         `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    CustomerOrg      CustomerOrg    `gorm:"foreignKey:CustomerOrgID" json:"customer_org,omitempty"`
    Contract         Contract       `gorm:"foreignKey:ContractID" json:"contract,omitempty"`
    Product          Product        `gorm:"foreignKey:ProductID" json:"product,omitempty"`
    ProductVersion   ProductVersion `gorm:"foreignKey:ProductVersionID" json:"product_version,omitempty"`
    Assignee         *User          `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
    Messages         []TicketMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
    Events           []TicketEvent  `gorm:"foreignKey:TicketID" json:"events,omitempty"`
}

// TicketMessage 工单消息
type TicketMessage struct {
    ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TicketID    string    `gorm:"type:varchar(36);not null;index" json:"ticket_id"`
    AuthorID    string    `gorm:"type:varchar(36);not null" json:"author_id"`
    Type        string    `gorm:"type:varchar(20);default:'public'" json:"type"` // public/internal
    Content     string    `gorm:"type:text;not null" json:"content"`
    Attachments string    `gorm:"type:text" json:"attachments"` // JSON
    CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Ticket Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
    Author User   `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
}

// TicketEvent 工单事件/审计
type TicketEvent struct {
    ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TicketID  string    `gorm:"type:varchar(36);not null;index" json:"ticket_id"`
    Type      string    `gorm:"type:varchar(50);not null" json:"type"`
    Payload   string    `gorm:"type:text" json:"payload"` // JSON
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    Actor     string    `gorm:"type:varchar(36);not null" json:"actor"`
    TenantID  string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`

    Ticket Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
    User   User   `gorm:"foreignKey:Actor" json:"user,omitempty"`
}

// KnowledgeArticle 知识文章
type KnowledgeArticle struct {
    ID             string     `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID       string     `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Title          string     `gorm:"type:varchar(500);not null" json:"title"`
    Body           string     `gorm:"type:text;not null" json:"body"`
    Tags           string     `gorm:"type:text" json:"tags"` // JSON
    Visibility     string     `gorm:"type:varchar(20);default:'internal'" json:"visibility"` // internal/customer/public
    Status         string     `gorm:"type:varchar(20);default:'draft'" json:"status"` // draft/published/archived
    RelatedProducts string    `gorm:"type:text" json:"related_products"` // JSON
    Version        int        `gorm:"default:1" json:"version"`
    CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
    PublishedAt    *time.Time `json:"published_at"`
    AuthorID       string     `gorm:"type:varchar(36);not null" json:"author_id"`

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Author User   `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
}

// ImportExportJob 导入导出任务
type ImportExportJob struct {
    ID              string     `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID        string     `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    JobType         string     `gorm:"type:varchar(20);not null" json:"job_type"` // import/export
    EntityType      string     `gorm:"type:varchar(50);not null" json:"entity_type"`
    SourceFormat    string     `gorm:"type:varchar(20);not null" json:"source_format"`
    Status          string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
    Progress        int        `gorm:"default:0" json:"progress"`
    TotalRecords    int        `gorm:"default:0" json:"total_records"`
    ProcessedRecords int       `gorm:"default:0" json:"processed_records"`
    FailedRecords   int        `gorm:"default:0" json:"failed_records"`
    ErrorLog        string     `gorm:"type:text" json:"error_log"`
    CreatedBy       string     `gorm:"type:varchar(36);not null" json:"created_by"`
    CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
    StartedAt       *time.Time `json:"started_at"`
    CompletedAt     *time.Time `json:"completed_at"`
    FilePath        string     `gorm:"type:varchar(500)" json:"file_path"`

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Creator User   `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
```

关键索引策略：
```sql
-- 高频查询优化
CREATE INDEX idx_tickets_tenant_status_created ON tickets(tenant_id, status, created_at DESC);
CREATE INDEX idx_tickets_assignee_status ON tickets(assignee_id, status) WHERE status != 'closed';
CREATE INDEX idx_contracts_tenant_expiry ON contracts(tenant_id, end_at) WHERE end_at > datetime('now');
CREATE INDEX idx_knowledge_tenant_visibility ON knowledge_articles(tenant_id, visibility) WHERE status = 'published';

-- 审计查询优化
CREATE INDEX idx_ticket_events_tenant_created ON ticket_events(tenant_id, created_at DESC);
```

数据完整性约束：
```go
// GORM 验证器
func (t *Ticket) BeforeSave(tx *gorm.DB) error {
    // 验证优先级
    validPriorities := []string{"low", "medium", "high", "critical"}
    if !contains(validPriorities, t.Priority) {
        return fmt.Errorf("invalid priority: %s", t.Priority)
    }

    // 验证严重程度
    validSeverities := []string{"minor", "major", "critical"}
    if !contains(validSeverities, t.Severity) {
        return fmt.Errorf("invalid severity: %s", t.Severity)
    }

    // 验证状态
    validStatuses := []string{"new", "in_progress", "resolved", "closed"}
    if !contains(validStatuses, t.Status) {
        return fmt.Errorf("invalid status: %s", t.Status)
    }

    return nil
}
```

## 自定义 LLM Provider 系统

### 设计理念
**AI 能力完全可控** - 企业可以自由选择和配置任何 LLM Provider，包括公有云服务、私有化部署或本地模型，确保 AI 功能符合企业安全和合规要求。

### 支持的 LLM Provider 类型

**公有云服务**：
- **OpenAI**：GPT-4、GPT-3.5-Turbo、Embedding 模型
- **Azure OpenAI**：企业级 OpenAI 服务，支持私有部署
- **Anthropic Claude**：Claude-3 系列（Opus、Sonnet、Haiku）
- **Google Gemini**：Gemini Pro、Gemini Pro Vision
- **DeepSeek**：DeepSeek-V2、DeepSeek-Coder 系列
- **百度文心一言**：ERNIE-4.0、ERNIE-3.5
- **阿里通义千问**：Qwen-Max、Qwen-Plus

**私有化部署**：
- **Ollama**：本地部署的开源模型（Llama、Mistral、Qwen 等）
- **vLLM**：高性能推理引擎，支持自定义模型
- **Text Generation Inference**：Hugging Face 推理框架
- **LocalAI**：本地 AI 模型服务，兼容 OpenAI API
- **FastChat**：开源对话模型训练和部署平台

**企业专属模型**：
- **微调模型**：基于企业数据微调的专属模型
- **行业模型**：针对特定行业优化的专业模型
- **本地部署大模型**：企业内部部署的闭源或开源模型

### Provider 配置管理

**配置模型设计**：
```go
// LLM Provider 配置
type LlmProvider struct {
    ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID    string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    Name        string    `gorm:"type:varchar(100);not null" json:"name"`
    Type        string    `gorm:"type:varchar(50);not null" json:"type"` // openai/azure/anthropic/local

    // API 配置
    Endpoint    string    `gorm:"type:varchar(500)" json:"endpoint"`
    ApiKey      string    `gorm:"type:varchar(1000)" json:"api_key"` // 加密存储
    ApiSecret   string    `gorm:"type:varchar(1000)" json:"api_secret"` // 加密存储
    Region      string    `gorm:"type:varchar(100)" json:"region"`

    // 模型配置
    ModelName   string    `gorm:"type:varchar(200)" json:"model_name"`
    ModelConfig string    `gorm:"type:text" json:"model_config"` // JSON

    // 高级配置
    MaxTokens   int       `gorm:"default:4096" json:"max_tokens"`
    Temperature float64   `gorm:"default:0.7" json:"temperature"`
    TopP        float64   `gorm:"default:1.0" json:"top_p"`

    // 认证与安全
    AuthType    string    `gorm:"type:varchar(50);default:'api_key'" json:"auth_type"`
    CustomHeaders string  `gorm:"type:text" json:"custom_headers"` // JSON

    // 限制与监控
    RateLimit   int       `gorm:"default:100" json:"rate_limit"` // 每分钟请求数
    CostLimit   float64   `gorm:"default:0" json:"cost_limit"` // 每日费用限制

    Status      string    `gorm:"type:varchar(20);default:'active'" json:"status"`
    IsDefault   bool      `gorm:"default:false" json:"is_default"`
    CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

// 模型配置
type ModelConfig struct {
    // 任务类型映射
    ChatModel      string `json:"chat_model"`       // 对话模型
    EmbeddingModel string `json:"embedding_model"`  // 嵌入模型
    RerankModel    string `json:"rerank_model"`     // 重排模型

    // 性能配置
    Timeout        int    `json:"timeout"`          // 请求超时时间（秒）
    RetryAttempts  int    `json:"retry_attempts"`   // 重试次数
    RetryDelay     int    `json:"retry_delay"`      // 重试延迟（秒）

    // 功能开关
    SupportStream  bool   `json:"support_stream"`   // 支持流式输出
    SupportVision  bool   `json:"support_vision"`   // 支持多模态
    SupportTools   bool   `json:"support_tools"`    // 支持函数调用

    // 安全配置
    ContentFilter  bool   `json:"content_filter"`   // 内容过滤
    DataMasking    bool   `json:"data_masking"`     // 数据脱敏
}
```

### 任务-模型映射系统

**智能任务分配**：
```go
type TaskType string

const (
    TaskChat          TaskType = "chat"           // 对话问答
    TaskEmbedding     TaskType = "embedding"      // 文本嵌入
    TaskRerank        TaskType = "rerank"         // 结果重排
    TaskSummarization TaskType = "summarization"  // 文本摘要
    TaskTranslation   TaskType = "translation"    // 文本翻译
    TaskClassification TaskType = "classification" // 文本分类
    TaskGeneration    TaskType = "generation"     // 内容生成
    TaskRCA           TaskType = "rca"            // 根因分析
)

// 任务路由配置
type TaskRouting struct {
    TenantID     string    `gorm:"primaryKey" json:"tenant_id"`
    TaskType     TaskType  `gorm:"primaryKey" json:"task_type"`
    ProviderID   string    `gorm:"not null" json:"provider_id"`
    Priority     int       `gorm:"default:1" json:"priority"`
    Enabled      bool      `gorm:"default:true" json:"enabled"`
    FallbackProviders []string `json:"fallback_providers"`

    Tenant   Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Provider LlmProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}
```

### 成本监控与优化

**使用统计**：
```go
type LlmUsage struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    TenantID    string    `gorm:"not null;index" json:"tenant_id"`
    ProviderID  string    `gorm:"not null;index" json:"provider_id"`
    TaskType    TaskType  `gorm:"not null" json:"task_type"`
    ModelName   string    `gorm:"not null" json:"model_name"`

    // Token 使用量
    InputTokens  int  `json:"input_tokens"`
    OutputTokens int  `json:"output_tokens"`
    TotalTokens  int  `json:"total_tokens"`

    // 成本计算
    InputCost    float64 `json:"input_cost"`
    OutputCost   float64 `json:"output_cost"`
    TotalCost    float64 `json:"total_cost"`

    // 性能指标
    ResponseTime int     `json:"response_time"` // 毫秒
    Success      bool    `json:"success"`
    ErrorMessage string  `json:"error_message"`

    // 关联信息
    TicketID     *string `gorm:"index" json:"ticket_id"`
    UserID       string  `gorm:"index" json:"user_id"`
    CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}
```

**成本控制策略**：
- **预算限制**：按租户设置每日/每月使用预算
- **智能路由**：根据任务复杂度选择合适的模型
- **缓存优化**：缓存常用查询结果，减少 API 调用
- **批量处理**：合并相似请求，提高效率
- **降级策略**：当达到限制时自动切换到免费模型

### 私有化部署支持

**本地模型部署**：
```go
// 本地模型服务配置
type LocalModelService struct {
    ID           string    `gorm:"primaryKey" json:"id"`
    TenantID     string    `gorm:"not null;index" json:"tenant_id"`
    Name         string    `gorm:"not null" json:"name"`
    Endpoint     string    `gorm:"not null" json:"endpoint"`
    ModelPath    string    `json:"model_path"`

    // 硬件配置
    GPURequired  bool    `gorm:"default:false" json:"gpu_required"`
    MemoryRequired int   `json:"memory_required"` // MB
    DiskRequired   int   `json:"disk_required"`   // MB

    // 模型信息
    ModelSize    int     `json:"model_size"`      // 参数量（亿）
    ContextSize  int     `gorm:"default:4096" json:"context_size"`
    Quantization string  `json:"quantization"`   // 量化方式

    Status       string  `gorm:"default:'stopped'" json:"status"`
    HealthCheck  string  `gorm:"default:'unknown'" json:"health_check"`
    CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
```

**部署方式**：
- **Docker 容器**：提供预配置的 Docker 镜像
- **二进制文件**：独立的可执行文件，无需额外依赖
- **Kubernetes**：支持 K8s 集群部署
- **云原生**：支持各种云平台部署

### 安全与合规

**数据安全**：
- **端到端加密**：API 密钥和传输数据全程加密
- **访问控制**：基于角色的 Provider 管理权限
- **审计日志**：记录所有 AI 调用和配置变更
- **数据脱敏**：自动识别和脱敏敏感信息

**合规支持**：
- **数据驻留**：支持数据本地化部署
- **GDPR 合规**：符合欧盟数据保护法规
- **行业认证**：支持各种行业合规要求
- **审计支持**：提供详细的审计报告

### API 接口设计

```go
// LLM Provider 管理 API
type LlmProviderAPI struct {
    // Provider 管理
    POST   /api/v1/llm/providers           -> CreateProvider
    GET    /api/v1/llm/providers           -> ListProviders
    GET    /api/v1/llm/providers/{id}      -> GetProvider
    PUT    /api/v1/llm/providers/{id}      -> UpdateProvider
    DELETE /api/v1/llm/providers/{id}      -> DeleteProvider

    // 模型配置
    POST   /api/v1/llm/providers/{id}/test -> TestProvider
    GET    /api/v1/llm/providers/{id}/models -> ListModels

    // 使用统计
    GET    /api/v1/llm/usage               -> GetUsageStats
    GET    /api/v1/llm/costs               -> GetCostReport

    // 任务路由
    GET    /api/v1/llm/routing             -> GetTaskRouting
    PUT    /api/v1/llm/routing             -> UpdateTaskRouting
}

// RAG 功能 API
type RAGAPI struct {
    // 文档管理
    POST   /api/v1/rag/documents           -> UploadDocument
    GET    /api/v1/rag/documents           -> ListDocuments
    DELETE /api/v1/rag/documents/{id}      -> DeleteDocument

    // 知识检索
    POST   /api/v1/rag/query               -> QueryRAG
    POST   /api/v1/rag/suggest             -> SuggestReply
    POST   /api/v1/rag/generate-rca        -> GenerateRCA

    // 向量管理
    POST   /api/v1/rag/reindex             -> ReindexAll
    GET    /api/v1/rag/embeddings/{id}     -> GetEmbedding
}
```

### 用户界面

**Provider 配置界面**：
1. **Provider 向导**：引导式配置各种 LLM Provider
2. **连接测试**：实时测试 Provider 连接状态
3. **模型选择**：可视化模型配置和参数调整
4. **成本预估**：实时显示使用成本预估
5. **性能监控**：图表化展示响应时间和成功率

**使用统计仪表盘**：
1. **使用概览**：Token 使用量、费用统计
2. **趋势分析**：时间序列图表展示使用趋势
3. **成本分析**：按部门、用户、任务类型分析成本
4. **告警设置**：预算超支告警和异常监控

这个自定义 LLM Provider 系统确保企业能够：
- 完全控制 AI 能力的选择和配置
- 根据安全和合规要求选择合适的 Provider
- 优化 AI 使用成本
- 保持技术灵活性和可扩展性

SLA 与路由

- 按用户自定义支持计划与优先级定义响应/恢复/解决时限；工作时段感知（非工作时暂停或不同阈值）
- 智能路由：基于产品匹配、负载均衡；根据支持计划级别加权；相似工单聚类优先同一工程师处理
- 升级策略：超过阈值自动提醒 Team Lead/值班；必要时升级到研发

## 数据导入导出系统（核心功能）

### 设计理念
**用户数据用户做主** - SmartTicket 确保企业用户能够完全控制自己的数据，随时导出、备份和迁移所有业务数据。

### 全量数据导出功能

**支持的数据实体**：
- **工单数据**：工单、消息、事件、附件、SLA 记录
- **知识库**：文章、版本、标签、分类、访问日志
- **用户管理**：用户信息、权限分配、组织架构
- **配置数据**：产品目录、支持计划、SLA 策略、集成配置
- **审计日志**：操作记录、登录日志、数据变更历史

**导出格式支持**：
```yaml
export_formats:
  csv:
    description: "Excel 兼容格式，适合数据分析"
    encoding: "UTF-8"
    delimiter: ","
    max_size: "500MB"

  json:
    description: "完整数据结构，适合系统迁移"
    compression: "gzip"
    schema_validation: true

  xml:
    description: "标准化格式，适合企业集成"
    schema: "smartticket_v1.xsd"

  markdown:
    description: "知识文章导出，支持前端元数据"
    front_matter: true
    image_handling: "embed_or_link"

  sqlite:
    description: "完整数据库副本，可直接用于系统恢复"
    encryption: "optional"
```

### 智能导入功能

**数据源适配**：
- **第三方系统**：Zendesk、Jira Service Management、Freshdesk
- **通用格式**：CSV、JSON、XML 文件导入
- **数据库迁移**：MySQL、PostgreSQL、SQL Server 数据迁移
- **历史数据**：邮件、Excel 表格、遗留系统数据

**导入处理流程**：
```go
type ImportProcessor struct {
    // 数据检测与清理
    DataValidator    *DataValidator
    PIIDetector      *PIIDetector
    ConflictResolver *ConflictResolver

    // 字段映射与转换
    FieldMapper      *FieldMapper
    DataTransformer  *DataTransformer

    // 错误处理与报告
    ErrorHandler     *ErrorHandler
    ProgressReporter *ProgressReporter
}

// 导入策略配置
type ImportStrategy struct {
    ConflictResolution string // skip/overwrite/merge/manual
    ValidationLevel    string // strict/lenient/interactive
    BatchSize         int    // 批处理大小
    ProgressCallback  func(progress *ImportProgress)
}
```

### 自动备份与恢复

**备份策略**：
- **定时备份**：每日、每周、每月自动备份
- **增量备份**：仅备份变更数据，节省存储空间
- **全量备份**：定期完整备份，确保数据完整性
- **异地备份**：支持云存储、NAS、移动设备备份

**备份内容**：
- **数据库文件**：SQLite 数据库完整副本
- **文件存储**：附件、文档、导出文件
- **配置文件**：系统配置、环境变量、证书文件
- **日志文件**：操作日志、错误日志、审计日志

**恢复功能**：
- **时间点恢复**：恢复到指定时间点的数据状态
- **选择性恢复**：仅恢复特定租户或特定数据类型
- **灾难恢复**：完整的系统恢复流程
- **数据验证**：恢复后数据完整性验证

### API 接口设计

```go
// 数据导出 API
type ExportAPI struct {
    // 创建导出任务
    POST   /api/v1/data/export      -> CreateExportJob

    // 获取导出进度
    GET    /api/v1/data/export/{id} -> GetExportStatus

    // 下载导出文件
    GET    /api/v1/data/export/{id}/download -> DownloadExportFile

    // 取消导出任务
    DELETE /api/v1/data/export/{id} -> CancelExportJob
}

// 数据导入 API
type ImportAPI struct {
    // 上传导入文件
    POST   /api/v1/data/import/upload -> UploadImportFile

    // 创建导入任务
    POST   /api/v1/data/import      -> CreateImportJob

    // 预览导入数据
    POST   /api/v1/data/import/preview -> PreviewImportData

    // 执行导入
    POST   /api/v1/data/import/{id}/execute -> ExecuteImport
}

// 备份恢复 API
type BackupAPI struct {
    // 创建备份
    POST   /api/v1/data/backup      -> CreateBackup

    // 列出备份
    GET    /api/v1/data/backup      -> ListBackups

    // 恢复备份
    POST   /api/v1/data/backup/{id}/restore -> RestoreBackup

    // 下载备份
    GET    /api/v1/data/backup/{id}/download -> DownloadBackup
}
```

### 用户界面设计

**导入向导**：
1. **数据源选择**：选择文件类型或第三方系统
2. **字段映射**：可视化字段映射界面
3. **预览确认**：数据预览和导入配置确认
4. **进度监控**：实时导入进度和错误报告
5. **结果查看**：导入结果统计和错误详情

**导出向导**：
1. **数据选择**：选择要导出的数据类型和时间范围
2. **格式配置**：选择导出格式和编码选项
3. **过滤条件**：设置数据过滤和排序条件
4. **执行导出**：后台执行导出任务
5. **下载管理**：导出文件下载和管理

**备份管理**：
1. **备份计划**：配置自动备份策略
2. **备份监控**：查看备份状态和历史记录
3. **恢复向导**：引导式数据恢复流程
4. **存储管理**：管理备份文件存储位置和清理策略

### 性能与安全

**性能优化**：
- **并行处理**：多线程处理大批量数据
- **内存管理**：流式处理避免内存溢出
- **压缩存储**：压缩导出文件节省存储空间
- **缓存机制**：缓存常用的映射配置和验证规则

**安全保障**：
- **权限控制**：基于角色的数据导出权限管理
- **数据脱敏**：自动检测和脱敏敏感信息
- **加密存储**：敏感数据加密存储和传输
- **审计记录**：完整的数据操作审计日志

集成接口（概要）

- 身份与访问：SAML/OIDC SSO（通过 JWT）
- 工单入口：Email to Ticket（解析主题/正文/签名/附件）、API、Webhook
- 研发与项目管理：Jira/GitHub Issues 双向链接与状态同步
- 协作：Slack/Teams 机器人（创建/查询/订阅、内外部通道隔离）
- CRM/计费：Salesforce/HubSpot；合同/续费/折扣联动

安全与合规增强

密钥管理与加密：
```go
type Config struct {
    // 数据库密钥
    DatabaseEncryptionKey string `json:"database_encryption_key"`

    // JWT 密钥
    JWTSecret string `json:"jwt_secret"`

    // API 密钥加密
    ApiKeyEncryptionKey string `json:"api_key_encryption_key"`
}
```

审计追踪：
```go
type AuditLog struct {
    ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID  string    `gorm:"type:varchar(36);not null;index" json:"tenant_id"`
    EntityID  string    `gorm:"type:varchar(36);not null" json:"entity_id"`
    Action    string    `gorm:"type:varchar(50);not null" json:"action"`
    OldValues string    `gorm:"type:text" json:"old_values"` // JSON
    NewValues string    `gorm:"type:text" json:"new_values"` // JSON
    ActorID   string    `gorm:"type:varchar(36);not null" json:"actor_id"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    Hash      string    `gorm:"type:varchar(64);not null;index" json:"hash"` // SHA256

    Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
    Actor  User   `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
}
```

性能优化策略

内存缓存策略：
```go
type Cache struct {
    // 用户会话缓存
    UserSessions map[string]*UserSession

    // 租户配置缓存
    TenantConfigs map[string]*TenantConfig

    // SLA 策略缓存
    SLAPolicies map[string]*SLAPolicy

    // 文章缓存
    ArticleCache map[string]*KnowledgeArticle

    mutex sync.RWMutex
}
```

数据库优化：
```sql
-- VACUUM 优化
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;
PRAGMA temp_store = memory;

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_tickets_search ON tickets(title, description, tenant_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_search ON knowledge_articles(title, body, tenant_id);
```

异步处理策略：
```go
type TaskQueue struct {
    tasks chan Task
    workers int
    wg     sync.WaitGroup
}

type Task struct {
    ID     string
    Type   string // sla_notification, rag_processing, report_generation
    Data   interface{}
    Status string // pending, processing, completed, failed
}
```

API 设计（REST）

基础路由结构：
```go
// API 路由定义
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
        tickets.POST("/:id/messages", addMessage)
        tickets.POST("/:id/assign", assignTicket)
        tickets.POST("/:id/resolve", resolveTicket)
        tickets.POST("/:id/close", closeTicket)
    }

    // 知识库
    knowledge := v1.Group("/knowledge")
    {
        knowledge.POST("", createArticle)
        knowledge.GET("", listArticles)
        knowledge.GET("/:id", getArticle)
        knowledge.PUT("/:id", updateArticle)
        knowledge.POST("/:id/publish", publishArticle)
        knowledge.GET("/search", searchArticles)
    }

    // RAG 查询
    rag := v1.Group("/rag")
    {
        rag.POST("/query", queryRAG)
        rag.POST("/suggest", suggestReply)
        rag.POST("/generate-rca", generateRCA)
    }

    // 数据管理
    data := v1.Group("/data")
    {
        data.POST("/import", createImportJob)
        data.POST("/export", createExportJob)
        data.GET("/jobs/:id/status", getJobStatus)
        data.GET("/jobs/:id/download", downloadFile)
    }

    // 管理接口
    admin := v1.Group("/admin")
    {
        admin.Use(adminMiddleware())
        admin.GET("/tenants", listTenants)
        admin.POST("/tenants", createTenant)
        admin.PUT("/tenants/:id", updateTenant)
        admin.GET("/users", listUsers)
        admin.POST("/users", createUser)
        admin.PUT("/users/:id", updateUser)
        admin.GET("/audit-logs", getAuditLogs)
    }
}
```

API 响应格式：
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

type Meta struct {
    Total      int `json:"total,omitempty"`
    Page       int `json:"page,omitempty"`
    PageSize   int `json:"page_size,omitempty"`
    TotalPages int `json:"total_pages,omitempty"`
}
```

实施路线（MVP → GA）

阶段 0：基础设施与多租户基座（2-3周）

**核心交付物：**
- 基础 Golang 项目结构
- SQLite 数据库设计与 GORM 模型
- 基础认证与租户中间件
- REST API 基础框架
- 基础审计日志功能

**成功标准：**
- API 服务可以启动并响应基本请求
- 数据库连接和基础 CRUD 操作正常
- 租户隔离功能正常

阶段 1：工单与SLA核心功能（3-4周）

**核心交付物：**
- 工单 CRUD API 完整实现
- SLA 策略引擎
- 基础通知功能
- 工单状态流转
- 前端基础界面

**成功标准：**
- 端到端工单流程可用
- SLA 监控准确
- 基础邮件通知功能正常

阶段 2：知识库与基础RAG（4-5周）

**核心交付物：**
- 知识库管理 API
- 文档摄取功能
- 简化向量检索
- RAG 查询接口
- LLM 集成（外部 API）

**成功标准：**
- 支持基础文档格式摄取
- 检索响应时间 < 2 秒
- 基础 RAG 功能可用

阶段 3：集成与优化（3-4周）

**核心交付物：**
- 外部系统集成（Jira/GitHub）
- 通知系统完善
- 批量操作功能
- 性能优化
- 基础报表功能

**成功标准：**
- 外部集成稳定可用
- 系统性能满足要求
- 批量操作功能正常

阶段 4：生产部署与完善（2-3周）

**核心交付物：**
- 数据导入导出系统
- 安全加固
- 监控与日志完善
- 部署文档
- 用户文档

**成功标准：**
- 系统具备生产就绪性
- 安全认证通过
- 文档完整

**总体时间线：14-19周（3.5-4.5个月）**

**资源分配建议：**
- 后端开发：1-2人
- 前端开发：1人
- DevOps/基础设施：0.5人（兼职）
- 产品管理：1人（兼职）

技术栈总结

- **后端**：Golang 1.21+
- **Web 框架**：Gin v1.9+
- **ORM**：GORM v1.25+
- **数据库**：SQLite 3.41+
- **认证**：JWT (golang-jwt/jwt)
- **配置**：Viper
- **日志**：Logrus 或 Zap
- **测试**：Go 标准库 + Testify
- **构建**：Go modules + Docker
- **前端**：React + TypeScript（可选）
- **部署**：Docker + Docker Compose

端口配置

- **API 服务**：6533
- **开发数据库**：SQLite 文件路径
- **测试数据库**：独立的 SQLite 文件

开发环境要求

- Go 1.21+
- Docker & Docker Compose
- Git
- SQLite 命令行工具（可选）

性能目标

- API 响应时间 P95 < 200ms
- 工单搜索 P95 < 300ms
- RAG 查询 P95 < 2s
- 支持并发用户数 100+
- 数据库大小支持到 10GB

安全要求

- JWT 认证
- API 密钥加密存储
- 输入验证与 SQL 注入防护
- CORS 配置
- 审计日志记录
- 数据加密存储

后续工作

- 细化 API 契约与数据库设计
- 建立项目仓库与 CI/CD 流水线
- 实施基础 UI 界面
- 建立测试策略与自动化测试
- 制定部署计划与运维文档