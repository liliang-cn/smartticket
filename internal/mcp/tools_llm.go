package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/llm"
	"github.com/company/smartticket/internal/models"
)

// llmProviderView is the schema-safe MCP view of an LLM provider. It NEVER
// includes the API key (which is stored encrypted and is json:"-" on the model).
type llmProviderView struct {
	ID           uint      `json:"id" jsonschema:"provider ID"`
	Name         string    `json:"name" jsonschema:"display name"`
	ProviderType string    `json:"provider_type" jsonschema:"openai, azure, deepseek, ollama, etc."`
	APIEndpoint  string    `json:"api_endpoint,omitempty" jsonschema:"base URL of the provider API"`
	Model        string    `json:"model,omitempty" jsonschema:"model identifier"`
	Dimensions   int       `json:"dimensions" jsonschema:"embedding output dimension (for embedding tasks)"`
	MaxTokens    int       `json:"max_tokens" jsonschema:"max tokens per request"`
	Temperature  float64   `json:"temperature" jsonschema:"sampling temperature"`
	TaskTypes    []string  `json:"task_types,omitempty" jsonschema:"task types this provider serves: chat, embedding, etc."`
	IsDefault    bool      `json:"is_default" jsonschema:"whether this is the default provider"`
	IsEnabled    bool      `json:"is_enabled" jsonschema:"whether this provider is enabled"`
	HasAPIKey    bool      `json:"has_api_key" jsonschema:"whether an API key is configured (the key itself is never returned)"`
	CreatedAt    time.Time `json:"created_at" jsonschema:"when the provider was created"`
	UpdatedAt    time.Time `json:"updated_at" jsonschema:"when the provider was last updated"`
}

func llmProviderViewFrom(p *models.LLMProvider) llmProviderView {
	if p == nil {
		return llmProviderView{}
	}
	var tasks []string
	if p.TaskTypes != "" {
		_ = json.Unmarshal([]byte(p.TaskTypes), &tasks)
	}
	return llmProviderView{
		ID: p.ID, Name: p.Name, ProviderType: p.ProviderType, APIEndpoint: p.APIEndpoint,
		Model: p.Model, Dimensions: p.Dimensions, MaxTokens: p.MaxTokens, Temperature: p.Temperature,
		TaskTypes: tasks, IsDefault: p.IsDefault, IsEnabled: p.IsEnabled,
		HasAPIKey: p.APIKey != "", CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// registerLLMTools registers the LLM-provider-domain MCP tools.
func registerLLMTools(s *mcp.Server, b Backend) {
	registerTool(s, "llm_list",
		"List configured LLM providers (API keys are never returned).",
		"llm:read",
		func(ctx context.Context, _ struct{}) (llmListOutput, string, error) {
			return llmList(ctx, b)
		})

	registerTool(s, "llm_get",
		"Fetch a single LLM provider by its numeric ID (API key is never returned).",
		"llm:read",
		func(ctx context.Context, in llmIDInput) (llmProviderView, string, error) {
			return llmGet(ctx, b, in)
		})

	registerTool(s, "llm_create",
		"Create an OpenAI-compatible LLM provider. The api_key is encrypted at rest.",
		"llm:write",
		func(ctx context.Context, in llmProviderInput) (llmProviderView, string, error) {
			return llmCreate(ctx, b, in)
		})

	registerTool(s, "llm_update",
		"Update an LLM provider by ID. Leave api_key empty to keep the existing key.",
		"llm:write",
		func(ctx context.Context, in llmUpdateInput) (llmProviderView, string, error) {
			return llmUpdate(ctx, b, in)
		})

	registerTool(s, "llm_delete",
		"Delete an LLM provider by its numeric ID.",
		"llm:write",
		func(ctx context.Context, in llmIDInput) (deleteOutput, string, error) {
			return llmDelete(ctx, b, in)
		})

	registerTool(s, "llm_test",
		"Exercise a provider (chat/embedding round-trip) and report which capabilities work.",
		"llm:read",
		func(ctx context.Context, in llmIDInput) (llmTestOutput, string, error) {
			return llmTest(ctx, b, in)
		})
}

// ---- schemas ----

type llmIDInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the LLM provider"`
}

type llmListOutput struct {
	Providers []llmProviderView `json:"providers,omitempty" jsonschema:"the configured providers"`
	Total     int               `json:"total" jsonschema:"number of providers"`
}

type llmProviderInput struct {
	Name         string   `json:"name" jsonschema:"display name (required)"`
	ProviderType string   `json:"provider_type" jsonschema:"openai, azure, deepseek, ollama, etc. (required)"`
	APIEndpoint  string   `json:"api_endpoint" jsonschema:"base URL of the provider API (required)"`
	APIKey       string   `json:"api_key,omitempty" jsonschema:"API key (stored encrypted; omit if not needed)"`
	Model        string   `json:"model" jsonschema:"model identifier (required)"`
	TaskTypes    []string `json:"task_types" jsonschema:"task types served, e.g. [\"chat\"] or [\"embedding\"] (required)"`
	Dimensions   int      `json:"dimensions,omitempty" jsonschema:"embedding output dimension (default 1024)"`
	MaxTokens    int      `json:"max_tokens,omitempty" jsonschema:"max tokens per request"`
	Temperature  float64  `json:"temperature,omitempty" jsonschema:"sampling temperature"`
	IsDefault    bool     `json:"is_default,omitempty" jsonschema:"mark as the default provider"`
	IsEnabled    bool     `json:"is_enabled,omitempty" jsonschema:"enable this provider"`
}

type llmUpdateInput struct {
	ID           uint     `json:"id" jsonschema:"numeric ID of the provider to update"`
	Name         string   `json:"name" jsonschema:"display name (required)"`
	ProviderType string   `json:"provider_type" jsonschema:"openai, azure, deepseek, ollama, etc. (required)"`
	APIEndpoint  string   `json:"api_endpoint" jsonschema:"base URL of the provider API (required)"`
	APIKey       string   `json:"api_key,omitempty" jsonschema:"new API key; leave empty to keep the existing key"`
	Model        string   `json:"model" jsonschema:"model identifier (required)"`
	TaskTypes    []string `json:"task_types" jsonschema:"task types served (required)"`
	Dimensions   int      `json:"dimensions,omitempty" jsonschema:"embedding output dimension"`
	MaxTokens    int      `json:"max_tokens,omitempty" jsonschema:"max tokens per request"`
	Temperature  float64  `json:"temperature,omitempty" jsonschema:"sampling temperature"`
	IsDefault    bool     `json:"is_default,omitempty" jsonschema:"mark as the default provider"`
	IsEnabled    bool     `json:"is_enabled,omitempty" jsonschema:"enable this provider"`
}

type llmTestOutput struct {
	ChatOK      bool   `json:"chat_ok" jsonschema:"whether a chat completion succeeded"`
	EmbeddingOK bool   `json:"embedding_ok" jsonschema:"whether an embedding call succeeded"`
	CortexOK    bool   `json:"cortex_ok" jsonschema:"whether the RAG store round-trip succeeded"`
	LatencyMS   int64  `json:"latency_ms" jsonschema:"total test latency in milliseconds"`
	Error       string `json:"error,omitempty" jsonschema:"first error encountered, if any"`
}

// ---- closures ----

func llmList(_ context.Context, b Backend) (llmListOutput, string, error) {
	ps, err := b.ListLLMProviders()
	if err != nil {
		return llmListOutput{}, "", err
	}
	views := make([]llmProviderView, 0, len(ps))
	for i := range ps {
		views = append(views, llmProviderViewFrom(&ps[i]))
	}
	return llmListOutput{Providers: views, Total: len(views)}, fmt.Sprintf("%d LLM provider(s) configured.", len(views)), nil
}

func llmGet(_ context.Context, b Backend, in llmIDInput) (llmProviderView, string, error) {
	p, err := b.GetLLMProvider(in.ID)
	if err != nil {
		return llmProviderView{}, "", err
	}
	return llmProviderViewFrom(p), fmt.Sprintf("Provider %q (#%d, %s).", p.Name, p.ID, p.ProviderType), nil
}

func llmInputToService(in llmProviderInput) llm.CreateProviderInput {
	return llm.CreateProviderInput{
		Name: in.Name, ProviderType: in.ProviderType, APIEndpoint: in.APIEndpoint,
		APIKey: in.APIKey, Model: in.Model, TaskTypes: in.TaskTypes, Dimensions: in.Dimensions,
		MaxTokens: in.MaxTokens, Temperature: in.Temperature, IsDefault: in.IsDefault, IsEnabled: in.IsEnabled,
	}
}

func llmCreate(_ context.Context, b Backend, in llmProviderInput) (llmProviderView, string, error) {
	p, err := b.CreateLLMProvider(llmInputToService(in))
	if err != nil {
		return llmProviderView{}, "", err
	}
	return llmProviderViewFrom(p), fmt.Sprintf("Created LLM provider %q (#%d).", p.Name, p.ID), nil
}

func llmUpdate(_ context.Context, b Backend, in llmUpdateInput) (llmProviderView, string, error) {
	svcIn := llm.CreateProviderInput{
		Name: in.Name, ProviderType: in.ProviderType, APIEndpoint: in.APIEndpoint,
		APIKey: in.APIKey, Model: in.Model, TaskTypes: in.TaskTypes, Dimensions: in.Dimensions,
		MaxTokens: in.MaxTokens, Temperature: in.Temperature, IsDefault: in.IsDefault, IsEnabled: in.IsEnabled,
	}
	p, err := b.UpdateLLMProvider(in.ID, svcIn)
	if err != nil {
		return llmProviderView{}, "", err
	}
	return llmProviderViewFrom(p), fmt.Sprintf("Updated LLM provider %q (#%d).", p.Name, p.ID), nil
}

func llmDelete(_ context.Context, b Backend, in llmIDInput) (deleteOutput, string, error) {
	if err := b.DeleteLLMProvider(in.ID); err != nil {
		return deleteOutput{}, "", err
	}
	return deleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("Deleted LLM provider #%d.", in.ID), nil
}

func llmTest(ctx context.Context, b Backend, in llmIDInput) (llmTestOutput, string, error) {
	res, err := b.TestLLMProvider(ctx, in.ID)
	if err != nil {
		return llmTestOutput{}, "", err
	}
	out := llmTestOutput{
		ChatOK: res.ChatOK, EmbeddingOK: res.EmbeddingOK, CortexOK: res.CortexOK,
		LatencyMS: res.LatencyMS, Error: res.Error,
	}
	return out, fmt.Sprintf("Tested provider #%d (chat=%v embedding=%v).", in.ID, res.ChatOK, res.EmbeddingOK), nil
}
