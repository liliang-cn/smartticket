# AI Foundation — LLM Providers + CortexDB — Design

**Date:** 2026-05-31
**Status:** Approved (brainstorming) — implementing directly per user request
**Branch:** `feat/ai-foundation`

## 1. Goal

Build the **foundation layer** for SmartTicket's Phase 3 AI integration. This slice
delivers the plumbing only — **no end-user-facing AI feature yet**:

- **Admin-editable LLM providers**: operators configure OpenAI-compatible providers
  (chat + embedding) from the console; credentials encrypted at rest.
- **LLM client abstraction**: one OpenAI-compatible client exposing `Chat()` and
  `Embed()`, plus a task→provider mapping (which provider serves chat, which serves
  embedding).
- **CortexDB integration**: open/manage the CortexDB knowledge+vector store, with its
  `Embedder` backed by the configured embedding provider.
- **Storage migration**: move the whole project off CGO (`mattn/go-sqlite3`) to pure-Go
  `modernc.org/sqlite`, unifying the SQLite driver stack with CortexDB.

The end-to-end chain `provider → embedder → CortexDB → chat` must be provably working
via a provider **Test** action. Knowledge-article ingest, semantic search, RAG Q&A,
and AI-assisted replies are explicitly **out of scope** for this slice (next round).

## 2. Decisions (from brainstorming)

| Topic | Decision |
|-------|----------|
| First slice | Foundation only: LLM provider management (admin-editable) + CortexDB integration + embedder/chat abstraction. No user-facing AI feature. |
| SQLite driver | Migrate the entire project from `mattn/go-sqlite3` (CGO) to `github.com/glebarez/sqlite` (modernc, pure Go). Drop CGO; simplify Docker. |
| Provider abstraction | Single generic **OpenAI-compatible** client (base_url + api_key + model). Provider config separates **chat** and **embedding** purposes (task→model mapping). |
| Chat provider | DeepSeek (`https://api.deepseek.com`). Model name configured by admin in UI, not hardcoded. |
| Embedding provider | Aliyun Bailian / Model Studio (OpenAI-compatible), `text-embedding-v4`, dimension **1024**. base_url `https://dashscope.aliyuncs.com/compatible-mode/v1`. Batch cap 10/request. |
| Vector dimension | Fixed at **1024** for the CortexDB index (chosen at index creation; not changeable later). |
| Credentials | Stored in DB, AES-GCM encrypted with `SMARTTICKET_SECRET_KEY` (env, like the JWT secret). API responses mask `api_key`. Real keys entered post-deploy via admin UI; **never committed to git**. |
| RBAC | New permission codes `llm:read` / `llm:write`, granted to admin (aligned with `internal/database/permissions.go`). |
| Test action | Provider `Test` performs a real chat + embedding call, and additionally a CortexDB embed→store→vector-recall round-trip, to prove the full chain. |

## 3. Storage Migration (modernc, drop CGO)

`internal/database`:
- Replace GORM dialector `gorm.io/driver/sqlite` (mattn) → `github.com/glebarez/sqlite` (modernc).
- Preserve current pragmas/settings: WAL journal mode, foreign keys ON, busy timeout,
  connection pool config.
- `Dockerfile`: `CGO_ENABLED=0`; remove `gcc musl-dev` build deps.

**Sequencing:** this migration lands **first, as its own commit**, gated on a full
`go test ./...` pass + a manual smoke (login, create ticket, run migrations) before
any AI code is layered on top. If a GORM/modernc incompatibility surfaces, it is
isolated to this commit.

## 4. LLM Provider Domain (`internal/llm`)

### 4.1 Model — reuse existing `models.LLMProvider`
The existing table already fits a one-row-per-purpose model. Each provider row serves
specific task types via the existing `TaskTypes` JSON array, so chat and embedding are
**separate rows** rather than new columns:
- Chat row: `ProviderType="openai-compatible"`, `APIEndpoint=https://api.deepseek.com`,
  `Model=deepseek-chat`, `TaskTypes=["chat"]`.
- Embedding row: `APIEndpoint=https://dashscope.aliyuncs.com/compatible-mode/v1`,
  `Model=text-embedding-v4`, `TaskTypes=["embedding"]`, `Dimensions=1024`.

Existing columns reused as-is: `Name, ProviderType, APIEndpoint, APIKey, Model,
MaxTokens, Temperature, TaskTypes, IsDefault, IsEnabled, QuotaLimit, QuotaUsed,
Configuration`.

**Changes to the model (migration):**
- `APIKey`: keep the column but change the JSON tag to `json:"-"` (it currently
  serializes as `api_key` — a plaintext leak). Store AES-GCM ciphertext. Expose a
  computed masked value in API responses only.
- Add `Dimensions int` (`gorm:"default:1024"`) for embedding output dim.

Task→provider resolution keys off `TaskTypes` + `IsEnabled` (+ `IsDefault` to break ties).

### 4.2 Encryption (`internal/llm/crypto.go` or `internal/utils`)
- AES-256-GCM. Key derived from `SMARTTICKET_SECRET_KEY` (base64/hex, 32 bytes).
- `Encrypt(plaintext) -> ciphertext`, `Decrypt(ciphertext) -> plaintext`.
- API layer never serializes plaintext keys; list/detail return a masked form
  (e.g. `sk-…e86` or `********`).

### 4.3 Service (`internal/llm/service.go`)
- Provider CRUD.
- Task→provider resolution: `ResolveChat()` / `ResolveEmbedding()` returns the
  default (or only) enabled provider for that purpose.
- `Test(providerID)`: decrypts key, calls chat + embedding, and runs the CortexDB
  round-trip; returns `{chatOK, embeddingOK, cortexOK, latencies, error}`.

### 4.4 Client (`internal/llm/client.go`)
- OpenAI-compatible HTTP client (reuse `openai-go/v3`, already transitively present
  via CortexDB; pin it as a direct dependency).
- `Chat(ctx, messages, opts) (string, error)`.
- `Embed(ctx, texts []string) ([][]float32, error)` — chunks to ≤10 inputs/request
  for the Aliyun batch cap; respects configured `Dimensions`.

### 4.5 API (`/api/v1/llm/providers`, admin-only)
- `GET    /llm/providers` — list (keys masked)
- `POST   /llm/providers` — create
- `GET    /llm/providers/:id` — detail (key masked)
- `PUT    /llm/providers/:id` — update (api_key optional; unchanged if omitted)
- `DELETE /llm/providers/:id` — delete
- `POST   /llm/providers/:id/test` — Test action

RBAC: read endpoints require `llm:read`; mutations + test require `llm:write`.

## 5. CortexDB Integration (`internal/knowledgebase`)

- Wrap `cortexdb.Open(cortexdb.DefaultConfig("data/cortex.db"))` with
  `WithEmbedder(...)`. Lifecycle managed in server startup/shutdown (open on boot,
  close on graceful shutdown).
- **Embedder adapter** (`internal/knowledgebase/embedder.go`): implements CortexDB's
  `Embedder` interface (`Embed`, `EmbedBatch`, `Dim`) by delegating to the configured
  embedding provider via `internal/llm`. `Dim()` returns 1024. `EmbedBatch` chunks to ≤10.
- `/health` (or `/api/v1/health`) reports CortexDB open status.
- **No** article ingest pipeline, **no** search/RAG endpoints this slice. CortexDB is
  exercised only by the provider `Test` round-trip.

## 6. Admin UI (`web`)

- New route `/llm` (team/admin-only via `TeamOnly` guard); nav entry added (admin-visible).
- Provider list (name, provider_type, task_types, model, default/enabled badges, masked key).
- Create/edit form: name, provider_type, api_endpoint, api_key (write-only; placeholder
  shows masked existing), model, task_types (chat/embedding multi-select), dimensions
  (shown when embedding selected), max_tokens, temperature, is_default, is_enabled.
- **Test** button per provider → shows chat/embedding/cortex OK + latencies or error.

## 7. Config & Secrets

- `config.go`: add `SMARTTICKET_SECRET_KEY` (encryption key; required when LLM features
  used). Provider data is DB-driven and managed via the admin UI — not config files.
- Prod: generate `SMARTTICKET_SECRET_KEY` on the box (mode 600, not in git), inject as
  `-e` like the JWT secret. Document in the deployment memory.
- DeepSeek + Aliyun keys: entered post-deploy via admin UI, encrypted in DB. Never
  committed. (Both pasted in plaintext during design — rotate after setup.)

## 8. Testing

- **Migration regression:** full `go test ./...` green after the modernc switch + manual smoke.
- **Unit:** provider CRUD; AES-GCM encrypt/decrypt round-trip; client `Chat`/`Embed`
  against a mock HTTP server (incl. ≤10 batch chunking); CortexDB embedder adapter
  against CortexDB's `DummyEmbedder`.
- **Integration (mocked external APIs):** `Test` action path end-to-end with a stub
  OpenAI-compatible server + a temp CortexDB file.

## 9. Out of Scope (next slices)

- Knowledge-article embedding/ingest into CortexDB.
- Semantic search endpoint + UI.
- RAG Q&A ("ask AI") and AI-assisted ticket replies.
- Native Anthropic client; usage/quota/cost tracking.
