package knowledgebase

import (
	"context"
	"testing"
)

func TestOpenStoreWithEmbedder(t *testing.T) {
	dir := t.TempDir()
	fn := func(ctx context.Context, texts []string) ([][]float32, error) {
		out := make([][]float32, len(texts))
		for i := range texts {
			out[i] = make([]float32, 3) // dim 3
		}
		return out, nil
	}
	st, err := Open(dir+"/cortex.db", NewProviderEmbedder(fn, 3))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if !st.Healthy() {
		t.Fatal("store should be healthy")
	}
}
