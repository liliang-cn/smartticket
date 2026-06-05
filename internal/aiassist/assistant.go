package aiassist

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

// Draft is a structured AI reply suggestion.
type Draft struct {
	Reply              string   `json:"reply"`
	Confidence         float64  `json:"confidence"`          // 0..1
	NeedsClarification bool     `json:"needs_clarification"`
	UsedKB             bool     `json:"used_kb"`
	Sources            []string `json:"sources"`
}

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
	gen      domain.Generator // direct generator for structured path
	kb       KBSearcher       // direct KB searcher for structured path
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

	return &Assistant{svc: svc, gen: gen, kb: kb, settings: settings}, nil
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
// It is a thin wrapper over SuggestReplyStructured.
func (a *Assistant) SuggestReply(ctx context.Context, in SuggestInput) (string, error) {
	draft, err := a.SuggestReplyStructured(ctx, in)
	if err != nil {
		return "", err
	}
	return draft.Reply, nil
}

// draftSchema is the JSON schema passed to GenerateStructured so that
// providers that support constrained output return a well-formed Draft.
var draftSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"reply": map[string]interface{}{
			"type":        "string",
			"description": "The drafted reply body to send to the customer. Friendly, professional, no placeholders, never invent facts.",
		},
		"confidence": map[string]interface{}{
			"type":        "number",
			"description": "0 to 1. How confident you are this reply fully resolves the ticket. Use LOW (< 0.5) when the ticket lacks enough detail.",
		},
		"needs_clarification": map[string]interface{}{
			"type":        "boolean",
			"description": "true if you must ask the customer for more information instead of resolving.",
		},
		"used_kb": map[string]interface{}{
			"type":        "boolean",
			"description": "true if you used the knowledge base context below to draft the reply.",
		},
		"sources": map[string]interface{}{
			"type":        "array",
			"items":       map[string]interface{}{"type": "string"},
			"description": "Titles or identifiers of knowledge base snippets you used.",
		},
	},
	"required": []string{"reply", "confidence", "needs_clarification", "used_kb", "sources"},
}

// SuggestReplyStructured produces a structured Draft for a ticket, using the
// BYO-LLM directly (no agent loop) so the JSON schema instruction reaches the
// model as-is. Falls back gracefully if the model ignores the schema.
func (a *Assistant) SuggestReplyStructured(ctx context.Context, in SuggestInput) (Draft, error) {
	if a == nil || a.gen == nil {
		return Draft{}, ErrNotConfigured
	}
	set, err := a.settings.Get()
	if err != nil {
		return Draft{}, err
	}
	if !set.Enabled || !set.SuggestReplies {
		return Draft{}, ErrDisabled
	}

	// RAG: gather KB snippets.
	var snippets []string
	usedKB := false
	if a.kb != nil {
		query := strings.TrimSpace(in.Title + " " + in.Description)
		// Append the last customer turn to improve retrieval.
		for i := len(in.Conversation) - 1; i >= 0; i-- {
			if in.Conversation[i].IsCustomer {
				query += " " + in.Conversation[i].Content
				break
			}
		}
		snippets = a.kb.SnippetsFor(ctx, strings.TrimSpace(query), 4)
	}

	// Build prompt.
	var b strings.Builder
	b.WriteString(agentSystemPrompt)
	b.WriteString("\n\n")
	b.WriteString(buildGoalStructured(in, set.ReplyInstructions))
	b.WriteString("\n\nKnowledge base context:\n")
	if len(snippets) == 0 {
		b.WriteString("(no relevant articles)\n")
	} else {
		for _, s := range snippets {
			b.WriteString("- " + s + "\n")
		}
		usedKB = true
	}
	b.WriteString(`
Output fields:
- reply: the drafted reply body (friendly, professional, no placeholders, never invent facts)
- confidence: float 0-1 how sure this reply fully resolves the ticket (low if ticket lacks detail)
- needs_clarification: true if you must ask the customer for more info instead of resolving
- used_kb: true if you used the knowledge base context above
- sources: list of KB snippet titles/identifiers you used`)

	result, err := a.gen.GenerateStructured(ctx, b.String(), draftSchema, &domain.GenerationOptions{Temperature: 0.4})
	if err != nil {
		return Draft{}, fmt.Errorf("structured generation failed: %w", err)
	}

	// Happy path: parse the structured map into Draft.
	if result.Valid {
		if dataMap, ok := result.Data.(map[string]interface{}); ok {
			marshaled, merr := json.Marshal(dataMap)
			if merr == nil {
				var d Draft
				if uerr := json.Unmarshal(marshaled, &d); uerr == nil {
					// Clamp confidence.
					if d.Confidence < 0 {
						d.Confidence = 0
					} else if d.Confidence > 1 {
						d.Confidence = 1
					}
					// Override used_kb based on whether we actually found snippets,
					// but trust the model's value if snippets were available.
					if !usedKB {
						d.UsedKB = false
					}
					return d, nil
				}
			}
		}
	}

	// Fallback: model returned prose — use raw text as reply, zero confidence.
	raw := strings.TrimSpace(result.Raw)
	if raw == "" {
		raw = result.Raw
	}
	return Draft{Reply: raw, Confidence: 0, NeedsClarification: false, UsedKB: false}, nil
}

// buildGoalStructured is like buildGoal but omits the "Return only the reply
// text" instruction since the structured path handles output format separately.
func buildGoalStructured(in SuggestInput, custom string) string {
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
	return b.String()
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
