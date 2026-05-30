# SmartTicket MCP Server — Design

**Date:** 2026-05-30
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `feature/mcp-server`

## 1. Goal

Expose SmartTicket's existing business operations (tickets, knowledge base, users,
products, services, SLA, import/export, RBAC) as **MCP tools** so AI agents can
operate the ticketing system. Built with the official Go SDK
`github.com/modelcontextprotocol/go-sdk`.

## 2. Decisions (from brainstorming)

| Topic | Decision |
|-------|----------|
| Target | An MCP server **inside this repo** wrapping SmartTicket's service layer |
| Backend coupling | Tools program against a `Backend` interface. Ship the **in-process `DirectBackend`** (calls services directly) now; design so an `HTTPBackend` (calls the REST API) can be added later **without touching tools or schemas**. |
| Transports | Both **stdio** and **Streamable HTTP** |
| Auth | **Per-connection credential** (one JWT per session). Validated, resolved to a user, then existing **RBAC enforced per tool call**. |
| Scope | **All** service operations, including mutations |
| Token refresh | Out of scope for v1 |

## 3. Architecture & Components

New package `internal/mcp/`, peer to the service layer. It depends only on the
service packages and the permission service — **never on HTTP/GIN**.

```
internal/mcp/
├── backend.go        # Backend interface (all operations, grouped by domain) + Session/Identity types
├── direct.go         # DirectBackend: wraps existing services (built from *gorm.DB + each NewService)
├── server.go         # Assembles *mcp.Server, registers all tools (filtered by toolsets)
├── auth.go           # Authenticator: credential -> ValidateToken -> Session; RBAC checks
├── transport.go      # stdio vs Streamable HTTP selection + graceful shutdown
├── tools_ticket.go        # Per-domain tool files: parse input -> authorize -> call Backend -> format output
├── tools_knowledge.go
├── tools_user.go
├── tools_product.go
├── tools_service.go
├── tools_sla.go
├── tools_importexport.go
└── tools_rbac.go
cmd/server/main.go          # New `smartticket mcp` subcommand
```

**Data flow:** agent → MCP tool call (typed args) → handler → read `Session` from
`ctx` (injected when the connection is authenticated) → `permissionService` RBAC
check → `Backend` method → existing service → DB → result struct → MCP output.

**Boundary discipline:** tools only know the `Backend` interface. `DirectBackend`
is the sole implementation today; a future `HTTPBackend` implements the same
interface, leaving tools and schemas unchanged. Each tool file is unit-testable in
isolation against a mock `Backend`.

## 4. Session & Authentication

**`Authenticator`** wraps the existing `auth.Service` and `permissionService`:
`Authenticate(token string) (*Session, error)` → `auth.Service.ValidateToken` parses
the JWT → take `UserID` → load the user and its permission set → return
`Session{UserID, Permissions}`. The `Session` is placed in the connection `context`.

**Credential entry (once per connection):**
- **HTTP:** a thin `http.Handler` middleware wraps the go-sdk Streamable HTTP
  handler. It reads `Authorization: Bearer <jwt>`, calls `Authenticate`, and injects
  the `Session` into the request context (which flows to tool handlers via `ctx`).
  Missing/invalid → **401**, never reaching MCP.
- **stdio:** one process == one session. Credential supplied at startup via
  `--token <jwt>` or env `SMARTTICKET_MCP_TOKEN` (alternatively `--user/--password`
  → `Login` once at boot). Validated at boot; `Session` injected into the root
  context passed to `server.Run`. **No credential → refuse to start (fail-closed).**

**Per-tool RBAC:** each tool declares a required permission code (e.g.
`ticket_create` → `ticket:create`). The handler checks `session.Can(code)` before
calling `Backend`; failure → structured MCP error (`IsError` result), not a panic.

**Token expiry:** `ValidateToken` checks `exp`. HTTP re-validates on each new
connection; a long-lived stdio session that hits an expired token returns an auth
error (agent restarts with a fresh token). Refresh flow is out of scope for v1.

## 5. Tool Catalog & Schema Conventions

**Naming:** `<domain>_<action>` for a clear namespace. ~60 tools:

| Domain | Tools | Count |
|--------|-------|-------|
| ticket | create / get / list / update / delete / assign / stats | 7 |
| knowledge | create / get / list / update / delete / stats | 6 |
| product | create / get / list / update / delete / activate / deactivate | 7 |
| service | create / get / list / update / delete / activate / deactivate | 7 |
| sla | template ×8 (incl. set_default, activate, deactivate) + rule ×7 | 15 |
| importexport | import_create / export_create / job_get / job_list / job_cancel / job_delete / job_stats | 7 |
| user | create / get / update / delete / list / activate / deactivate / change_password / stats | 9 |
| rbac | role & permission CRUD + assign/remove + list user roles/permissions | ~10 |
| auth | whoami (returns current session user) | 1 |

Internal validation helpers (`ValidateProductService`, `ValidateFileFormat`) are **not**
exposed as tools. `Login` is not a tool (credential is session-level).

**Tool-surface control:** all tools enabled by default; a `--toolsets=ticket,knowledge,...`
flag lets an operator trim the exposed surface (so ~60 tools don't overwhelm a given
agent's context).

**Schema conventions:**
- Input/output are typed Go structs with `json` + `jsonschema` tags; `mcp.AddTool`
  auto-generates the JSON schema.
- **Define MCP-facing input structs** (do not reuse service DTOs directly) and map to
  service DTOs inside the handler — this keeps internal fields (e.g. `is_deleted`) out
  of tool schemas and keeps schemas LLM-friendly and stable. Output may reuse the
  service response structs (already clean).
- IDs are integers. `list` tools take `page`/`page_size` + filters; output includes
  `items` + `total` + pagination meta.
- Each tool returns a structured `Output` plus a short human-readable text summary
  (go-sdk supports structured content).
- Destructive tools (`*_delete`) note this in their description; v1 adds no extra
  confirmation gating (host/agent approval handles that) and relies on RBAC.

**Error mapping:** service errors (NotFound / Validation / Conflict / Forbidden) map to
structured MCP tool errors (`IsError` + code + message); raw Go errors are never leaked.

## 6. Subcommand, Transport Wiring, Error Handling

**`smartticket mcp` subcommand** (cobra; reuses existing config/DB/logger wiring,
same service construction as `serve`, no business-logic duplication):
- Flags: `--config`; `--http <addr>` (set → Streamable HTTP, else stdio);
  `--token` / env `SMARTTICKET_MCP_TOKEN` (stdio credential); `--toolsets`.
- Flow: load config → init DB → construct services → build `DirectBackend` → build
  `mcp.Server` and register tools → build `Authenticator`.

**Transport wiring (`transport.go`):**
- **stdio:** validate startup token → `Session` in root ctx → `server.Run(ctx, &mcp.StdioTransport{})`.
- **HTTP:** wrap the go-sdk Streamable HTTP handler with the auth middleware on a
  standard `http.Server` listening on `--http`; graceful shutdown on signal. Default
  address uses a non-common high port per project convention (configurable, separate
  from the main API's 6533).

**Error handling:**
- Service errors inside tools → structured `IsError` result (code + message).
- Auth failure → HTTP 401 / stdio refuse-to-start.
- Handler panics → recovered → converted to an error result; process never crashes.
- Every tool call logged via the existing zap structured logger: user, tool, latency, outcome.

## 7. Testing Strategy

- **Tool unit tests:** each handler against a **mock `Backend`** + a fake `Session` —
  verify argument mapping, permission allow/deny, and error mapping. No DB.
- **DirectBackend:** in-memory SQLite (like existing service tests) to verify it
  delegates correctly to services.
- **Authenticator:** valid / invalid / expired tokens.
- **Integration:** use the go-sdk **in-memory transport** to run a client/server pair,
  call several tools end-to-end with a test token, assert results and RBAC denials.
- **HTTP middleware:** returns 401 without / with a bad token.
- Coverage aligned with the repo target (75%+ per CLAUDE.md).

## 8. Out of Scope (v1)

- `HTTPBackend` implementation (interface seam only).
- JWT refresh flow.
- Extra per-call confirmation gating for destructive tools.
- MCP resources/prompts (tools only for v1).
- Syncing the hand-maintained `docs/api/*.yaml` OpenAPI specs.

## 9. Open Questions to Resolve During Planning

- Exact go-sdk API for Streamable HTTP + reading request headers / per-session
  context (confirm `StreamableHTTPHandler` signature and how request context reaches
  tool handlers).
- Exact go-sdk in-memory transport API for integration tests.
- The authoritative list of permission codes to map each tool to (read from the
  permission seeding / model).
