package aiassist

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/liliang-cn/agent-go/v2/pkg/agent"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
)

// Sentinel errors so callers can map AI unavailability to the right HTTP status.
var (
	// ErrDisabled — the feature is turned off in AI settings.
	ErrDisabled = errors.New("ai feature is disabled")
	// ErrNotConfigured — no LLM provider / assistant is available.
	ErrNotConfigured = errors.New("no AI provider configured")
)

// KBSearcher returns relevant knowledge-base snippets for a query (RAG). A nil
// searcher just means the agent has no knowledge-base tool to call.
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

// Assistant runs a real agent-go agent on the deployment's BYO-LLM. The agent
// has a knowledge-base search tool and decides on its own whether to use it
// (prompt-tool-calling, so it works even with text-only providers). Gated by
// the AI settings singleton.
type Assistant struct {
	svc      *agent.Service
	settings *SettingsStore
	mu       sync.Mutex // serialize runs — suggestions are low-QPS
}

const agentSystemPrompt = `You are an experienced customer-support agent. Your job is to draft the agent's next reply to the customer on a support ticket.

You have a tool, search_knowledge_base, that searches the team's knowledge base. Call it when you need product facts, steps, or policies you are not sure about — do not guess. You may call it more than once. If the knowledge base has nothing useful, rely on the ticket context.

Write only the reply body the agent will send to the customer: clear, friendly and professional. Never invent facts, prices or commitments. If you cannot resolve it, say a teammate will follow up. Do not include a subject line or placeholders like [Name].`

// NewAssistant builds the agent (once) wiring the BYO-LLM generator and the
// knowledge-base tool. dbPath contains agent-go's own SQLite store.
func NewAssistant(gen domain.Generator, kb KBSearcher, settings *SettingsStore, dbPath string) (*Assistant, error) {
	svc, err := agent.New("support-assistant").
		WithLLM(gen).
		WithDBPath(dbPath).
		WithPrompt(agentSystemPrompt).
		Build()
	if err != nil {
		return nil, fmt.Errorf("build support agent: %w", err)
	}

	svc.AddToolWithMetadata(
		"search_knowledge_base",
		"Search the support knowledge base for articles relevant to the customer's question. Returns matching snippets.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "What to look up, e.g. 'reset password' or 'export tickets to CSV'.",
				},
			},
			"required": []string{"query"},
		},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			query, _ := args["query"].(string)
			if kb == nil || strings.TrimSpace(query) == "" {
				return "knowledge base is unavailable", nil
			}
			hits := kb.SnippetsFor(ctx, query, 4)
			if len(hits) == 0 {
				return "no relevant knowledge base articles found", nil
			}
			return strings.Join(hits, "\n---\n"), nil
		},
		agent.ToolMetadata{ReadOnly: true, ConcurrencySafe: true, InterruptBehavior: agent.InterruptBehaviorCancel},
	)

	return &Assistant{svc: svc, settings: settings}, nil
}

// Close releases the agent's resources.
func (a *Assistant) Close() error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.Close()
}

// SuggestReply runs the agent to draft the agent's next reply for a ticket.
// Returns ErrDisabled / ErrNotConfigured when unavailable.
func (a *Assistant) SuggestReply(ctx context.Context, in SuggestInput) (string, error) {
	if a == nil || a.svc == nil {
		return "", ErrNotConfigured
	}
	set, err := a.settings.Get()
	if err != nil {
		return "", err
	}
	if !set.Enabled || !set.SuggestReplies {
		return "", ErrDisabled
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	res, err := a.svc.Run(ctx, buildGoal(in, set.ReplyInstructions),
		agent.WithSessionID(newSessionID()),
		agent.WithMaxTurns(6),
		agent.WithTemperature(0.4),
	)
	if err != nil {
		return "", fmt.Errorf("agent run failed: %w", err)
	}
	return strings.TrimSpace(finalText(res)), nil
}

// buildGoal assembles the per-run instruction (ticket context + dynamic
// operator guidance). Dynamic guidance lives here, not in the built system
// prompt, so settings changes take effect without rebuilding the agent.
func buildGoal(in SuggestInput, custom string) string {
	var b strings.Builder
	b.WriteString("Draft the agent's next reply for this support ticket.\n\n")
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
	if c := strings.TrimSpace(custom); c != "" {
		b.WriteString("\nTeam guidance to follow:\n" + c + "\n")
	}
	b.WriteString("\nReturn only the reply text.")
	return b.String()
}

func finalText(r *agent.ExecutionResult) string {
	if r == nil {
		return ""
	}
	switch v := r.FinalResult.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func newSessionID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "suggest-" + hex.EncodeToString(b)
}
