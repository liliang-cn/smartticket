// Package aiassist bridges SmartTicket's bring-your-own LLM to the agent-go
// framework, so AI features (suggested replies, triage, and later a full
// tool-using auto-resolver) run on the operator's own configured model — no
// extra service, still a single binary.
package aiassist

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/liliang-cn/agent-go/v2/pkg/domain"

	"github.com/company/smartticket/internal/llm"
)

// Chatter is the slice of the LLM service this package needs: a plain chat
// completion over the operator's configured provider. *llm.Service satisfies it.
type Chatter interface {
	Chat(ctx context.Context, msgs []llm.ChatMessage) (string, error)
}

// Generator adapts a Chatter to agent-go's domain.Generator. The underlying
// BYO-LLM path is text-only (no native tool-calling), so tool definitions are
// accepted but not advertised to the model; the agent runtime still drives
// prompting, history and lints on top of this.
type Generator struct {
	chat Chatter
}

// NewGenerator wraps a Chatter as a domain.Generator.
func NewGenerator(chat Chatter) *Generator { return &Generator{chat: chat} }

var _ domain.Generator = (*Generator)(nil)

func (g *Generator) Generate(ctx context.Context, prompt string, _ *domain.GenerationOptions) (string, error) {
	return g.chat.Chat(ctx, []llm.ChatMessage{{Role: "user", Content: prompt}})
}

func (g *Generator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, cb func(string)) error {
	out, err := g.Generate(ctx, prompt, opts)
	if err != nil {
		return err
	}
	cb(out)
	return nil
}

func (g *Generator) GenerateWithTools(ctx context.Context, msgs []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions) (*domain.GenerationResult, error) {
	out, err := g.chat.Chat(ctx, toChatMessages(msgs))
	if err != nil {
		return nil, err
	}
	return &domain.GenerationResult{Content: out, Finished: true, FinishReason: "stop"}, nil
}

func (g *Generator) StreamWithTools(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, cb domain.ToolCallCallback) error {
	res, err := g.GenerateWithTools(ctx, msgs, tools, opts)
	if err != nil {
		return err
	}
	return cb(res)
}

func (g *Generator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	fullPrompt := prompt
	if schema != nil {
		schemaBytes, err := json.Marshal(schema)
		if err == nil {
			fullPrompt = prompt + "\n\nRespond with ONLY a single JSON object matching this schema (no prose, no markdown, no code fences):\n" + string(schemaBytes)
		}
	}

	out, err := g.Generate(ctx, fullPrompt, opts)
	if err != nil {
		return nil, err
	}

	extracted := extractJSON(out)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(extracted), &parsed); err != nil {
		return &domain.StructuredResult{Data: nil, Raw: out, Valid: false}, nil
	}
	return &domain.StructuredResult{Data: parsed, Raw: extracted, Valid: true}, nil
}

// extractJSON strips markdown code fences and returns the substring from the
// first '{' to the last '}'. Returns empty string if no JSON object is found.
func extractJSON(s string) string {
	// Strip ```json ... ``` or ``` ... ``` fences.
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence line.
		newline := strings.Index(s, "\n")
		if newline >= 0 {
			s = s[newline+1:]
		}
		// Remove closing fence.
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return ""
	}
	return s[start : end+1]
}

func (g *Generator) RecognizeIntent(_ context.Context, _ string) (*domain.IntentResult, error) {
	return &domain.IntentResult{Intent: domain.IntentAction, Confidence: 0.5}, nil
}

// toChatMessages maps agent-go messages onto the LLM service's chat format.
func toChatMessages(msgs []domain.Message) []llm.ChatMessage {
	out := make([]llm.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		role := m.Role
		switch role {
		case "system", "user", "assistant":
		default:
			role = "user" // tool/other → fold into user turn
		}
		out = append(out, llm.ChatMessage{Role: role, Content: m.Content})
	}
	return out
}
