# SmartTicket 对外集成设计:API Key + 出站 Webhook

> 日期:2026-06-12 · Program「对外开放 + AI 顾问 + Live Chat 配置」三子项之一(实现顺序 **B → A → C**)。
> 决策:API Key 绑服务账号、继承 RBAC(选项 A) · 出站 Webhook 用 DB 队列 + worker(选项 A) · SSRF 内网拦截开关默认关。

## 目标与非目标

**目标**:让第三方系统能把 SmartTicket 接入自己的产品做自动化,补齐「集成生态」缺口的两块地基:

1. **API Key** —— 长效、可吊销的机器凭证,替代「拿登录 JWT 凑合」,用于程序化读写 REST / MCP。
2. **出站 Webhook** —— SmartTicket 在工单事件发生时主动推送到用户配置的 URL,支撑外部系统的事件驱动自动化。

**非目标(YAGNI)**:OAuth2 授权码流 / 第三方应用市场、按 key 独立权限列表(选项 B,暂不做)、Webhook 的可视化规则编排(沿用事件类型过滤即可)、入站 Webhook(邮件→工单已存在,不在本批)。

## 既有架构约定(必须沿用)

- 每个功能 = `internal/<module>/{service.go, handlers.go}`,在 `internal/server/server.go` 注册路由,在 `cmd/server/main.go` 的 `dbModels` 追加模型。
- 模型集中 `internal/models/models.go`,GORM `AutoMigrate`,Unix 时间戳,软删除用 `is_deleted`。
- Service 配 `_test.go`,testify 风格,覆盖率 ≥75%。端口 6533。
- 鉴权:坐席端 JWT + RBAC(`internal/authz`,`Actor{UserID,Role,CustomerID}`);事件总线 `internal/automation`(`ticket.created/updated/resolved`、`message.created`、`sla_warning`)。

---

## 第一部分:API Key

### 数据模型 — `internal/models/models.go`

> 实现发现:`models.APIKey` **已存在**且已注册 `AutoMigrate`,但全仓**无任何读写**(死壳),其字段偏向未选的选项 B(带 `Permissions` JSON,无 `UserID`)。本批**改形复用**它(不新建表,不考虑迁移):加 `UserID` 绑服务账号;**吊销沿用现有 `IsActive`**(置 false);`Permissions` 字段为选项 B 残留,选项 A 下不使用(留作 DB 列,代码忽略)。

```go
type APIKey struct {
    BaseModel
    Name        string     // 人类可读标签,如 "Zapier 集成"
    KeyHash     string     `gorm:"uniqueIndex"` // SHA-256(完整密钥),库内只存哈希
    KeyPrefix   string     // 明文前 12 位,如 "stk_live_a1b2",列表展示
    UserID      uint       `gorm:"index"`       // 【新增】绑定的服务账号;认证后 ActorFromUser 继承其 Role+权限
    User        *User      `gorm:"foreignKey:UserID"`
    IsActive    bool       // 吊销 = 置 false
    ExpiresAt   *time.Time // nil = 永不过期
    LastUsedAt  *time.Time
    CreatorID   uint       // 创建该 key 的管理员 userID(现有字段)
    // Permissions string —— 选项 B 残留,本批不使用
}
```
密钥生成复用现有 `utils.GenerateAPIKey(prefix, length)`。

### 模块 — `internal/apikey/`

- `service.go`
  - `Create(name string, userID uint, expiresAt *int64, createdBy uint) (plaintext string, key *APIKey, err)`
    - 生成 `stk_live_` + 32 字节 `crypto/rand` 十六进制;`KeyPrefix` = 明文前 12 位;`KeyHash` = `sha256(plaintext)`。
    - **明文仅此一次返回**,不落库。
  - `Authenticate(plaintext string) (*models.User, error)`
    - `sha256` 后按 `key_hash` 查;校验未吊销、未过期;异步更新 `LastUsedAt`(best-effort,不阻塞请求)。
  - `List() / Revoke(id)`。
- `token.go`:密钥生成/哈希,配 `token_test.go`。

### 认证中间件 — 扩展 `internal/server/server.go` 现有 `authMiddleware`

`Authorization: Bearer <token>` 按前缀分流(也接受 `X-API-Key:` 头):

```
eyJ...        → 现有 validateJWTToken(JWT 路径,不变)
stk_live_...  → apikey.Authenticate → 载 User → ActorFromUser(u) → 同一个 Actor
其它/失效     → 401
```

API Key 走完得到的 `Actor` 与 JWT 路径完全一致,**下游所有 RBAC 检查零改动**。最小权限通过「为集成新建一个低权限服务账号用户」实现。

### 管理后台 API(admin 组,需 admin 权限)

```
GET    /api/v1/admin/api-keys        列表(只回 KeyPrefix,绝不回明文)
POST   /api/v1/admin/api-keys        创建 → 响应体含明文密钥(仅此一次)
DELETE /api/v1/admin/api-keys/:id    吊销(置 RevokedAt)
```

前端:`web/src/pages/` 下「Settings → API Keys」页;创建后弹窗一次性展示明文 + 复制按钮 + 「关闭后无法再查看」提示。7 语言 i18n。

---

## 第二部分:出站 Webhook

### 数据模型 — `internal/models/models.go`

```go
type Webhook struct {
    ID        uint
    Name      string
    URL       string
    Secret    string   // HMAC 签名密钥,创建时生成
    Events    string   // JSON 数组,如 ["ticket.created","ticket.resolved"]
    Active    bool
    CreatedBy uint
    CreatedAt int64
}

type WebhookDelivery struct {
    ID            uint
    WebhookID     uint
    EventType     string
    Payload       string  // 投递的 JSON 体
    Status        string  // pending / success / failed
    StatusCode    int
    Attempts      int
    LastAttemptAt *int64
    Error         string
    CreatedAt     int64
}
```

### 模块 — `internal/webhook/`

- `service.go`:Webhook 的 CRUD;`enqueue(eventType, payload)` 查订阅了该事件且 `Active` 的 webhook,各写一条 `WebhookDelivery{Status:pending}`。
- `worker.go`:后台投递循环(`go worker.Run(ctx)`,随 server 启动/优雅关闭)。
  - 轮询 `status=pending` 或可重试的 `failed`,POST 投递。
  - **重试**:指数退避,最多 3 次;结果写回 `Status/StatusCode/Attempts/Error`。
  - **重启不丢**:投递状态在库里,worker 重启后继续拉取未完成项。
- `sign.go`:`HMAC-SHA256(body, secret)`,配测试。

### 事件订阅器 — `internal/server/server.go` 装配层

与现有 CSAT 订阅器并列(让 `webhook` 包不依赖 ticket/hub,避免循环 import):

```go
for _, et := range []automation.EventType{
    EventTicketCreated, EventTicketUpdated, EventTicketResolved,
    EventMessageCreated, EventSLAWarning,
} {
    s.bus.Subscribe(et, func(ev automation.Event) {
        payload := buildPayload(ev)        // 载入工单/消息快照
        webhookSvc.Enqueue(string(ev.Type), payload)  // 失败仅记日志,绝不阻塞工单主流程
    })
}
```

### 投递报文

```
POST <Webhook.URL>
  Content-Type: application/json
  X-SmartTicket-Event: ticket.created
  X-SmartTicket-Delivery: <uuid>
  X-SmartTicket-Signature: sha256=<HMAC-SHA256(rawBody, Secret)>

  { "event": "ticket.created",
    "occurred_at": 1749700000,
    "data": { ...工单或消息快照... } }
```

接收方用 `Secret` 重算 HMAC 验真,防伪造。

### SSRF 护栏

URL 由管理员(高权限)填写,风险较低。提供配置项 `webhook.block_private_ips`(默认 **false**,因自托管常需打内网);开启时投递前解析目标 IP,命中私网段(10/8、172.16/12、192.168/16、127/8、::1 等)则拒投并记 `failed`。

### 管理后台 API(admin 组)

```
GET/POST/DELETE /api/v1/admin/webhooks
GET            /api/v1/admin/webhooks/:id/deliveries   投递日志(排障/可观测)
POST           /api/v1/admin/webhooks/:id/test         发一条 ping 事件试通
```

前端:「Settings → Webhooks」页,含端点 CRUD、事件多选、投递历史(状态码/重试次数/错误)、发送测试。7 语言 i18n。

---

## 测试策略

- `apikey`:生成→哈希→认证往返;过期/吊销→拒;`LastUsedAt` 更新;中间件 JWT 与 API Key 双路径分流。
- `webhook`:Enqueue 按事件过滤命中正确订阅者;worker 重试与退避;HMAC 签名稳定且可被独立验证;SSRF 开关命中私网拦截。
- 集成:用 API Key 调一个受 RBAC 保护的 REST 端点,验证 Actor 权限与等价 JWT 一致;工单事件触发 → 投递日志落 `success`。

## 实现顺序(spec B 内部)

1. `models` 加 `APIKey`/`Webhook`/`WebhookDelivery` + `AutoMigrate`。
2. `internal/apikey` 模块 + 中间件分流 + admin CRUD + 前端页。
3. `internal/webhook` 模块(service/worker/sign)+ 事件订阅器 + admin CRUD + 前端页。
4. 文档:`swag init --parseDependency`(见记忆 swagger 重生成),刷新 `docs/swagger.yaml`。
