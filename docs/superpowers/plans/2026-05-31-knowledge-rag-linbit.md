# Knowledge RAG + LINBIT Demo — Implementation Plan

> Executed via subagent-driven-development on branch `feat/ai-foundation` (continues the AI foundation). Steps use `- [ ]`.

**Goal:** Turn the AI foundation into a working RAG: index knowledge articles into CortexDB, expose semantic search + "Ask AI" (retrieval-augmented DeepSeek answers with citations), import the LINBIT UG9 docs as a knowledge base, and create a LINBIT customer to demo it.

**Architecture:** `internal/knowledgebase` wraps CortexDB's high-level Knowledge API (`SaveKnowledge`/`SearchKnowledge`/`DeleteKnowledge` — chunking+embedding+hybrid retrieval are internal). The `internal/knowledge` domain hooks indexing into article create/update/delete and adds search/ask endpoints; ask uses `llm.Service.ResolveChat` to generate an answer from the retrieved `Context`. A CLI command imports LINBIT `.adoc` docs as articles and creates the LINBIT customer.

**CortexDB API (verified, package `pkg/cortexdb`):**
- `(*DB).SaveKnowledge(ctx, KnowledgeSaveRequest{KnowledgeID, Title, Content, SourceURL, Author, Collection, ChunkSize, ChunkOverlap, Metadata}) (*KnowledgeSaveResponse, error)`
- `(*DB).SearchKnowledge(ctx, KnowledgeSearchRequest{Query, Collection, TopK, MaxContextChars, ...}) (*KnowledgeSearchResponse, error)` → `Results []KnowledgeSearchHit{KnowledgeID, Title, SourceURL, Snippet, Score}` + packed `Context string`.
- `(*DB).DeleteKnowledge(ctx, KnowledgeDeleteRequest{KnowledgeID}) (*KnowledgeDeleteResponse, error)` (verify field name via `go doc`).
- `(*DB).HasEmbedder() bool`.

---

## Task RAG-1: knowledgebase indexing + search methods

**Files:** `internal/knowledgebase/index.go` (create), `internal/knowledgebase/index_test.go` (create).

Add methods on `*Store`:
- `SaveArticle(ctx, id uint, title, content, sourceURL string) error` → `cortex.SaveKnowledge` with `KnowledgeID=fmt.Sprintf("article-%d", id)`, `Collection="knowledge"`.
- `DeleteArticle(ctx, id uint) error` → `cortex.DeleteKnowledge` (verify request field; ignore not-found).
- `Search(ctx, query string, topK int) (*SearchResult, error)` → `cortex.SearchKnowledge{Query, Collection:"knowledge", TopK:topK, MaxContextChars: 6000}`; map to a local `SearchResult{Context string; Hits []SearchHit{ArticleID uint, Title, Snippet string, Score float64}}`. Parse `ArticleID` back from the `article-%d` KnowledgeID (best-effort; 0 if unparseable).

`SearchHit`/`SearchResult` are package-local cycle-free structs (own JSON tags). Add `var _ = ...` only if needed.

- [ ] Write `index_test.go`: open a temp store with a `ProviderEmbedder` over a deterministic fake EmbedFunc (fixed-dim vectors, e.g. dim 8 hashing the text so similar text → similar vec is NOT required; just non-zero). Save two articles, Search a query, assert ≥1 hit returned and the round-trip `ArticleID` parse works. (This is an integration test against real CortexDB in `t.TempDir()`.) If CortexDB requires an embedder that returns consistent dims, ensure the fake returns the store's dim.
- [ ] Run → fail (methods undefined). Implement `index.go`. Verify `go doc` for exact `SaveKnowledge`/`SearchKnowledge`/`DeleteKnowledge` request/response field names and adapt. Run → pass.
- [ ] `go test ./internal/knowledgebase/ -v -count=1`; `go vet`. Commit: `feat(knowledgebase): article indexing + semantic search over CortexDB`.

## Task RAG-2: knowledge search/ask endpoints + indexing hooks

**Files:** `internal/knowledge/service.go` + `handlers.go` (modify), `internal/server/server.go` (modify), tests.

- Inject `*knowledgebase.Store` and `*llm.Service` into the knowledge service/handlers (constructor params; nil-safe — if store nil or `!HasEmbedder`, search/ask return a 503-style "AI not configured" error and indexing hooks are skipped).
- On article Create/Update success → `store.SaveArticle(...)` (best-effort: log warn on error, do not fail the request). On Delete → `store.DeleteArticle(...)`.
- `POST /api/v1/knowledge/search` (auth required, any role): body `{query, top_k?}` → `store.Search` → `{success, data:{hits}}`.
- `POST /api/v1/knowledge/ask` (auth required): body `{question, top_k?}` → `store.Search(question, topK)` → if no hits, answer "I don't have information on that." Else call `llm.Service` chat: resolve chat provider, `client.Chat(model, [{system: "Answer ONLY from the provided context. Cite article titles. If the context lacks the answer, say so."},{user: "Context:\n"+ctx+"\n\nQuestion: "+question}])` → `{success, data:{answer, citations:[{article_id,title,score}]}}`.
- `POST /api/v1/knowledge/reindex` (admin only): iterate all non-deleted articles, `SaveArticle` each; return count. 
- Add an `llm.Service` method if needed: a convenience `Chat(ctx, []ChatMessage) (string, error)` that resolves the chat provider, decrypts, and calls the client (so knowledge doesn't duplicate resolution). Put it in `internal/llm/service.go`.
- Wire in `server.go`: pass the existing `kbStore` + `llmService` into the knowledge handlers constructor; register the three routes (search/ask under `protected`, reindex under admin).

- [ ] TDD where practical (service Chat convenience method unit-tested with httptest; search/ask handler happy-path with a temp store + httptest chat server). Best-effort indexing hooks tested for non-fatal behavior (save error doesn't fail create).
- [ ] `go build ./... && go test ./internal/knowledge/ ./internal/llm/ ./internal/server/ -count=1`; vet. Commit: `feat(knowledge): semantic search + RAG ask endpoints + auto-indexing`.

## Task RAG-3: LINBIT import command + LINBIT customer

**Files:** `cmd/server/main.go` (add command) or `internal/importexport`/a new `internal/linbit` helper; `internal/customer` reuse.

- Add a Cobra command `importlinbit` that:
  - Fetches the file list of `LINBIT/linbit-documentation/UG9/en` via the GitHub API (`https://api.github.com/repos/LINBIT/linbit-documentation/contents/UG9/en`), downloads each `.adoc` raw.
  - Light AsciiDoc→text cleanup: strip `ifdef/endif/ifndef`, attribute lines (`:name: val`), image/include macros, convert `== Heading` to plain headings, drop block delimiters (`----`, `====`); keep prose + code. (A simple regex pass is fine — perfection not required for RAG.)
  - Creates one `KnowledgeArticle` per file (Title from the first `= `/`== ` heading or filename; Content = cleaned text; Status published; Visibility public; SourceURL = the GitHub blob URL). Skips if an article with that title already exists (idempotent re-run).
  - After creating each article, calls `store.SaveArticle` to index it (or relies on the service hook if it routes through the service).
  - Creates a `Customer{Name:"LINBIT", Code:"LINBIT", IsActive:true}` if absent (idempotent).
  - Flags: `--config`. Logs counts (articles created, indexed, skipped).
- Keep network failures non-fatal per file (log + continue).

- [ ] Manual verification (network): run `go run ./cmd/server importlinbit --config configs/config.dev.yaml` against a local DB with providers configured; confirm ~40 articles created + indexed and a LINBIT customer exists. (Requires Aliyun embedding provider configured, else indexing is skipped but articles still import.)
- [ ] Commit: `feat(cli): importlinbit — load LINBIT UG9 docs as knowledge + create LINBIT customer`.

## Task RAG-4: Knowledge UI — search + Ask AI

**Files:** `web/src/features/knowledge/api.ts` (extend), `web/src/pages/knowledge*.tsx` (modify), maybe a new `AskPanel` component.

- Add `search(query, topK)` and `ask(question, topK)` to the knowledge feature api.
- On the Knowledge list page: a search input that switches the list to ranked semantic hits (title + snippet + score), clearing returns to the normal list.
- An "Ask AI" panel/dialog: textarea question → calls `/knowledge/ask` → renders the answer (markdown) + a "Sources" list of cited articles (link to article detail). Loading + error states. Gracefully handle the "AI not configured" error with a hint to configure providers.
- Follow existing feature/page patterns; full-width; sonner for errors.

- [ ] `cd web && pnpm build`. Commit: `feat(web): knowledge semantic search + Ask AI (RAG) panel`.

## Task RAG-5: integrate, deploy, demo (runtime — keys NOT committed)

- [ ] Full `go test ./...` + `pnpm build`. Merge `feat/ai-foundation` → main (`--no-ff`), push.
- [ ] Deploy backend (see production-deployment memory): generate `/opt/smartticket/.secret_key` (openssl rand -hex 32, 600), rebuild image (CGO_ENABLED=0), run with `-e SMARTTICKET_SECRET_KEY=$(cat ...)`. Deploy frontend.
- [ ] Via admin UI at /llm: create CHAT provider (DeepSeek: base_url `https://api.deepseek.com`, model `deepseek-chat`, key sk-6e37…) and EMBEDDING provider (Aliyun: base_url `https://dashscope.aliyuncs.com/compatible-mode/v1`, model `text-embedding-v4`, dim 1024, key sk-18725…). Test both (chat_ok/embedding_ok/cortex_ok).
- [ ] On the box: `docker exec smartticket ./smartticket importlinbit --config configs/config.prod.yaml`. Verify articles + LINBIT customer.
- [ ] Demo: open Knowledge, run a search ("how to configure DRBD?") and Ask AI; confirm an answer with LINBIT citations. Report the result.
- [ ] Update `production-deployment` memory with the secret-key step + importlinbit.

## Notes
- DeepSeek = chat only (no embeddings); Aliyun text-embedding-v4 (dim 1024) = embeddings. Both OpenAI-compatible via the existing `internal/llm` client.
- Keys appeared in plaintext during design → rotate after the demo.
- Knowledge is global (not customer-scoped) in this slice; the LINBIT customer demonstrates the org/team surface. Customer-scoped KB is out of scope.
