package llm

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newTestService(t *testing.T) *Service {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
}
