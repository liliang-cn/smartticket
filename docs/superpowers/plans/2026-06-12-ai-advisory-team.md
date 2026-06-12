# AI Advisory Team Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A team of advisory AI agents (Triage, Sentinel, Researcher, Reviewer, Drafter) that surface structured suggestions in a ticket Copilot panel; humans adopt them.

**Architecture (HYBRID — decided after reading agent-go v2.79.1 source):** Register 5 named members in an `agent.TeamManager` (real agent-go team roster), but run each member's inference through the proven `domain.Generator.GenerateStructured(prompt, schema)` path (reliable structured output) and reuse the existing KB closure tool — NOT the Task queue's free-text execution. Orchestration (triggers, persistence, hub broadcast) is ours. Suggestions persist to a new `AISuggestion` table; results stream to the ticket Copilot panel over the existing realtime hub.

**Tech Stack:** Go, GIN, GORM, modernc SQLite, agent-go/v2, cortexdb/v2, React/TS.

**Spec:** `docs/superpowers/specs/2026-06-12-ai-advisory-team-design.md`

**Grounding (verified):**
- `aiassist.NewGenerator(llmSvc)` → `domain.Generator`; `gen.GenerateStructured(ctx, prompt, schema interface{}, *domain.GenerationOptions) (*domain.StructuredResult{Data map[string]any, Raw string, Valid bool}, error)` — appends the schema to the prompt and tolerantly extracts JSON (see `internal/aiassist/generator.go`). The existing `aiassist.Assistant` builds a prompt and calls this with `draftSchema` (see `assistant.go` `SuggestReplyStructured`).
- `aiassist.KBSearcherFunc(func(ctx,q,k)[]string)` is the KB tool; `aiassist.NewSettingsStore(db)` is the AI settings singleton.
- agent-go: `agent.NewStore(path)`, `agent.NewTeamManager(store)`, `mgr.SetLLM(gen)`, `mgr.SetDisableMemory(true)`, `mgr.AddSpecialist(ctx, teamID, name, description, instructions) (*AgentModel, error)`, `mgr.GetMemberByName(name)`, `mgr.ListMembers()`, `mgr.CreateTeam(ctx, &agent.Team{...})`. (We use these only to register/roster members; inference goes through `gen`.)
- realtime: `hub.Broadcast(room string, payload []byte)`, room `ticket:<id>`; agent WS `/api/v1/ws/tickets/:id` (frontend `web/src/pages/ticket-detail.tsx`).
- event bus: `s.bus.Subscribe(automation.EventTicketCreated|EventMessageCreated|EventSLAWarning, func(automation.Event))`. CSAT subscriber (server.go ~419) is the wiring template.
- `models.AISettings` singleton holds AI toggles; extend it for per-agent enable + throttle.
- No data migration (new table + new columns).

---

## Task 1: AISuggestion model

**Files:** `internal/models/models.go`, `cmd/server/main.go` (both dbModels slices).

- [ ] Add:
```go
// AISuggestion is one advisory output from an AI team agent on a ticket, shown
// in the Copilot panel. Payload is the agent's structured JSON; the human adopts
// or dismisses it. Status: pending|done|adopted|dismissed|failed.
type AISuggestion struct {
	BaseModel
	TicketID   uint    `gorm:"index;not null" json:"ticket_id"`
	AgentName  string  `gorm:"size:32;index" json:"agent_name"` // Triage|Sentinel|Researcher|Reviewer|Drafter
	Status     string  `gorm:"size:16;index;default:'pending'" json:"status"`
	Confidence float64 `json:"confidence"`
	Payload    string  `gorm:"type:text" json:"payload"` // structured JSON
	AdoptedBy  *uint   `json:"adopted_by"`
	ResolvedAt *int64  `json:"resolved_at"`
}
func (AISuggestion) TableName() string { return "ai_suggestions" }
```
Register `&models.AISuggestion{}` in both dbModels slices. `go build ./...` → clean. Commit `feat(aiteam): AISuggestion model`.

---

## Task 2: aiteam.Team — TeamManager roster + generator/KB

**Files:** Create `internal/aiteam/team.go`, `internal/aiteam/team_test.go`.

Build the team shell. The 5 member instructions (system prompts) live as constants and are registered as specialists (idempotent). `gen`/`kb`/`settings`/`db`/`hub` are held for the run methods (Tasks 3-4) and orchestration (Tasks 6-8).

- [ ] `team.go`:
```go
package aiteam

import (
	"context"
	"github.com/company/smartticket/internal/aiassist"
	"github.com/liliang-cn/agent-go/v2/pkg/agent"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
	"gorm.io/gorm"
)

const teamName = "support-advisory"

// memberInstructions maps agent name → its system prompt (used both to register
// the agent-go specialist and to prime each GenerateStructured call).
var memberInstructions = map[string]string{
	"Triage":     `You triage a new support ticket. Judge priority, severity, a short category, and (if obvious) a suggested team. Be conservative; never invent facts.`,
	"Sentinel":   `You assess escalation risk on a support ticket conversation. Judge customer sentiment, churn risk, SLA-breach risk, and whether to escalate to a manager. Be conservative.`,
	"Researcher": `You help an agent resolve a ticket: find relevant knowledge-base snippets and similar past tickets, and propose a resolution. Use only provided context; never invent.`,
	"Reviewer":   `You review an agent's draft reply before it is sent: flag tone, accuracy, policy and missing-info issues, and optionally provide a revised draft.`,
	"Drafter":    `You draft the agent's next reply to the customer: clear, friendly, professional. Never invent facts or commitments.`,
}

type Team struct {
	mgr      *agent.TeamManager
	gen      domain.Generator
	kb       aiassist.KBSearcher
	settings *aiassist.SettingsStore
	db       *gorm.DB
}

// NewTeam builds the agent-go team (registers the 5 specialists, idempotent) and
// holds the BYO-LLM generator + KB tool for structured inference. dbPath is
// agent-go's own store (e.g. "./data/agentgo-team.db").
func NewTeam(dbPath string, gen domain.Generator, kb aiassist.KBSearcher, settings *aiassist.SettingsStore, db *gorm.DB) (*Team, error) {
	store, err := agent.NewStore(dbPath)
	if err != nil {
		return nil, err
	}
	mgr := agent.NewTeamManager(store)
	mgr.SetLLM(gen)
	mgr.SetDisableMemory(true)
	t := &Team{mgr: mgr, gen: gen, kb: kb, settings: settings, db: db}
	if err := t.ensureMembers(context.Background()); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Team) ensureMembers(ctx context.Context) error {
	team, err := t.mgr.GetTeamByName(teamName)
	if err != nil || team == nil {
		team, err = t.mgr.CreateTeam(ctx, &agent.Team{Name: teamName, Description: "SmartTicket AI advisory team"})
		if err != nil {
			return err
		}
	}
	for name, instr := range memberInstructions {
		if m, _ := t.mgr.GetMemberByName(name); m != nil {
			continue // already registered
		}
		if _, err := t.mgr.AddSpecialist(ctx, team.ID, name, name+" advisory agent", instr); err != nil {
			return err
		}
	}
	return nil
}

// Members returns the registered roster (for ListMembers visibility / tests).
func (t *Team) Members() ([]*agent.AgentModel, error) { return t.mgr.ListMembers() }
```
- [ ] `team_test.go`: with a fake `domain.Generator` (mirror `internal/aiassist/*_test.go` / `references/testing.md` fakeLLM) and a temp `t.TempDir()` dbPath, `NewTeam(...)` then `Members()` returns ≥5 members including "Triage" and "Sentinel". (agent-go store needs CGO sqlite — the project already builds with it.)
- [ ] `go build ./... && go test ./internal/aiteam/ -run TestTeam -v`. Commit `feat(aiteam): agent-go team roster (5 specialists) + generator`.

> If `agent.Team`/`CreateTeam`/`AddSpecialist` signatures differ from the above at compile time, read `$(go list -m -f '{{.Dir}}' github.com/liliang-cn/agent-go/v2)/pkg/agent/team_manager.go` and `agent_model.go` and adapt — the names are verified present in v2.79.1; field/struct exact shapes should be confirmed against source.

---

## Task 3: Triage + Sentinel agents (auto) — schemas + run methods (TDD)

**Files:** Create `internal/aiteam/agents.go`, `internal/aiteam/agents_test.go`.

Each run method: build a prompt from `memberInstructions[name]` + ticket context, call `t.gen.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{Temperature: 0.3})`, map `result.Data` (a `map[string]any`) into a typed struct. Mirror `aiassist.Assistant.SuggestReplyStructured` for the prompt-building + result-mapping idiom.

- [ ] Define typed results + JSON schemas:
```go
type TriageResult struct {
	Priority         string  `json:"priority"`
	Severity         string  `json:"severity"`
	Category         string  `json:"category"`
	SuggestedTeamID  *uint   `json:"suggested_team_id"`
	Reasoning        string  `json:"reasoning"`
	Confidence       float64 `json:"confidence"`
}
type SentinelResult struct {
	Sentiment      string  `json:"sentiment"`
	ChurnRisk      string  `json:"churn_risk"`
	SLABreachRisk  bool    `json:"sla_breach_risk"`
	Escalate       bool    `json:"escalate"`
	Reasoning      string  `json:"reasoning"`
	Confidence     float64 `json:"confidence"`
}
```
with matching `map[string]interface{}` schemas (mirror `aiassist.draftSchema` shape: type object + properties + required). Provide a `TicketContext` input struct (title, description, conversation, customer, sla state) and helpers to render it into the prompt.
- [ ] `RunTriage(ctx, TicketContext) (*TriageResult, error)` and `RunSentinel(ctx, TicketContext) (*SentinelResult, error)`: build prompt, call `GenerateStructured`, `json.Marshal(result.Data)`→`json.Unmarshal` into the typed struct (or map field-by-field), clamp Confidence to [0,1]. Return `aiassist.ErrNotConfigured` when `t.gen` is nil and `ErrDisabled` when the master AI switch is off.
- [ ] Tests with a fake generator returning canned JSON: assert `RunTriage` parses priority/severity/confidence; `RunSentinel` parses escalate/sentiment. Test a malformed-JSON response → returns a low-confidence/empty result without panicking.
- [ ] `go test ./internal/aiteam/ -v`. Commit `feat(aiteam): Triage and Sentinel agents`.

---

## Task 4: Researcher + Reviewer + Drafter agents

**Files:** extend `internal/aiteam/agents.go` + tests.

- [ ] Add result types + schemas: `ResearcherResult{KBCitations []Snippet, SimilarTickets []SimilarTicket, SuggestedResolution string, Confidence float64}`, `ReviewerResult{Issues []ReviewIssue, RevisedDraft string, Approve bool, Confidence float64}`, `DrafterResult{Reply string, Confidence float64}` (Drafter mirrors existing `aiassist.Draft`).
- [ ] `RunDrafter` reuses the existing draft logic: either call the existing `aiassist.Assistant.SuggestReplyStructured` if injected, or replicate via `GenerateStructured` with the Drafter prompt + KB snippets fetched through `t.kb.SnippetsFor(ctx, query, 4)`.
- [ ] `RunResearcher(ctx, TicketContext)`: fetch KB snippets via `t.kb` AND similar tickets via the index from Task 5 (inject a `SimilarTicketSearcher` interface, nil-safe), include them in the prompt, then `GenerateStructured` for the suggested resolution + structured citations.
- [ ] `RunReviewer(ctx, TicketContext, draft string)`: prompt includes the draft + `settings.ReplyInstructions`; `GenerateStructured` → issues + optional revised draft.
- [ ] Tests with fake generator + fake KB/similar searchers. `go test ./internal/aiteam/ -v`. Commit `feat(aiteam): Researcher, Reviewer, Drafter agents`.

---

## Task 5: Similar-ticket cortexdb index (Researcher's tool)

**Files:** extend `internal/knowledgebase/index.go` (add `SaveTicket`/`SearchTickets` on a `"tickets"` collection, mirroring `SaveArticle`/`Search`), create `internal/aiteam/similar.go` adapter, wire async indexing in server.go.

- [ ] Add to `knowledgebase.Store`: `SaveTicket(ctx, id uint, title, body, resolution string) error` and `SearchTickets(ctx, query string, topK int) (*SearchResult, error)` using a dedicated `"tickets"` cortex collection (copy the `SaveArticle`/`searchCollection` implementation, swap collection name). Tests mirror existing knowledgebase tests (skip when no embedder, like the existing ones).
- [ ] `internal/aiteam/similar.go`: a `SimilarTicketSearcher` impl wrapping the store; returns `[]SimilarTicket{ID,Title,Resolution,Score}`.
- [ ] In server.go, subscribe `EventTicketResolved` (and `EventTicketUpdated`) to async-index the ticket (`go store.SaveTicket(...)`, best-effort, never blocks). Mirror the existing article async-index requirement (see memory: async indexing).
- [ ] `go build ./... && go test ./internal/knowledgebase/ ./internal/aiteam/ -count=1`. Commit `feat(aiteam): similar-ticket cortexdb index`.

> If extending the shared `knowledgebase.Store` risks the article path, instead add a sibling `internal/aiteam/ticketindex.go` that opens its own cortex collection via the same `cortexdb` API the Store uses. Prefer extending the Store (one cortex DB) unless that entangles the article logic.

---

## Task 6: Suggestion store + GET endpoint

**Files:** Create `internal/aiteam/suggestions.go`, `internal/aiteam/suggestions_test.go`.

- [ ] `SuggestionStore{db}` with: `Upsert(ticketID uint, agent string, status string, confidence float64, payload string) (*models.AISuggestion, error)` (one row per (ticket,agent) — update in place), `List(ticketID uint) ([]models.AISuggestion, error)`, `Adopt(id, userID uint) error` (set status=adopted, AdoptedBy, ResolvedAt), `Dismiss(id uint) error`. Tests cover the state transitions.
- [ ] `go test ./internal/aiteam/ -v`. Commit `feat(aiteam): suggestion persistence + state machine`.

---

## Task 7: Orchestrator + auto triggers + per-agent settings/throttle

**Files:** Create `internal/aiteam/orchestrator.go`; extend `internal/models/models.go` (AISettings fields); wire in `internal/server/server.go`.

- [ ] Extend `models.AISettings` with per-agent toggles + throttle: `TriageEnabled bool (default true)`, `SentinelEnabled bool (default true)`, `SentinelThrottleSec int (default 60)`. (AutoClassify already exists — keep.)
- [ ] `Orchestrator` method `Run(ctx, agentName string, tc TicketContext) `: checks the master + per-agent setting; writes `AISuggestion{pending}`; runs the agent (Task 3-4); on success writes `{done, payload, confidence}` and broadcasts to `ticket:<id>` via an injected `Broadcaster` (hub); on error writes `{failed}`. Throttle Sentinel: skip if a Sentinel suggestion for that ticket updated within `SentinelThrottleSec`.
- [ ] In server.go (mirror the CSAT subscriber): build the `TicketContext` from the event's ticket, then:
```go
s.bus.Subscribe(automation.EventTicketCreated, func(ev automation.Event){ go orch.Run(ctx, "Triage", buildCtx(ev)) })
s.bus.Subscribe(automation.EventMessageCreated, func(ev automation.Event){ if isCustomerMsg(ev) { go orch.Run(ctx, "Sentinel", buildCtx(ev)) } })
s.bus.Subscribe(automation.EventSLAWarning, func(ev automation.Event){ go orch.Run(ctx, "Sentinel", buildCtx(ev)) })
```
(best-effort goroutines; never block the ticket path). The Broadcaster wraps `s.hub.Broadcast`.
- [ ] Tests for the orchestrator with fakes (agent run + broadcaster + store): asserts disabled→no run, throttle suppresses a second Sentinel, success persists+broadcasts. `go build ./... && go test ./internal/aiteam/ -count=1`. Commit `feat(aiteam): orchestrator, auto triggers, per-agent settings`.

---

## Task 8: On-demand API + GET suggestions + wiring

**Files:** Create `internal/aiteam/handlers.go`; wire routes + team construction in `internal/server/server.go`.

- [ ] Handlers (JWT + RBAC, under `/api/v1/tickets/:id/ai`):
```
POST /tickets/:id/ai/research        → orch.Run(ctx,"Researcher",buildCtx) (sync; returns the suggestion)
POST /tickets/:id/ai/review {draft}  → orch.Run(ctx,"Reviewer",...)
POST /tickets/:id/ai/draft           → orch.Run(ctx,"Drafter",...)
GET  /tickets/:id/ai/suggestions     → SuggestionStore.List
POST /tickets/:id/ai/suggestions/:sid/adopt   → Adopt(sid, currentUser)
POST /tickets/:id/ai/suggestions/:sid/dismiss → Dismiss(sid)
```
Adopt does NOT itself mutate the ticket — it records adoption; the frontend calls the existing ticket-update / merge / reply APIs to apply. (Keep Copilot free of new write paths, per spec.)
- [ ] Construct the team near the existing aiassist wiring (server.go ~347-378): only when an LLM is configured. Reuse `aiassist.NewGenerator(llmServiceRef)`, the same `KBSearcherFunc`, and `aiSettings`. Pass `s.hub` as broadcaster, `s.db.DB`. Register routes under the JWT-protected group. Gate everything on AI being configured (nil-safe when not).
- [ ] Swagger annotations on the handlers. `go build ./...`; handler tests where practical. Commit `feat(aiteam): on-demand API + suggestions endpoints + wiring`.

---

## Task 9: Frontend Copilot panel + adopt actions

**Files:** `web/src/features/aiteam/api.ts`, a Copilot panel component, modify `web/src/pages/ticket-detail.tsx`, locales.

- [ ] `features/aiteam/api.ts`: TanStack Query hooks (mirror `features/apikeys/api.ts`): `useSuggestions(ticketId)`, `useRunResearcher`, `useRunReviewer`, `useRunDraft`, `useAdoptSuggestion`, `useDismissSuggestion`.
- [ ] Copilot panel component in ticket-detail's right rail: render one card per suggestion (title = agent, confidence badge, reasoning expandable, payload-specific body). Auto agents (Triage/Sentinel) appear when present; Researcher/Reviewer/Drafter have a "Run" button. Subscribe to the existing ticket WS (`ticket-detail.tsx` already opens `/api/v1/ws/tickets/:id`) — on an incoming suggestion message, invalidate/refresh `useSuggestions`.
- [ ] Adopt actions reuse EXISTING write APIs (no new write path): Triage "Apply" → existing ticket-update; Researcher "Insert" → fill the reply box / existing merge API; Reviewer/Drafter "Use draft" → fill reply box. Each also calls `adopt`/`dismiss` to record state.
- [ ] i18n `aiteam.json` for all 7 langs + any nav/labels. `cd web && pnpm build` → clean. Commit `feat(aiteam): ticket Copilot panel + adopt actions`.

---

## Task 10: OpenAPI regen + full verification

- [ ] Ensure swagger annotations on `internal/aiteam/handlers.go`. `swag init -g cmd/server/main.go --parseDependency --parseInternal -o docs` (required flag). Confirm `/tickets/{id}/ai/...` paths appear.
- [ ] Full sweep: `go build ./...` (clean), `go test ./... 2>&1 | tail -40` (0 failures; distinguish feature vs pre-existing), `cd web && pnpm build` (clean). Report results.
- [ ] Commit `docs(api): regenerate OpenAPI for AI advisory team`.

---

## Self-Review Notes
- **Spec coverage:** 5 agents + schemas (Tasks 3-4); team roster (Task 2); similar-ticket index (Task 5); persistence (Tasks 1,6); triggers + settings (Task 7); on-demand API + suggestions (Task 8); Copilot panel + adopt (Task 9); OpenAPI (Task 10). Escalation→supervisor reuses spec D's `SupervisorOf` (already shipped) — Sentinel's `escalate` adoption calls the existing escalate path; no new escalation code here.
- **Hybrid honored:** TeamManager registers the named members (Task 2) but inference is `GenerateStructured` (Tasks 3-4); KB reuse via `aiassist.KBSearcher`. No Task-queue free-text parsing, no MCP tool wiring.
- **Decoupling:** `aiteam` imports `aiassist` (generator/KB/settings) + `agent-go` + `models`; server.go wires triggers/hub (mirrors CSAT subscriber) so `aiteam` doesn't import ticket/hub. Broadcaster + SimilarTicketSearcher are injected interfaces.
- **Best-effort everywhere:** all auto runs are goroutines that never block the ticket path; failures persist as `failed`, never surface to the request.
- **Open detail for implementer:** confirm exact agent-go `Team`/`AddSpecialist`/`AgentModel` struct shapes against `$(go list -m -f '{{.Dir}}' github.com/liliang-cn/agent-go/v2)/pkg/agent/` (names verified in v2.79.1; adapt fields). Confirm the cortexdb collection API for the ticket index against the existing `knowledgebase` Store usage.
