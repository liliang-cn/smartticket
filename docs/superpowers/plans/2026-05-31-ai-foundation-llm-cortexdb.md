# AI Foundation — LLM Providers + CortexDB — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build SmartTicket's AI foundation — admin-editable LLM providers (independent chat & embedding config), an OpenAI-compatible client, and CortexDB integration — with no end-user AI feature yet.

**Architecture:** Migrate the whole project to pure-Go SQLite (modernc via glebarez) first. Then add an `internal/llm` domain (provider CRUD + AES-GCM encrypted keys + OpenAI-compatible Chat/Embed client + task→provider resolution) and an `internal/knowledgebase` package wrapping CortexDB with an Embedder backed by the configured embedding provider. Each provider row independently configures base_url + model + api_key and is tagged chat and/or embedding via `TaskTypes`.

**Tech Stack:** Go 1.25, GORM, GIN, `github.com/glebarez/sqlite` (modernc), `github.com/openai/openai-go/v3`, `github.com/liliang-cn/cortexdb/v2`, React/Vite/TS frontend.

**Spec:** `docs/superpowers/specs/2026-05-31-ai-foundation-llm-cortexdb-design.md`

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/database/database.go` (modify) | Swap GORM dialector mattn→glebarez; translate DSN to modernc `_pragma` syntax |
| `Dockerfile` (modify) | `CGO_ENABLED=0`; drop gcc/musl-dev |
| `internal/config/config.go` (modify) | Add `SMARTTICKET_SECRET_KEY` loading |
| `internal/llm/crypto.go` (create) | AES-256-GCM encrypt/decrypt + key loading |
| `internal/llm/crypto_test.go` (create) | Crypto round-trip tests |
| `internal/models/models.go` (modify) | `LLMProvider`: `APIKey` json:"-", add `Dimensions` |
| `internal/llm/client.go` (create) | OpenAI-compatible `Chat()` + `Embed()` (batch ≤10) |
| `internal/llm/client_test.go` (create) | Client tests vs mock HTTP server |
| `internal/llm/service.go` (create) | Provider CRUD, encrypt-on-save, `ResolveChat`/`ResolveEmbedding`, `Test` |
| `internal/llm/service_test.go` (create) | Service tests |
| `internal/llm/handlers.go` (create) | REST handlers (keys masked) |
| `internal/llm/handlers_test.go` (create) | Handler tests |
| `internal/knowledgebase/store.go` (create) | CortexDB open/close lifecycle |
| `internal/knowledgebase/embedder.go` (create) | CortexDB `Embedder` adapter → llm embedding provider |
| `internal/knowledgebase/embedder_test.go` (create) | Adapter tests |
| `internal/database/permissions.go` (modify) | Add `llm:read`/`llm:write` to catalog + engineer grants |
| `internal/server/server.go` (modify) | Construct llm service/handlers + knowledgebase store; register `/llm/providers` routes; health includes cortex |
| `web/src/features/llm/*` (create) | API client, types, hooks |
| `web/src/pages/llm-providers.tsx` (create) | Admin provider list + form + Test |
| `web/src/App.tsx`, `web/src/components/app-shell.tsx` (modify) | Route + nav entry (team-only) |

---

## Task 1: Migrate SQLite driver to modernc (drop CGO)

**Files:**
- Modify: `internal/database/database.go:10` (import), `:95-103` (openSQLite)
- Modify: `go.mod` / `go.sum`
- Modify: `Dockerfile`

- [ ] **Step 1: Add glebarez dependency**

Run:
```bash
go get github.com/glebarez/sqlite@v1.11.0
```
Expected: `go: added github.com/glebarez/sqlite v1.11.0`

- [ ] **Step 2: Swap the dialector import**

In `internal/database/database.go`, change line 10:
```go
// from:
"gorm.io/driver/sqlite"
// to:
"github.com/glebarez/sqlite"
```

- [ ] **Step 3: Translate the DSN to modernc pragma syntax**

`modernc.org/sqlite` does NOT understand mattn's `_journal_mode=`/`_foreign_keys=` params. Replace `openSQLite` (lines 95-103) with:
```go
// openSQLite creates a new SQLite database connection (pure-Go modernc driver).
func openSQLite(dsn string, gormLogger logger.Interface) (*gorm.DB, error) {
	// modernc uses _pragma=NAME(value) syntax. Foreign keys are disabled during
	// migration (re-enabled by EnableForeignKeys) to avoid constraint churn.
	if dsn != ":memory:" {
		dsn = fmt.Sprintf(
			"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(0)&_pragma=busy_timeout(5000)",
			dsn,
		)
	}

	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:      gormLogger,
		PrepareStmt: true,
	})
}
```

- [ ] **Step 4: Verify the foreign-keys pragma toggles still work**

`EnableForeignKeys`/`DisableForeignKeys` use `PRAGMA foreign_keys = ON/OFF` (Exec), which modernc supports — no change needed. Confirm by reading `internal/database/database.go:121-134`.

- [ ] **Step 5: Run the full test suite**

Run:
```bash
go test ./... 2>&1 | tail -40
```
Expected: all packages `ok` / `[no test files]`, no `FAIL`. Pay attention to `internal/database`, `internal/repositories`, `internal/customer`, `internal/ticket`.

- [ ] **Step 6: Tidy and confirm CGO is gone**

Run:
```bash
go mod tidy && CGO_ENABLED=0 go build ./cmd/server 2>&1 | tail -5 && echo "BUILD OK (no cgo)"
```
Expected: `BUILD OK (no cgo)` (mattn/go-sqlite3 should drop out of go.mod as a dependency).

- [ ] **Step 7: Smoke test the server**

Run:
```bash
rm -f ./data/smoke.db
SMARTTICKET_DATABASE_CONNECTION_URL=./data/smoke.db go run ./cmd/server migrate 2>&1 | tail -5
```
Expected: migrations complete without error; `./data/smoke.db` created.

- [ ] **Step 8: Update Dockerfile to drop CGO**

In `Dockerfile`, change the builder stage:
```dockerfile
# from:
RUN CGO_ENABLED=1 GOOS=linux go build ...
# to:
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o smartticket ./cmd/server
```
Remove the `apk add ... gcc musl-dev` (and `git` if only there for cgo) line from the builder stage. Keep any line needed for fetching modules.

- [ ] **Step 9: Commit**

```bash
git add internal/database/database.go go.mod go.sum Dockerfile
git commit -m "refactor(db): migrate SQLite to pure-Go modernc (glebarez), drop CGO"
```

---

## Task 2: AES-256-GCM credential crypto

**Files:**
- Create: `internal/llm/crypto.go`
- Test: `internal/llm/crypto_test.go`

- [ ] **Step 1: Write the failing test**

`internal/llm/crypto_test.go`:
```go
package llm

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32) // 32 zero bytes is a valid AES-256 key for the test
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	plain := "sk-secret-12345"
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plain || enc == "" {
		t.Fatalf("ciphertext not transformed: %q", enc)
	}
	got, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plain)
	}
}

func TestEncryptNonceVaries(t *testing.T) {
	c, _ := NewCipher(make([]byte, 32))
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("expected different ciphertext per call (random nonce)")
	}
}

func TestNewCipherRejectsBadKey(t *testing.T) {
	if _, err := NewCipher(make([]byte, 16)); err == nil {
		t.Fatal("expected error for non-32-byte key")
	}
}

func TestLoadKeyFromHexAndBase64(t *testing.T) {
	// 32 bytes hex = 64 chars
	hexKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	if _, err := LoadKey(hexKey); err != nil {
		t.Fatalf("hex key: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/llm/ -run TestEncrypt -v`
Expected: FAIL — undefined `NewCipher`.

- [ ] **Step 3: Implement crypto.go**

`internal/llm/crypto.go`:
```go
// Package llm provides LLM provider management, an OpenAI-compatible client,
// and credential encryption for SmartTicket's AI foundation.
package llm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// Cipher encrypts and decrypts provider API keys with AES-256-GCM.
type Cipher struct {
	gcm cipher.AEAD
}

// NewCipher builds a Cipher from a 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{gcm: gcm}, nil
}

// Encrypt returns base64(nonce||ciphertext).
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	ns := c.gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := c.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// LoadKey decodes a 32-byte key from a hex (64 chars) or base64 string.
func LoadKey(s string) ([]byte, error) {
	if len(s) == 64 {
		if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, errors.New("SMARTTICKET_SECRET_KEY must be a 32-byte key (hex or base64)")
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/llm/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/llm/crypto.go internal/llm/crypto_test.go
git commit -m "feat(llm): AES-256-GCM credential cipher"
```

---

## Task 3: Extend LLMProvider model

**Files:**
- Modify: `internal/models/models.go:147-162`

- [ ] **Step 1: Change APIKey JSON tag and add Dimensions**

In `internal/models/models.go`, edit the `LLMProvider` struct:
```go
// APIKey holds AES-GCM ciphertext; json:"-" so it never serializes to clients.
APIKey     string `gorm:"size:500" json:"-"`
// Dimensions is the embedding output dimension (used when TaskTypes includes "embedding").
Dimensions int    `gorm:"default:1024" json:"dimensions"`
```
(Keep all other fields. Place `Dimensions` after `Model`.)

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/models/ && echo OK`
Expected: `OK`. GORM AutoMigrate (run on startup/migrate) adds the `dimensions` column automatically.

- [ ] **Step 3: Run model tests**

Run: `go test ./internal/models/ -v 2>&1 | tail -15`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/models/models.go
git commit -m "feat(models): LLMProvider hide api_key from JSON, add dimensions"
```

---

## Task 4: OpenAI-compatible client (Chat + Embed)

**Files:**
- Create: `internal/llm/client.go`
- Test: `internal/llm/client_test.go`

- [ ] **Step 1: Add the openai-go dependency as direct**

Run:
```bash
go get github.com/openai/openai-go/v3@v3.37.0
```
Expected: added/promoted to direct require.

- [ ] **Step 2: Write the failing test (mock HTTP server)**

`internal/llm/client_test.go`:
```go
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"role": "assistant", "content": "hi there"},
			}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	out, err := c.Chat(context.Background(), "deepseek-chat", []ChatMessage{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi there" {
		t.Fatalf("got %q", out)
	}
}

func TestClientEmbedBatchesAtTen(t *testing.T) {
	var batchSizes []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []string `json:"input"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		batchSizes = append(batchSizes, len(body.Input))
		data := make([]any, len(body.Input))
		for i := range body.Input {
			data[i] = map[string]any{"embedding": []float32{0.1, 0.2}, "index": i}
		}
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	texts := make([]string, 23)
	for i := range texts {
		texts[i] = "t"
	}
	vecs, err := c.Embed(context.Background(), "text-embedding-v4", 2, texts)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 23 {
		t.Fatalf("want 23 vectors, got %d", len(vecs))
	}
	// 23 inputs, cap 10 => batches of 10,10,3
	want := []int{10, 10, 3}
	if len(batchSizes) != 3 || batchSizes[0] != want[0] || batchSizes[2] != want[2] {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/llm/ -run TestClient -v`
Expected: FAIL — undefined `NewClient`.

- [ ] **Step 4: Implement client.go**

`internal/llm/client.go`:
```go
package llm

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// maxEmbeddingBatch is the per-request input cap (Aliyun text-embedding-v4 = 10).
const maxEmbeddingBatch = 10

// ChatMessage is a single chat turn.
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Client talks to any OpenAI-compatible endpoint.
type Client struct {
	api openai.Client
}

// NewClient builds a client for the given base URL and API key.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		api: openai.NewClient(
			option.WithBaseURL(baseURL),
			option.WithAPIKey(apiKey),
		),
	}
}

// Chat sends messages and returns the assistant's text.
func (c *Client) Chat(ctx context.Context, model string, msgs []ChatMessage) (string, error) {
	oa := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			oa = append(oa, openai.SystemMessage(m.Content))
		case "assistant":
			oa = append(oa, openai.AssistantMessage(m.Content))
		default:
			oa = append(oa, openai.UserMessage(m.Content))
		}
	}
	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: oa,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}

// Embed returns one vector per input text, batching to maxEmbeddingBatch per
// request. dimensions is sent when > 0 (v3/v4 support it).
func (c *Client) Embed(ctx context.Context, model string, dimensions int, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += maxEmbeddingBatch {
		end := start + maxEmbeddingBatch
		if end > len(texts) {
			end = len(texts)
		}
		params := openai.EmbeddingNewParams{
			Model: openai.EmbeddingModel(model),
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: texts[start:end],
			},
		}
		if dimensions > 0 {
			params.Dimensions = openai.Int(int64(dimensions))
		}
		resp, err := c.api.Embeddings.New(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, d := range resp.Data {
			v := make([]float32, len(d.Embedding))
			for i, f := range d.Embedding {
				v[i] = float32(f)
			}
			out = append(out, v)
		}
	}
	return out, nil
}
```

> Note: the exact openai-go v3 param type names (`EmbeddingNewParamsInputUnion`, `OfArrayOfStrings`, `option.WithBaseURL`) must match the installed SDK. If a symbol differs, run `go doc github.com/openai/openai-go/v3 <Type>` to find the current name and adjust. The mock test pins the wire behavior regardless of SDK surface.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/llm/ -run TestClient -v`
Expected: PASS (batch sizes `[10 10 3]`).

- [ ] **Step 6: Commit**

```bash
git add internal/llm/client.go internal/llm/client_test.go go.mod go.sum
git commit -m "feat(llm): OpenAI-compatible Chat/Embed client with batch chunking"
```

---

## Task 5: CortexDB embedder adapter + store

**Files:**
- Create: `internal/knowledgebase/embedder.go`
- Create: `internal/knowledgebase/store.go`
- Test: `internal/knowledgebase/embedder_test.go`

- [ ] **Step 1: Add the cortexdb dependency**

Run:
```bash
go get github.com/liliang-cn/cortexdb/v2@v2.20.2
```
Expected: added to require.

- [ ] **Step 2: Write the failing test**

`internal/knowledgebase/embedder_test.go`:
```go
package knowledgebase

import (
	"context"
	"testing"
)

// fakeEmbed implements EmbedFunc for the adapter.
func TestProviderEmbedderDelegates(t *testing.T) {
	called := 0
	fn := func(ctx context.Context, texts []string) ([][]float32, error) {
		called++
		out := make([][]float32, len(texts))
		for i := range texts {
			out[i] = []float32{1, 2, 3}
		}
		return out, nil
	}
	e := NewProviderEmbedder(fn, 3)

	if e.Dim() != 3 {
		t.Fatalf("Dim=%d want 3", e.Dim())
	}
	v, err := e.Embed(context.Background(), "hello")
	if err != nil || len(v) != 3 {
		t.Fatalf("Embed got %v err %v", v, err)
	}
	vs, err := e.EmbedBatch(context.Background(), []string{"a", "b"})
	if err != nil || len(vs) != 2 {
		t.Fatalf("EmbedBatch got %d vecs err %v", len(vs), err)
	}
	if called != 2 {
		t.Fatalf("delegate called %d times, want 2", called)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/knowledgebase/ -run TestProviderEmbedder -v`
Expected: FAIL — undefined `NewProviderEmbedder`.

- [ ] **Step 4: Implement embedder.go**

`internal/knowledgebase/embedder.go`:
```go
// Package knowledgebase integrates the CortexDB vector/knowledge store, with an
// embedder backed by SmartTicket's configured LLM embedding provider.
package knowledgebase

import "context"

// EmbedFunc embeds a batch of texts (typically internal/llm's Embed bound to the
// resolved embedding provider).
type EmbedFunc func(ctx context.Context, texts []string) ([][]float32, error)

// ProviderEmbedder adapts an EmbedFunc to CortexDB's Embedder interface
// (Embed, EmbedBatch, Dim).
type ProviderEmbedder struct {
	fn  EmbedFunc
	dim int
}

// NewProviderEmbedder wraps fn with a fixed output dimension.
func NewProviderEmbedder(fn EmbedFunc, dim int) *ProviderEmbedder {
	return &ProviderEmbedder{fn: fn, dim: dim}
}

func (e *ProviderEmbedder) Dim() int { return e.dim }

func (e *ProviderEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vs, err := e.fn(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vs) == 0 {
		return nil, nil
	}
	return vs[0], nil
}

func (e *ProviderEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.fn(ctx, texts)
}
```

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/knowledgebase/ -run TestProviderEmbedder -v`
Expected: PASS.

- [ ] **Step 6: Implement store.go (verify Embedder interface name first)**

First confirm CortexDB's open/config/embedder API:
```bash
go doc github.com/liliang-cn/cortexdb/v2 Open
go doc github.com/liliang-cn/cortexdb/v2 DefaultConfig
go doc github.com/liliang-cn/cortexdb/v2 WithEmbedder
go doc github.com/liliang-cn/cortexdb/v2 Embedder
```
Then `internal/knowledgebase/store.go`:
```go
package knowledgebase

import (
	"fmt"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

// Store wraps a CortexDB instance.
type Store struct {
	db *cortexdb.DB
}

// Open opens (or creates) the CortexDB file at path, using embedder for
// vectorization. Adjust the option call to match the verified API surface.
func Open(path string, embedder cortexdb.Embedder) (*Store, error) {
	cfg := cortexdb.DefaultConfig(path)
	db, err := cortexdb.Open(cfg, cortexdb.WithEmbedder(embedder))
	if err != nil {
		return nil, fmt.Errorf("open cortexdb: %w", err)
	}
	return &Store{db: db}, nil
}

// DB exposes the underlying CortexDB handle.
func (s *Store) DB() *cortexdb.DB { return s.db }

// Healthy reports whether the store is open.
func (s *Store) Healthy() bool { return s != nil && s.db != nil }

// Close closes the store.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
```
> The exact `cortexdb.Open` signature / `WithEmbedder` option / `Embedder` interface must match `go doc` output. Adjust the types in `Open` accordingly; the `ProviderEmbedder` methods (`Embed`/`EmbedBatch`/`Dim`) are written to satisfy the documented interface.

- [ ] **Step 7: Compile + test**

Run: `go build ./internal/knowledgebase/ && go test ./internal/knowledgebase/ -v 2>&1 | tail -15`
Expected: build OK, tests PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/knowledgebase/ go.mod go.sum
git commit -m "feat(knowledgebase): CortexDB store + provider-backed embedder adapter"
```

---

## Task 6: LLM provider service (CRUD + resolve + test)

**Files:**
- Create: `internal/llm/service.go`
- Test: `internal/llm/service_test.go`

- [ ] **Step 1: Write failing tests**

`internal/llm/service_test.go`:
```go
package llm

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newTestService(t *testing.T) *Service {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LLMProvider{}); err != nil {
		t.Fatal(err)
	}
	cipher, _ := NewCipher(make([]byte, 32))
	return NewService(db, cipher)
}

func TestCreateEncryptsKeyAndResolves(t *testing.T) {
	s := newTestService(t)
	p, err := s.Create(CreateProviderInput{
		Name: "embed", ProviderType: "openai-compatible",
		APIEndpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:      "sk-secret", Model: "text-embedding-v4",
		TaskTypes: []string{"embedding"}, Dimensions: 1024,
		IsDefault: true, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Stored key must be ciphertext, not plaintext.
	var row models.LLMProvider
	s.db.First(&row, p.ID)
	if row.APIKey == "sk-secret" || row.APIKey == "" {
		t.Fatalf("api key not encrypted at rest: %q", row.APIKey)
	}
	// Resolve embedding provider + decrypt.
	rp, key, err := s.ResolveEmbedding()
	if err != nil {
		t.Fatal(err)
	}
	if rp.ID != p.ID || key != "sk-secret" {
		t.Fatalf("resolve mismatch: id=%d key=%q", rp.ID, key)
	}
}

func TestResolveChatErrorsWhenNone(t *testing.T) {
	s := newTestService(t)
	if _, _, err := s.ResolveChat(); err == nil {
		t.Fatal("expected error when no chat provider configured")
	}
}

func TestMaskedKey(t *testing.T) {
	if got := MaskKey("sk-1234567890abcdef"); got == "sk-1234567890abcdef" || got == "" {
		t.Fatalf("mask failed: %q", got)
	}
	if MaskKey("") != "" {
		t.Fatal("empty stays empty")
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/llm/ -run 'TestCreate|TestResolve|TestMasked' -v`
Expected: FAIL — undefined `Service`/`NewService`.

- [ ] **Step 3: Implement service.go**

`internal/llm/service.go`:
```go
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// Service manages LLM providers and resolves them by task.
type Service struct {
	db     *gorm.DB
	cipher *Cipher
}

// NewService builds the service.
func NewService(db *gorm.DB, cipher *Cipher) *Service {
	return &Service{db: db, cipher: cipher}
}

// CreateProviderInput is the create payload (plaintext APIKey).
type CreateProviderInput struct {
	Name         string   `json:"name" binding:"required"`
	ProviderType string   `json:"provider_type" binding:"required"`
	APIEndpoint  string   `json:"api_endpoint" binding:"required"`
	APIKey       string   `json:"api_key"`
	Model        string   `json:"model" binding:"required"`
	TaskTypes    []string `json:"task_types" binding:"required"`
	Dimensions   int      `json:"dimensions"`
	MaxTokens    int      `json:"max_tokens"`
	Temperature  float64  `json:"temperature"`
	IsDefault    bool     `json:"is_default"`
	IsEnabled    bool     `json:"is_enabled"`
}

func (s *Service) Create(in CreateProviderInput) (*models.LLMProvider, error) {
	enc := ""
	if in.APIKey != "" {
		var err error
		if enc, err = s.cipher.Encrypt(in.APIKey); err != nil {
			return nil, err
		}
	}
	tt, _ := json.Marshal(in.TaskTypes)
	dim := in.Dimensions
	if dim == 0 {
		dim = 1024
	}
	p := &models.LLMProvider{
		Name: in.Name, ProviderType: in.ProviderType, APIEndpoint: in.APIEndpoint,
		APIKey: enc, Model: in.Model, TaskTypes: string(tt), Dimensions: dim,
		MaxTokens: in.MaxTokens, Temperature: in.Temperature,
		IsDefault: in.IsDefault, IsEnabled: in.IsEnabled,
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// Update applies non-zero fields; APIKey is only re-encrypted when non-empty.
func (s *Service) Update(id uint, in CreateProviderInput) (*models.LLMProvider, error) {
	var p models.LLMProvider
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	p.Name, p.ProviderType, p.APIEndpoint = in.Name, in.ProviderType, in.APIEndpoint
	p.Model = in.Model
	tt, _ := json.Marshal(in.TaskTypes)
	p.TaskTypes = string(tt)
	if in.Dimensions > 0 {
		p.Dimensions = in.Dimensions
	}
	p.MaxTokens, p.Temperature = in.MaxTokens, in.Temperature
	p.IsDefault, p.IsEnabled = in.IsDefault, in.IsEnabled
	if in.APIKey != "" {
		enc, err := s.cipher.Encrypt(in.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKey = enc
	}
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) List() ([]models.LLMProvider, error) {
	var ps []models.LLMProvider
	return ps, s.db.Order("id").Find(&ps).Error
}

func (s *Service) Get(id uint) (*models.LLMProvider, error) {
	var p models.LLMProvider
	return &p, s.db.First(&p, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.LLMProvider{}, id).Error
}

func (s *Service) resolve(task string) (*models.LLMProvider, string, error) {
	var ps []models.LLMProvider
	if err := s.db.Where("is_enabled = ?", true).Order("is_default desc, id").Find(&ps).Error; err != nil {
		return nil, "", err
	}
	for _, p := range ps {
		var tt []string
		_ = json.Unmarshal([]byte(p.TaskTypes), &tt)
		for _, t := range tt {
			if t == task {
				key := ""
				if p.APIKey != "" {
					var err error
					if key, err = s.cipher.Decrypt(p.APIKey); err != nil {
						return nil, "", err
					}
				}
				pp := p
				return &pp, key, nil
			}
		}
	}
	return nil, "", fmt.Errorf("no enabled provider configured for task %q", task)
}

// ResolveChat returns the chat provider and its decrypted key.
func (s *Service) ResolveChat() (*models.LLMProvider, string, error) { return s.resolve("chat") }

// ResolveEmbedding returns the embedding provider and its decrypted key.
func (s *Service) ResolveEmbedding() (*models.LLMProvider, string, error) {
	return s.resolve("embedding")
}

// TestResult reports the outcome of a provider self-test.
type TestResult struct {
	ChatOK      bool   `json:"chat_ok"`
	EmbeddingOK bool   `json:"embedding_ok"`
	CortexOK    bool   `json:"cortex_ok"`
	LatencyMS   int64  `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
}

// Test exercises the resolved chat + embedding providers. cortexProbe, if
// non-nil, runs an embed→store→recall round-trip and sets CortexOK.
func (s *Service) Test(ctx context.Context, cortexProbe func(ctx context.Context, vec []float32) error) TestResult {
	start := time.Now()
	res := TestResult{}

	if cp, key, err := s.ResolveChat(); err == nil {
		c := NewClient(cp.APIEndpoint, key)
		if _, err := c.Chat(ctx, cp.Model, []ChatMessage{{Role: "user", Content: "ping"}}); err == nil {
			res.ChatOK = true
		} else {
			res.Error = "chat: " + err.Error()
		}
	}

	if ep, key, err := s.ResolveEmbedding(); err == nil {
		c := NewClient(ep.APIEndpoint, key)
		vecs, err := c.Embed(ctx, ep.Model, ep.Dimensions, []string{"hello"})
		if err == nil && len(vecs) == 1 {
			res.EmbeddingOK = true
			if cortexProbe != nil {
				if err := cortexProbe(ctx, vecs[0]); err == nil {
					res.CortexOK = true
				} else if res.Error == "" {
					res.Error = "cortex: " + err.Error()
				}
			}
		} else if res.Error == "" {
			res.Error = "embedding: " + errString(err)
		}
	}

	res.LatencyMS = time.Since(start).Milliseconds()
	return res
}

func errString(err error) string {
	if err == nil {
		return "no vectors returned"
	}
	return err.Error()
}

// ErrNoProvider is returned by resolve helpers when nothing matches.
var ErrNoProvider = errors.New("no provider")
```

- [ ] **Step 4: Implement MaskKey (in service.go or a small helper)**

Append to `internal/llm/service.go`:
```go
// MaskKey returns a display-safe form of a plaintext key (never store output).
func MaskKey(plain string) string {
	if plain == "" {
		return ""
	}
	if len(plain) <= 6 {
		return "****"
	}
	return plain[:3] + "…" + plain[len(plain)-3:]
}
```

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/llm/ -v 2>&1 | tail -25`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/llm/service.go internal/llm/service_test.go
git commit -m "feat(llm): provider service — CRUD, encrypted keys, task resolution, self-test"
```

---

## Task 7: HTTP handlers + routes + RBAC

**Files:**
- Create: `internal/llm/handlers.go`
- Test: `internal/llm/handlers_test.go`
- Modify: `internal/database/permissions.go:34` (catalog) + engineer grants
- Modify: `internal/server/server.go` (construct + route + health)

- [ ] **Step 1: Add permission codes**

In `internal/database/permissions.go`, add to `permissionCatalog` (after the rbac entries):
```go
	{"llm:read", "Read LLM providers", "llm"},
	{"llm:write", "Manage LLM providers", "llm"},
```
Add to the `engineer` grant list:
```go
		"llm:read",
```
(admin already gets `*`.)

- [ ] **Step 2: Write the handler (mirror `internal/customer/handlers.go`)**

`internal/llm/handlers.go`:
```go
package llm

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/company/smartticket/internal/models"
)

// CortexProbe runs an embed→store→recall round-trip with a sample vector.
type CortexProbe func(ctx context.Context, vec []float32) error

// Handlers exposes LLM provider REST endpoints.
type Handlers struct {
	svc   *Service
	probe CortexProbe
}

// NewHandlers builds handlers.
func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

// SetCortexProbe injects the CortexDB round-trip probe (set by the server after
// the store opens; keeps this package free of a knowledgebase import).
func (h *Handlers) SetCortexProbe(fn CortexProbe) { h.probe = fn }

// providerView is the masked, client-safe representation.
type providerView struct {
	models.LLMProvider
	APIKeyMasked string `json:"api_key_masked"`
}

func (h *Handlers) view(p models.LLMProvider) providerView {
	masked := ""
	if p.APIKey != "" {
		masked = "********" // ciphertext on disk; show only that a key exists
	}
	return providerView{LLMProvider: p, APIKeyMasked: masked}
}

func (h *Handlers) List(c *gin.Context) {
	ps, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	views := make([]providerView, len(ps))
	for i, p := range ps {
		views[i] = h.view(p)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": views})
}

func (h *Handlers) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "provider not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.view(*p)})
}

func (h *Handlers) Create(c *gin.Context) {
	var in CreateProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	p, err := h.svc.Create(in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": h.view(*p)})
}

func (h *Handlers) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in CreateProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	p, err := h.svc.Update(uint(id), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.view(*p)})
}

func (h *Handlers) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Test runs the provider self-test. The cortex probe is injected by the server
// (see SetCortexProbe) so this package doesn't import knowledgebase.
func (h *Handlers) Test(c *gin.Context) {
	var probe func(context.Context, []float32) error
	if h.probe != nil {
		probe = h.probe
	}
	res := h.svc.Test(c.Request.Context(), probe)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}
```

- [ ] **Step 3: Write handler test**

`internal/llm/handlers_test.go`:
```go
package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newTestHandlers(t *testing.T) *Handlers {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&models.LLMProvider{})
	cipher, _ := NewCipher(make([]byte, 32))
	return NewHandlers(NewService(db, cipher))
}

func TestCreateAndListMasksKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandlers(t)
	r := gin.New()
	r.POST("/llm/providers", h.Create)
	r.GET("/llm/providers", h.List)

	body := `{"name":"chat","provider_type":"openai-compatible","api_endpoint":"https://api.deepseek.com","api_key":"sk-deepseek","model":"deepseek-chat","task_types":["chat"],"is_enabled":true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/llm/providers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create code %d body %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sk-deepseek") {
		t.Fatal("plaintext key leaked in create response")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/llm/providers", nil)
	r.ServeHTTP(w, req)
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 provider, got %d", len(resp.Data))
	}
	if _, ok := resp.Data[0]["api_key"]; ok {
		t.Fatal("api_key must not be serialized")
	}
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/llm/ -run TestCreateAndList -v`
Expected: PASS.

- [ ] **Step 5: Wire into server.go**

In `internal/server/server.go` near the customer wiring (line ~163):
```go
// LLM providers + knowledge store (AI foundation).
// Encryption key: SMARTTICKET_SECRET_KEY (see config task). Dev fallback:
// SHA-256(JWT secret) so local runs work without extra setup (NOT for prod).
key, err := llm.LoadKey(s.config.SecretKeyRaw)
if err != nil {
	sum := sha256.Sum256([]byte(s.config.JWT.Secret))
	key = sum[:]
}
cipher, err := llm.NewCipher(key)
if err != nil {
	return nil, fmt.Errorf("llm cipher: %w", err)
}
llmService := llm.NewService(s.db.DB, cipher)
llmHandlers := llm.NewHandlers(llmService)

// CortexDB store, embedder bound to the resolved embedding provider.
embedder := knowledgebase.NewProviderEmbedder(func(ctx context.Context, texts []string) ([][]float32, error) {
	ep, key, err := llmService.ResolveEmbedding()
	if err != nil {
		return nil, err
	}
	return llm.NewClient(ep.APIEndpoint, key).Embed(ctx, ep.Model, ep.Dimensions, texts)
}, 1024)
kbStore, err := knowledgebase.Open("./data/cortex.db", embedder)
if err != nil {
	s.logger.Warn("cortexdb unavailable", zap.Error(err)) // non-fatal
}
llmHandlers.SetCortexProbe(func(ctx context.Context, vec []float32) error {
	if kbStore == nil || !kbStore.Healthy() {
		return fmt.Errorf("cortexdb not open")
	}
	return nil // round-trip probe filled in once ingest API is added (next slice)
})
```
Register routes (near the customers group, ~line 290), admin-only:
```go
llmGroup := protected.Group("/llm/providers")
llmGroup.Use(s.adminMiddleware())
{
	llmGroup.GET("", llmHandlers.List)
	llmGroup.POST("", llmHandlers.Create)
	llmGroup.GET("/:id", llmHandlers.Get)
	llmGroup.PUT("/:id", llmHandlers.Update)
	llmGroup.DELETE("/:id", llmHandlers.Delete)
	llmGroup.POST("/:id/test", llmHandlers.Test)
}
```
Add imports: `"context"`, `"github.com/company/smartticket/internal/llm"`, `"github.com/company/smartticket/internal/knowledgebase"`. Store `kbStore` on the server struct and `kbStore.Close()` in the server's shutdown path.

- [ ] **Step 6: Build + full test**

Run: `go build ./... && go test ./internal/llm/ ./internal/server/ ./internal/database/ -v 2>&1 | tail -30`
Expected: build OK; tests PASS (permissions test sees the new codes).

- [ ] **Step 7: Commit**

```bash
git add internal/llm/handlers.go internal/llm/handlers_test.go internal/database/permissions.go internal/server/server.go
git commit -m "feat(llm): REST handlers + routes (admin) + llm RBAC codes + CortexDB wiring"
```

---

## Task 8: Config — SMARTTICKET_SECRET_KEY

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add the field + accessor**

Find the top-level `Config` struct in `internal/config/config.go`. Add:
```go
// SecretKeyRaw is the AES key for credential encryption (hex 64 or base64 32B).
SecretKeyRaw string `mapstructure:"secret_key"`
```
Bind the env var (where other env bindings / `viper.BindEnv` live):
```go
v.BindEnv("secret_key", "SMARTTICKET_SECRET_KEY")
```
That is the only change to config: it exposes the **raw string** `SecretKeyRaw`. Key
decoding + the dev fallback live in `server.go` (Task 7 step 5), which calls
`llm.LoadKey(s.config.SecretKeyRaw)` and falls back to `sha256.Sum256(JWT secret)`.
This keeps `config` free of any `llm` import (no cycle). Add `crypto/sha256` to
`server.go`'s imports.

Confirm the exact field path for the JWT secret used in the fallback (`s.config.JWT.Secret`)
by checking the `Config` struct; adjust if the field is named differently.

- [ ] **Step 2: Build + config test**

Run: `go build ./internal/config/ ./internal/server/ && go test ./internal/config/ -v 2>&1 | tail -15`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go internal/server/server.go
git commit -m "feat(config): SMARTTICKET_SECRET_KEY for credential encryption"
```

---

## Task 9: Web admin UI — LLM providers

**Files:**
- Create: `web/src/features/llm/api.ts`, `web/src/features/llm/types.ts`
- Create: `web/src/pages/llm-providers.tsx`
- Modify: `web/src/App.tsx`, `web/src/components/app-shell.tsx`

- [ ] **Step 1: Types**

`web/src/features/llm/types.ts`:
```ts
export type LLMTaskType = "chat" | "embedding";

export interface LLMProvider {
  id: number;
  name: string;
  provider_type: string;
  api_endpoint: string;
  model: string;
  task_types: string; // JSON-encoded string[] from backend
  dimensions: number;
  max_tokens: number;
  temperature: number;
  is_default: boolean;
  is_enabled: boolean;
  api_key_masked: string;
}

export interface ProviderInput {
  name: string;
  provider_type: string;
  api_endpoint: string;
  api_key?: string;
  model: string;
  task_types: LLMTaskType[];
  dimensions?: number;
  max_tokens?: number;
  temperature?: number;
  is_default: boolean;
  is_enabled: boolean;
}

export interface TestResult {
  chat_ok: boolean;
  embedding_ok: boolean;
  cortex_ok: boolean;
  latency_ms: number;
  error?: string;
}
```

- [ ] **Step 2: API client (follow `web/src/features/rbac/api.ts` pattern)**

`web/src/features/llm/api.ts`:
```ts
import { api, unwrap } from "@/lib/api";
import type { LLMProvider, ProviderInput, TestResult } from "./types";

export const llmApi = {
  list: () => api.get("/llm/providers").then(unwrap<LLMProvider[]>),
  get: (id: number) => api.get(`/llm/providers/${id}`).then(unwrap<LLMProvider>),
  create: (input: ProviderInput) =>
    api.post("/llm/providers", input).then(unwrap<LLMProvider>),
  update: (id: number, input: ProviderInput) =>
    api.put(`/llm/providers/${id}`, input).then(unwrap<LLMProvider>),
  remove: (id: number) => api.delete(`/llm/providers/${id}`).then(unwrap<void>),
  test: (id: number) =>
    api.post(`/llm/providers/${id}/test`, {}).then(unwrap<TestResult>),
};
```
> Verify `unwrap` and the `api` axios instance signatures against `web/src/lib/api.ts`; match whatever the rbac feature uses.

- [ ] **Step 3: Page (list + create/edit dialog + Test) — follow `web/src/pages/rbac.tsx`**

`web/src/pages/llm-providers.tsx`: build a full-width page (`w-full`) using the same query/mutation/toast/dialog primitives as the rbac page. It must let the admin independently configure a **chat** provider (e.g. DeepSeek base_url/model/key) and an **embedding** provider (e.g. Aliyun base_url/model/key) as separate rows, with a per-row **Test** button surfacing `chat_ok/embedding_ok/cortex_ok/latency`. Mask the existing key (show `api_key_masked`); the api_key input is write-only and left blank means "unchanged".

Minimum structure (adapt imports/components to the existing UI kit):
```tsx
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { llmApi } from "@/features/llm/api";
import type { LLMProvider, ProviderInput, TestResult } from "@/features/llm/types";
// ...table, dialog, button, input, switch, badge imports from @/components/ui

export function LLMProvidersPage() {
  const qc = useQueryClient();
  const { data: providers = [] } = useQuery({ queryKey: ["llm-providers"], queryFn: llmApi.list });
  const createM = useMutation({
    mutationFn: (in_: ProviderInput) => llmApi.create(in_),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["llm-providers"] }); toast.success("Provider created"); },
    onError: (e: unknown) => toast.error(String(e)),
  });
  const testM = useMutation({
    mutationFn: (id: number) => llmApi.test(id),
    onSuccess: (r: TestResult) =>
      toast[r.error ? "error" : "success"](
        `chat:${r.chat_ok} embed:${r.embedding_ok} cortex:${r.cortex_ok} (${r.latency_ms}ms)` + (r.error ? ` — ${r.error}` : "")
      ),
  });
  // render full-width table of `providers` with task_types badges + masked key + Edit/Delete/Test,
  // plus a "New provider" dialog backed by createM/updateM. (Mirror rbac.tsx layout.)
  return <div className="w-full">{/* ... */}</div>;
}
```

- [ ] **Step 4: Route + nav**

In `web/src/App.tsx`, add inside the team-guarded routes:
```tsx
<Route path="/llm" element={<TeamOnly><LLMProvidersPage /></TeamOnly>} />
```
(import `LLMProvidersPage` from `@/pages/llm-providers`.)

In `web/src/components/app-shell.tsx`, add to `NAV` (after `rbac`/Access), team-only:
```tsx
{ to: "/llm", label: "AI Providers", icon: Sparkles, team: true },
```
(import `Sparkles` from `lucide-react`.)

- [ ] **Step 5: Build the frontend**

Run: `cd web && pnpm build 2>&1 | tail -5`
Expected: build succeeds, new `index-*.js` emitted.

- [ ] **Step 6: Commit**

```bash
git add web/src/features/llm web/src/pages/llm-providers.tsx web/src/App.tsx web/src/components/app-shell.tsx
git commit -m "feat(web): admin LLM providers page (independent chat/embedding config + test)"
```

---

## Task 10: Integration verification + deploy

**Files:** none (runtime)

- [ ] **Step 1: Full backend test + vet**

Run: `go test ./... 2>&1 | tail -30 && go vet ./... 2>&1 | tail -10`
Expected: all `ok`, no vet errors.

- [ ] **Step 2: Local end-to-end (real keys, NOT committed)**

Run locally with the keys exported in the shell only:
```bash
export SMARTTICKET_SECRET_KEY=$(openssl rand -hex 32)
go run ./cmd/server serve --config configs/config.dev.yaml
```
Create a chat provider (DeepSeek) and an embedding provider (Aliyun `text-embedding-v4`, dim 1024) via the UI, then click Test on each. Expected: `chat_ok` true for DeepSeek, `embedding_ok` + `cortex_ok` true for Aliyun.

- [ ] **Step 3: Deploy backend (see production-deployment memory)**

On `linode-jp`:
```bash
# generate + persist the encryption key once (mode 600, NOT in git)
test -f /opt/smartticket/.secret_key || openssl rand -hex 32 > /opt/smartticket/.secret_key && chmod 600 /opt/smartticket/.secret_key
cd /opt/smartticket/src && git fetch && git reset --hard origin/feat/ai-foundation
docker build -t smartticket:latest .
docker rm -f smartticket
docker run -d --name smartticket --restart unless-stopped -p 127.0.0.1:6533:6533 \
  -v smartticket-data:/app/data \
  -e SMARTTICKET_JWT_SECRET=$(cat /opt/smartticket/.jwt_secret) \
  -e SMARTTICKET_SECRET_KEY=$(cat /opt/smartticket/.secret_key) \
  smartticket:latest serve --config configs/config.prod.yaml
```
(Docker image now builds with CGO_ENABLED=0 — no gcc needed.)

- [ ] **Step 4: Deploy frontend**

```bash
cd web && pnpm build
ssh linode-jp 'rm -rf /opt/smartticket/web/*'
scp -r dist/* linode-jp:/opt/smartticket/web/
ssh linode-jp 'chmod -R a+rX /opt/smartticket/web'
```

- [ ] **Step 5: Verify live + enter real keys via UI**

Log in at https://smartticket.superleo.app/llm, create the two providers with the real (to-be-rotated) keys, Test both. Confirm `/api/v1/health` reports cortex open.

- [ ] **Step 6: Merge the branch**

```bash
git checkout main && git merge --no-ff feat/ai-foundation -m "Merge: AI foundation (LLM providers + CortexDB)"
git push
```

---

## Notes
- Keys (`sk-…` DeepSeek + Aliyun) are entered via the UI post-deploy and stored AES-GCM encrypted; **never committed**. Rotate both after setup (they appeared in plaintext during design).
- `openai-go/v3` and `cortexdb/v2` exact symbol names must be confirmed with `go doc` during Tasks 4–5; the tests pin behavior so SDK-surface drift is caught at compile/test time.
- Out of scope this plan (next slice): knowledge-article ingest into CortexDB, semantic search endpoint/UI, RAG Q&A, AI-assisted replies.
