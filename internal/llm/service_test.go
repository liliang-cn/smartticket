package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newTestService(t *testing.T) *Service {
	// Unique shared-cache name per test: shared across this test's pooled
	// connections, but isolated from every other test so state cannot leak
	// (which previously broke -shuffle=on runs).
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LLMProvider{}); err != nil {
		t.Fatal(err)
	}
	cipher, _ := NewCipher(make([]byte, 32))
	return NewService(db, cipher)
}

func TestCreateEncryptsKeyAndResolves(t *testing.T) {
	s := newTestService(t)
	p, err := s.Create(CreateProviderInput{
		Name: "embed", ProviderType: "openai-compatible",
		APIEndpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:      "sk-secret", Model: "text-embedding-v4",
		TaskTypes: []string{"embedding"}, Dimensions: 1024,
		IsDefault: true, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var row models.LLMProvider
	s.db.First(&row, p.ID)
	if row.APIKey == "sk-secret" || row.APIKey == "" {
		t.Fatalf("api key not encrypted at rest: %q", row.APIKey)
	}
	rp, key, err := s.ResolveEmbedding()
	if err != nil {
		t.Fatal(err)
	}
	if rp.ID != p.ID || key != "sk-secret" {
		t.Fatalf("resolve mismatch: id=%d key=%q", rp.ID, key)
	}
}

func TestResolveChatErrorsWhenNone(t *testing.T) {
	s := newTestService(t)
	if _, _, err := s.ResolveChat(); err == nil {
		t.Fatal("expected error when no chat provider configured")
	}
}

func TestUpdateKeepsKeyWhenBlank(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create(CreateProviderInput{
		Name: "chat", ProviderType: "openai-compatible", APIEndpoint: "https://api.deepseek.com",
		APIKey: "sk-original", Model: "deepseek-chat", TaskTypes: []string{"chat"}, IsEnabled: true,
	})
	// Update with blank APIKey must NOT wipe the stored key.
	if _, err := s.Update(p.ID, CreateProviderInput{
		Name: "chat2", ProviderType: "openai-compatible", APIEndpoint: "https://api.deepseek.com",
		APIKey: "", Model: "deepseek-chat", TaskTypes: []string{"chat"}, IsEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	_, key, err := s.ResolveChat()
	if err != nil || key != "sk-original" {
		t.Fatalf("key after blank update: %q err %v", key, err)
	}
}

func TestMaskedKey(t *testing.T) {
	if got := MaskKey("sk-1234567890abcdef"); got == "sk-1234567890abcdef" || got == "" {
		t.Fatalf("mask failed: %q", got)
	}
	if MaskKey("") != "" {
		t.Fatal("empty stays empty")
	}
	if MaskKey("abc") != "****" {
		t.Fatal("short key must be ****")
	}
}

func TestListGetDelete(t *testing.T) {
	s := newTestService(t)
	p1, err := s.Create(CreateProviderInput{
		Name: "first", ProviderType: "openai-compatible", APIEndpoint: "https://a.example",
		APIKey: "sk-a", Model: "m1", TaskTypes: []string{"chat"}, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := s.Create(CreateProviderInput{
		Name: "second", ProviderType: "openai-compatible", APIEndpoint: "https://b.example",
		APIKey: "sk-b", Model: "m2", TaskTypes: []string{"embedding"}, Dimensions: 8, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 providers, got %d", len(list))
	}
	if list[0].ID != p1.ID || list[1].ID != p2.ID {
		t.Fatalf("list not ordered by id: %d, %d", list[0].ID, list[1].ID)
	}

	got, err := s.Get(p1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "first" || got.Model != "m1" {
		t.Fatalf("Get returned wrong fields: name=%q model=%q", got.Name, got.Model)
	}

	if err := s.Delete(p1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(p1.ID); err == nil {
		t.Fatal("expected error getting deleted provider")
	}
	list, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != p2.ID {
		t.Fatalf("after delete want only p2, got %+v", list)
	}
}

func TestServiceTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/chat/completions":
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{
					"message": map[string]any{"role": "assistant", "content": "pong"},
				}},
			})
		case "/embeddings":
			var body struct {
				Input []string `json:"input"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			data := make([]any, len(body.Input))
			for i := range body.Input {
				data[i] = map[string]any{"embedding": []float32{0.1, 0.2, 0.3, 0.4}, "index": i}
			}
			json.NewEncoder(w).Encode(map[string]any{"data": data})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	setup := func() (*Service, uint) {
		s := newTestService(t)
		// A single provider tagged for BOTH chat and embedding, hitting one
		// httptest server that serves /chat/completions and /embeddings.
		p, err := s.Create(CreateProviderInput{
			Name: "both", ProviderType: "openai-compatible", APIEndpoint: srv.URL,
			APIKey: "sk-both", Model: "x", TaskTypes: []string{"chat", "embedding"},
			Dimensions: 4, IsEnabled: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		return s, p.ID
	}

	// With a cortex probe: chat, embedding, and cortex all succeed.
	t.Run("with_probe", func(t *testing.T) {
		s, id := setup()
		var gotVec []float32
		probe := func(ctx context.Context, vec []float32) error {
			gotVec = vec
			return nil
		}
		res, err := s.TestProvider(context.Background(), id, probe)
		if err != nil {
			t.Fatal(err)
		}
		if !res.ChatOK {
			t.Fatalf("ChatOK false, err=%q", res.Error)
		}
		if !res.EmbeddingOK {
			t.Fatalf("EmbeddingOK false, err=%q", res.Error)
		}
		if !res.CortexOK {
			t.Fatalf("CortexOK false, err=%q", res.Error)
		}
		want := []float32{0.1, 0.2, 0.3, 0.4}
		if !reflect.DeepEqual(gotVec, want) {
			t.Fatalf("probe got vector %v, want %v", gotVec, want)
		}
	})

	// Without a cortex probe: embedding succeeds but cortex is not exercised.
	t.Run("nil_probe", func(t *testing.T) {
		s, id := setup()
		res, err := s.TestProvider(context.Background(), id, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.EmbeddingOK {
			t.Fatalf("EmbeddingOK false, err=%q", res.Error)
		}
		if res.CortexOK {
			t.Fatal("CortexOK must be false with nil probe")
		}
	})

	// Unknown provider id returns an error.
	t.Run("not_found", func(t *testing.T) {
		s, _ := setup()
		if _, err := s.TestProvider(context.Background(), 99999, nil); err == nil {
			t.Fatal("expected error for unknown provider id")
		}
	})
}
