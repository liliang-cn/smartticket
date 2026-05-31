package knowledgebase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

// collection is the CortexDB collection that holds SmartTicket knowledge articles.
const collection = "knowledge"

// articleIDPrefix namespaces SmartTicket article IDs inside CortexDB.
const articleIDPrefix = "article-"

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
func (s *Store) SaveArticle(ctx context.Context, id uint, title, content, sourceURL string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store not available")
	}
	_, err := s.db.SaveKnowledge(ctx, cortexdb.KnowledgeSaveRequest{
		KnowledgeID: knowledgeID(id),
		Title:       title,
		Content:     content,
		SourceURL:   sourceURL,
		Collection:  collection,
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

// DeleteArticle removes an article from the knowledge index. A missing article
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
// plus packed RAG context.
func (s *Store) Search(ctx context.Context, query string, topK int) (*SearchResult, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store not available")
	}
	if topK <= 0 {
		topK = 5
	}
	resp, err := s.db.SearchKnowledge(ctx, cortexdb.KnowledgeSearchRequest{
		Query:           query,
		Collection:      collection,
		TopK:            topK,
		MaxContextChars: 6000,
	})
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}
	out := &SearchResult{
		Hits:    make([]SearchHit, 0, len(resp.Results)),
		Context: resp.Context,
	}
	for _, h := range resp.Results {
		out.Hits = append(out.Hits, SearchHit{
			ArticleID: parseArticleID(h.KnowledgeID),
			Title:     h.Title,
			Snippet:   h.Snippet,
			Score:     h.Score,
			SourceURL: h.SourceURL,
		})
	}
	return out, nil
}

// isNotFound reports whether err indicates the knowledge item did not exist.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no rows")
}
