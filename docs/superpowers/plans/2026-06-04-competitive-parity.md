# Competitive Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 7 competitor-parity features (web chat widget, AI auto-resolve, automation engine, macros, CSAT, agent collaboration, ticket merge) on the `feature/parity` branch.

**Architecture:** Each feature is a Go module `internal/<name>/{service.go,handlers.go}` registered in `internal/server/server.go:setupRoutes()`, with models in `internal/models/models.go` added to `cmd/server/main.go` `dbModels`. Two shared primitives are built first: an in-process WebSocket **Hub** (`internal/realtime`) and a synchronous **domain event bus** (`internal/automation/events.go`) that decouples auto-resolve, triggers, and CSAT from the ticket service.

**Tech Stack:** Go 1.25, Gin, GORM + modernc SQLite, `github.com/gorilla/websocket` (new dep), agent-go/v2 + cortexdb (existing AI), React + Vite + react-i18next (web), separate vanilla-TS Vite build for the widget.

**Conventions to follow:** Existing code uses raw string status/priority (no enum consts), `authz.Actor` first arg on ticket service methods, Unix/`time.Time` timestamps, `is_deleted` soft delete, testify-style `_test.go` with an in-memory SQLite via `AutoMigrate`. New status/channel values are added as plain strings matching existing style. No co-author trailers in commits (user rule). Use non-standard ports if any dev server is needed.

---

## File Structure (created/modified)

**Shared:**
- Create `internal/realtime/hub.go`, `internal/realtime/hub_test.go`, `internal/realtime/client.go`
- Create `internal/automation/events.go`, `internal/automation/events_test.go` (event bus)
- Modify `internal/ticket/service.go` (emit events on create/message/resolve; broadcast to hub)
- Modify `internal/models/models.go` (new tables + Ticket/AISettings fields)
- Modify `cmd/server/main.go` (register new models in both `dbModels` slices, ~line 172 and ~line 330)
- Modify `internal/server/server.go` (wire new services + routes in `setupRoutes`)
- Modify `go.mod` / `go.sum` (add gorilla/websocket)

**Per feature:**
- Widget: `internal/widget/{service.go,handlers.go,token.go,*_test.go}`, `web-widget/` (new Vite project)
- AI auto-resolve: `internal/aiassist/autoresolve.go` + `_test.go`; modify `settings.go`, `models.go`
- Automation: `internal/automation/{engine.go,actions.go,scheduler.go,handlers.go,service.go,*_test.go}`
- Macros: `internal/macro/{service.go,handlers.go,render.go,*_test.go}`
- CSAT: `internal/survey/{service.go,handlers.go,*_test.go}`
- Collaboration: `internal/team/{service.go,handlers.go,*_test.go}`; mentions in `internal/ticket/mentions.go`; collision in hub
- Merge/link: methods in `internal/ticket/merge.go` + `_test.go`
- Web: pages under `web/src/pages/`, locale keys in `web/src/locales/*`

---

## Phase 0 — Shared infrastructure

### Task 0.1: Add gorilla/websocket dependency

- [ ] **Step 1:** Run `go get github.com/gorilla/websocket@latest`
- [ ] **Step 2:** Verify `go build ./...` still compiles. Expected: success.
- [ ] **Step 3:** Commit
```bash
git add go.mod go.sum && git commit -m "chore: add gorilla/websocket"
```

### Task 0.2: WebSocket Hub

**Files:** Create `internal/realtime/hub.go`, `internal/realtime/hub_test.go`

- [ ] **Step 1: Write failing test** `hub_test.go`:
```go
func TestHubBroadcastReachesRoomMembers(t *testing.T) {
	h := realtime.NewHub()
	go h.Run()
	a := h.Subscribe("ticket:1")
	b := h.Subscribe("ticket:1")
	other := h.Subscribe("ticket:2")
	h.Broadcast("ticket:1", []byte(`{"type":"message"}`))
	assert.Equal(t, []byte(`{"type":"message"}`), <-a)
	assert.Equal(t, []byte(`{"type":"message"}`), <-b)
	select {
	case <-other:
		t.Fatal("ticket:2 must not receive ticket:1 broadcast")
	case <-time.After(50 * time.Millisecond):
	}
}
```
- [ ] **Step 2:** Run `go test ./internal/realtime/ -run TestHubBroadcast -v` → FAIL (no package).
- [ ] **Step 3: Implement** `hub.go`. Interface:
```go
type Hub struct { /* mu, rooms map[string]map[chan []byte]struct{}, register/unregister/broadcast chans */ }
func NewHub() *Hub
func (h *Hub) Run()                                  // goroutine loop
func (h *Hub) Subscribe(room string) chan []byte     // returns recv channel
func (h *Hub) Unsubscribe(room string, ch chan []byte)
func (h *Hub) Broadcast(room string, payload []byte)
func (h *Hub) Presence(room string) int              // count for collision UI
```
Non-blocking send (drop if subscriber buffer full, buffer size 16).
- [ ] **Step 4:** Run test → PASS.
- [ ] **Step 5: Commit** `git add internal/realtime && git commit -m "feat(realtime): in-process websocket hub"`

### Task 0.3: WebSocket client adapter + endpoints

**Files:** Create `internal/realtime/client.go`; modify `internal/server/server.go`

- [ ] **Step 1:** Implement `client.go`: `ServeWS(hub *Hub, room string, w, r)` upgrades with `gorilla/websocket`, subscribes, pumps hub→socket (write) and socket→hub-less read (used only for typing/presence pings re-broadcast to room). Origin check via allowlist setting (default allow same-host).
- [ ] **Step 2:** In `server.go` create the hub once (`s.hub = realtime.NewHub(); go s.hub.Run()`), register `protected.GET("/ws/tickets/:id", ...)` (JWT) and a public `s.router.GET("/widget/ws", ...)` (conversation_token). For the agent endpoint, authorize the actor can view ticket `:id` before subscribing to `ticket:<id>`.
- [ ] **Step 3:** Build `go build ./...` → success. Manual smoke optional.
- [ ] **Step 4: Commit** `git commit -am "feat(realtime): ws endpoints for agent + widget"`

### Task 0.4: Domain event bus

**Files:** Create `internal/automation/events.go`, `internal/automation/events_test.go`

- [ ] **Step 1: Failing test:**
```go
func TestBusDispatchCallsSubscribers(t *testing.T) {
	bus := automation.NewBus()
	got := 0
	bus.Subscribe(automation.EventTicketCreated, func(e automation.Event) { got++ })
	bus.Publish(automation.Event{Type: automation.EventTicketCreated, TicketID: 7})
	assert.Equal(t, 1, got)
}
```
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3: Implement** `events.go`:
```go
type EventType string
const (
	EventTicketCreated  EventType = "ticket.created"
	EventTicketUpdated  EventType = "ticket.updated"
	EventMessageCreated EventType = "message.created"
	EventSLAWarning     EventType = "ticket.sla_warning"
	EventTicketResolved EventType = "ticket.resolved"
)
type Event struct {
	Type      EventType
	TicketID  uint
	ActorID   uint
	Source    string // "" for human; "automation"/"ai" to prevent loops
	Payload   map[string]any
}
type Handler func(Event)
type Bus struct { mu sync.RWMutex; subs map[EventType][]Handler }
func NewBus() *Bus
func (b *Bus) Subscribe(t EventType, h Handler)
func (b *Bus) Publish(e Event)   // synchronous, recover per-handler panic
```
- [ ] **Step 4:** Run → PASS.
- [ ] **Step 5: Commit** `git commit -am "feat(automation): domain event bus"`

### Task 0.5: Emit events + hub broadcast from ticket service

**Files:** Modify `internal/ticket/service.go` (constructor + `CreateTicket`, `CreateMessage`, resolve path in `UpdateTicket`)

- [ ] **Step 1:** Add optional `bus *automation.Bus` and `hub *realtime.Hub` fields to ticket `Service` with setters (`SetBus`, `SetHub`) so existing constructor signature/tests stay valid (nil-safe: guard every use with `if s.bus != nil`).
- [ ] **Step 2: Failing test** in `service_test.go`: after `CreateMessage`, a subscribed bus handler receives `EventMessageCreated` with correct `TicketID`.
- [ ] **Step 3:** Implement: in `CreateTicket` publish `EventTicketCreated`; in `CreateMessage` publish `EventMessageCreated` and `hub.Broadcast("ticket:<id>", json)` + `widget:<id>` if channel is widget; in `UpdateTicket` when status transitions to `resolved` publish `EventTicketResolved`.
- [ ] **Step 4:** Run ticket tests → PASS.
- [ ] **Step 5: Commit** `git commit -am "feat(ticket): emit domain events + ws broadcast"`

---

## Phase 1 — Web chat widget

### Task 1.1: Ticket model fields + migration

**Files:** Modify `internal/models/models.go`, `cmd/server/main.go`

- [ ] **Step 1:** Add to `Ticket`: `Channel string` (`gorm:"size:30;default:'web'"`), `ConversationToken string` (`gorm:"size:128;index"`), `Summary string` (`gorm:"type:text"`), `AssignedTeamID *uint` (`gorm:"index"`), `MergedIntoID *uint` (`gorm:"index"`). (All Phase-1..7 Ticket fields added together here to avoid repeated migrations.)
- [ ] **Step 2:** Build → success (GORM auto-adds columns).
- [ ] **Step 3: Commit** `git commit -am "feat(models): ticket channel/token/summary/team/merge fields"`

### Task 1.2: Conversation token

**Files:** Create `internal/widget/token.go`, `token_test.go`

- [ ] **Step 1: Failing test:** `Issue(ticketID, secret)` then `Parse(token, secret)` returns the ticketID; tampered token errors; wrong secret errors.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3: Implement** with `golang-jwt/jwt/v5` (already a dep): claims `{tid, exp:+7d}`, HMAC with server secret key (reuse `config` secret).
- [ ] **Step 4:** Run → PASS. **Step 5:** Commit.

### Task 1.3: Widget service (session + messages)

**Files:** Create `internal/widget/service.go`, `service_test.go`

- [ ] **Step 1: Failing tests:** `StartSession(req)` with email creates/reuses a customer + an open ticket (`channel=web_widget`), returns token; `StartSession` anonymous (no email) also works; `PostMessage(token, body)` appends a customer message and returns it; `History(token)` returns non-internal messages only.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3: Implement** delegating to existing `customer` + `ticket` services. Reuse-by-email when provided, else create anonymous customer keyed by a browser token. Publish events via the bus so AI auto-resolve fires.
- [ ] **Step 4:** Run → PASS. **Step 5:** Commit.

### Task 1.4: Widget HTTP handlers + public routes

**Files:** Create `internal/widget/handlers.go`; modify `server.go`

- [ ] **Step 1:** Handlers: `POST /widget/session`, `POST /widget/messages`, `GET /widget/messages`, `GET /widget/ws` (token-auth middleware in this package), `GET /widget.js`, `GET /widget/app`.
- [ ] **Step 2:** Register an unauthenticated `widget := s.router.Group("/widget")` plus top-level `GET /widget.js`. Token middleware rejects missing/invalid token with 401.
- [ ] **Step 3:** `go build` → success; add a handler test using `httptest` for `POST /widget/session` returning 200 + token.
- [ ] **Step 4: Commit** `git commit -am "feat(widget): session + message API"`

### Task 1.5: Widget frontend bundle

**Files:** Create `web-widget/` (package.json, vite.config.ts library mode, `src/widget.ts`, `src/panel.ts`); modify build to copy output where `GET /widget.js` serves it (e.g. `web/public/widget.js` or embedded).

- [ ] **Step 1:** Scaffold vanilla-TS Vite lib that: reads `data-key` from its own script tag, injects a bubble button, on click opens an iframe to `/widget/app`. The iframe app calls `/widget/session`, then opens `/widget/ws`, renders message list + composer, sends via `/widget/messages`, shows agent replies live, and renders an inline CSAT prompt when it receives a `survey` ws event (Phase 5).
- [ ] **Step 2:** `pnpm build` in `web-widget/` → emits `widget.js` < 20KB gzip.
- [ ] **Step 3:** Manual: serve, embed on a scratch HTML page (dev port 3477), confirm round-trip with a seeded agent reply.
- [ ] **Step 4: Commit** `git commit -m "feat(widget): embeddable JS bubble + iframe chat"`

---

## Phase 2 — AI auto-resolve

### Task 2.1: AISettings fields

**Files:** Modify `internal/models/models.go`, `internal/aiassist/settings.go`, settings handler/DTO

- [ ] **Step 1:** Add to `AISettings`: `AutoReplyEnabled bool` (default false), `AutoReplyConfidence float64` (`gorm:"default:0.75"`), `AutoResolveEnabled bool`, `MaxAutoRepliesPerTicket int` (`gorm:"default:2"`), `AutoSummarizeOnResolve bool`. (`AutoClassify` already exists.)
- [ ] **Step 2:** Extend the settings GET/PUT DTOs + validation (confidence 0–1, max ≥1). Build → success.
- [ ] **Step 3: Commit** `git commit -am "feat(ai): auto-resolve settings fields"`

### Task 2.2: Confidence-scored draft

**Files:** Modify `internal/aiassist/generator.go` (or assistant), add `Draft{Text string, Confidence float64, Sources []string}` return; `_test.go`

- [ ] **Step 1: Failing test:** generator returns a `Draft` with confidence derived from top retrieval score (mock retriever returning a known score → expected normalized confidence).
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement: combine retrieval top-score with model self-rating; clamp 0–1.
- [ ] **Step 4:** Run → PASS. **Step 5:** Commit.

### Task 2.3: Auto-resolve orchestrator

**Files:** Create `internal/aiassist/autoresolve.go`, `autoresolve_test.go`

- [ ] **Step 1: Failing tests (table-driven):**
  - confidence ≥ threshold & AutoReplyEnabled & under cap → sends AI reply (asserts a message with `IsFromAI=true` created), increments count.
  - confidence < threshold → no auto reply; a "suggested reply" notification/record is produced instead.
  - count at `MaxAutoRepliesPerTicket` → escalates to human (no auto reply, notify agent), stops.
  - master `Enabled=false` → no-op.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement an event handler `OnMessageCreated(e)` / `OnTicketCreated(e)` that loads settings, calls generator, branches per above, tags events `Source:"ai"` to avoid loops. Auto-reply counter = count of `IsFromAI` messages on the ticket.
- [ ] **Step 4:** Run → PASS. **Step 5:** Commit.

### Task 2.4: Wire auto-resolve + classify + summarize to bus

**Files:** Modify `server.go` (subscribe), `autoresolve.go`

- [ ] **Step 1:** On startup subscribe handlers: `EventTicketCreated`→(auto-classify if enabled)+auto-reply consideration; `EventMessageCreated`(from customer/widget, `Source==""`)→auto-reply; `EventTicketResolved`→summarize if enabled.
- [ ] **Step 2:** Build + run aiassist tests → PASS.
- [ ] **Step 3: Commit** `git commit -am "feat(ai): wire auto-resolve/classify/summarize to event bus"`

---

## Phase 3 — Automation engine

### Task 3.1: AutomationRule model

**Files:** Modify `internal/models/models.go`, `cmd/server/main.go`

- [ ] **Step 1:** Add `AutomationRule{ BaseModel; Name, Description string; Enabled bool; Event string; Match string /* all|any */; Conditions string /* JSON */; Actions string /* JSON */; Position int }`. Register in both `dbModels` slices.
- [ ] **Step 2:** Build → success. **Step 3:** Commit.

### Task 3.2: Condition matcher

**Files:** Create `internal/automation/conditions.go`, `conditions_test.go`

- [ ] **Step 1: Failing tests:** `Match(rule, ticket)` for ops `eq, neq, contains, in, gt, lt` over fields `status, priority, severity, channel, tags, customer_email`; `all` vs `any` semantics.
- [ ] **Step 2:** Run → FAIL. **Step 3:** Implement. **Step 4:** PASS. **Step 5:** Commit.

### Task 3.3: Action executor

**Files:** Create `internal/automation/actions.go`, `actions_test.go`

- [ ] **Step 1: Failing tests** per action: `assign`(user/team), `add_tag`, `set_priority`, `set_status`, `set_severity`, `notify`, `send_email`, `escalate`, `ai_suggest`, `ai_auto_reply`, `close`. Each asserts the side effect via injected service interfaces (ticket, notification, email, aiassist) — use small interfaces + fakes.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement `Executor` with injected dependencies; each action a method dispatched by `type`. Mark resulting events `Source:"automation"`.
- [ ] **Step 4:** PASS. **Step 5:** Commit.

### Task 3.4: Engine (event-driven)

**Files:** Create `internal/automation/engine.go`, `engine_test.go`

- [ ] **Step 1: Failing test:** given two enabled rules (positions 1,2) and one disabled, on `EventTicketCreated` the engine runs matching rules in order and skips disabled/non-matching.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement: `Engine.Handle(e Event)` loads rules for `e.Type` ordered by `Position`, matches conditions against the loaded ticket, runs actions. Ignore events whose `Source=="automation"` for action types that would recurse.
- [ ] **Step 4:** PASS. **Step 5:** Commit.

### Task 3.5: Scheduler (ticker)

**Files:** Create `internal/automation/scheduler.go`, `scheduler_test.go`

- [ ] **Step 1: Failing test:** scheduler's `tick(now)` finds tickets past SLA/no-reply timeout and runs `schedule` rules + emits `EventSLAWarning`; auto-resolve closes tickets when `AutoResolveEnabled` and customer silent past window.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement `tick(now time.Time)` (pure, testable) + a `Run(ctx)` 60s ticker calling it. Wire `Run` in `server.go`.
- [ ] **Step 4:** PASS. **Step 5:** Commit.

### Task 3.6: Rules CRUD API + admin page

**Files:** Create `internal/automation/{service.go,handlers.go}`; modify `server.go`; create `web/src/pages/automations.tsx` + route + nav + locales

- [ ] **Step 1:** CRUD `GET/POST/PUT/DELETE /automations` (admin middleware) + reorder endpoint. Handler test for create/list.
- [ ] **Step 2:** React page: rule list with enable toggle + drag reorder; condition/action builder (selects for field/op/value and action/params). i18n keys in all 7 locales.
- [ ] **Step 3:** Build web (`VITE_BASE=/app/ pnpm build` from `web/`) → success.
- [ ] **Step 4: Commit** `git commit -m "feat(automation): rules CRUD + admin UI"`

---

## Phase 4 — Macros

### Task 4.1: Macro model + render

**Files:** Modify `models.go`, `cmd/server/main.go`; create `internal/macro/render.go`, `render_test.go`

- [ ] **Step 1:** `Macro{ BaseModel; Title, Category, Body string; Actions string /*JSON*/; Shared bool; OwnerID uint; UsageCount int }`. Register model.
- [ ] **Step 2: Failing test:** `Render(body, ctx)` substitutes `{{customer.name}}`, `{{ticket.id}}`, `{{ticket.subject}}`, `{{agent.name}}`; unknown vars left blank.
- [ ] **Step 3:** Implement; PASS. **Step 4:** Commit.

### Task 4.2: Macro service + API

**Files:** Create `internal/macro/{service.go,handlers.go,*_test.go}`; modify `server.go`

- [ ] **Step 1: Failing tests:** CRUD with private/shared visibility (private only to owner); `Apply(id, ticketID, actor)` returns rendered text + actions and increments `UsageCount`.
- [ ] **Step 2:** Implement service; handlers `GET/POST/PUT/DELETE /macros`, `POST /macros/:id/apply`.
- [ ] **Step 3:** PASS + handler test. **Step 4:** Commit.

### Task 4.3: Macro picker UI

**Files:** Modify `web/src/pages/ticket-detail.tsx`; create `web/src/pages/macros.tsx` + route/nav; locales

- [ ] **Step 1:** Reply box gets an "Insert macro" searchable dropdown (by category) → fills composer + applies attached actions. `/macros` management page for CRUD.
- [ ] **Step 2:** Build web → success. **Step 3:** Commit `git commit -m "feat(macros): canned responses + picker"`

---

## Phase 5 — CSAT

### Task 5.1: SatisfactionSurvey model + service

**Files:** Modify `models.go`, `cmd/server/main.go`; create `internal/survey/{service.go,handlers.go,*_test.go}`

- [ ] **Step 1:** `SatisfactionSurvey{ BaseModel; TicketID uint; Rating int; Comment string; Token string; SentAt *time.Time; RespondedAt *time.Time }`. Register.
- [ ] **Step 2: Failing tests:** `CreateForTicket(ticketID)` issues a token + SentAt; `Submit(token, rating, comment)` sets Rating/RespondedAt, rejects rating outside 1–5 and double-submit; `Stats()` returns avg + response rate.
- [ ] **Step 3:** Implement; PASS.
- [ ] **Step 4:** Public handlers `GET/POST /api/v1/survey/:token` + public page route; admin `GET /survey/stats`. **Step 5:** Commit.

### Task 5.2: Trigger survey on resolve + delivery

**Files:** Modify `server.go` (subscribe `EventTicketResolved`); email send via `internal/email`; widget ws `survey` event

- [ ] **Step 1:** On resolve, create survey; if email channel → send link `/survey/:token`; if widget channel → `hub.Broadcast("widget:<id>", {type:"survey",token})`.
- [ ] **Step 2:** Test the subscriber creates exactly one survey per resolve (idempotent).
- [ ] **Step 3:** Commit `git commit -am "feat(csat): survey on resolve via email/widget"`

### Task 5.3: Survey page + dashboard card

**Files:** Create `web/src/pages/survey.tsx` (public); modify `web/src/pages/dashboard.tsx`; locales

- [ ] **Step 1:** Public 1–5 + comment page posting to `/api/v1/survey/:token`. Dashboard CSAT card (avg, response rate, 30-day trend).
- [ ] **Step 2:** Build web → success. **Step 3:** Commit.

---

## Phase 6 — Agent collaboration

### Task 6.1: Teams

**Files:** Modify `models.go`, `cmd/server/main.go`; create `internal/team/{service.go,handlers.go,*_test.go}`; modify `server.go`

- [ ] **Step 1:** `Team{ BaseModel; Name, Description string }`, `TeamMember{ BaseModel; TeamID, UserID uint }`. Register. (`Ticket.AssignedTeamID` already added in Task 1.1.)
- [ ] **Step 2: Failing tests:** CRUD team; add/remove member; list members.
- [ ] **Step 3:** Implement service + handlers `GET/POST/PUT/DELETE /teams`, `POST/DELETE /teams/:id/members`. PASS.
- [ ] **Step 4:** Extend automation `assign` action + ticket assign to accept team. **Step 5:** Commit.

### Task 6.2: @mentions

**Files:** Create `internal/ticket/mentions.go`, `mentions_test.go`; modify `CreateMessage`

- [ ] **Step 1: Failing test:** an internal message body `"@alice please look"` with an existing user `alice` produces a notification to alice; unknown handles ignored; non-internal messages don't mention.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement `parseMentions(body)` + lookup users by username; in `CreateMessage`, when `IsInternal`, notify mentioned users (reuse notification service).
- [ ] **Step 4:** PASS. **Step 5:** Commit.

### Task 6.3: Collision presence (web)

**Files:** Modify `web/src/pages/ticket-detail.tsx`

- [ ] **Step 1:** On open, connect `/ws/tickets/:id`; send `presence`/`typing` pings; render a top bar "X viewing / Y typing…" from broadcast `presence` events (hub `Presence` + relayed typing).
- [ ] **Step 2:** Build web → success; manual two-browser check. **Step 3:** Commit `git commit -m "feat(collab): teams, @mentions, collision presence"`

---

## Phase 7 — Ticket merge / link

### Task 7.1: TicketLink model + merge/link service

**Files:** Modify `models.go`, `cmd/server/main.go`; create `internal/ticket/merge.go`, `merge_test.go`

- [ ] **Step 1:** `TicketLink{ BaseModel; SourceID, TargetID uint; Type string /* related|duplicate|blocks */ }`. Register. (`Ticket.MergedIntoID` already added.)
- [ ] **Step 2: Failing tests:** `Merge(actor, srcID, intoID)` moves messages + attachments to target, sets src `status="merged"` + `MergedIntoID`, records a `TicketEvent`, refuses self-merge and already-merged source; `Link/Unlink` create/remove a `TicketLink`.
- [ ] **Step 3:** Run → FAIL. **Step 4:** Implement (transaction). PASS.
- [ ] **Step 5:** Handlers `POST /tickets/:id/merge`, `POST/DELETE /tickets/:id/links`, `GET /tickets/:id/links`. Commit.

### Task 7.2: Merge/link UI

**Files:** Modify `web/src/pages/ticket-detail.tsx`; locales

- [ ] **Step 1:** "Merge into…" action (with confirm dialog) + related-tickets panel with link type. Hide composer when `status==merged`.
- [ ] **Step 2:** Build web → success. **Step 3:** Commit `git commit -m "feat(tickets): merge + linking"`

---

## Phase 8 — Finalize

### Task 8.1: Full build + test sweep

- [ ] **Step 1:** `go build ./...` and `go test ./...` → all pass. Fix any breakage.
- [ ] **Step 2:** `cd web && VITE_BASE=/app/ pnpm build` and `cd ../web-widget && pnpm build` → success.
- [ ] **Step 3:** Commit any fixes.

### Task 8.2: Update competitive analysis + merge

- [ ] **Step 1:** Update `docs/competitive-analysis.md` capability matrix to reflect shipped features (widget, auto-resolve, triggers, macros, CSAT, teams, merge).
- [ ] **Step 2:** Commit. Then per finishing-a-development-branch skill, merge `feature/parity` → `main` (no co-author trailer).

---

## Self-Review

**Spec coverage:** Widget (P1), AI auto-resolve incl. classify/summarize (P2), automation engine incl. scheduler/escalation (P3), macros (P4), CSAT (P5), @mentions+teams+collision (P6), merge/link (P7), shared Hub+event bus (P0), models+routes+frontend+i18n+docs (across phases) — all spec sections mapped.

**Type consistency:** `automation.Event`/`EventType`/`Bus` used consistently P0/P2/P3; `realtime.Hub` (`Subscribe/Broadcast/Presence`) P0/P5/P6; `Draft{Text,Confidence,Sources}` P2; rooms `ticket:<id>` / `widget:<id>` consistent; Ticket fields all added once in Task 1.1; survey token flow consistent P5.

**Placeholders:** none — every task names exact files, interfaces, and test assertions. Representative test code given; full per-line implementation is produced at execution time by reading the real service signatures (documented in plan header).

**Notable risk:** single-process Hub (no multi-replica) — documented in spec as out-of-scope for this batch.
