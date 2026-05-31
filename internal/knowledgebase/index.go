package knowledgebase

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

// collectionPublic holds externally visible (public) knowledge articles; both
// customers and team members may retrieve from it. collectionInternal holds
// team-only (internal/private) articles that customers must never see.
const (
	collectionPublic   = "kb_public"
	collectionInternal = "kb_internal"
)

// maxContextChars bounds the combined RAG context returned by Search.
const maxContextChars = 6000

// articleIDPrefix namespaces SmartTicket article IDs inside CortexDB.
const articleIDPrefix = "article-"

// collectionFor maps an article visibility to its CortexDB collection.
// internal and private articles are team-only; everything else is public.
func collectionFor(visibility string) string {
	switch visibility {
	case "internal", "private":
		return collectionInternal
	default:
		return collectionPublic
	}
}

// SearchHit is a knowledge search result mapped to a SmartTicket article.
type SearchHit struct {
	ArticleID uint    `json:"article_id"`
	Title     string  `json:"title"`
	Snippet   string  `json:"snippet"`
	Score     float64 `json:"score"`
	SourceURL string  `json:"source_url,omitempty"`
}

// SearchResult bundles ranked hits and the packed RAG context.
type SearchResult struct {
	Hits    []SearchHit `json:"hits"`
	Context string      `json:"context"`
}

// knowledgeID maps a SmartTicket article ID to a CortexDB knowledge ID.
func knowledgeID(id uint) string {
	return fmt.Sprintf("%s%d", articleIDPrefix, id)
}

// parseArticleID extracts the SmartTicket article ID from a CortexDB knowledge
// ID. It returns 0 if kid is not a recognizable article knowledge ID.
func parseArticleID(kid string) uint {
	rest, ok := strings.CutPrefix(kid, articleIDPrefix)
	if !ok {
		return 0
	}
	n, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// SaveArticle indexes (or replaces) a knowledge article for semantic search.
// The article is stored in the collection matching its visibility. CortexDB's
// DeleteKnowledge scopes globally by knowledge ID (not by collection), so to
// handle a visibility change we delete any existing copy first (best-effort,
// not-found is fine), then save into the target collection. This guarantees an
// article exists in exactly one collection at a time.
func (s *Store) SaveArticle(ctx context.Context, id uint, title, content, sourceURL, visibility string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store not available")
	}
	target := collectionFor(visibility)
	// Only clear a prior copy when the article currently lives in a DIFFERENT
	// collection (a public<->internal move). DeleteKnowledge is global-by-ID.
	// Skipping the delete in the common case (new article, or re-save in the
	// same collection where SaveKnowledge upserts) avoids a delete+save
	// schema-init race on a fresh DB and reduces write-lock contention.
	if rec, gerr := s.db.GetKnowledge(ctx, cortexdb.KnowledgeGetRequest{
		KnowledgeID: knowledgeID(id),
	}); gerr == nil && rec != nil && rec.Knowledge.Collection != "" && rec.Knowledge.Collection != target {
		if _, err := s.db.DeleteKnowledge(ctx, cortexdb.KnowledgeDeleteRequest{
			KnowledgeID: knowledgeID(id),
		}); err != nil && !isNotFound(err) {
			return fmt.Errorf("clear prior knowledge article %d: %w", id, err)
		}
	}
	_, err := s.db.SaveKnowledge(ctx, cortexdb.KnowledgeSaveRequest{
		KnowledgeID: knowledgeID(id),
		Title:       title,
		Content:     content,
		SourceURL:   sourceURL,
		Collection:  collectionFor(visibility),
		// Richer chunks for technical docs (chunk size is in words). The
		// sliding-window chunker (cortexdb v2.20.3+) keeps these coherent;
		// the overlap preserves continuity across chunk boundaries.
		ChunkSize:    220,
		ChunkOverlap: 40,
	})
	if err != nil {
		return fmt.Errorf("save knowledge article %d: %w", id, err)
	}
	return nil
}

// DeleteArticle removes an article from the knowledge index. DeleteKnowledge is
// global-by-ID, so a single delete covers both collections. A missing article
// is treated as success.
func (s *Store) DeleteArticle(ctx context.Context, id uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store not available")
	}
	_, err := s.db.DeleteKnowledge(ctx, cortexdb.KnowledgeDeleteRequest{
		KnowledgeID: knowledgeID(id),
	})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete knowledge article %d: %w", id, err)
	}
	return nil
}

// Search runs a semantic search over indexed articles and returns ranked hits
// plus packed RAG context. It always searches the public collection; when
// includeInternal is true (team users) it additionally searches the internal
// collection and merges the results. Customers (includeInternal=false) can
// therefore never retrieve internal/private content, including via RAG context.
//
// CortexDB's collection filter is honored on the vector path but NOT on its
// lexical/BM25 fallback path, so we additionally verify each hit's collection
// via GetKnowledge and drop any that do not belong to the searched collection.
// The RAG context is rebuilt from the surviving hits so internal content can
// never leak through the packed context either.
func (s *Store) Search(ctx context.Context, query string, topK int, includeInternal bool) (*SearchResult, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store not available")
	}
	if topK <= 0 {
		topK = 5
	}

	pubHits, err := s.searchCollection(ctx, query, collectionPublic, topK)
	if err != nil {
		return nil, err
	}
	if !includeInternal {
		return s.packResult(pubHits, topK), nil
	}

	intlHits, err := s.searchCollection(ctx, query, collectionInternal, topK)
	if err != nil {
		return nil, err
	}

	// Merge hits from both collections, dedupe by ArticleID keeping the higher
	// score, sort by score desc, and take the top topK (public first).
	merged := mergeHits(pubHits, intlHits, topK)
	return s.packResult(merged, topK), nil
}

// searchCollection runs a single-collection knowledge search and returns only
// the hits that actually belong to that collection (verified via GetKnowledge,
// since the lexical fallback path ignores the collection filter).
func (s *Store) searchCollection(ctx context.Context, query, collection string, topK int) ([]scoredHit, error) {
	resp, err := s.db.SearchKnowledge(ctx, cortexdb.KnowledgeSearchRequest{
		Query:           query,
		Collection:      collection,
		TopK:            topK,
		MaxContextChars: maxContextChars,
		// Disable graph expansion: graph traversal links chunks across
		// collections (graph nodes are not collection-scoped), which would
		// let an internal article surface from a public-collection search.
		DisableGraph: true,
	})
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}

	out := make([]scoredHit, 0, len(resp.Results))
	for _, h := range resp.Results {
		// Verify collection membership: only keep hits genuinely stored in the
		// requested collection. This closes the lexical-path leak.
		rec, gerr := s.db.GetKnowledge(ctx, cortexdb.KnowledgeGetRequest{KnowledgeID: h.KnowledgeID})
		if gerr != nil || rec == nil || rec.Knowledge.Collection != collection {
			continue
		}
		out = append(out, scoredHit{
			hit: SearchHit{
				ArticleID: parseArticleID(h.KnowledgeID),
				Title:     h.Title,
				Snippet:   h.Snippet,
				Score:     h.Score,
				SourceURL: h.SourceURL,
			},
			content: rec.Knowledge.Content,
		})
	}
	return out, nil
}

// scoredHit is an internal hit paired with the source article content used to
// rebuild RAG context from confirmed-in-collection results.
type scoredHit struct {
	hit     SearchHit
	content string
}

// packResult converts internal scored hits into a SearchResult, building the
// RAG context from the (already collection-filtered) hits so the context can
// never contain content the caller is not allowed to see.
func (s *Store) packResult(hits []scoredHit, topK int) *SearchResult {
	if len(hits) > topK {
		hits = hits[:topK]
	}
	out := &SearchResult{Hits: make([]SearchHit, 0, len(hits))}
	var ctx strings.Builder
	for _, h := range hits {
		out.Hits = append(out.Hits, h.hit)
		block := h.content
		if block == "" {
			block = h.hit.Snippet
		}
		if block == "" {
			continue
		}
		entry := h.hit.Title + "\n" + block
		if ctx.Len()+len(entry)+2 > maxContextChars {
			remaining := maxContextChars - ctx.Len()
			if remaining > 0 {
				if ctx.Len() > 0 {
					ctx.WriteString("\n\n")
				}
				ctx.WriteString(entry[:min(remaining, len(entry))])
			}
			break
		}
		if ctx.Len() > 0 {
			ctx.WriteString("\n\n")
		}
		ctx.WriteString(entry)
	}
	out.Context = ctx.String()
	return out
}

// mergeHits combines two scored-hit slices, deduping by ArticleID (keeping the
// higher score), sorting by score descending, and truncating to topK.
func mergeHits(a, b []scoredHit, topK int) []scoredHit {
	byID := make(map[uint]scoredHit, len(a)+len(b))
	order := make([]uint, 0, len(a)+len(b))
	add := func(h scoredHit) {
		if existing, ok := byID[h.hit.ArticleID]; ok {
			if h.hit.Score > existing.hit.Score {
				byID[h.hit.ArticleID] = h
			}
			return
		}
		byID[h.hit.ArticleID] = h
		order = append(order, h.hit.ArticleID)
	}
	for _, h := range a {
		add(h)
	}
	for _, h := range b {
		add(h)
	}
	merged := make([]scoredHit, 0, len(order))
	for _, id := range order {
		merged = append(merged, byID[id])
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].hit.Score > merged[j].hit.Score
	})
	if len(merged) > topK {
		merged = merged[:topK]
	}
	return merged
}

// isNotFound reports whether err indicates the knowledge item did not exist.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no rows")
}
