# Follow-ups

Outstanding items discovered while implementing the program (B/D/A) and deploying
to the gui01/02/03 HA cluster. None block current functionality; listed roughly
by priority.

## AI

### 1. AI async job queue (in-process, SQLite-backed) — HIGH for production
Today every AI run (auto Triage/Sentinel + on-demand Research/Review/Draft +
aiassist auto-classify/auto-reply) is a fire-and-forget goroutine (recover + 2min
timeout). It works at small scale but has three gaps:
- **No concurrency cap / backpressure** — a burst of tickets/messages spawns
  unbounded goroutines, each holding an LLM + embedding call; can hit provider
  rate limits / exhaust connections / spike memory.
- **No crash/failover recovery** — on the DRBD HA cluster a failover kills
  in-flight goroutines, leaving suggestions stuck in `pending` (orphaned). Seen
  in testing.
- **No retry** — transient LLM failures (timeout/5xx/rate-limit) just mark
  `failed`.

**Recommended design (no new deps, single-binary-friendly):** mirror the existing
webhook delivery worker (`internal/webhook/worker.go`) — a DB-backed queue + a
bounded worker pool.
- Add `attempts int` + `next_attempt_at *int64` to `models.AISuggestion` (and the
  Reviewer draft input, since it isn't persisted today).
- All AI triggers ENQUEUE (write a `pending` row) instead of spawning a goroutine.
- `internal/aiteam/worker.go`: poll `pending`/retryable rows, run with
  `min(NumCPU, 4)` concurrency, exponential-backoff retry, broadcast on done.
- On startup, re-enqueue orphaned `pending` rows (failover/crash recovery).
References: `internal/webhook/worker.go` (DB queue + worker pattern), agent-go
TeamManager's own persistent Task queue.

### 2. RunReviewer loads AI settings twice
`internal/aiteam/agents.go` `RunReviewer` calls `settings.Get()` twice — one
redundant DB round-trip per Reviewer call. Minor; collapse to one.

## Deployment / robustness

### 3. Store paths should `mkdir -p` their parent dir
cortex (`./data/cortex.db`), aiassist (`./data/agentgo.db`), and aiteam
(`./data/agentgo-team.db`) open relative `./data/...` paths but don't create the
`data/` dir. On the HA deploy this CANTOPEN'd ("out of memory (14)") until
`/var/lib/smartticket/data` was created by hand. Add `os.MkdirAll(dir, 0o755)`
before opening these stores so a fresh deployment just works.
NOTE: agent-go requires CGO at runtime (the no-CGO build compiles but CANTOPENs).
Linux builds must use a CGO cross-compile (we use `zig cc`).

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
