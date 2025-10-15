SmartTicket 设计文档（草案）

更新时间：2025-10-15

概览

SmartTicket 是一款面向 B2B 软件服务商的多租户工单与知识协作平台，目标用户是一家位于欧洲、约 40 人规模的软件公司，为企业客户提供如“高可用存储”等产品与服务，并按照不同支持层级（如铂金/标准）交付 SLA。系统为其客户与内部工程师/售前/销售提供端到端的问题受理、协同处理、知识沉淀与 AI 辅助（原生 RAG/LLM）能力，同时满足 GDPR 与审计合规。

目标与非目标

- 目标
  - 降低平均首次响应时间（FRT）与平均解决时间（MTTR），提升 SLA 兑现率
  - 让客户在创建工单前通过 RAG 自助排查，减少无效/重复工单
  - 支持按“产品/版本/支持层级/合同”细粒度的 SLA 策略、升级路径与排班
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
  - 数据管理：批量导入/导出工单、知识库、用户数据；支持CSV/JSON/XML格式；增量同步与冲突处理
  - LLM 配置：管理 LLM 供应商（OpenAI、Azure OpenAI、DeepSeek、本地/私有化），设置/轮换 API Key/EndPoint/Region，模型白名单与默认模型，任务到模型映射（检索/重排/生成/函数调用），配额与费用上限，速率限制
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

- 前端
  - 客户门户（Web，移动友好）：创建工单、自助 RAG、查看进度、知识库
  - 内部控制台（Web）：工作台、队列、排班、知识编辑、集成配置、审计
- 后端服务（Rust + gRPC）- 简化架构
  - gRPC 网关：统一入口（mTLS、JWT/OIDC/SAML 会话注入），通过 Envoy/Traefik 支持 gRPC-Web；提供可选 JSON 转码以便第三方系统集成
  - Core Service：合并 Ticket + Knowledge 服务，统一处理工单生命周期、知识库管理、SLA 引擎、智能路由
  - AI Service：RAG/LLM 文档摄取、分片、嵌入、混合检索（BM25+向量）、重排、提示编排与安全；内置 AI Provider Adapter 与路由策略
  - Platform Service：合并 Identity + Integration，处理多租户、SSO、RBAC、审计拦截器和外部系统集成
  - Notification Service：邮件/聊天/Push 模板、节流与重试
- 数据层
  - 关系型数据库：PostgreSQL（多租户隔离：tenant_id + RLS；支持强一致事务）
  - 全文检索：OpenSearch/Elasticsearch（标题/描述/备注/知识全文搜索）
  - 向量数据库：PgVector/Weaviate/Qdrant（多租户命名空间 + 细粒度 ACL）
  - 对象存储：S3 兼容（附件、导出、模型缓存）
  - 缓存与队列：Redis（缓存/会话/限流/分布式锁）、Kafka/NATS（异步事件总线）
- 可观测与运维
  - Metrics/Tracing/Logging：Prometheus + Grafana，OpenTelemetry
  - 审计与合规：不可变日志、签名与留存策略（按租户与法规）
  - 备份恢复：数据库/索引/对象存储快照，演练 Runbook

数据模型（核心实体草图）

关键索引策略：
```sql
-- 高频查询优化
CREATE INDEX idx_tickets_tenant_status_created ON tickets(tenant_id, status, created_at DESC);
CREATE INDEX idx_tickets_assignee_status ON tickets(assignee_id, status) WHERE status != 'closed';
CREATE INDEX idx_contracts_tenant_expiry ON contracts(tenant_id, end_at) WHERE end_at > NOW();
CREATE INDEX idx_knowledge_tenant_visibility ON knowledge_articles(tenant_id, visibility) WHERE status = 'published';
CREATE INDEX idx_embedding_tenant_source ON embedding_chunks(tenant_id, source_type, source_id);

-- 审计查询优化
CREATE INDEX idx_audit_tenant_created ON ticket_events(tenant_id, created_at DESC);
CREATE INDEX idx_llm_usage_tenant_date ON llm_usage(tenant_id, created_at DESC);
```

数据完整性约束：
```sql
-- 业务规则约束
ALTER TABLE tickets ADD CONSTRAINT check_priority CHECK (priority IN ('low','medium','high','critical'));
ALTER TABLE tickets ADD CONSTRAINT check_severity CHECK (severity IN ('minor','major','critical'));
ALTER TABLE tickets ADD CONSTRAINT check_status CHECK (status IN ('new','in_progress','resolved','closed'));
ALTER TABLE knowledge_articles ADD CONSTRAINT check_visibility CHECK (visibility IN ('internal','customer','public'));

-- 外键级联删除
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_contract FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE RESTRICT;
ALTER TABLE ticket_messages ADD CONSTRAINT fk_messages_ticket FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE;
```

- Tenant（租户）: id, name, region, data_residency, settings
- User: id, tenant_id, email, role(admin/customer/engineer/se/sales), profile, status
- CustomerOrg（客户组织）: id, tenant_id, name, domain, billing_info
- Product: id, tenant_id, name, versioning_policy
- ProductVersion: id, product_id, version, lifecycle(status: GA/LTS/EOL)
- SupportPlan: id, tenant_id, name, description, sla_policies(json), channels, features(json), pricing_model, is_active
- Contract: id, tenant_id, customer_org_id, products[], seats, support_plan_id, start_at, end_at, discounts[]
- SLA Policy: id, tenant_id, response_time, restore_time, resolve_time, calendar_id, priority_mapping
- Schedule/Calendar: id, tenant_id, business_hours, holidays, oncall_rotations
- Ticket: id, tenant_id, customer_org_id, contract_id, product_id, product_version_id, priority(enum: low/medium/high/critical), severity(enum: minor/major/critical), status(enum: new/in_progress/resolved/closed), assignee_group_id, assignee_id, created_by, created_at, updated_at, due_at, is_deleted(default: false)
- TicketMessage: id, ticket_id, author_id, type(enum: public/internal), content, attachments[], created_at, updated_at
- TicketEvent/Audit: id, ticket_id, type, payload(json), created_at, actor, tenant_id
- KnowledgeArticle(Post): id, tenant_id, title, body, tags[], visibility(enum: internal/customer/public), status(enum: draft/published/archived), related_products[], version(integer), created_at, updated_at, published_at, author_id
- EmbeddingChunk: id, tenant_id, source_type(article/ticket/faq/doc), source_id, chunk_index, vector, metadata(json)
- LlmProvider: id, tenant_id, name(OpenAI/Azure/DeepSeek/local), endpoint, region, status
- LlmCredential: id, tenant_id, provider_id, key_ciphertext, key_kms_ref, created_at, rotated_at, expires_at
- LlmModelConfig: id, tenant_id, provider_id, model_name, purpose(embedding/rerank/generation/tool), default(bool), rate_limits, cost_policy
- PromptTemplate: id, tenant_id, name, content, variables[], version, status
- LlmUsage: id, tenant_id, provider_id, model_name, tokens_in, tokens_out, cost, trace_id, created_at
- IntegrationMapping: id, tenant_id, external_system, external_id, local_ref, direction
- Notification: id, tenant_id, channel, template_id, payload, status, retries
- ImportExportJob: id, tenant_id, job_type(enum: import/export), entity_type, source_format, status, progress, total_records, processed_records, failed_records, error_log, created_by, created_at, started_at, completed_at, file_path

RAG/LLM 方案

- 文档来源
  - 产品手册、部署指南、FAQ、Runbook、历史已解决 Ticket 摘要、变更公告
- 摄取与预处理
  - 支持 PDF/HTML/Markdown/Confluence/Jira/Repo 内 README；统一抽取 → 清洗 → 分片（如 500-1000 tokens overlap）→ 元数据标注（产品/版本/可见性/租户）
  - PII/密钥脱敏，客户私有与公共知识分域
- 嵌入与索引
  - 向量模型：开源或商用（支持欧盟区域托管）；多租户命名空间隔离；配合 BM25 全文检索做 Hybrid Search
  - 定期重建/增量更新；embedding 版本治理
- 检索与重排
  - 先基于意图分类选择索引/命名空间 → 向量召回若干段落 → 语义重排 → 构造上下文包
  - 基于模板的提示编排，控制引用、格式与安全边界（避免泄露其他租户数据）
- 生成与安全
  - 生成回复草稿/RCA/下一步建议；要求带可点击引用；对外回复需人工审核或阈值门控
  - 审计与反馈：用户对 AI 结果打分，自动回流训练与规则优化
  - 供应商与密钥管理：Admin 在控制台配置 OpenAI/DeepSeek 等 Provider 与 API Key；密钥加密存储（KMS/Vault），按租户/环境隔离；为不同任务选择最优/最经济模型；跨境数据流评估与开关（禁止将特定数据发往非 EU 区域）
RAG/LLM 质量评估体系

检索质量指标：
```rust
#[derive(Debug, Serialize)]
struct RAGMetrics {
    // 检索准确性
    precision_at_k: f64,        // Top-K准确率
    recall_at_k: f64,           // Top-K召回率
    mean_reciprocal_rank: f64,  // 平均倒数排名

    // 引用质量
    citation_accuracy: f64,     // 引用准确率
    source_relevance: f64,      // 源文档相关性

    // 业务影响
    deflection_rate: f64,       // 工单自助化率
    user_satisfaction: f64,     // 用户满意度
    time_to_resolution: Duration, // 解决时间改善
}
```

幻觉检测机制：
```python
class HallucinationDetector:
    def __init__(self):
        self.fact_checker = FactCheckModel()
        self.contradiction_detector = ContradictionModel()

    def detect_hallucination(self, response: str, sources: List[str]) -> float:
        # 1. 事实一致性检查
        factual_score = self.fact_checker.verify(response, sources)

        # 2. 逻辑矛盾检测
        contradiction_score = self.contradiction_detector.detect(response)

        # 3. 源文覆盖率检查
        coverage_score = self.calculate_coverage(response, sources)

        return self.aggregate_score(factual_score, contradiction_score, coverage_score)
```

A/B测试框架：
```yaml
ab_test_config:
  rag_experiment:
    name: "embedding_model_comparison"
    variants:
      - name: "control"
        embedding_model: "text-embedding-ada-002"
        rerank_model: "none"
        traffic: 50%
      - name: "treatment"
        embedding_model: "bge-m3"
        rerank_model: "cross-encoder"
        traffic: 50%
    success_metrics:
      - "deflection_rate"
      - "user_satisfaction"
      - "response_time_p95"
    duration: "14_days"
```

SLA 与路由

- 按用户自定义支持计划与优先级定义响应/恢复/解决时限；工作时段感知（非工作时暂停或不同阈值）
- 智能路由：技能标签、负载均衡、排班/值班；根据支持计划级别加权；相似工单聚类优先同一工程师处理
- 升级策略：超过阈值自动提醒 Team Lead/值班；触发 PagerDuty；必要时升级到研发

支持计划配置示例：
```json
{
  "support_plans": [
    {
      "name": "企业VIP",
      "description": "24x7全天候企业级支持",
      "features": {
        "response_time": {
          "critical": "15分钟",
          "high": "30分钟",
          "medium": "1小时",
          "low": "2小时"
        },
        "channels": ["电话", "邮件", "在线聊天", "专属客服"],
        "escalation": "直达技术专家",
        "business_hours": "24x7",
        "extra_features": ["月度健康报告", "专属技术经理", "现场支持"]
      }
    },
    {
      "name": "技术支持Plus",
      "description": "工作日 extended hours 支持",
      "features": {
        "response_time": {
          "critical": "1小时",
          "high": "2小时",
          "medium": "4小时",
          "low": "8小时"
        },
        "channels": ["邮件", "在线聊天"],
        "escalation": "二级技术支持",
        "business_hours": "周一至周五 8:00-20:00",
        "extra_features": ["季度报告", "电话回访"]
      }
    },
    {
      "name": "基础支持",
      "description": "标准工作时间支持",
      "features": {
        "response_time": {
          "critical": "4小时",
          "high": "8小时",
          "medium": "1个工作日",
          "low": "2个工作日"
        },
        "channels": ["邮件"],
        "escalation": "标准升级流程",
        "business_hours": "周一至周五 9:00-17:00"
      }
    }
  ]
}
```

数据导入导出系统

支持的格式与实体：
```yaml
supported_formats:
  tickets:
    csv:
      schema: "ticket_id,title,description,priority,severity,status,customer_email,product,version,created_at"
      encoding: "UTF-8"
      max_size: "100MB"
    json:
      schema: "ticket_schema_v1.json"
      validation: "json_schema"
    xml:
      schema: "ticket_schema_v1.xsd"

  knowledge_articles:
    csv:
      schema: "article_id,title,category,tags,status,visibility,created_at"
      encoding: "UTF-8"
    json:
      schema: "article_schema_v1.json"
    markdown:
      front_matter: true
      image_handling: "base64_or_url"

  users:
    csv:
      schema: "user_id,email,role,name,department,status"
      encoding: "UTF-8"
    json:
      schema: "user_schema_v1.json"

  contracts:
    csv:
      schema: "contract_id,customer_name,product,start_date,end_date,support_plan"
      encoding: "UTF-8"
    json:
      schema: "contract_schema_v1.json"
```

导入流程设计：
```rust
pub struct ImportJob {
    pub id: UUID,
    pub tenant_id: UUID,
    pub entity_type: EntityType,
    pub source_format: Format,
    pub validation_strategy: ValidationStrategy,
    pub conflict_resolution: ConflictResolution,
}

#[derive(Debug)]
pub enum ConflictResolution {
    Skip,                    // 跳过冲突记录
    Overwrite,              // 覆盖现有记录
    CreateNew,              // 创建新记录
    Merge,                  // 智能合并
    RequireManual,          // 需要人工处理
}

#[derive(Debug)]
pub enum ValidationStrategy {
    Strict,                 // 严格模式，任何错误都停止
    Lenient,                // 宽松模式，跳过错误记录
    Interactive,            // 交互模式，逐条确认
}
```

导出功能设计：
```rust
pub struct ExportRequest {
    pub entity_type: EntityType,
    pub filters: ExportFilters,
    pub format: ExportFormat,
    pub fields: Vec<String>,        // 选择导出字段
    pub pagination: Option<Pagination>,
    pub compression: CompressionType,
}

#[derive(Debug)]
pub struct ExportFilters {
    pub date_range: Option<DateRange>,
    pub status_filter: Option<Vec<String>>,
    pub customer_filter: Option<Vec<String>>,
    pub product_filter: Option<Vec<String>>,
    pub custom_filters: HashMap<String, String>,
}
```

批量操作限制：
```yaml
batch_limits:
  max_records_per_file: 50000
  max_file_size: "500MB"
  max_concurrent_jobs: 3
  timeout_per_job: "2 hours"
  rate_limiting: "1000 requests/minute"
```

导入导出API设计：
```protobuf
// gRPC Service Definition
service DataManagementService {
  // 导入操作
  rpc CreateImportJob(CreateImportJobRequest) returns (ImportJob);
  rpc UploadImportFile(stream UploadImportFileRequest) returns (UploadResponse);
  rpc StartImportJob(StartImportJobRequest) returns (ImportJobStatus);
  rpc GetImportJobStatus(GetImportJobStatusRequest) returns (ImportJobStatus);
  rpc CancelImportJob(CancelImportJobRequest) returns (CancelResponse);

  // 导出操作
  rpc CreateExportJob(CreateExportJobRequest) returns (ExportJob);
  rpc GetExportJobStatus(GetExportJobStatusRequest) returns (ExportJobStatus);
  rpc DownloadExportFile(DownloadExportFileRequest) returns (stream FileChunk);
  rpc CancelExportJob(CancelExportJobRequest) returns (CancelResponse);

  // 模板和验证
  rpc GetImportTemplate(GetImportTemplateRequest) returns (ImportTemplate);
  rpc ValidateImportFile(ValidateImportFileRequest) returns (ValidationResult);
}
```

用户界面设计：
```typescript
// 导入界面组件
interface ImportWizardProps {
  entityType: EntityType;
  onImportComplete: (jobId: string) => void;
}

// 导出界面组件
interface ExportWizardProps {
  entityType: EntityType;
  availableFilters: FilterDefinition[];
  onExportComplete: (downloadUrl: string) => void;
}

// 进度追踪组件
interface JobProgressProps {
  jobId: string;
  showRealTimeUpdates: boolean;
}
```

集成接口（概要）

- 身份与访问：SAML/OIDC SSO；SCIM 用户同步
- 工单入口：Email to Ticket（解析主题/正文/签名/附件）、API、Webhook
- 研发与项目管理：Jira/GitHub Issues 双向链接与状态同步
- 协作：Slack/Teams 机器人（创建/查询/订阅、内外部通道隔离）
- On-Call：PagerDuty 事件联动
- CRM/计费：Salesforce/HubSpot、Stripe；合同/续费/折扣联动

安全与合规增强

密钥管理与加密：
```rust
// 密钥分层加密策略
struct EncryptionConfig {
    // 数据库字段级加密（敏感信息）
    field_encryption: AEAD256Config,
    // API密钥存储加密
    key_encryption: RSAKeyEnvelope,
    // 传输中数据加密
    transport: TLSConfig,
    // 备份加密
    backup: AES256GCMConfig,
}
```

审计追踪完整性：
```sql
-- 审计日志防篡改设计
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    actor_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    hash VARCHAR(64) NOT NULL, -- SHA256哈希防篡改
    prev_hash VARCHAR(64), -- 链式哈希
    CONSTRAINT uq_entity_action UNIQUE (entity_type, entity_id, action, created_at)
);

-- 每日哈希校验
CREATE TABLE audit_integrity (
    date DATE PRIMARY KEY,
    daily_hash VARCHAR(64) NOT NULL,
    verified_at TIMESTAMP
);
```

数据脱敏策略：
```yaml
pii_masking:
  email: "partial"      # user***@domain.com
  phone: "partial"      # +1***-***-1234
  ip_address: "hash"    # SHA256哈希
  name: "initial"       # John D. -> J. D.
  custom_fields:
    contract_number: "mask_middle"
    license_key: "hash"
```

GDPR 合规实现：
- 数据主体权利自动化：30天内响应DSR请求
- 数据映射与血缘追踪：完整的数据流图
- 影响评估自动化：DPIA模板与风险评估
- 违约通知自动化：72小时内通知机制

导入导出安全控制：

数据验证与清理：
```rust
pub struct DataValidator {
    pub pii_detector: PIIDetector,
    pub malicious_content_scanner: MaliciousContentScanner,
    pub schema_validator: SchemaValidator,
}

#[derive(Debug)]
pub struct ImportSecurityContext {
    pub max_records_per_batch: usize,
    pub allowed_file_types: Vec<String>,
    pub scan_for_malware: bool,
    pub detect_pii: bool,
    pub quarantine_suspicious: bool,
}
```

权限控制矩阵：
```yaml
import_export_permissions:
  admin:
    can_import: ["tickets", "knowledge_articles", "users", "contracts"]
    can_export: ["tickets", "knowledge_articles", "users", "contracts"]
    max_records: 100000
    can_include_pii: true

  support_manager:
    can_import: ["tickets", "knowledge_articles"]
    can_export: ["tickets", "knowledge_articles"]
    max_records: 50000
    can_include_pii: false

  knowledge_manager:
    can_import: ["knowledge_articles"]
    can_export: ["knowledge_articles"]
    max_records: 20000
    can_include_pii: false

  customer_admin:
    can_import: ["users"]
    can_export: ["tickets", "users"]
    max_records: 10000
    can_include_pii: false
```

数据脱敏与过滤：
```rust
pub struct DataFilter {
    pub exclude_fields: Vec<String>,           // 排除敏感字段
    pub anonymize_fields: Vec<String>,         // 匿名字段
    pub pseudonymize_fields: Vec<String>,      // 假名化字段
    pub exclude_internal_notes: bool,          // 排除内部备注
    pub date_range_limit: Option<DateRange>,   // 限制日期范围
}
```

审计追踪：
```sql
CREATE TABLE import_export_audit (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    job_id UUID REFERENCES import_export_jobs(id),
    operation_type VARCHAR(20) NOT NULL, -- import/export
    entity_type VARCHAR(50) NOT NULL,
    record_count INTEGER,
    file_hash VARCHAR(64),
    pii_detected BOOLEAN DEFAULT FALSE,
    malicious_content_detected BOOLEAN DEFAULT FALSE,
    user_id UUID NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

性能优化策略

缓存分层架构：
```rust
// L1: 应用内存缓存 (热点数据)
struct L1Cache {
    user_sessions: LruCache<UserId, Session>,
    tenant_config: LruCache<TenantId, TenantConfig>,
    sla_policies: LruCache<ContractId, SLAPolicy>,
}

// L2: Redis分布式缓存 (查询结果)
struct L2Cache {
    ticket_search: RedisCache<SearchQuery, SearchResult>,
    rag_results: RedisCache<RAGQuery, RAGResult>,
    user_permissions: RedisCache<UserId, PermissionSet>,
}

// L3: 数据库查询缓存 (复杂查询)
struct L3Cache {
    materialized_views: MaterializedViewCache,
    analytics_queries: QueryResultCache,
}
```

数据库优化：
```sql
-- 分区表策略 (按租户+时间)
CREATE TABLE tickets (
    id UUID,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    -- 其他字段
) PARTITION BY RANGE (created_at);

-- 按月分区
CREATE TABLE tickets_2024_01 PARTITION OF tickets
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- 连接池配置
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET work_mem = '64MB';
```

异步处理策略：
```rust
#[derive(Debug)]
pub struct AsyncTaskQueue {
    // 高优先级队列 (SLA相关)
    sla_notifications: PriorityQueue<SLANotification>,

    // 中优先级队列 (AI处理)
    rag_processing: PriorityQueue<RAGTask>,

    // 低优先级队列 (报告生成)
    report_generation: Queue<ReportTask>,
}

// 背压控制
impl BackpressureControl for AsyncTaskQueue {
    fn should_throttle(&self) -> bool {
        self.queue_size() > self.max_queue_size * 0.8
    }
}
```

CDN与静态资源优化：
```yaml
cdn_config:
  static_assets:
    js_css: "CloudFront"     # 全球CDN
    images: "CloudFront"     # 图片压缩+WebP
    fonts: "Google Fonts"    # 字体CDN

  api_caching:
    knowledge_articles: "1_hour"
    user_profiles: "30_minutes"
    tenant_settings: "5_minutes"

  edge_locations:
    - "eu-west-1"  # 爱尔兰
    - "eu-central-1" # 法兰克福
    - "eu-north-1"  # 斯德哥尔摩
```

- 可靠性：核心服务 SLO 99.9%；关键路径冗余；幂等与重试；限流与熔断
- 性能：典型规模（40 人内部 + 百家客户）下工单检索 P95 < 300ms；RAG 响应 P95 < 2s（不含生成）
- 扩展性：水平扩展支持，分区分片策略，缓存优化，异步处理
- 安全：字段级与记录级访问控制；多层加密策略；零信任架构；审计不可篡改
- 合规：GDPR 数据主体请求（导出/删除）、数据驻留（EU 区域优先），处理者协议与日志留存策略
- 备份恢复：RPO<=15min，RTO<=1h，定期演练

实施路线（MVP → GA）

实施路线（MVP → GA）- 优化依赖关系

阶段 0：基础设施与多租户基座（4-6周）

**核心交付物：**
- 用户认证与SSO集成（Keycloak/Auth0）
- 多租户架构（PostgreSQL RLS + Redis缓存）
- 基础RBAC权限系统
- 审计日志框架
- gRPC服务基础架构

**关键依赖：**
- 云环境配置（AWS/Azure EU区域）
- 数据库schema设计
- 身份提供商集成

**成功标准：**
- 用户可通过SSO登录
- 租户数据完全隔离
- 基础审计功能正常

阶段 1：工单与SLA核心功能（6-8周）

**核心交付物：**
- 工单CRUD操作与状态机
- SLA策略引擎与计时器
- 智能路由算法（基础版）
- 邮件通知系统
- 客户门户基础界面

**并行开发：**
- 前端工单管理界面
- 后端Core Service开发
- SLA监控仪表盘

**关键依赖：**
- 阶段0的认证系统
- 产品与合同数据模型

**成功标准：**
- 端到端工单流程可用
- SLA监控准确
- 邮件通知及时送达

阶段 2：知识库与基础RAG（8-10周）

**核心交付物：**
- 知识库管理系统
- 文档摄取管道
- 基础向量检索
- RAG查询接口
- AI Provider抽象层

**并行开发：**
- 文档解析器开发
- 向量数据库集成
- 检索算法优化

**关键依赖：**
- 阶段1的工单系统（用于知识关联）
- LLM API集成（OpenAI/DeepSeek）

**成功标准：**
- 支持常见文档格式摄取
- 检索响应时间 < 1秒
- 基础RAG功能可用

阶段 3：智能路由与集成（6-8周）

**核心交付物：**
- 高级智能路由（ML增强）
- Jira/GitHub双向集成
- Slack/Teams机器人
- PagerDuty集成
- 相似工单聚类

**并行开发：**
- 外部API适配器
- 机器学习模型训练
- 自动化规则引擎

**关键依赖：**
- 阶段2的RAG系统
- 外部系统API访问权限

**成功标准：**
- 自动路由准确率 > 80%
- 外部集成稳定可用
- 智能建议功能上线

阶段 4：数据管理与生产优化（6-8周）

**核心交付物：**
- 数据导入导出系统（支持CSV/JSON/XML格式）
- 批量操作与作业调度
- 性能优化与缓存策略
- 高级安全功能（字段加密、审计增强）
- GDPR合规工具包
- 监控与告警系统
- 备份与恢复流程

**并行开发：**
- 导入导出界面开发
- 批量处理队列实现
- 性能测试与调优
- 安全审计与渗透测试
- 文档与培训材料

**关键依赖：**
- 前序阶段功能稳定
- 生产环境部署就绪
- 数据处理管道优化

**成功标准：**
- 支持大规模数据导入导出（5万+记录）
- 批量作业成功率 > 99%
- 性能指标达到设计目标
- 安全合规认证通过
- 系统具备生产就绪性

**总体时间线：30-40周（7.5-10个月）**

**资源分配建议：**
- 后端开发：2-3人
- 前端开发：1-2人
- DevOps/基础设施：1人
- 产品管理：1人（兼职）

**风险缓解：**
- 每阶段末进行Go/No-Go决策
- 保持MVP功能最小化
- 建立用户反馈循环机制

技术栈建议（示例）

- 前端：React + TypeScript（Next.js/SPA），Ant Design 或 MUI
- 后端：Rust（tonic gRPC + prost + tokio）
  - 接入：gRPC + gRPC-Web（经 Envoy/Traefik 转发）；可选 JSON 转码供第三方 HTTP 客户端
  - 数据访问：sqlx（异步）、或 sea-orm（可选）；PostgreSQL + RLS
  - 缓存：Redis；消息：Kafka/NATS；任务：tokio + cron/队列
  - 安全：mTLS、JWT/OIDC、SAML（Keycloak/Auth0），tonic 拦截器注入 tenant_id/user_id/roles
- 数据库：PostgreSQL + RLS，多租户；PgVector 作为向量索引（或 Qdrant/Weaviate）
- 检索：OpenSearch/Elasticsearch；缓存：Redis；消息：Kafka/NATS
- 身份：Keycloak/Auth0（SAML/OIDC、SCIM）；对象存储：S3 兼容（AWS/GCP/MinIO）
- 可观测：OpenTelemetry + Prometheus + Grafana；日志：Loki/ELK
- 基础 AI：本地/区域 LLM 推理或 API（EU 区域），开源嵌入模型（bge/multilingual）

gRPC 接口与 proto 纲要（示例）

- 公共约定
  - Metadata：x-tenant-id、x-user-id、x-roles；所有服务要求元数据用于租户隔离与审计；支持请求 ID 与幂等键
  - 错误模型：统一 status code + details（含可本地化 message、可操作建议）
  - 分页：page_size、page_token；排序：order_by、order
- TicketService
  - CreateTicket, GetTicket, ListTickets, UpdateTicket, AddMessage, AssignTicket, MergeTickets, SplitTicket, ResolveTicket, CloseTicket
- KnowledgeService
  - CreateArticle, UpdateArticle, PublishArticle, GetArticle, SearchArticles, LinkTickets
- RAGService
  - IngestDocument, BatchIngest, Reindex, QueryRAG（返回段落+引用）、SuggestReply、GenerateRCA
- IdentityService
  - Authenticate（OIDC/SAML 交换）、WhoAmI、ListUsers、AssignRoles
- IntegrationService
  - ConfigureWebhook, SyncJiraIssue, LinkGitHubIssue, EmailToTicket
- NotificationService

  - Send, TemplatePreview, Subscribe, Unsubscribe

- LlmConfigService
  - ListProviders, UpsertProvider, RotateCredential, ListModels, SetDefaultModel, GetUsage, SetRateLimit, TestConnectivity
- DataManagementService
  - CreateImportJob, UploadImportFile, StartImportJob, GetImportJobStatus, CancelImportJob
  - CreateExportJob, GetExportJobStatus, DownloadExportFile, CancelExportJob
  - GetImportTemplate, ValidateImportFile

关键页面草图

- 客户门户
  - 创建工单页：智能表单 + RAG 建议 + 相似工单提示
  - 工单详情：时间线（客户/内部分层显示）、SLA 计时、引用与附件
  - 知识库：搜索/筛选（产品/版本/可见性），文章与票据摘要
- 内部控制台
  - 工单工作台：队列、SLA 红黄灯、批量操作、宏
  - 知识编辑器：所见即所得 + 引用校验 + 审核发布
  - 配置中心：产品/版本/支持计划、合同与折扣、集成、排班

度量与 OKR 样例

- KR1：客户自助化率 ≥ 25%（由 RAG 引导无须创建工单）
- KR2：铂金客户 FRT P50 ≤ 10 分钟，MTTR P50 ≤ 4 小时
- KR3：知识复用率（月内被引用 ≥1 次的文章比例）≥ 40%
- KR4：AI 建议采用率 ≥ 50%，且用户评分 ≥4/5

后端技术要求规范

代码质量与开发标准

1. 测试覆盖率要求
   - 单元测试覆盖率 ≥ 75%（关键模块 ≥ 85%）
   - 集成测试覆盖所有公共 API 端点
   - 端到端测试覆盖核心业务流程
   - 性能测试覆盖所有数据库查询和 API 调用
   - 安全测试覆盖所有认证和授权路径

2. 测试策略与工具
```rust
// 测试分层结构
tests/
├── unit/                    # 单元测试
│   ├── services/
│   ├── repositories/
│   └── utils/
├── integration/             # 集成测试
│   ├── api/
│   ├── database/
│   └── external_services/
├── e2e/                     # 端到端测试
│   ├── ticket_lifecycle/
│   ├── sla_monitoring/
│   └── rag_pipeline/
├── performance/             # 性能测试
│   ├── load_tests/
│   └── benchmarks/
└── security/                # 安全测试
    ├── auth_flows/
    └── data_isolation/
```

3. 测试工具配置
```toml
# Cargo.toml
[dev-dependencies]
tokio-test = "0.4"
wiremock = "0.5"
testcontainers = "0.15"
criterion = "0.5"
proptest = "1.0"
mockall = "0.11"
quickcheck = "1.0"

[[bench]]
name = "ticket_processing"
harness = false

[[bench]]
name = "rag_performance"
harness = false
```

API 文档与 Swagger 规范

1. gRPC 到 OpenAPI 映射
```protobuf
// 使用 gRPC-Gateway 生成 OpenAPI 规范
syntax = "proto3";

package smartticket.v1;

import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";

service TicketService {
  rpc CreateTicket(CreateTicketRequest) returns (CreateTicketResponse) {
    option (google.api.http) = {
      post: "/api/v1/tickets"
      body: "*"
    };
  }

  rpc ListTickets(ListTicketsRequest) returns (ListTicketsResponse) {
    option (google.api.http) = {
      get: "/api/v1/tickets"
    };
  }
}

message CreateTicketRequest {
  string title = 1;
  string description = 2;
  Priority priority = 3;
  string customer_org_id = 4;
  string product_id = 5;
  map<string, string> metadata = 6;
}
```

2. OpenAPI 配置
```yaml
# openapi.yaml
openapi: 3.0.3
info:
  title: SmartTicket API
  description: B2B Multi-tenant Ticketing and Knowledge Platform
  version: 1.0.0
  contact:
    name: SmartTicket Team
    email: support@smartticket.com
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: https://api.smartticket.com/v1
    description: Production server
  - url: https://staging-api.smartticket.com/v1
    description: Staging server
  - url: http://localhost:8080/v1
    description: Development server

security:
  - BearerAuth: []
  - ApiKeyAuth: []

paths:
  /tickets:
    post:
      tags:
        - Tickets
      summary: Create a new ticket
      description: Create a new support ticket with automatic SLA calculation
      operationId: createTicket
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateTicketRequest'
      responses:
        '201':
          description: Ticket created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Ticket'
        '400':
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
        '401':
          description: Unauthorized
        '403':
          description: Forbidden
```

3. 组件定义
```yaml
components:
  schemas:
    Ticket:
      type: object
      required:
        - id
        - title
        - status
        - created_at
        - tenant_id
      properties:
        id:
          type: string
          format: uuid
          example: "550e8400-e29b-41d4-a716-446655440000"
        title:
          type: string
          minLength: 1
          maxLength: 200
          example: "Database connection issues in production"
        description:
          type: string
          maxLength: 5000
          example: "Cannot connect to PostgreSQL database from application server"
        status:
          $ref: '#/components/schemas/TicketStatus'
        priority:
          $ref: '#/components/schemas/Priority'
        severity:
          $ref: '#/components/schemas/Severity'
        assignee_id:
          type: string
          format: uuid
          nullable: true
        created_at:
          type: string
          format: date-time
          example: "2024-01-15T10:30:00Z"
        updated_at:
          type: string
          format: date-time
          example: "2024-01-15T11:45:00Z"
        due_at:
          type: string
          format: date-time
          nullable: true
          example: "2024-01-15T14:30:00Z"

    TicketStatus:
      type: string
      enum: [new, in_progress, resolved, closed]
      description: |
        - `new`: Newly created ticket, not yet assigned
        - `in_progress`: Ticket is being worked on
        - `resolved`: Ticket has been resolved, waiting for customer confirmation
        - `closed`: Ticket is closed and confirmed
      example: "new"

    Priority:
      type: string
      enum: [low, medium, high, critical]
      description: |
        - `low`: Minor issue, no business impact
        - `medium`: Issue affecting some functionality
        - `high`: Major issue affecting business operations
        - `critical`: Critical issue requiring immediate attention
      example: "high"
```

性能要求

1. 响应时间指标
```rust
// 性能基准测试
use criterion::{black_box, criterion_group, criterion_main, Criterion};

fn benchmark_ticket_creation(c: &mut Criterion) {
    c.bench_function("create_ticket", |b| {
        b.iter(|| {
            // 创建工单的性能测试
            let request = CreateTicketRequest {
                title: "Test Ticket".to_string(),
                description: "Performance test ticket".to_string(),
                priority: Priority::Medium,
                ..Default::default()
            };

            let start = std::time::Instant::now();
            let result = ticket_service.create_ticket(black_box(request));
            let duration = start.elapsed();

            assert!(duration.as_millis() < 100); // < 100ms
            result
        })
    });
}

fn benchmark_rag_query(c: &mut Criterion) {
    c.bench_function("rag_query", |b| {
        b.iter(|| {
            // RAG 查询性能测试
            let query = RAGQuery {
                text: "How to configure database connection?",
                max_results: 5,
                tenant_id: "test-tenant".to_string(),
            };

            let start = std::time::Instant::now();
            let result = rag_service.query(black_box(query));
            let duration = start.elapsed();

            assert!(duration.as_millis() < 2000); // < 2s
            result
        })
    });
}
```

2. 负载测试要求
```yaml
# load-test-config.yml
load_tests:
  ticket_creation:
    concurrent_users: 100
    duration: "5m"
    ramp_up: "30s"
    target_rps: 50
    success_rate: 99.5
    p95_response_time: "200ms"
    p99_response_time: "500ms"

  rag_query:
    concurrent_users: 50
    duration: "3m"
    ramp_up: "15s"
    target_rps: 20
    success_rate: 99.0
    p95_response_time: "1500ms"
    p99_response_time: "3000ms"

  concurrent_tickets:
    scenario: "mixed_operations"
    concurrent_users: 200
    duration: "10m"
    operations:
      - create_ticket: 40%
      - list_tickets: 30%
      - update_ticket: 20%
      - search_knowledge: 10%
```

安全要求

1. 代码安全扫描
```yaml
# .github/workflows/security.yml
name: Security Scan
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run security audit
        run: cargo audit

      - name: Run cargo-deny
        uses: EmbarkStudios/cargo-deny-action@v1

      - name: Run clippy
        run: cargo clippy --all-targets --all-features -- -D warnings

      - name: Run cargo check
        run: cargo check --all-targets --all-features

      - name: Security static analysis
        uses: github/super-linter@v4
        env:
          DEFAULT_BRANCH: main
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          VALIDATE_RUST: true
```

2. 认证与授权测试
```rust
#[cfg(test)]
mod auth_tests {
    use super::*;
    use reqwest::StatusCode;

    #[tokio::test]
    async fn test_unauthorized_access() {
        let app = test_app().await;

        let response = app
            .request(http::Method::GET, "/api/v1/tickets")
            .send()
            .await;

        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn test_tenant_isolation() {
        let app = test_app().await;

        // 租户 A 创建工单
        let tenant_a_token = create_test_token("tenant-a");
        let response = app
            .request(http::Method::POST, "/api/v1/tickets")
            .header("Authorization", format!("Bearer {}", tenant_a_token))
            .json(&test_ticket_request())
            .send()
            .await;

        assert_eq!(response.status(), StatusCode::CREATED);

        // 租户 B 尝试访问租户 A 的工单
        let tenant_b_token = create_test_token("tenant-b");
        let ticket_id = response.json::<TicketResponse>().await.id;

        let response = app
            .request(http::Method::GET, &format!("/api/v1/tickets/{}", ticket_id))
            .header("Authorization", format!("Bearer {}", tenant_b_token))
            .send()
            .await;

        assert_eq!(response.status(), StatusCode::FORBIDDEN);
    }
}
```

监控与日志

1. 结构化日志配置
```rust
use tracing::{info, warn, error, debug, instrument};
use serde_json::json;

#[instrument(skip(self))]
impl TicketService {
    pub async fn create_ticket(&self, request: CreateTicketRequest) -> Result<Ticket> {
        let start = std::time::Instant::now();

        info!(
            tenant_id = %request.tenant_id,
            title = %request.title,
            priority = ?request.priority,
            "Creating new ticket"
        );

        match self.internal_create_ticket(request).await {
            Ok(ticket) => {
                let duration = start.elapsed();
                info!(
                    ticket_id = %ticket.id,
                    duration_ms = duration.as_millis(),
                    "Ticket created successfully"
                );
                Ok(ticket)
            }
            Err(e) => {
                error!(
                    error = %e,
                    duration_ms = start.elapsed().as_millis(),
                    "Failed to create ticket"
                );
                Err(e)
            }
        }
    }
}
```

2. 指标收集
```rust
use prometheus::{Counter, Histogram, Gauge, register_counter, register_histogram, register_gauge};

lazy_static! {
    static ref TICKET_CREATED_TOTAL: Counter = register_counter!(
        "smartticket_tickets_created_total",
        "Total number of tickets created"
    ).unwrap();

    static ref TICKET_PROCESSING_DURATION: Histogram = register_histogram!(
        "smartticket_ticket_processing_duration_seconds",
        "Time spent processing tickets"
    ).unwrap();

    static ref ACTIVE_TICKETS: Gauge = register_gauge!(
        "smartticket_active_tickets",
        "Number of currently active tickets"
    ).unwrap();
}

impl TicketService {
    pub async fn create_ticket(&self, request: CreateTicketRequest) -> Result<Ticket> {
        let timer = TICKET_PROCESSING_DURATION.start_timer();

        let result = self.internal_create_ticket(request).await;

        timer.observe_duration();

        if result.is_ok() {
            TICKET_CREATED_TOTAL.inc();
            ACTIVE_TICKETS.inc();
        }

        result
    }
}
```

部署与 CI/CD 要求

1. Docker 镜像优化
```dockerfile
# Dockerfile
FROM rust:1.75 as builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release && rm -rf src

COPY . .
RUN touch src/main.rs && cargo build --release

FROM debian:bookworm-slim as runtime

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/release/smartticket /usr/local/bin/

EXPOSE 8080
CMD ["smartticket"]
```

2. 健康检查端点
```rust
#[derive(Serialize)]
struct HealthResponse {
    status: String,
    timestamp: chrono::DateTime<chrono::Utc>,
    version: String,
    dependencies: HashMap<String, DependencyStatus>,
}

#[derive(Serialize)]
struct DependencyStatus {
    status: String,
    response_time_ms: Option<u64>,
    error: Option<String>,
}

#[get("/health")]
async fn health_check(
    app_state: web::Data<AppState>,
) -> impl Responder {
    let mut dependencies = HashMap::new();

    // 检查数据库连接
    match sqlx::query("SELECT 1").fetch_one(&app_state.db).await {
        Ok(_) => {
            dependencies.insert("database".to_string(), DependencyStatus {
                status: "healthy".to_string(),
                response_time_ms: Some(10),
                error: None,
            });
        }
        Err(e) => {
            dependencies.insert("database".to_string(), DependencyStatus {
                status: "unhealthy".to_string(),
                response_time_ms: None,
                error: Some(e.to_string()),
            });
        }
    }

    // 检查 Redis 连接
    match app_state.redis.get("health_check").await {
        Ok(_) => {
            dependencies.insert("redis".to_string(), DependencyStatus {
                status: "healthy".to_string(),
                response_time_ms: Some(5),
                error: None,
            });
        }
        Err(e) => {
            dependencies.insert("redis".to_string(), DependencyStatus {
                status: "unhealthy".to_string(),
                response_time_ms: None,
                error: Some(e.to_string()),
            });
        }
    }

    let response = HealthResponse {
        status: "healthy".to_string(),
        timestamp: chrono::Utc::now(),
        version: env!("CARGO_PKG_VERSION").to_string(),
        dependencies,
    };

    HttpResponse::Ok().json(response)
}
```

后续工作

- 细化 API 契约与事件模型；出 ER 图与序列图
- 出最小可行 Schema 与迁移脚本；脚手架项目仓库
- RAG 基线评测集与自动化回归；提示模板库
- 安全部署基线（EU 区域），数据驻留策略落地
- 实施后端技术规范，包括测试覆盖率、API 文档、性能基准
