package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/knowledgebase"
	"github.com/company/smartticket/internal/llm"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// fakeEmbed mirrors the deterministic embedder used in the knowledgebase tests:
// distinct texts map to distinct, non-zero, fixed-dim vectors.
func fakeEmbed(dim int) knowledgebase.EmbedFunc {
	return func(ctx context.Context, texts []string) ([][]float32, error) {
		out := make([][]float32, len(texts))
		for i, t := range texts {
			v := make([]float32, dim)
			for _, r := range t {
				v[int(r)%dim] += 1.0
			}
			if len(t) == 0 {
				v[0] = 1.0
			}
			out[i] = v
		}
		return out, nil
	}
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.KnowledgeArticle{}, &models.LLMProvider{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestStore(t *testing.T) *knowledgebase.Store {
	t.Helper()
	st, err := knowledgebase.Open(t.TempDir()+"/cortex.db",
		knowledgebase.NewProviderEmbedder(fakeEmbed(16), 16))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func seedArticle(t *testing.T, db *gorm.DB, title, content string) {
	t.Helper()
	if err := db.Create(&models.KnowledgeArticle{
		Title: title, Slug: generateSlug(title), Content: content, Status: "published", Version: 1,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}
}

func TestSearchHappyPath(t *testing.T) {
	db := newTestDB(t)
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SaveArticle(ctx, 1, "DRBD Configuration",
		"DRBD is configured via drbd.conf with resource sections.", ""); err != nil {
		t.Fatalf("SaveArticle: %v", err)
	}

	svc := NewService(db, store, nil)
	hits, err := svc.Search(ctx, "how to configure drbd", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected at least one hit")
	}
}

func TestSearchNilStoreReturns503(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, nil, nil) // no store -> not AI ready
	_, err := svc.Search(context.Background(), "anything", 5)
	if err == nil {
		t.Fatal("expected error from nil store")
	}
	if got := errorStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", got)
	}
}

func TestAskHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"role": "assistant", "content": "DRBD is configured via drbd.conf."},
			}},
		})
	}))
	defer srv.Close()

	db := newTestDB(t)
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.SaveArticle(ctx, 1, "DRBD Configuration",
		"DRBD is configured via drbd.conf with resource sections.", ""); err != nil {
		t.Fatalf("SaveArticle: %v", err)
	}

	cipher, _ := llm.NewCipher(make([]byte, 32))
	llmSvc := llm.NewService(db, cipher)
	if _, err := llmSvc.Create(llm.CreateProviderInput{
		Name: "chat", ProviderType: "openai-compatible", APIEndpoint: srv.URL,
		APIKey: "sk-chat", Model: "x", TaskTypes: []string{"chat"}, IsEnabled: true,
	}); err != nil {
		t.Fatalf("create chat provider: %v", err)
	}

	svc := NewService(db, store, llmSvc)
	res, err := svc.Ask(ctx, "how do I configure drbd?", 5)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if res.Answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected citations")
	}
}

func TestAskNoHitsFallback(t *testing.T) {
	db := newTestDB(t)
	store := newTestStore(t)
	cipher, _ := llm.NewCipher(make([]byte, 32))
	llmSvc := llm.NewService(db, cipher)
	// chat provider exists but should not be called when there are no hits
	if _, err := llmSvc.Create(llm.CreateProviderInput{
		Name: "chat", ProviderType: "openai-compatible", APIEndpoint: "http://127.0.0.1:1",
		APIKey: "sk-chat", Model: "x", TaskTypes: []string{"chat"}, IsEnabled: true,
	}); err != nil {
		t.Fatalf("create chat provider: %v", err)
	}

	svc := NewService(db, store, llmSvc)
	res, err := svc.Ask(context.Background(), "an unindexed question", 5)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if res.Answer != askFallback {
		t.Fatalf("expected fallback answer, got %q", res.Answer)
	}
	if len(res.Citations) != 0 {
		t.Fatalf("expected no citations, got %d", len(res.Citations))
	}
}

func TestReindexCountsArticles(t *testing.T) {
	db := newTestDB(t)
	store := newTestStore(t)
	seedArticle(t, db, "First Article", "Some content about networking and storage.")
	seedArticle(t, db, "Second Article", "More content about clusters and replication.")

	svc := NewService(db, store, nil)
	indexed, failed, err := svc.Reindex(context.Background())
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if indexed != 2 || failed != 0 {
		t.Fatalf("reindex counts: indexed=%d failed=%d, want 2/0", indexed, failed)
	}
}

// errorStatus extracts the HTTP status from an AppError, returning 0 otherwise.
func errorStatus(err error) int {
	if ae, ok := err.(*apperrors.AppError); ok {
		return ae.HTTPStatus
	}
	return 0
}
