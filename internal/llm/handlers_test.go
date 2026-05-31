package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newTestHandlers(t *testing.T) *Handlers {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&models.LLMProvider{})
	cipher, _ := NewCipher(make([]byte, 32))
	return NewHandlers(NewService(db, cipher))
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
