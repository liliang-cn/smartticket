# SmartTicket AI 顾问 Team 设计

> 日期:2026-06-12 · Program「对外开放 + AI 顾问 + Live Chat 配置」三子项之 A(实现顺序 B → **A** → C)。
> 决策:agent-go TeamManager + 持久任务队列(方案 A) · 事件→agent 显式 wiring(不用 LLM Dispatcher) · 混合触发(Triage/Sentinel 自动,Researcher/Reviewer 按需) · 全部只给建议,人拍板 · 相似工单用 cortexdb 向量。

## 目标与非目标

**目标**:把现有「单个 AI 助手草稿回复」升级为一组**专家 Agent 团队**,在工单页实时给内部坐席**决策建议**(分类分派、升级风险、相似工单/知识、回复质检),人来采纳。强化「AI 原生」这条产品楔子。

**非目标(YAGNI)**:Agent 自主写回工单(一律建议,人采纳)、LLM Dispatcher 动态路由(用固定事件→agent 映射)、对话式多轮 agent(本批是「一次分析出一张建议卡」)、客户端可见的 AI(本批纯内部坐席辅助;客户侧 AI 先答属 Live Chat 子项 C)。

## 既有架构约定(必须沿用)

- AI 层:`internal/llm`(provider 配置,管理员在 AI Providers 页配)→ `internal/aiassist`(agent-go 编排,`NewGenerator(llmSvc)` 适配 `domain.Generator`,`GenerateStructured` 出结构化结果)。
- RAG:`internal/knowledgebase`(cortexdb 向量)+ `KBSearcher` 工具;现有 `support-assistant`(`agent.New(...)`)产 `Draft{reply,confidence,...}`。
- 事件总线 `internal/automation`:`ticket.created/updated/resolved`、`message.created`、`sla_warning`。
- 实时:`internal/realtime` hub,房间 `ticket:<id>`(坐席 WS `/api/v1/ws/tickets/:id`)。
- 模型集中 `models.go`,Unix 时间戳;Service 配 `_test.go` ≥75%;端口 6533;7 语言 i18n。

## 现状 → 目标的架构跃迁(混合方案,2026-06-12 定)

现在只有一个 agent-go agent(`support-assistant`,直接 `agent.New`,无 TeamManager)。

> **实现决策(读 agent-go v2.79.1 源码后)**:纯 TeamManager 有两个硬约束——① 成员工具只能 MCP/Skills,不能挂 Go 闭包(现有 KB 工具是闭包);② Task 输出是自由文本 `ResultText`,无 schema 校验(结构化卡片得解析文本 JSON,脆)。故采用**混合**:用 `agent.NewTeamManager(store)` + `AddSpecialist(...)` **真实注册** 5 个具名成员(`ListMembers` 可见、prompt 存团队库、是真正的 agent-go team),但每个成员的**实际推理走已验证的 `gen.GenerateStructured(prompt, schema)`**(可靠结构化)+ **复用现有 KB 闭包工具**。编排(触发/持久化/hub 广播)由我们做,不依赖 Task 队列的 LLM 执行。

新模块:`internal/aiteam/`(TeamManager 装配 + 5 成员注册;各 agent 的 prompt 取自成员 Instructions;各 agent 输出 schema + `GenerateStructured` 调用;编排:建上下文→跑 agent→落 `AISuggestion`→hub 广播)。复用 `internal/aiassist` 的 `NewGenerator` / `KBSearcher`,不重写 AI 调用。`support-assistant`(Drafter)逻辑并入。

---

## 第 1 段:Team 成员 + 工具 + 输出 schema

每个成员 = 独立 system prompt + 工具 + `GenerateStructured` 结构化输出(沿用 `aiassist.draftSchema` 模式)。

| Agent | 触发 | 输入 | 工具 | 输出 schema |
|---|---|---|---|---|
| **Triage** | 自动·`ticket.created` | 标题+描述+客户上下文 | products/services 分类目录 | `{priority, severity, category, suggested_team_id?, suggested_assignee_id?, reasoning, confidence}` |
| **Sentinel** | 自动·`message.created`(客户) / `sla_warning` | 完整会话 + SLA 状态 | 无(数据注入) | `{sentiment, churn_risk, sla_breach_risk, escalate, escalate_to?, reasoning, confidence}` |
| **Researcher** | 按需·按钮 | 工单会话 | `KBSearcher` + 新「相似工单」cortexdb 检索 | `{kb_citations[], similar_tickets[{id,title,resolution,merge_candidate}], suggested_resolution, confidence}` |
| **Reviewer** | 按需·发送前 | 坐席草稿 + 工单上下文 + 政策/语气(设置) | 无 | `{issues[{type,severity,note}], revised_draft?, approve, confidence}` |
| **Drafter** | 现有,并入 | 工单会话 | `KBSearcher` | `Draft{reply, confidence, ...}`(已有) |

**相似工单检索(唯一新增工具)**:加一条历史工单索引管线,与 KB 同构 —— 复用 `internal/knowledgebase` 的 `ProviderEmbedder` + cortexdb,单独一个工单 collection,**异步索引**(工单 resolved/updated 时入队建向量,不阻塞主流程)。Researcher 用它做语义检索 + `merge_candidate` 判定。

输出统一带 `confidence`(0..1)+ `reasoning`;低置信卡片降级显示。

---

## 第 2 段:触发与事件 wiring

**自动 Agent**(`server.go` 装配层订阅,让 `aiteam` 包不依赖 ticket/hub):
```go
bus.Subscribe(EventTicketCreated,  → aiteam.Submit("Triage",  ticketID))
bus.Subscribe(EventMessageCreated, → if 发信人是客户 → Submit("Sentinel", ticketID))
bus.Subscribe(EventSLAWarning,     → Submit("Sentinel", ticketID))  // 走 SLA 风险
```

**按需 Agent**(新 API,挂工单下,坐席 JWT + RBAC):
```
POST /api/v1/tickets/:id/ai/research        → Submit("Researcher")
POST /api/v1/tickets/:id/ai/review {draft}  → Submit("Reviewer")
POST /api/v1/tickets/:id/ai/draft           → Drafter(现有逻辑接入)
```

**结果回流**:`aiteam` 通过 `manager.SubscribeTask(taskID)` 收 agent-go 事件 → upsert `AISuggestion` + `hub.Broadcast("ticket:<id>", 卡片状态)` → Copilot 面板实时刷新。

**成本护栏**:Triage 仅 create 跑一次(不随 update 重跑);Sentinel 节流(同工单 N 秒内不重复);均受 AISettings 每-agent 开关控制。失败/超时 → 建议置 `failed`,不影响工单流程。

---

## 第 3 段:Copilot 面板 与 采纳动作

工单详情页(`ticket-detail.tsx`)右侧加 **AI Copilot 面板**,复用现有坐席 WS。每个 agent 一张建议卡:标题 + 结论 + `confidence`(低置信标灰)+ `reasoning` 可展开;过程经 `SubscribeTask` 显示「分析中…」→ 结论。自动 agent 进单即出卡;按需 agent 卡上有「运行」。

**采纳动作(人点了才写回,全部复用既有写 API,Copilot 不引入新写路径):**

| 卡 | 采纳 | 落地动作 |
|---|---|---|
| Triage | 应用优先级/分派 | 写 `priority/severity/category`、team/assignee(现有工单更新 API) |
| Sentinel | 升级 | 改优先级 / 通知主管;仅 `escalate` 时显示红卡 |
| Researcher | 插入方案 / 合并工单 | `suggested_resolution` 填回复框;`merge_candidate` 调现有合并 API |
| Reviewer | 采用修订稿 | `revised_draft` 替换回复框 |
| Drafter | 采用草稿 | 现有逻辑填回复框 |

每张卡可「忽略/收起」,不强塞。

---

## 第 4 段:落库 与 审计

**两个 SQLite 分工**:agent-go `./data/agentgo.db`(Task 队列/checkpoint,框架内部)vs 应用主库新增 `AISuggestion`(面向 UI 的投影,面板读它),`task_id` 关联。

```go
type AISuggestion struct {
    ID         uint
    TicketID   uint
    AgentName  string
    TaskID     string   // 关联 agent-go Task,可回溯/重放
    Status     string   // pending / done / adopted / dismissed / failed
    Confidence float64
    Payload    string   // 结构化输出 JSON
    AdoptedBy  *uint    // 采纳人 userID(审计)
    ResolvedAt *int64
    CreatedAt  int64
}
```

**生命周期**:事件/按钮 → Submit Task + `AISuggestion{pending}` → SubscribeTask 更新 → 完成 `{done,payload,confidence}` → 采纳 `{adopted,AdoptedBy,ResolvedAt}` / 忽略 `{dismissed}`。

**读取**(面板打开即加载,刷新不丢):`GET /api/v1/tickets/:id/ai/suggestions`。

**审计**:采纳写 `AdoptedBy/ResolvedAt` + 复用现有不可变哈希链审计记一条;实际改动走既有 API、本就被审计,Copilot 不绕过。

**管理员自助配置**:扩展 `aiassist.AISettings`,加每-agent 开关 + 自动触发节流秒数;在 Settings → AI 页暴露。自托管方按 LLM 预算自控。

---

## 测试策略

- 各 agent:给定工单/会话,`GenerateStructured` 返回符合 schema(用 mock Generator,见 agent-go testing helpers);低质量输出被 OutputLint 拦截重试。
- 触发:`ticket.created` → 提交 Triage Task 且仅一次;`message.created` 区分客户/坐席;Sentinel 节流生效。
- 相似工单:索引管线异步入库;cortexdb 检索返回相关历史工单;`merge_candidate` 阈值正确。
- 落库/采纳:Task 事件驱动 `AISuggestion` 状态机;采纳调既有写 API 且 `AdoptedBy` 记录;失败置 `failed` 不抛到工单流程。

## 实现顺序(spec A 内部)

1. `internal/aiteam` 骨架:`NewTeamManager` + 5 成员注册(Drafter 并入)+ 各 schema;复用现有 Generator/KBSearcher。
2. 历史工单 cortexdb 索引管线(`knowledgebase` 同构)+ 相似工单工具。
3. 事件订阅器(自动)+ 按需 API + `SubscribeTask` 回流 + `hub` 广播。
4. `AISuggestion` 模型 + 状态机 + `GET .../ai/suggestions`;`AISettings` 扩展每-agent 开关/节流。
5. 前端 Copilot 面板(`ticket-detail.tsx`)+ 采纳动作接既有 API + 7 语言 i18n。
6. `swag init --parseDependency` 刷新 OpenAPI。
