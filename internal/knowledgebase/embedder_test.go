package knowledgebase

import (
	"context"
	"testing"
)

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
