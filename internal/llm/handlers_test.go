package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newTestServiceForHandlers(t *testing.T) *Service {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&models.LLMProvider{})
	cipher, _ := NewCipher(make([]byte, 32))
	return NewService(db, cipher)
}

func newTestHandlers(t *testing.T) *Handlers {
	return NewHandlers(newTestServiceForHandlers(t))
}

func TestCreateAndListMasksKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandlers(t)
	r := gin.New()
	r.POST("/llm/providers", h.Create)
	r.GET("/llm/providers", h.List)

	body := `{"name":"chat","provider_type":"openai-compatible","api_endpoint":"https://api.deepseek.com","api_key":"sk-deepseek","model":"deepseek-chat","task_types":["chat"],"is_enabled":true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/llm/providers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create code %d body %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sk-deepseek") {
		t.Fatal("plaintext key leaked in create response")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/llm/providers", nil)
	r.ServeHTTP(w, req)
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 provider, got %d", len(resp.Data))
	}
	if _, ok := resp.Data[0]["api_key"]; ok {
		t.Fatal("api_key must not be serialized")
	}
	if resp.Data[0]["api_key_masked"] != "********" {
		t.Fatalf("expected masked key marker, got %v", resp.Data[0]["api_key_masked"])
	}
}

func TestGetHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := newTestServiceForHandlers(t)
	p, err := s.Create(CreateProviderInput{
		Name: "g", ProviderType: "openai-compatible", APIEndpoint: "https://a.example",
		APIKey: "sk-a", Model: "m", TaskTypes: []string{"chat"}, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(s)
	r := gin.New()
	r.GET("/llm/providers/:id", h.Get)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/llm/providers/%d", p.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"api_key_masked":"********"`) {
		t.Fatalf("get found unexpected: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/llm/providers/99999", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get missing want 404, got %d", w.Code)
	}
}

func TestTestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer up.Close()

	s := newTestServiceForHandlers(t)
	p, err := s.Create(CreateProviderInput{
		Name: "both", ProviderType: "openai-compatible", APIEndpoint: up.URL,
		APIKey: "sk", Model: "x", TaskTypes: []string{"chat", "embedding"},
		Dimensions: 4, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(s)
	h.SetCortexProbe(func(ctx context.Context, vec []float32) error { return nil })
	r := gin.New()
	r.POST("/llm/providers/:id/test", h.Test)

	// Success: tests the specific provider.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/llm/providers/%d/test", p.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("test ok want 200, got %d body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"chat_ok":true`) ||
		!strings.Contains(w.Body.String(), `"embedding_ok":true`) ||
		!strings.Contains(w.Body.String(), `"cortex_ok":true`) {
		t.Fatalf("unexpected test body: %s", w.Body.String())
	}

	// Invalid id -> 400.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/llm/providers/abc/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("test invalid id want 400, got %d", w.Code)
	}

	// Unknown id -> 404.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/llm/providers/99999/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("test unknown id want 404, got %d", w.Code)
	}
}

func TestUpdateAndDeleteHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := newTestServiceForHandlers(t)
	p, err := s.Create(CreateProviderInput{
		Name: "n", ProviderType: "openai-compatible", APIEndpoint: "https://a.example",
		APIKey: "sk", Model: "m", TaskTypes: []string{"chat"}, IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(s)
	r := gin.New()
	r.PUT("/llm/providers/:id", h.Update)
	r.DELETE("/llm/providers/:id", h.Delete)

	body := `{"name":"n2","provider_type":"openai-compatible","api_endpoint":"https://a.example","model":"m","task_types":["chat"],"is_enabled":true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/llm/providers/%d", p.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"name":"n2"`) {
		t.Fatalf("update unexpected: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/llm/providers/%d", p.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete want 200, got %d", w.Code)
	}
}
