package aiassist

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/liliang-cn/agent-go/v2/pkg/domain"
)

// Sentinel errors so callers can map AI unavailability to the right HTTP status.
var (
	// ErrDisabled — the feature is turned off in AI settings.
	ErrDisabled = errors.New("ai feature is disabled")
	// ErrNotConfigured — no LLM provider is available.
	ErrNotConfigured = errors.New("no AI provider configured")
)

// KBSearcher returns relevant knowledge-base snippets for a query (RAG context).
// A nil searcher simply means replies are drafted without KB grounding.
type KBSearcher interface {
	SnippetsFor(ctx context.Context, query string, topK int) []string
}

// KBSearcherFunc adapts a function to KBSearcher.
type KBSearcherFunc func(ctx context.Context, query string, topK int) []string

func (f KBSearcherFunc) SnippetsFor(ctx context.Context, query string, topK int) []string {
	return f(ctx, query, topK)
}

// Turn is one message in a ticket conversation.
type Turn struct {
	Author     string
	IsCustomer bool
	Content    string
}

// SuggestInput is the ticket context used to draft a reply.
type SuggestInput struct {
	Title        string
	Description  string
	CustomerName string
	Conversation []Turn
}

// Assistant runs AI features on the deployment's BYO-LLM (via the agent-go
// generator), gated by the AI settings singleton.
type Assistant struct {
	gen      domain.Generator
	kb       KBSearcher // optional
	settings *SettingsStore
}

// NewAssistant wires the generator, optional KB searcher and settings.
func NewAssistant(gen domain.Generator, kb KBSearcher, settings *SettingsStore) *Assistant {
	return &Assistant{gen: gen, kb: kb, settings: settings}
}

// SuggestReply drafts the agent's next reply for a ticket. Returns ErrDisabled
// or ErrNotConfigured when the feature is unavailable.
func (a *Assistant) SuggestReply(ctx context.Context, in SuggestInput) (string, error) {
	if a == nil || a.gen == nil {
		return "", ErrNotConfigured
	}
	set, err := a.settings.Get()
	if err != nil {
		return "", err
	}
	if !set.Enabled || !set.SuggestReplies {
		return "", ErrDisabled
	}

	var kbContext string
	if a.kb != nil {
		query := strings.TrimSpace(in.Title + " " + in.Description)
		if hits := a.kb.SnippetsFor(ctx, query, 4); len(hits) > 0 {
			kbContext = "Relevant knowledge base entries:\n" + strings.Join(hits, "\n---\n")
		}
	}

	res, err := a.gen.GenerateWithTools(ctx, []domain.Message{
		{Role: "system", Content: buildSystemPrompt(set.ReplyInstructions)},
		{Role: "user", Content: buildUserPrompt(in, kbContext)},
	}, nil, &domain.GenerationOptions{Temperature: 0.4})
	if err != nil {
		return "", fmt.Errorf("ai generation failed: %w", err)
	}
	return strings.TrimSpace(res.Content), nil
}

func buildSystemPrompt(custom string) string {
	var b strings.Builder
	b.WriteString("You are a helpful customer-support agent. Draft a clear, friendly, professional reply to the customer's latest message. ")
	b.WriteString("Use only the information provided; never invent facts, prices or commitments. If unsure, say a teammate will follow up. ")
	b.WriteString("Write the reply body only — no subject line and no placeholders like [Name]. Keep it concise.")
	if c := strings.TrimSpace(custom); c != "" {
		b.WriteString("\n\nAdditional guidance from the team:\n")
		b.WriteString(c)
	}
	return b.String()
}

func buildUserPrompt(in SuggestInput, kbContext string) string {
	var b strings.Builder
	b.WriteString("Ticket: " + in.Title + "\n")
	if d := strings.TrimSpace(in.Description); d != "" {
		b.WriteString("Description: " + d + "\n")
	}
	if in.CustomerName != "" {
		b.WriteString("Customer: " + in.CustomerName + "\n")
	}
	b.WriteString("\nConversation so far:\n")
	if len(in.Conversation) == 0 {
		b.WriteString("(no replies yet — respond to the description above)\n")
	}
	for _, t := range in.Conversation {
		who := strings.TrimSpace(t.Author)
		if who == "" {
			if t.IsCustomer {
				who = "Customer"
			} else {
				who = "Agent"
			}
		}
		b.WriteString(who + ": " + t.Content + "\n")
	}
	if kbContext != "" {
		b.WriteString("\n" + kbContext + "\n")
	}
	b.WriteString("\nDraft the agent's next reply:")
	return b.String()
}
