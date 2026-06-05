# SmartTicket 竞品补齐设计(feature/parity)

> 日期:2026-06-04 · 基于 2026-06-02 竞品分析(`docs/competitive-analysis.md`)中识别的 7 个一/二级缺口。
> 决策:WebSocket 实时传输 · AI 默认建议可开自动 · 通用规则引擎 · 单分支全做完再合 main。

## 目标与非目标

**目标**:在不偏离"自托管 + 数据主权 + 可换 AI"楔子的前提下,补齐 7 个对 demo / 范式至关重要的缺口:

| # | 功能 | 等级 |
|---|---|---|
| 1 | 网页聊天 Widget(可嵌入 JS) | 🔴 |
| 2 | AI 自动结案(默认建议、可开自动) | 🔴 |
| 3 | 通用触发器 / 自动化引擎 | 🔴 |
| 4 | Macros / 预设回复 | 🟡 |
| 5 | CSAT 满意度调查 | 🟡 |
| 6 | 坐席协作(@mentions · teams · collision) | 🟡 |
| 7 | 工单合并 / 关联 | 🟡 |

**非目标(YAGNI)**:社交渠道(WhatsApp/Telegram/Slack/Discord)、SSO/SAML/OAuth、自定义报表构建器、知识文章多语翻译、iOS 客户端(本批暂缓,单独跟进)。

## 既有架构约定(必须沿用)

- 每个功能 = `internal/<module>/{service.go, handlers.go}`,在 `internal/server/server.go` 的 `setupRoutes()` 注册路由,在 `cmd/server/main.go` 的 `dbModels` 切片追加模型。
- 模型集中在 `internal/models/models.go`,GORM `AutoMigrate`,Unix 时间戳,软删除用 `is_deleted`。
- Service 配 `_test.go`,testify 风格,目标覆盖率 ≥75%。
- 端口沿用 6533;前端 `web/src/pages/*` + 7 语言 i18n(`web/src/locales/`)。
- 鉴权:坐席端 JWT + RBAC(`internal/authz`);公开端用签名 token。

---

## 横向基础设施(被多功能复用)

### A. WebSocket Hub — `internal/realtime/`

单进程内存 hub,无外部依赖(契合单二进制 / SQLite)。

- `hub.go`:`map[room]map[*client]struct{}`,带 `Register/Unregister/Broadcast(room, payload)`,goroutine + channel,读写加锁。
- 房间命名:
  - `ticket:<ticketID>` — 坐席端:实时新消息、typing、collision(谁在查看/输入)。
  - `widget:<ticketID>` — 访客端:实时收发本会话消息。
- 传输库:`github.com/gorilla/websocket`。
- 端点:
  - `GET /api/v1/ws`(坐席,JWT 鉴权,中间件复用现有 auth)。
  - `GET /widget/ws`(访客,`conversation_token` 查询参数鉴权)。
- 触发点:`ticket.Service.CreateMessage` 成功后 `hub.Broadcast("ticket:<id>", msg)` 与(若 widget 会话)`hub.Broadcast("widget:<id>", msg)`。现有轮询保留为降级路径。
- 消息信封:`{type: "message"|"typing"|"presence"|"status", payload: {...}}`。

### B. Domain Event Bus — `internal/automation/events.go`

轻量进程内事件总线(`map[EventType][]Handler`,同步派发)。领域事件:`ticket.created`、`ticket.updated`、`message.created`、`ticket.sla_warning`、`ticket.resolved`。触发器引擎(功能 3)、自动结案(功能 2)、CSAT 发送(功能 5)都订阅这条总线,避免在 ticket service 里硬编码各功能调用。

---

## 功能 1 — 网页聊天 Widget

### 嵌入
客户网站插入:
```html
<script src="https://<实例>/widget.js" data-key="<widget_public_key>" async></script>
```
`widget.js` 是独立构建产物(目录 `web-widget/`,纯 TS,无 React,目标 < 20KB gzip,Vite library 模式)。渲染右下角气泡 + 点击展开的 iframe 面板(iframe 指向 `/widget/app`,样式隔离)。

### 会话模型(复用 Ticket)
- Ticket 新增 `Channel` 枚举值 `web_widget`,新增字段 `ConversationToken`。
- 访客首次打开:`POST /widget/session`(body 可含 pre-chat 的 email/name;**pre-chat 收 email 为可选**,匿名也能开)→ 创建/复用匿名 `Customer`(按 email 或浏览器 token)+ 一个 `open` Ticket(`channel=web_widget`)→ 返回签名 `conversation_token`(JWT,claims 含 ticketID,7 天有效)。
- 后续消息用该 token 鉴权,绑定到对应 ticket。

### 实时
访客 WebSocket 接 `widget:<ticketID>`;坐席在现有 ticket-detail 页正常回复 → 经 Hub 实时推到 widget。访客发消息 → `POST /widget/messages`(token 鉴权)+ Hub 推到 `ticket:<id>`。

### 后端
- `internal/widget/{service.go, handlers.go}`。
- 公开路由组 `/widget/*`(无 JWT;`conversation_token` 鉴权):`POST /widget/session`、`POST /widget/messages`、`GET /widget/messages`(历史)、`GET /widget/ws`。
- 静态:`GET /widget.js` 服务构建产物;`GET /widget/app` 服务 iframe 页。
- Widget 公钥配置:`SystemSetting` 存 `widget.public_key`(admin 在 `/settings` 生成/查看,用于校验来源域名白名单,可选)。

### AI 接入
widget 会话的 `message.created` 事件照常进自动结案流程(功能 2);widget 面板内支持 CSAT 内嵌评分(功能 5)。

---

## 功能 2 — AI 自动结案(默认建议、可开自动)

扩展 `internal/aiassist` 与 `models.AISettings`。

### 新增设置(`models.AISettings`,均可在 `/settings` 改)
| 字段 | 默认 | 说明 |
|---|---|---|
| `AutoReplyEnabled` | false | 开启高置信度自动回复 |
| `AutoReplyConfidence` | 0.75 | 自动发送的置信度阈值 |
| `AutoResolveEnabled` | false | AI 回复后允许自动结案 |
| `MaxAutoRepliesPerTicket` | 2 | 连续自动回复上限,超过转人工 |
| `AutoClassifyEnabled` | false | 建单时自动设 category/priority/tags |
| `AutoSummarizeOnResolve` | false | 结案时生成摘要 |

### 流程(订阅 `message.created` / `ticket.created`)
1. RAG 检索知识库(现有 `knowledgebase`)→ agent-go 生成草稿,产出 **置信度**(检索 top 分数 + 模型自评,归一到 0–1)。
2. 若 `AutoReplyEnabled && 置信度 ≥ 阈值 && 本工单自动回复数 < Max`:自动发回复(`IsFromAI=true`),计数,写 `TicketEvent`。
3. 否则:落为"建议回复"交坐席(现有 `SuggestReply` UI / 通知)。
4. 连续 N 次自动回复未结案、或检测到客户负面情绪 → 自动转人工(通知坐席,停止自动回复)。
5. `AutoClassify`:建单事件里 AI 设分类/优先级/标签。
6. `AutoSummarizeOnResolve`:`ticket.resolved` 事件生成摘要写 `Ticket.Summary`。

### 自动结案触发
AI 回复后,由功能 3 的"超时定时规则"驱动:客户 X 时间无回复且 `AutoResolveEnabled` → 置 `resolved`。所有自动动作写 `TicketEvent` 可审计。

---

## 功能 3 — 通用触发器 / 自动化引擎

### 模型 `AutomationRule`
```
ID, Name, Description, Enabled, Position(执行顺序)
Event:    ticket.created | ticket.updated | message.created | ticket.sla_warning | schedule
Match:    "all" | "any"
Conditions JSON: [{field, op, value}]   // field 如 status/priority/severity/channel/tags/customer_email
Actions    JSON: [{type, params}]
```

### 执行器 `internal/automation/engine.go`
- 订阅 Domain Event Bus;事件到达 → 按 `Position` 取 `Enabled` 规则 → 条件匹配 → 顺序执行动作。
- 动作类型:`assign`(user/team)、`add_tag`、`set_priority`、`set_status`、`set_severity`、`notify`(站内/邮件)、`send_email`、`escalate`、`ai_suggest`、`ai_auto_reply`、`close`。
- **定时**:进程内 ticker(每 60s)扫 `event=schedule` 规则 + SLA 超时 / 客户无回复超时 → 驱动升级、自动结案(功能 2)、CSAT 发送(功能 5)。
- 接管 `Service` 模型现有 escalation JSON 字段(此前无执行器)。
- 防循环:自动化产生的事件标记来源,`ai_auto_reply` / `close` 等动作不再二次触发同类规则。

### 管理 UI
`/automations` 页(admin):规则列表 + 启停 + 拖拽排序;条件/动作可视化编辑器(下拉选 field/op/action,无需写 DSL)。

---

## 功能 4 — Macros / 预设回复

### 模型 `Macro`
```
ID, Title, Category, Body(支持变量), Actions JSON(可选:顺带改状态/打标签)
Shared(bool), OwnerID(私有归属), UsageCount, CreatedAt/UpdatedAt
```
变量:`{{customer.name}}`、`{{ticket.id}}`、`{{ticket.subject}}`、`{{agent.name}}`。

### 后端
`internal/macro/{service, handlers}`:
- `GET/POST/PUT/DELETE /macros`(CRUD;私有 macro 仅 owner 可见)。
- `POST /macros/:id/apply?ticket_id=` → 返回渲染后文本 + 待执行动作,`UsageCount++`。

### 前端
ticket-detail 回复框上方加"插入 Macro"选择器(搜索 + 分类),选中 → 填入正文 + 执行附带动作。

---

## 功能 5 — CSAT 满意度调查

### 模型 `SatisfactionSurvey`
```
ID, TicketID, Rating(1-5), Comment, Token(签名), SentAt, RespondedAt
```

### 流程
- 工单 `resolved`/`closed` 事件 → 触发器(功能 3)发调查:
  - 邮件渠道:发带签名链接 `/survey/:token` 的邮件(复用 `internal/email`)。
  - widget 渠道:在 widget 面板内直接弹 1–5 评分。
- 公开路由 `/api/v1/survey/:token`:`GET`(取问卷)/ `POST`(提交评分+评论),写 `RespondedAt`。
- 公开页 `/survey/:token`(无需登录)。

### 报表
Dashboard 加 CSAT 卡片:平均分、回收率、近 30 天趋势。

---

## 功能 6 — 坐席协作

### @mentions
- 内部备注(`IsInternal`)正文解析 `@username` → 给被提及坐席发 `Notification`(复用 `internal/notification`)。
- ticket-detail 渲染时高亮 @提及。

### Teams / Groups
- 新模型 `Team`(`Name, Description`)+ `TeamMember`(user ↔ team,带角色可选)。
- Ticket 增 `AssignedTeamID`(与 `AssignedTo` 并存:可只分到组,或组内某人)。
- 触发器 `assign` 动作支持 assign-to-team。
- `/teams` 管理页(admin):建组、加/减成员。

### Agent collision
- 经 WebSocket Hub `ticket:<id>` 房间广播 presence + typing。
- ticket-detail 顶部条显示"张三正在查看 / 李四正在输入…",防止重复回复。

---

## 功能 7 — 工单合并 / 关联

### 合并
- `POST /tickets/:id/merge { into: <targetID> }`:源工单 messages + attachments 迁移到目标;源置 `status=merged`(新枚举值)+ `MergedIntoID`;写 `TicketEvent`;通知相关方。
- 合并不可逆(软关联保留可追溯)。

### 关联
- 模型 `TicketLink`(`SourceID, TargetID, Type: related | duplicate | blocks`)。
- `POST/DELETE /tickets/:id/links`;ticket-detail 显示关联工单列表与跳转。

---

## 数据模型增量

**新表**:`AutomationRule`、`Macro`、`SatisfactionSurvey`、`Team`、`TeamMember`、`TicketLink`。

**Ticket 增字段**:`Summary`、`AssignedTeamID`、`MergedIntoID`、`ConversationToken`;`Channel` 增枚举 `web_widget`;`Status` 增枚举 `merged`。

**AISettings 增字段**:见功能 2 表格。

全部追加到 `cmd/server/main.go` 的 `dbModels`(两处:`migrate` 与 `serve` 路径)。

## 前端增量

- 新页:`/automations`、`/macros`、`/teams`。
- ticket-detail:macro 选择器、合并/关联操作、collision 提示条、@mention 高亮、WebSocket 实时消息(替代/补充轮询)。
- `/settings` AI 区:自动结案开关 + 阈值滑块。
- dashboard:CSAT 卡片。
- 独立 widget 构建 `web-widget/`(气泡 + iframe 面板 + WebSocket)。
- 公开页:`/survey/:token`、`/widget/app`。
- 7 语言 i18n:所有新文案进 `web/src/locales/`。

## 落地顺序(同一 `feature/parity` 分支,逐功能 commit + 测试)

```
0. Hub + Event Bus 基础设施
1. Widget(后端会话 + WS + widget.js + iframe 页)
2. AI 自动结案(AISettings + 流程 + 接 Event Bus)
3. 触发器引擎(模型 + engine + ticker + /automations)
4. Macros
5. CSAT
6. 协作(@mentions + teams + collision)
7. 合并 / 关联
8. 前端收尾 + i18n + 文档更新 competitive-analysis.md
```

每个 Go service 配 `_test.go`。每完成一块即可本地 `make build` + 部署验证。

## 风险与缓解

- **WebSocket 与单进程**:hub 在内存,多副本部署会分裂房间 —— 当前单二进制单实例部署,无影响;未来多副本需外部 pub/sub(记入 TODO,不在本批)。
- **自动回复跑偏**:默认全关 + 置信度阈值 + 每单上限 + 负面情绪转人工,多重闸门。
- **触发器死循环**:动作来源标记 + 自动动作不二次触发。
- **合并不可逆**:UI 二次确认 + 全程 `TicketEvent` 审计。
