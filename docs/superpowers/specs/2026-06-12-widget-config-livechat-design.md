# SmartTicket Widget 后台配置 + Live Chat 设计

> 日期:2026-06-12 · Program「对外开放 + AI 顾问 + Live Chat 配置」三子项之 C(实现顺序 B → A → **C**)。
> 决策:部署方管理员后台自助配置 · 架构方案 A(新 `widget` 设置模块,照搬 branding 公开 GET + 管理员 PUT,外观继承 branding) · 营业时间用完整周计划+时区+离线转邮件(选项 A) · AI 先答复用 spec A 的 Drafter。

## 背景与目标

**现状(已存在,无需重建)**:Live Chat 实时双向已跑通 —— 访客 widget 开 WS `/widget/ws` 订阅 `widget:<ticketID>`,坐席工单页订阅 `ticket:<ticketID>`,`ticket/service.go` 每条新消息同时广播到两个房间。接入靠一行 `<script src="/widget.js" data-key=... data-accent=...>`。

**问题**:配置全写死或埋在 embed 脚本属性里,部署方管理员**无法在后台自助配置** widget。`data-key` 读了没用;branding 模块未接到 widget;后台无 Widget 设置页。

**目标**:加一个**管理员可自助配置**的 Widget 设置面,覆盖外观/预聊/营业时间/路由,改完即时生效、无需碰 embed 脚本。

**非目标(YAGNI)**:多站点多 widget(单租户单 widget)、widget 内嵌知识库自助检索 UI(本批不做)、访客端自定义 CSS 注入。

## 既有架构约定(必须沿用)

- 配置模式照搬 `internal/branding`(公开 `GET /settings/branding` + 管理员 `PUT`)与 `aiassist.SettingsStore`(单例 Get/Update)。
- widget:`internal/widget/{service,handlers,token}.go`,客户端 `web-widget/src/{widget,ui,api}.ts`,bundle `/widget.js`。
- email 模块 `internal/email`(IMAP 收 + SMTP 发);Teams `internal/team`;AI Drafter 见 spec A `internal/aiteam`。
- 模型集中 `models.go`,Unix 时间戳;端口 6533;后台前端 `web/src/pages/`,7 语言 i18n。

---

## C.1 模型 + 存储

### 数据模型 — `internal/models/models.go`

```go
type WidgetSettings struct {
    ID      uint   // 单例,id=1
    Enabled bool
    // 外观(AccentColor 为空 → 继承 branding 主题色)
    AccentColor      string
    AppName          string
    WelcomeMessage   string
    AgentDisplayName string
    AgentAvatarURL   string
    LauncherPosition string // bottom-right / bottom-left
    // 预聊
    PrechatEnabled      bool
    PrechatRequireName  bool
    PrechatRequireEmail bool
    PrechatGreeting     string
    // 营业时间 / 离线
    Timezone            string // IANA,如 "Asia/Shanghai"
    BusinessHours       string // JSON: [{"day":1,"open":"09:00","close":"18:00"}, ...]
    OfflineMessage      string
    OfflineCollectEmail bool
    // 路由
    RouteTeamID  *uint
    AIFirstReply bool   // true → 首条客户消息调 spec A Drafter 自动答
    UpdatedAt    int64
}
```

### 模块 — `internal/widget/settings.go`(扩展现有 widget 包)

- `SettingsStore{db}`:`Get() (*WidgetSettings, error)`(无行则返回默认值)、`Update(in UpdateWidgetSettings) error`(单例 upsert id=1)。
- 校验:`Timezone` 可被 `time.LoadLocation` 解析;`BusinessHours` 为合法 JSON 且时段格式正确;`RouteTeamID` 指向存在的 team。

### 路由

```
GET /widget/config        公开(widget.js 无登录读;widgetCORS),只回客户端需要的字段
PUT /api/v1/settings/widget   管理员(admin 组),全量更新
```

`GET /widget/config` 投影出客户端所需子集(外观 + 预聊配置 + 当前在线/离线状态),**不泄露**路由/内部字段。在线状态由服务端按 `Timezone`+`BusinessHours`+当前时刻计算后返回布尔。

---

## C.2 接到三个消费点

### ① widget.js 启动 — `web-widget/src/widget.ts`

- 启动时 `fetch('/widget/config')` → 渲染 `AccentColor`(空则用 branding)、`AppName`、`WelcomeMessage`、`LauncherPosition`、`AgentDisplayName/Avatar`。
- `Enabled=false` → 不渲染 launcher,直接 return。
- 现有 `data-accent` / appName 作**向后兼容回退**(config 拉取失败时用)。`data-key` = 站点开关/标识。
- 在线/离线:config 返回 `online` 布尔 → 离线时 launcher 展示 `OfflineMessage`、表单切为「留邮箱」模式。

### ② POST /widget/session — `internal/widget/service.go`

- **预聊校验**:`PrechatRequireName/Email` 为真时服务端强制校验,空则 400(不只靠前端)。
- **营业时间**:按 `Timezone`+`BusinessHours` 判当前在线;离线 → 仍建 `web_widget` 工单但打 `offline` 标记,响应带 `OfflineMessage`;`OfflineCollectEmail` 时要求邮箱。
- **路由**:`RouteTeamID` 非空 → 新工单分配给该 team(复用现有工单分配逻辑)。

### ③ AI 先答 — 复用 spec A

`AIFirstReply=true` 且在线时,首条客户消息落库后 → 调 `aiteam` 的 **Drafter**(spec A 已建)出草稿:高置信直接作为坐席消息回推 widget(经现有广播),低置信仅落 Copilot 面板给坐席。**不新写 AI 代码**,只多一个调用点。离线时不触发。

---

## C.3 后台配置页

`web/src/pages/widget-settings.tsx`(或并入 Settings):
- 分区:外观 / 预聊 / 营业时间(周计划编辑器 + 时区选择)/ 路由(team 下拉 + AI 先答开关)/ 总开关。
- **实时预览**:右侧渲染 widget 外观随表单变化。
- **Embed 片段**:展示 `<script src="https://<本后端>/widget.js" data-key="..." async></script>` + 复制按钮。
- 保存 → `PUT /api/v1/settings/widget`。7 语言 i18n。

---

## 离线 → 邮件回流

离线时段建的工单标 `offline`。坐席之后在工单页回复:
- 若访客 widget WS 仍在线 → 现有广播即时收到(无需改动)。
- 若访客已离开 → 复用 `internal/email` SMTP,把坐席回复发到访客预聊填写的邮箱(沿用现有 `ticket/mailer.go` 邮件通道)。判定:访客 `widget:<ticketID>` 房间无在线订阅者(`hub.Presence(room)==0`)时走邮件。

---

## 测试策略

- `widget.SettingsStore`:单例 upsert;默认值;`Timezone`/`BusinessHours`/`RouteTeamID` 校验。
- 在线判定:跨时区、跨午夜、周末时段的营业时间计算(表驱动)。
- session:预聊必填服务端拦截;离线建单打标 + 返回 OfflineMessage;`RouteTeamID` 分配生效。
- 离线回流:`hub.Presence==0` 时走 SMTP,在线时不发邮件(避免重复)。
- AI 先答:`AIFirstReply` 高/低置信分流;离线不触发。

## 实现顺序(spec C 内部)

1. `WidgetSettings` 模型 + `AutoMigrate` + `widget.SettingsStore` + 校验。
2. `GET /widget/config`(公开投影 + 在线状态计算)+ `PUT /api/v1/settings/widget`(admin)。
3. widget.js 启动读 config + 离线模式 UI(`web-widget/`,重新 `pnpm build`)。
4. session 预聊校验 / 营业时间 / 路由接线;离线→邮件回流。
5. AI 先答接 spec A Drafter。
6. 后台 Widget 设置页 + 实时预览 + embed 片段 + i18n。
7. `swag init --parseDependency` 刷新 OpenAPI。
