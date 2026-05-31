package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// Service manages LLM providers and resolves them by task.
type Service struct {
	db     *gorm.DB
	cipher *Cipher
}

// NewService builds the service.
func NewService(db *gorm.DB, cipher *Cipher) *Service {
	return &Service{db: db, cipher: cipher}
}

// CreateProviderInput is the create/update payload (plaintext APIKey).
type CreateProviderInput struct {
	Name         string   `json:"name" binding:"required"`
	ProviderType string   `json:"provider_type" binding:"required"`
	APIEndpoint  string   `json:"api_endpoint" binding:"required"`
	APIKey       string   `json:"api_key"`
	Model        string   `json:"model" binding:"required"`
	TaskTypes    []string `json:"task_types" binding:"required"`
	Dimensions   int      `json:"dimensions"`
	MaxTokens    int      `json:"max_tokens"`
	Temperature  float64  `json:"temperature"`
	IsDefault    bool     `json:"is_default"`
	IsEnabled    bool     `json:"is_enabled"`
}

// Create persists a new provider, encrypting the API key at rest.
func (s *Service) Create(in CreateProviderInput) (*models.LLMProvider, error) {
	enc := ""
	if in.APIKey != "" {
		var err error
		if enc, err = s.cipher.Encrypt(in.APIKey); err != nil {
			return nil, err
		}
	}
	tt, _ := json.Marshal(in.TaskTypes)
	dim := in.Dimensions
	if dim == 0 {
		dim = 1024
	}
	p := &models.LLMProvider{
		Name: in.Name, ProviderType: in.ProviderType, APIEndpoint: in.APIEndpoint,
		APIKey: enc, Model: in.Model, TaskTypes: string(tt), Dimensions: dim,
		MaxTokens: in.MaxTokens, Temperature: in.Temperature,
		IsDefault: in.IsDefault, IsEnabled: in.IsEnabled,
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// Update applies fields; APIKey is only re-encrypted when non-empty (blank = keep existing).
func (s *Service) Update(id uint, in CreateProviderInput) (*models.LLMProvider, error) {
	var p models.LLMProvider
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	p.Name, p.ProviderType, p.APIEndpoint = in.Name, in.ProviderType, in.APIEndpoint
	p.Model = in.Model
	tt, _ := json.Marshal(in.TaskTypes)
	p.TaskTypes = string(tt)
	if in.Dimensions > 0 {
		p.Dimensions = in.Dimensions
	}
	p.MaxTokens, p.Temperature = in.MaxTokens, in.Temperature
	p.IsDefault, p.IsEnabled = in.IsDefault, in.IsEnabled
	if in.APIKey != "" {
		enc, err := s.cipher.Encrypt(in.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKey = enc
	}
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// List returns all providers ordered by ID.
func (s *Service) List() ([]models.LLMProvider, error) {
	var ps []models.LLMProvider
	return ps, s.db.Order("id").Find(&ps).Error
}

// Get returns a single provider by ID.
func (s *Service) Get(id uint) (*models.LLMProvider, error) {
	var p models.LLMProvider
	return &p, s.db.First(&p, id).Error
}

// Delete removes a provider by ID.
func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.LLMProvider{}, id).Error
}

func (s *Service) resolve(task string) (*models.LLMProvider, string, error) {
	var ps []models.LLMProvider
	if err := s.db.Where("is_enabled = ?", true).Order("is_default desc, id").Find(&ps).Error; err != nil {
		return nil, "", err
	}
	for _, p := range ps {
		var tt []string
		_ = json.Unmarshal([]byte(p.TaskTypes), &tt)
		for _, t := range tt {
			if t == task {
				key := ""
				if p.APIKey != "" {
					dec, err := s.cipher.Decrypt(p.APIKey)
					if err != nil {
						return nil, "", err
					}
					key = dec
				}
				pp := p
				return &pp, key, nil
			}
		}
	}
	return nil, "", fmt.Errorf("no enabled provider configured for task %q", task)
}

// ResolveChat returns the chat provider and its decrypted key.
func (s *Service) ResolveChat() (*models.LLMProvider, string, error) { return s.resolve("chat") }

// ResolveEmbedding returns the embedding provider and its decrypted key.
func (s *Service) ResolveEmbedding() (*models.LLMProvider, string, error) {
	return s.resolve("embedding")
}

// TestResult reports the outcome of a provider self-test.
type TestResult struct {
	ChatOK      bool   `json:"chat_ok"`
	EmbeddingOK bool   `json:"embedding_ok"`
	CortexOK    bool   `json:"cortex_ok"`
	LatencyMS   int64  `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
}

// TestProvider exercises a SPECIFIC provider (by id) for the task types it
// declares. cortexProbe, if non-nil, runs an embed->store->recall round-trip
// with the produced vector and sets CortexOK. Returns an error only when the
// provider cannot be loaded.
func (s *Service) TestProvider(ctx context.Context, id uint, cortexProbe func(ctx context.Context, vec []float32) error) (TestResult, error) {
	var p models.LLMProvider
	if err := s.db.First(&p, id).Error; err != nil {
		return TestResult{}, err
	}

	key := ""
	if p.APIKey != "" {
		dec, err := s.cipher.Decrypt(p.APIKey)
		if err != nil {
			return TestResult{}, err
		}
		key = dec
	}

	var tasks []string
	_ = json.Unmarshal([]byte(p.TaskTypes), &tasks)
	has := func(task string) bool {
		for _, t := range tasks {
			if t == task {
				return true
			}
		}
		return false
	}

	start := time.Now()
	res := TestResult{}
	client := NewClient(p.APIEndpoint, key)

	if has("chat") {
		if _, err := client.Chat(ctx, p.Model, []ChatMessage{{Role: "user", Content: "ping"}}); err == nil {
			res.ChatOK = true
		} else if res.Error == "" {
			res.Error = "chat: " + err.Error()
		}
	}

	if has("embedding") {
		vecs, err := client.Embed(ctx, p.Model, p.Dimensions, []string{"hello"})
		if err == nil && len(vecs) == 1 {
			res.EmbeddingOK = true
			if cortexProbe != nil {
				if err := cortexProbe(ctx, vecs[0]); err == nil {
					res.CortexOK = true
				} else if res.Error == "" {
					res.Error = "cortex: " + err.Error()
				}
			}
		} else if res.Error == "" {
			res.Error = "embedding: " + embErr(err)
		}
	}

	res.LatencyMS = time.Since(start).Milliseconds()
	return res, nil
}

func embErr(err error) string {
	if err == nil {
		return "no vectors returned"
	}
	return err.Error()
}

// MaskKey returns a display-safe form of a plaintext key (never store output).
func MaskKey(plain string) string {
	if plain == "" {
		return ""
	}
	if len(plain) <= 6 {
		return "****"
	}
	return plain[:3] + "…" + plain[len(plain)-3:]
}
