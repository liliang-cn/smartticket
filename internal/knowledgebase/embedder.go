// Package knowledgebase integrates the CortexDB vector/knowledge store, with an
// embedder backed by SmartTicket's configured LLM embedding provider.
package knowledgebase

import "context"

// EmbedFunc embeds a batch of texts (typically internal/llm's Embed bound to the
// resolved embedding provider).
type EmbedFunc func(ctx context.Context, texts []string) ([][]float32, error)

// ProviderEmbedder adapts an EmbedFunc to CortexDB's Embedder interface.
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
