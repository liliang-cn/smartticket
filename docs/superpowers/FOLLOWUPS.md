# Follow-ups

Outstanding items discovered while implementing the program (B/D/A) and deploying
to the gui01/02/03 HA cluster. None block current functionality; listed roughly
by priority.

## AI

### 1. AI async job queue — MOSTLY RESOLVED (agent-go v2.81.0 task queue)
The 5 advisory agents now execute through agent-go's **persistent TeamManager
task queue** (`Tasks().Submit` + `OutputSchema`), which persists each run to
SQLite. On top of that:
- ✅ **Concurrency cap / backpressure** — a package-wide semaphore in
  `Team.structured()` caps concurrent in-flight tasks at `min(NumCPU-2,4)` (>=2).
  agent-go's own submit has no cap, so this is ours.
- ✅ **Crash/failover recovery** — `Orchestrator.RecoverPending` (wired into
  startup) re-runs suggestions left `pending` by a dead process: idempotent
  re-run from the persisted ticket id. Reviewer (no persisted draft) is marked
  `failed` for re-trigger.
- ⚠️ **Retry** — partial. agent-go's StructuredOutput lint retries malformed
  output (bounded); transient LLM/network failures still mark `failed` (the user
  re-triggers, or the next auto-trigger / startup recovery re-runs). A real
  backoff-retry loop is still a possible enhancement, but no longer urgent.

NOTE: `aiassist` auto-classify/auto-reply still uses fire-and-forget goroutines
(it doesn't go through the team task queue). Lower volume; left as-is for now.

### 2. RunReviewer loads AI settings twice — RESOLVED
Collapsed to a single `settings.Get()` (gate on Enabled + reuse for
ReplyInstructions).

## Deployment / robustness

### 3. Store paths should `mkdir -p` their parent dir — RESOLVED (agent-go v2.81.0)
~~cortex (`./data/cortex.db`), aiassist (`./data/agentgo.db`), and aiteam
(`./data/agentgo-team.db`) open relative `./data/...` paths but don't create the
`data/` dir.~~
- agent-go v2.81.0 `NewAgentGoDB` now `os.MkdirAll`s the parent — covers
  `agentgo.db` + `agentgo-team.db`.
- cortexdb still does not self-mkdir, so `server.go` now `os.MkdirAll("./data")`
  before `knowledgebase.Open`.

**CGO requirement also RESOLVED.** agent-go v2.81.0 switched its store to
`modernc.org/sqlite` (pure Go) — no more mattn/CGO. The whole binary now builds
and runs under `CGO_ENABLED=0` (verified: build + aiteam/aiassist tests pass with
CGO disabled). The `zig cc` CGO cross-compile is no longer needed; the standard
`CGO_ENABLED=0 GOOS=linux go build` (Makefile + Dockerfile already use it) just
works.

## Departments (spec D review — non-blocking)

### 4. Validate department parent existence + manager is staff
`internal/department/service.go`:
- `guardParent` treats a missing parent as "no cycle" → a non-existent `parent_id`
  is silently accepted (no FK enforcement on SQLite). Add an existence check.
- `ManagerID` is never validated to be a real, team-side user. A department can be
  created with a manager pointing at a customer user or a non-existent id. Admin-
  only + actor role still gates dept scoping, so low impact — add a check anyway.

### 5. DepartmentIsolation: confirm intended GetMyTickets behavior
Under `DepartmentIsolation`, a plain agent's "my tickets" is the intersection of
`assigned_to = self` and the department subtree. Members now see their own
department (fixed), but verify this matches product intent for edge cases.

## Program

### 6. Spec C (Widget config + Live Chat) not implemented
The fourth program sub-spec (`docs/superpowers/specs/2026-06-12-widget-config-livechat-design.md`)
— admin-self-serve widget settings (appearance/prechat/business-hours/routing),
offline-to-email, AI-first reply reusing the Drafter — is designed but not built.
