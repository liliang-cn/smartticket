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
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
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
	want := []int{10, 10, 3}
	if len(batchSizes) != 3 || batchSizes[0] != want[0] || batchSizes[2] != want[2] {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
	}
}
