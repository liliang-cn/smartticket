package knowledgebase

import (
	"context"
	"strings"
	"testing"
)

// fakeEmbed returns deterministic fixed-dim vectors derived from the input text,
// so that distinct texts yield distinct (and non-zero) embeddings. CortexDB
// needs consistent, non-zero vectors to index and retrieve knowledge.
func fakeEmbed(dim int) EmbedFunc {
	return func(ctx context.Context, texts []string) ([][]float32, error) {
		out := make([][]float32, len(texts))
		for i, t := range texts {
			v := make([]float32, dim)
			for _, r := range t {
				v[int(r)%dim] += 1.0
			}
			// ensure non-zero even for empty strings
			if len(t) == 0 {
				v[0] = 1.0
			}
			out[i] = v
		}
		return out, nil
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	st, err := Open(dir+"/cortex.db", NewProviderEmbedder(fakeEmbed(16), 16))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestSaveSearchDeleteRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SaveArticle(ctx, 1, "DRBD Configuration",
		"DRBD is configured via drbd.conf with resource sections that define peers, disks and meta-disks.",
		"http://x/1", "public"); err != nil {
		t.Fatalf("SaveArticle 1: %v", err)
	}
	if err := st.SaveArticle(ctx, 2, "LINSTOR Volumes",
		"LINSTOR manages volumes and storage pools across nodes in a cluster.",
		"http://x/2", "public"); err != nil {
		t.Fatalf("SaveArticle 2: %v", err)
	}

	res, err := st.Search(ctx, "how to configure drbd", 5, true)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) < 1 {
		t.Fatalf("expected at least 1 hit, got %d", len(res.Hits))
	}

	foundValid := false
	for _, h := range res.Hits {
		if h.ArticleID == 1 || h.ArticleID == 2 {
			foundValid = true
		}
	}
	if !foundValid {
		t.Fatalf("expected a hit with article id 1 or 2, got hits: %+v", res.Hits)
	}

	t.Logf("context len=%d, hits=%d", len(res.Context), len(res.Hits))

	if err := st.DeleteArticle(ctx, 2); err != nil {
		t.Fatalf("DeleteArticle 2: %v", err)
	}
	// deleting a non-existent article is treated as success
	if err := st.DeleteArticle(ctx, 9999); err != nil {
		t.Fatalf("DeleteArticle non-existent: %v", err)
	}
}

func TestKnowledgeIDRoundTrip(t *testing.T) {
	cases := []uint{0, 1, 42, 9999}
	for _, id := range cases {
		if got := parseArticleID(knowledgeID(id)); got != id {
			t.Errorf("round-trip id=%d -> %q -> %d", id, knowledgeID(id), got)
		}
	}
	if got := parseArticleID("not-an-article"); got != 0 {
		t.Errorf("parseArticleID(garbage) = %d, want 0", got)
	}
	if got := parseArticleID("article-abc"); got != 0 {
		t.Errorf("parseArticleID(non-numeric) = %d, want 0", got)
	}
}

func TestIndexMethodsNilGuard(t *testing.T) {
	ctx := context.Background()
	var s *Store
	if err := s.SaveArticle(ctx, 1, "t", "c", "", "public"); err == nil {
		t.Error("SaveArticle on nil store should error")
	}
	if err := s.DeleteArticle(ctx, 1); err == nil {
		t.Error("DeleteArticle on nil store should error")
	}
	if _, err := s.Search(ctx, "q", 5, true); err == nil {
		t.Error("Search on nil store should error")
	}
}

// TestVisibilityIsolation proves customers (includeInternal=false) only retrieve
// public articles, while team users (includeInternal=true) can retrieve internal
// content too.
func TestVisibilityIsolation(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SaveArticle(ctx, 100, "Public Failover Guide",
		"Public failover guide: how customers trigger a cluster failover via the dashboard.",
		"http://x/100", "public"); err != nil {
		t.Fatalf("SaveArticle public: %v", err)
	}
	if err := st.SaveArticle(ctx, 200, "Internal Failover Runbook",
		"Internal failover runbook: SSH to the node and run the privileged failover script with the ops key.",
		"http://x/200", "internal"); err != nil {
		t.Fatalf("SaveArticle internal: %v", err)
	}

	// Customer search must NEVER return the internal article.
	ext, err := st.Search(ctx, "failover", 10, false)
	if err != nil {
		t.Fatalf("Search external: %v", err)
	}
	for _, h := range ext.Hits {
		if h.ArticleID == 200 {
			t.Fatalf("external search leaked internal article 200: %+v", ext.Hits)
		}
	}
	if strings.Contains(ext.Context, "privileged failover script") {
		t.Fatalf("external RAG context leaked internal content: %q", ext.Context)
	}
	foundPublic := false
	for _, h := range ext.Hits {
		if h.ArticleID == 100 {
			foundPublic = true
		}
	}
	if !foundPublic {
		t.Fatalf("external search should return public article 100, got: %+v", ext.Hits)
	}

	// Team search can return both.
	team, err := st.Search(ctx, "failover", 10, true)
	if err != nil {
		t.Fatalf("Search team: %v", err)
	}
	foundInternal := false
	for _, h := range team.Hits {
		if h.ArticleID == 200 {
			foundInternal = true
		}
	}
	if !foundInternal {
		t.Fatalf("team search should be able to return internal article 200, got: %+v", team.Hits)
	}
}

// TestVisibilityChangeMovesCollections verifies that re-saving an article with a
// new visibility moves it between collections so the old copy no longer leaks.
func TestVisibilityChangeMovesCollections(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SaveArticle(ctx, 300, "Migration Notes",
		"Migration notes describe the steps to migrate storage pools between cluster nodes safely.",
		"http://x/300", "public"); err != nil {
		t.Fatalf("SaveArticle public: %v", err)
	}
	// Move it to internal.
	if err := st.SaveArticle(ctx, 300, "Migration Notes",
		"Migration notes describe the steps to migrate storage pools between cluster nodes safely.",
		"http://x/300", "internal"); err != nil {
		t.Fatalf("SaveArticle internal: %v", err)
	}

	// External search must no longer find it.
	ext, err := st.Search(ctx, "migrate storage pools", 10, false)
	if err != nil {
		t.Fatalf("Search external: %v", err)
	}
	for _, h := range ext.Hits {
		if h.ArticleID == 300 {
			t.Fatalf("article 300 still visible externally after move to internal: %+v", ext.Hits)
		}
	}
}
