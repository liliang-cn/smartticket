package aiteam

import (
	"context"
	"fmt"

	"github.com/company/smartticket/internal/knowledgebase"
)

// mergeCandidateThreshold is the cosine-similarity score above which two
// tickets are flagged as candidates for merging.
const mergeCandidateThreshold = 0.9

// TicketSearchStore is a narrow interface satisfied by *knowledgebase.Store.
// Using an interface here keeps aiteam free of a hard dependency on the
// concrete store type and makes unit-testing straightforward with a fake.
type TicketSearchStore interface {
	// SearchTickets runs a semantic search over the indexed ticket collection.
	SearchTickets(ctx context.Context, query string, topK int) (*knowledgebase.SearchResult, error)
	// Healthy reports whether the backing store is open and ready.
	Healthy() bool
}

// StoreSimilarSearcher implements SimilarTicketSearcher by delegating to a
// TicketSearchStore (typically a *knowledgebase.Store).
type StoreSimilarSearcher struct {
	store TicketSearchStore
}

// NewStoreSimilarSearcher wraps store as a SimilarTicketSearcher.
// Returns an error when store is nil or not healthy.
func NewStoreSimilarSearcher(store TicketSearchStore) (*StoreSimilarSearcher, error) {
	if store == nil || !store.Healthy() {
		return nil, fmt.Errorf("ticket search store is not available")
	}
	return &StoreSimilarSearcher{store: store}, nil
}

// SearchSimilar queries the ticket index and maps the hits to []SimilarTicket.
// MergeCandidate is set when the hit score exceeds mergeCandidateThreshold (0.9).
// An unavailable store returns an empty slice (never an error) so callers degrade
// gracefully when no ticket index is configured.
func (s *StoreSimilarSearcher) SearchSimilar(ctx context.Context, query string, topK int) ([]SimilarTicket, error) {
	if s == nil || s.store == nil {
		return []SimilarTicket{}, nil
	}
	res, err := s.store.SearchTickets(ctx, query, topK)
	if err != nil {
		// Non-fatal: return empty slice so the Researcher can still respond.
		return []SimilarTicket{}, nil
	}
	out := make([]SimilarTicket, 0, len(res.Hits))
	for _, h := range res.Hits {
		out = append(out, SimilarTicket{
			ID:             h.ArticleID, // ArticleID field carries the ticket ID
			Title:          h.Title,
			Resolution:     h.Snippet, // Snippet field carries the resolution
			MergeCandidate: h.Score > mergeCandidateThreshold,
			Score:          h.Score,
		})
	}
	return out, nil
}
