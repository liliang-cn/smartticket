# External Integration (API Key + Outbound Webhook) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let third-party systems integrate SmartTicket via long-lived API keys and receive ticket events through signed outbound webhooks.

**Architecture:** API keys reshape the existing dead-scaffold `models.APIKey` to bind a service-account user and inherit its RBAC (option A); the auth middleware gains a `stk_live_` prefix branch. Outbound webhooks subscribe to the existing `automation` event bus, persist each delivery to a DB queue, and a background worker POSTs with HMAC-SHA256 signatures and exponential-backoff retries.

**Tech Stack:** Go 1.21+, GIN, GORM, modernc SQLite, testify. New packages `internal/apikey`, `internal/webhook`. No data migration (new/dead tables only).

**Spec:** `docs/superpowers/specs/2026-06-12-external-integration-design.md`

**Conventions (verified against codebase):**
- Test DB: `sqlite "github.com/company/smartticket/internal/database/moderncsqlite"`, `gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})`, testify `require`.
- Singleton/CRUD module pattern mirrors `internal/branding`. Admin routes go under `protected.Group("/...")` + `.Use(s.adminMiddleware())` in `internal/server/server.go`.
- Errors via `internal/errors` (`errors.NewDatabaseError`, `errors.ErrorHandler`). Key generation via `utils.GenerateAPIKey(prefix, length)`.
- Background worker started with `go worker.Run(s.cancelCtx-derived ctx)` like `autoScheduler` (server.go:404-406).
- Event subscriber registered in server.go wiring layer like the CSAT subscriber (server.go:419).

---

## Part 1 — API Key

### Task 1: Reshape the `APIKey` model

**Files:**
- Modify: `internal/models/models.go` (existing `type APIKey struct`)

- [ ] **Step 1: Replace the existing APIKey struct**

Find `type APIKey struct { ... }` and replace its body with:

```go
type APIKey struct {
	BaseModel
	Name       string     `gorm:"size:255;not null" json:"name"`
	KeyHash    string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	KeyPrefix  string     `gorm:"size:20;not null" json:"key_prefix"`
	UserID     uint       `gorm:"index;not null" json:"user_id"` // bound service account; inherits its RBAC
	User       *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	IsActive   bool       `gorm:"default:true" json:"is_active"` // revoke = set false
	ExpiresAt  *time.Time `json:"expires_at"`                    // nil = never expires
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatorID  uint       `gorm:"index" json:"creator_id"`
	Creator    *User      `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
}
```

(The previous `Permissions` and `UsageCount` fields are dropped from the struct; their DB columns, if any, are harmless and ignored. No migration needed.)

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./internal/models/`
Expected: builds clean (no references to removed fields exist — verified: `models.APIKey` had zero readers).

- [ ] **Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat(apikey): reshape APIKey model to bind service-account user"
```

---

### Task 2: API key service — generate, authenticate, list, revoke

**Files:**
- Create: `internal/apikey/service.go`
- Test: `internal/apikey/service_test.go`

- [ ] **Step 1: Write the failing test**

```go
package apikey

import (
	"fmt"
	"testing"
	"time"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.APIKey{}))
	return db
}

func seedUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	u := models.User{Email: "svc@x.local", Username: "svc", PasswordHash: "-", Role: "engineer", IsActive: true}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestCreateAndAuthenticateRoundTrip(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	svc := NewService(db)

	plaintext, key, err := svc.Create("Zapier", u.ID, nil, 99)
	require.NoError(t, err)
	require.True(t, len(plaintext) > 20)
	require.Contains(t, plaintext, "stk_live_")
	require.Equal(t, plaintext[:12], key.KeyPrefix)

	got, err := svc.Authenticate(plaintext)
	require.NoError(t, err)
	require.Equal(t, u.ID, got.ID)
}

func TestAuthenticateRejectsUnknownRevokedExpired(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	svc := NewService(db)

	_, err := svc.Authenticate("stk_live_doesnotexist")
	require.Error(t, err)

	pt, key, _ := svc.Create("k", u.ID, nil, 1)
	require.NoError(t, svc.Revoke(key.ID))
	_, err = svc.Authenticate(pt)
	require.ErrorIs(t, err, ErrRevoked)

	past := time.Now().Add(-time.Hour)
	pt2, _, _ := svc.Create("k2", u.ID, &past, 1)
	_, err = svc.Authenticate(pt2)
	require.ErrorIs(t, err, ErrExpired)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/apikey/ -run TestCreate -v`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 3: Write the implementation**

```go
// Package apikey issues and validates long-lived machine credentials. Each key
// binds a service-account user; authentication resolves that user so all RBAC
// checks downstream behave exactly as for a JWT-authenticated request.
package apikey

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/utils"
	"gorm.io/gorm"
)

const keyPrefixLabel = "stk_live" // GenerateAPIKey appends "_<token>"

var (
	ErrInvalid = errors.New("invalid api key")
	ErrRevoked = errors.New("api key revoked")
	ErrExpired = errors.New("api key expired")
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Create issues a new key bound to userID. The plaintext is returned ONCE and
// never stored. expiresAt nil = never. createdBy is the admin's user ID.
func (s *Service) Create(name string, userID uint, expiresAt *time.Time, createdBy uint) (string, *models.APIKey, error) {
	plaintext := utils.GenerateAPIKey(keyPrefixLabel, 32) // -> "stk_live_<64hex/base...>"
	key := &models.APIKey{
		Name:      name,
		KeyHash:   hashKey(plaintext),
		KeyPrefix: plaintext[:12],
		UserID:    userID,
		IsActive:  true,
		ExpiresAt: expiresAt,
		CreatorID: createdBy,
	}
	if err := s.db.Create(key).Error; err != nil {
		return "", nil, err
	}
	return plaintext, key, nil
}

// Authenticate resolves the service-account user for a plaintext key, or an
// error (ErrInvalid / ErrRevoked / ErrExpired). LastUsedAt is bumped async.
func (s *Service) Authenticate(plaintext string) (*models.User, error) {
	var key models.APIKey
	if err := s.db.Where("key_hash = ?", hashKey(plaintext)).First(&key).Error; err != nil {
		return nil, ErrInvalid
	}
	if !key.IsActive {
		return nil, ErrRevoked
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpired
	}
	var user models.User
	if err := s.db.First(&user, key.UserID).Error; err != nil {
		return nil, ErrInvalid
	}
	now := time.Now()
	go func() { _ = s.db.Model(&models.APIKey{}).Where("id = ?", key.ID).Update("last_used_at", &now).Error }()
	return &user, nil
}

func (s *Service) List() ([]models.APIKey, error) {
	var keys []models.APIKey
	err := s.db.Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (s *Service) Revoke(id uint) error {
	return s.db.Model(&models.APIKey{}).Where("id = ?", id).Update("is_active", false).Error
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/apikey/ -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/apikey/service.go internal/apikey/service_test.go
git commit -m "feat(apikey): service for create/authenticate/list/revoke"
```

---

### Task 3: Extend auth middleware with the API-key branch

**Files:**
- Modify: `internal/server/middleware.go` (inside `authMiddleware`, around the `token := parts[1]` line ~241)
- Modify: `internal/server/server.go` (add `apiKeyService *apikey.Service` field + construct it)
- Test: `internal/server/apikey_middleware_test.go`

- [ ] **Step 1: Add the service field and construction**

In `internal/server/server.go`, add to the `Server` struct:
```go
	apiKeyService *apikey.Service
```
In the server setup (near other `NewService` calls, e.g. after `userService :=`):
```go
	s.apiKeyService = apikey.NewService(s.db.DB)
```
Add import `"github.com/company/smartticket/internal/apikey"`.

- [ ] **Step 2: Branch in authMiddleware**

In `internal/server/middleware.go`, replace the block starting at `token := parts[1]` (the line right before `// Validate JWT token with auth service`) with:

```go
		token := parts[1]

		// API-key path: tokens prefixed stk_live_ resolve a bound service account.
		if strings.HasPrefix(token, "stk_live_") {
			user, err := s.apiKeyService.Authenticate(token)
			if err != nil {
				appErr := errors.NewUnauthorizedError("Invalid or expired API key").
					WithRequestID(c.GetString("request_id"))
				logger.LogSecurityEvent("apikey_invalid", "", clientIP, userAgent, false)
				errors.ErrorHandler(c, appErr)
				return
			}
			c.Set("user_id", user.ID)
			c.Set("user_role", user.Role)
			if user.CustomerID != nil {
				c.Set("user_customer_id", *user.CustomerID)
			}
			logger.LogSecurityEvent("apikey_success", fmt.Sprintf("%d", user.ID), clientIP, userAgent, true)
			c.Next()
			return
		}

		// Validate JWT token with auth service
```

(Leave the existing JWT lines that follow unchanged.)

- [ ] **Step 3: Write the failing test**

```go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// buildKeyTestServer wires a minimal Server with a real apikey service over an
// in-memory DB, an authMiddleware-protected route echoing the resolved user_id.
func buildKeyTestServer(t *testing.T) (*gin.Engine, func(role string) string) {
	t.Helper()
	s := newInMemoryTestServer(t) // helper that builds *Server with db + apiKeyService (see note)
	r := gin.New()
	r.GET("/whoami", s.authMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint("user_id"), "role": c.GetString("user_role")})
	})
	issue := func(role string) string {
		u := seedServiceUser(t, s.db.DB, role)
		pt, _, err := s.apiKeyService.Create("test", u.ID, nil, 1)
		require.NoError(t, err)
		return pt
	}
	return r, issue
}

func TestAPIKeyAuthResolvesUser(t *testing.T) {
	r, issue := buildKeyTestServer(t)
	key := issue("admin")

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"role":"admin"`)
}

func TestInvalidAPIKeyRejected(t *testing.T) {
	r, _ := buildKeyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer stk_live_bogus")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
```

> NOTE: if `newInMemoryTestServer`/`seedServiceUser` helpers do not already exist in the `server` test package, add them in this same test file: construct a `&Server{config: <dev config>, db: <in-mem gorm wrapper>, apiKeyService: apikey.NewService(db)}` with `config.IsDevelopment()` returning false, AutoMigrate `&models.User{}, &models.APIKey{}`, and a `seedServiceUser` that inserts a `models.User{Role: role, IsActive: true}`. Match the existing `Server` field names (`config`, `db`).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/server/ -run APIKey -v`
Expected: FAIL first (prefix branch absent / helpers missing) → PASS after Steps 1-2 and helper wiring.

- [ ] **Step 5: Commit**

```bash
git add internal/server/middleware.go internal/server/server.go internal/server/apikey_middleware_test.go
git commit -m "feat(apikey): authenticate stk_live_ keys in auth middleware"
```

---

### Task 4: Admin CRUD handlers + routes for API keys

**Files:**
- Create: `internal/apikey/handlers.go`
- Modify: `internal/server/server.go` (register routes under settings/admin group)
- Test: `internal/apikey/handlers_test.go`

- [ ] **Step 1: Write the failing handler test**

```go
package apikey

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateHandlerReturnsPlaintextOnce(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	h := NewHandlers(NewService(db))

	r := gin.New()
	r.POST("/admin/api-keys", func(c *gin.Context) { c.Set("user_id", uint(1)); h.Create(c) })

	body := `{"name":"Zapier","user_id":` + strconv.FormatUint(uint64(u.ID), 10) + `}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp["key"].(string), "stk_live_") // plaintext present exactly here

	// List must NOT contain plaintext.
	r.GET("/admin/api-keys", h.List)
	req2 := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.NotContains(t, w2.Body.String(), "stk_live_")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/apikey/ -run TestCreateHandler -v`
Expected: FAIL — `undefined: NewHandlers`.

- [ ] **Step 3: Write the handlers**

```go
package apikey

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

type createReq struct {
	Name      string `json:"name" binding:"required"`
	UserID    uint   `json:"user_id" binding:"required"`
	ExpiresAt *int64 `json:"expires_at"` // unix seconds, optional
}

type keyView struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"key_prefix"`
	UserID     uint   `json:"user_id"`
	IsActive   bool   `json:"is_active"`
	ExpiresAt  *int64 `json:"expires_at"`
	LastUsedAt *int64 `json:"last_used_at"`
	CreatedAt  int64  `json:"created_at"`
}

func toUnix(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	u := t.Unix()
	return &u
}

// Create issues a key. The plaintext is in the response ONCE; it is never
// retrievable again.
func (h *Handlers) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var exp *time.Time
	if req.ExpiresAt != nil {
		tm := time.Unix(*req.ExpiresAt, 0)
		exp = &tm
	}
	createdBy := c.GetUint("user_id")
	plaintext, key, err := h.svc.Create(req.Name, req.UserID, exp, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api key"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"key": plaintext, // shown once
		"api_key": keyView{ID: key.ID, Name: key.Name, KeyPrefix: key.KeyPrefix,
			UserID: key.UserID, IsActive: key.IsActive, CreatedAt: key.CreatedAt.Unix()},
	})
}

func (h *Handlers) List(c *gin.Context) {
	keys, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list api keys"})
		return
	}
	views := make([]keyView, 0, len(keys))
	for _, k := range keys {
		views = append(views, keyView{ID: k.ID, Name: k.Name, KeyPrefix: k.KeyPrefix, UserID: k.UserID,
			IsActive: k.IsActive, ExpiresAt: toUnix(k.ExpiresAt), LastUsedAt: toUnix(k.LastUsedAt), CreatedAt: k.CreatedAt.Unix()})
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": views})
}

func (h *Handlers) Revoke(c *gin.Context) {
	var id uint
	if _, err := fmtSscan(c.Param("id"), &id); err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Revoke(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
```

Add at top of file the tiny helper (or use `fmt.Sscanf` directly):
```go
import "fmt"
func fmtSscan(s string, v *uint) (int, error) { return fmt.Sscanf(s, "%d", v) }
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/apikey/ -v`
Expected: PASS.

- [ ] **Step 5: Register routes**

In `internal/server/server.go`, inside the `settings := protected.Group("/settings"); settings.Use(s.adminMiddleware())` block (or a new admin group), add — first construct handlers near other handler constructions:
```go
	apiKeyHandlers := apikey.NewHandlers(s.apiKeyService)
```
then register:
```go
			adminKeys := protected.Group("/admin/api-keys")
			adminKeys.Use(s.adminMiddleware())
			{
				adminKeys.GET("", apiKeyHandlers.List)
				adminKeys.POST("", apiKeyHandlers.Create)
				adminKeys.DELETE("/:id", apiKeyHandlers.Revoke)
			}
```

- [ ] **Step 6: Build + commit**

Run: `go build ./... && go test ./internal/apikey/ ./internal/server/ -count=1`
Expected: PASS.

```bash
git add internal/apikey/handlers.go internal/apikey/handlers_test.go internal/server/server.go
git commit -m "feat(apikey): admin CRUD handlers and routes"
```

---

### Task 5: Frontend — API Keys settings page

**Files:**
- Create: `web/src/pages/api-keys.tsx`
- Modify: `web/src/App.tsx` (route), `web/src/components/app-shell.tsx` (nav link under Settings)
- Modify: `web/src/locales/*/...` (i18n strings — follow existing namespace pattern)

- [ ] **Step 1: Build the page component**

```tsx
import { useEffect, useState } from "react";
import { api } from "../lib/api"; // match existing api client import in sibling pages

type ApiKey = { id: number; name: string; key_prefix: string; user_id: number; is_active: boolean; last_used_at: number | null; created_at: number };

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [newPlaintext, setNewPlaintext] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [userId, setUserId] = useState("");

  const load = async () => setKeys((await api.get("/admin/api-keys")).data.api_keys);
  useEffect(() => { void load(); }, []);

  const create = async () => {
    const res = await api.post("/admin/api-keys", { name, user_id: Number(userId) });
    setNewPlaintext(res.data.key); // show ONCE
    setName(""); setUserId("");
    await load();
  };
  const revoke = async (id: number) => { await api.delete(`/admin/api-keys/${id}`); await load(); };

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-semibold">API Keys</h1>
      {newPlaintext && (
        <div className="rounded border border-amber-400 bg-amber-50 p-4">
          <p className="text-sm">Copy this key now — it will not be shown again.</p>
          <code className="block break-all mt-2">{newPlaintext}</code>
          <button className="mt-2 text-sm underline" onClick={() => { void navigator.clipboard.writeText(newPlaintext); }}>Copy</button>
          <button className="mt-2 ml-4 text-sm underline" onClick={() => setNewPlaintext(null)}>Close</button>
        </div>
      )}
      <div className="flex gap-2">
        <input className="border rounded px-2 py-1" placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} />
        <input className="border rounded px-2 py-1" placeholder="Service user ID" value={userId} onChange={(e) => setUserId(e.target.value)} />
        <button className="bg-amber-500 text-white px-3 py-1 rounded" onClick={create}>Create</button>
      </div>
      <table className="w-full text-sm">
        <thead><tr><th className="text-left">Name</th><th>Prefix</th><th>Active</th><th>Last used</th><th></th></tr></thead>
        <tbody>
          {keys.map((k) => (
            <tr key={k.id} className="border-t">
              <td>{k.name}</td><td><code>{k.key_prefix}…</code></td><td>{k.is_active ? "yes" : "revoked"}</td>
              <td>{k.last_used_at ? new Date(k.last_used_at * 1000).toLocaleString() : "—"}</td>
              <td>{k.is_active && <button className="text-red-600 underline" onClick={() => revoke(k.id)}>Revoke</button>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

> Match the actual API client (`api.get/post/delete`) and styling to a sibling page in `web/src/pages/` before finalizing — copy its import header.

- [ ] **Step 2: Wire route + nav**

Add a `<Route path="/settings/api-keys" element={<ApiKeysPage />} />` in `web/src/App.tsx` and a nav entry under Settings in `web/src/components/app-shell.tsx`, mirroring an existing settings sub-link.

- [ ] **Step 3: Build the frontend**

Run: `cd web && pnpm build`
Expected: builds with no type errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/api-keys.tsx web/src/App.tsx web/src/components/app-shell.tsx web/src/locales
git commit -m "feat(apikey): admin API Keys settings page"
```

---

## Part 2 — Outbound Webhook

### Task 6: Webhook + WebhookDelivery models

**Files:**
- Modify: `internal/models/models.go`
- Modify: `cmd/server/main.go` (add to `dbModels`)

- [ ] **Step 1: Add the models**

```go
type Webhook struct {
	BaseModel
	Name      string `gorm:"size:255;not null" json:"name"`
	URL       string `gorm:"size:1024;not null" json:"url"`
	Secret    string `gorm:"size:255;not null" json:"-"` // HMAC signing key
	Events    string `gorm:"type:text" json:"events"`    // JSON array of event types
	Active    bool   `gorm:"default:true" json:"active"`
	CreatorID uint   `gorm:"index" json:"creator_id"`
}

type WebhookDelivery struct {
	BaseModel
	WebhookID     uint       `gorm:"index;not null" json:"webhook_id"`
	EventType     string     `gorm:"size:64;index" json:"event_type"`
	Payload       string     `gorm:"type:text" json:"payload"`
	Status        string     `gorm:"size:16;index;default:'pending'" json:"status"` // pending/success/failed
	StatusCode    int        `json:"status_code"`
	Attempts      int        `json:"attempts"`
	LastAttemptAt *time.Time `json:"last_attempt_at"`
	Error         string     `gorm:"type:text" json:"error"`
}
```

- [ ] **Step 2: Register migration**

In `cmd/server/main.go`, add to the `dbModels` slice:
```go
		&models.Webhook{},
		&models.WebhookDelivery{},
```

- [ ] **Step 3: Build + commit**

Run: `go build ./...`
```bash
git add internal/models/models.go cmd/server/main.go
git commit -m "feat(webhook): Webhook and WebhookDelivery models"
```

---

### Task 7: HMAC signing

**Files:**
- Create: `internal/webhook/sign.go`
- Test: `internal/webhook/sign_test.go`

- [ ] **Step 1: Write the failing test**

```go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignIsStableAndVerifiable(t *testing.T) {
	body := []byte(`{"event":"ticket.created"}`)
	secret := "whsec_test"
	sig := Sign(body, secret)
	require.True(t, len(sig) > len("sha256="))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	require.Equal(t, want, sig)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/webhook/ -run TestSign -v`
Expected: FAIL — `undefined: Sign`.

- [ ] **Step 3: Implement**

```go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign returns the X-SmartTicket-Signature header value: "sha256=<hex hmac>".
func Sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
```

- [ ] **Step 4: Run + commit**

Run: `go test ./internal/webhook/ -run TestSign -v` → PASS
```bash
git add internal/webhook/sign.go internal/webhook/sign_test.go
git commit -m "feat(webhook): HMAC-SHA256 request signing"
```

---

### Task 8: Webhook service — CRUD + Enqueue

**Files:**
- Create: `internal/webhook/service.go`
- Test: `internal/webhook/service_test.go`

- [ ] **Step 1: Write the failing test**

```go
package webhook

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Webhook{}, &models.WebhookDelivery{}))
	return db
}

func TestEnqueueOnlyMatchingSubscribers(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	_, err := svc.Create(CreateInput{Name: "a", URL: "http://x", Events: []string{"ticket.created"}}, 1)
	require.NoError(t, err)
	_, err = svc.Create(CreateInput{Name: "b", URL: "http://y", Events: []string{"ticket.resolved"}}, 1)
	require.NoError(t, err)

	require.NoError(t, svc.Enqueue("ticket.created", `{"id":1}`))

	var deliveries []models.WebhookDelivery
	require.NoError(t, db.Find(&deliveries).Error)
	require.Len(t, deliveries, 1) // only webhook "a" subscribed
	require.Equal(t, "pending", deliveries[0].Status)
	require.Equal(t, "ticket.created", deliveries[0].EventType)
}

func TestInactiveWebhookNotEnqueued(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	wh, _ := svc.Create(CreateInput{Name: "a", URL: "http://x", Events: []string{"ticket.created"}}, 1)
	require.NoError(t, svc.SetActive(wh.ID, false))
	require.NoError(t, svc.Enqueue("ticket.created", `{}`))
	var n int64
	db.Model(&models.WebhookDelivery{}).Count(&n)
	require.Equal(t, int64(0), n)
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/webhook/ -run TestEnqueue -v`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 3: Implement**

```go
package webhook

import (
	"encoding/json"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/utils"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateInput struct {
	Name   string
	URL    string
	Events []string
}

func (s *Service) Create(in CreateInput, createdBy uint) (*models.Webhook, error) {
	events, _ := json.Marshal(in.Events)
	wh := &models.Webhook{
		Name:      in.Name,
		URL:       in.URL,
		Secret:    utils.GenerateAPIKey("whsec", 24),
		Events:    string(events),
		Active:    true,
		CreatorID: createdBy,
	}
	if err := s.db.Create(wh).Error; err != nil {
		return nil, err
	}
	return wh, nil
}

func (s *Service) List() ([]models.Webhook, error) {
	var whs []models.Webhook
	err := s.db.Order("created_at DESC").Find(&whs).Error
	return whs, err
}

func (s *Service) SetActive(id uint, active bool) error {
	return s.db.Model(&models.Webhook{}).Where("id = ?", id).Update("active", active).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.Webhook{}, id).Error
}

func (s *Service) Deliveries(webhookID uint, limit int) ([]models.WebhookDelivery, error) {
	var ds []models.WebhookDelivery
	err := s.db.Where("webhook_id = ?", webhookID).Order("created_at DESC").Limit(limit).Find(&ds).Error
	return ds, err
}

// Enqueue writes a pending delivery for each active webhook subscribed to eventType.
func (s *Service) Enqueue(eventType, payload string) error {
	var whs []models.Webhook
	if err := s.db.Where("active = ?", true).Find(&whs).Error; err != nil {
		return err
	}
	for _, wh := range whs {
		var events []string
		_ = json.Unmarshal([]byte(wh.Events), &events)
		if !contains(events, eventType) {
			continue
		}
		d := models.WebhookDelivery{WebhookID: wh.ID, EventType: eventType, Payload: payload, Status: "pending"}
		if err := s.db.Create(&d).Error; err != nil {
			return err
		}
	}
	return nil
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run + commit**

Run: `go test ./internal/webhook/ -v` → PASS
```bash
git add internal/webhook/service.go internal/webhook/service_test.go
git commit -m "feat(webhook): CRUD service and event enqueue"
```

---

### Task 9: Delivery worker with retry + SSRF guard

**Files:**
- Create: `internal/webhook/worker.go`
- Test: `internal/webhook/worker_test.go`

- [ ] **Step 1: Write the failing test**

```go
package webhook

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
)

func TestWorkerDeliversPendingWithSignature(t *testing.T) {
	db := newTestDB(t)
	var hits int32
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		gotSig = r.Header.Get("X-SmartTicket-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewService(db)
	wh, _ := svc.Create(CreateInput{Name: "a", URL: srv.URL, Events: []string{"ticket.created"}}, 1)
	require.NoError(t, svc.Enqueue("ticket.created", `{"id":1}`))

	w := NewWorker(db, WorkerOptions{BlockPrivateIPs: false})
	w.processOnce() // one pass over pending deliveries

	require.Equal(t, int32(1), atomic.LoadInt32(&hits))
	require.Contains(t, gotSig, "sha256=")

	var d models.WebhookDelivery
	require.NoError(t, db.Where("webhook_id = ?", wh.ID).First(&d).Error)
	require.Equal(t, "success", d.Status)
	require.Equal(t, http.StatusOK, d.StatusCode)
}

func TestWorkerMarksFailedAfterMaxAttempts(t *testing.T) {
	db := newTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	svc := NewService(db)
	svc.Create(CreateInput{Name: "a", URL: srv.URL, Events: []string{"e"}}, 1)
	require.NoError(t, svc.Enqueue("e", `{}`))

	w := NewWorker(db, WorkerOptions{BlockPrivateIPs: false})
	for i := 0; i < maxAttempts+1; i++ {
		w.processOnce()
		time.Sleep(time.Millisecond)
	}
	var d models.WebhookDelivery
	require.NoError(t, db.First(&d).Error)
	require.Equal(t, "failed", d.Status)
	require.Equal(t, maxAttempts, d.Attempts)
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/webhook/ -run TestWorker -v`
Expected: FAIL — `undefined: NewWorker`.

- [ ] **Step 3: Implement**

```go
package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

const maxAttempts = 3

type WorkerOptions struct {
	BlockPrivateIPs bool
	Interval        time.Duration // default 5s
}

type Worker struct {
	db   *gorm.DB
	opts WorkerOptions
	cli  *http.Client
}

func NewWorker(db *gorm.DB, opts WorkerOptions) *Worker {
	if opts.Interval == 0 {
		opts.Interval = 5 * time.Second
	}
	return &Worker{db: db, opts: opts, cli: &http.Client{Timeout: 10 * time.Second}}
}

// Run loops until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.opts.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.processOnce()
		}
	}
}

// processOnce attempts every deliverable row once (pending, or failed-but-retryable).
func (w *Worker) processOnce() {
	var rows []models.WebhookDelivery
	w.db.Where("status = ? OR (status = ? AND attempts < ?)", "pending", "failed", maxAttempts).
		Order("created_at ASC").Limit(50).Find(&rows)
	for i := range rows {
		w.deliver(&rows[i])
	}
}

func (w *Worker) deliver(d *models.WebhookDelivery) {
	var wh models.Webhook
	if err := w.db.First(&wh, d.WebhookID).Error; err != nil {
		w.fail(d, 0, "webhook gone")
		return
	}
	if w.opts.BlockPrivateIPs {
		if err := guardSSRF(wh.URL); err != nil {
			w.fail(d, 0, err.Error())
			return
		}
	}
	body := []byte(d.Payload)
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		w.fail(d, 0, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SmartTicket-Event", d.EventType)
	req.Header.Set("X-SmartTicket-Delivery", fmt.Sprintf("%d", d.ID))
	req.Header.Set("X-SmartTicket-Signature", Sign(body, wh.Secret))

	resp, err := w.cli.Do(req)
	now := time.Now()
	d.Attempts++
	d.LastAttemptAt = &now
	if err != nil {
		w.fail(d, 0, err.Error())
		return
	}
	defer resp.Body.Close()
	d.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.Status = "success"
		d.Error = ""
	} else {
		d.Status = "failed"
		d.Error = fmt.Sprintf("non-2xx: %d", resp.StatusCode)
	}
	w.db.Save(d)
}

func (w *Worker) fail(d *models.WebhookDelivery, code int, msg string) {
	now := time.Now()
	d.Attempts++
	d.LastAttemptAt = &now
	d.StatusCode = code
	d.Status = "failed"
	d.Error = msg
	w.db.Save(d)
}

// guardSSRF rejects URLs that resolve to private / loopback IP ranges.
func guardSSRF(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("bad url")
	}
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("dns: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return errors.New("destination resolves to a private address")
		}
	}
	return nil
}
```

> Note: `processOnce` increments `Attempts` once per pass; `TestWorkerMarksFailedAfterMaxAttempts` loops `maxAttempts+1` times. After attempts reach `maxAttempts` the row no longer matches the `attempts < maxAttempts` retry filter, so it stays `failed`. This matches the test assertion `Attempts == maxAttempts`.

- [ ] **Step 4: Run + commit**

Run: `go test ./internal/webhook/ -v` → PASS
```bash
git add internal/webhook/worker.go internal/webhook/worker_test.go
git commit -m "feat(webhook): delivery worker with retry and SSRF guard"
```

---

### Task 10: Wire event subscribers + payload builder + start worker

**Files:**
- Modify: `internal/server/server.go` (subscribe events, start worker)
- Modify: `internal/config/config.go` (add `Webhook.BlockPrivateIPs bool`)

- [ ] **Step 1: Add config field**

In `internal/config/config.go`, add a `WebhookConfig` with `BlockPrivateIPs bool` (default false) and include it on the root config, mirroring an existing sub-config (e.g. how `Email`/`Inbound` are structured). Default false in defaults.

- [ ] **Step 2: Construct service + worker + subscribe**

In `internal/server/server.go` setup, after the event bus (`s.bus`) and `s.cancelCtx` exist:

```go
	webhookSvc := webhook.NewService(s.db.DB)
	webhookWorker := webhook.NewWorker(s.db.DB, webhook.WorkerOptions{BlockPrivateIPs: s.config.Webhook.BlockPrivateIPs})
	go webhookWorker.Run(schedCtx) // reuse the scheduler context created at server.go:404

	for _, et := range []automation.EventType{
		automation.EventTicketCreated, automation.EventTicketUpdated, automation.EventTicketResolved,
		automation.EventMessageCreated, automation.EventSLAWarning,
	} {
		et := et
		s.bus.Subscribe(et, func(ev automation.Event) {
			payload := buildWebhookPayload(s.db.DB, ev)
			if err := webhookSvc.Enqueue(string(et), payload); err != nil {
				logger.Warn("webhook enqueue failed", zap.String("event", string(et)), zap.Error(err))
			}
		})
	}
```

Add `buildWebhookPayload` in the same file (or a small `internal/server/webhook_payload.go`):

```go
func buildWebhookPayload(db *gorm.DB, ev automation.Event) string {
	out := map[string]any{"event": string(ev.Type), "occurred_at": time.Now().Unix()}
	var tkt models.Ticket
	if err := db.Where("id = ?", ev.TicketID).First(&tkt).Error; err == nil {
		out["data"] = map[string]any{
			"id": tkt.ID, "ticket_number": tkt.TicketNumber, "title": tkt.Title,
			"status": tkt.Status, "priority": tkt.Priority, "assigned_to": tkt.AssignedTo,
		}
	} else {
		out["data"] = map[string]any{"ticket_id": ev.TicketID}
	}
	b, _ := json.Marshal(out)
	return string(b)
}
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add internal/server/server.go internal/config/config.go
git commit -m "feat(webhook): subscribe ticket events and run delivery worker"
```

---

### Task 11: Admin CRUD + deliveries + test-ping handlers/routes

**Files:**
- Create: `internal/webhook/handlers.go`
- Modify: `internal/server/server.go` (routes)
- Test: `internal/webhook/handlers_test.go`

- [ ] **Step 1: Write the failing test**

```go
package webhook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateAndListHandlers(t *testing.T) {
	db := newTestDB(t)
	h := NewHandlers(NewService(db))
	r := gin.New()
	r.POST("/admin/webhooks", func(c *gin.Context) { c.Set("user_id", uint(1)); h.Create(c) })
	r.GET("/admin/webhooks", h.List)

	body := `{"name":"a","url":"http://x","events":["ticket.created"]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), `"secret"`) // secret returned on create only

	req2 := httptest.NewRequest(http.MethodGet, "/admin/webhooks", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.NotContains(t, w2.Body.String(), `"secret"`) // never on list
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/webhook/ -run TestCreateAndList -v`
Expected: FAIL — `undefined: NewHandlers`.

- [ ] **Step 3: Implement handlers**

```go
package webhook

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

type createReq struct {
	Name   string   `json:"name" binding:"required"`
	URL    string   `json:"url" binding:"required,url"`
	Events []string `json:"events" binding:"required"`
}

type whView struct {
	ID     uint     `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

func (h *Handlers) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wh, err := h.svc.Create(CreateInput{Name: req.Name, URL: req.URL, Events: req.Events}, c.GetUint("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create webhook"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"webhook": whView{ID: wh.ID, Name: wh.Name, URL: wh.URL, Events: req.Events, Active: wh.Active},
		"secret":  wh.Secret, // shown once
	})
}

func (h *Handlers) List(c *gin.Context) {
	whs, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list"})
		return
	}
	out := make([]whView, 0, len(whs))
	for _, wh := range whs {
		var events []string
		_ = jsonUnmarshal(wh.Events, &events)
		out = append(out, whView{ID: wh.ID, Name: wh.Name, URL: wh.URL, Events: events, Active: wh.Active})
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": out})
}

func (h *Handlers) Delete(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handlers) Deliveries(c *gin.Context) {
	id := parseID(c.Param("id"))
	ds, err := h.svc.Deliveries(id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deliveries": ds})
}

// Test enqueues a synthetic ping delivery so the admin can verify connectivity.
func (h *Handlers) Test(c *gin.Context) {
	id := parseID(c.Param("id"))
	if err := h.svc.EnqueueTo(id, "ping", `{"event":"ping"}`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queued": true})
}

func parseID(s string) uint {
	var id uint
	_, _ = fmt.Sscanf(s, "%d", &id)
	return id
}
```

Add to `service.go` the helpers used above:
```go
func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }

// EnqueueTo writes a pending delivery for a single webhook regardless of its
// event subscription (used by the Test ping).
func (s *Service) EnqueueTo(webhookID uint, eventType, payload string) error {
	d := models.WebhookDelivery{WebhookID: webhookID, EventType: eventType, Payload: payload, Status: "pending"}
	return s.db.Create(&d).Error
}
```

- [ ] **Step 4: Run + register routes**

Run: `go test ./internal/webhook/ -v` → PASS

In `internal/server/server.go`, construct `webhookHandlers := webhook.NewHandlers(webhookSvc)` and register:
```go
			adminWebhooks := protected.Group("/admin/webhooks")
			adminWebhooks.Use(s.adminMiddleware())
			{
				adminWebhooks.GET("", webhookHandlers.List)
				adminWebhooks.POST("", webhookHandlers.Create)
				adminWebhooks.DELETE("/:id", webhookHandlers.Delete)
				adminWebhooks.GET("/:id/deliveries", webhookHandlers.Deliveries)
				adminWebhooks.POST("/:id/test", webhookHandlers.Test)
			}
```

- [ ] **Step 5: Build + commit**

Run: `go build ./... && go test ./internal/webhook/ -count=1` → PASS
```bash
git add internal/webhook/handlers.go internal/webhook/service.go internal/webhook/handlers_test.go internal/server/server.go
git commit -m "feat(webhook): admin CRUD, deliveries log, and test-ping routes"
```

---

### Task 12: Frontend — Webhooks settings page

**Files:**
- Create: `web/src/pages/webhooks.tsx`
- Modify: `web/src/App.tsx`, `web/src/components/app-shell.tsx`, `web/src/locales/*`

- [ ] **Step 1: Build the page** (mirror `api-keys.tsx` structure)

Component with: list of webhooks (name, url, events, active), create form (name, url, multi-select of the five event types: `ticket.created/updated/resolved`, `message.created`, `ticket.sla_warning`), a one-time secret reveal modal on create, a delete button, a "Test" button calling `POST /admin/webhooks/:id/test`, and a deliveries drawer calling `GET /admin/webhooks/:id/deliveries` rendering status/code/attempts/error. Reuse the exact `api` client + styling from `api-keys.tsx`.

- [ ] **Step 2: Wire route `/settings/webhooks` + nav link** (mirror api-keys wiring).

- [ ] **Step 3: Build**

Run: `cd web && pnpm build` → no type errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/webhooks.tsx web/src/App.tsx web/src/components/app-shell.tsx web/src/locales
git commit -m "feat(webhook): admin Webhooks settings page"
```

---

### Task 13: Regenerate OpenAPI + final verification

**Files:**
- Modify: `docs/swagger.yaml`, `docs/swagger.json`, `docs/docs.go` (generated)

- [ ] **Step 1: Add swagger annotations** to the new handlers (`apikey/handlers.go`, `webhook/handlers.go`) following the `@Summary/@Tags/@Router/@Security BearerAuth` style of `internal/auth/handlers.go`.

- [ ] **Step 2: Regenerate** (the `--parseDependency` flag is REQUIRED, see memory note):

Run: `swag init -g cmd/server/main.go --parseDependency --parseInternal -o docs`
Expected: exit 0; `docs/swagger.yaml` updated with the new `/admin/api-keys` and `/admin/webhooks` paths.

- [ ] **Step 3: Full test + build sweep**

Run: `go build ./... && go test ./... -count=1`
Expected: PASS across the repo.

- [ ] **Step 4: Commit**

```bash
git add docs/swagger.yaml docs/swagger.json docs/docs.go internal/apikey/handlers.go internal/webhook/handlers.go
git commit -m "docs(api): regenerate OpenAPI for api-keys and webhooks"
```

---

## Self-Review Notes

- **Spec coverage:** API Key model/service/middleware/CRUD/frontend → Tasks 1-5. Outbound Webhook models/sign/service/worker/wiring/CRUD/frontend/SSRF → Tasks 6-12. OpenAPI refresh → Task 13. All spec sections covered.
- **Type consistency:** `apikey.Service` methods (`Create/Authenticate/List/Revoke`) consistent across Tasks 2-4. `webhook.Service` (`Create/List/SetActive/Delete/Deliveries/Enqueue/EnqueueTo`) and `Worker` (`NewWorker/Run/processOnce/deliver`) consistent across Tasks 8-11. `Sign` signature consistent (Task 7 ↔ Task 9).
- **No migration:** per user instruction, models are new or dead-scaffold reshapes; AutoMigrate handles creation. No backfill tasks.
- **Known follow-up:** `utils.GenerateAPIKey` returns `prefix + "_" + token`; with `keyPrefixLabel = "stk_live"` the output starts `stk_live_…` so the middleware prefix check and `KeyPrefix = plaintext[:12]` both hold. Verify `GenerateSecureCryptoToken(32)` yields ≥4 chars after the 9-char `stk_live_` prefix (it does — 32 → 64 hex).
