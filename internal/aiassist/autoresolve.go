package aiassist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
)

// TicketActions is the minimal interface the AutoResolver needs to interact
// with the ticket and notification layers.  A concrete adapter (in
// internal/server) implements this using the ticket + notification services,
// avoiding an import cycle (internal/ticket already imports internal/aiassist).
type TicketActions interface {
	// PostAIReply appends an AI-authored public reply to the ticket.
	// The underlying event bus publish MUST carry Source:"ai" so the
	// orchestrator's loop-guard (Source!="") fires and ignores it.
	PostAIReply(ticketID uint, body string) error

	// CountAIReplies returns the number of AI-authored messages on the ticket.
	CountAIReplies(ticketID uint) (int, error)

	// UpdateClassification sets the ticket's category, priority, and tags.
	// priority must be one of: low, medium, high, critical.
	UpdateClassification(ticketID uint, category, priority string, tags []string) error

	// SetSummary stores a generated summary on the ticket.
	SetSummary(ticketID uint, summary string) error

	// LoadContext assembles a SuggestInput for the ticket plus a flag that is
	// true when the last message originates from a customer (i.e. someone is
	// waiting for a reply).
	LoadContext(ticketID uint) (SuggestInput, bool, error)

	// NotifyAgentsSuggestion sends an in-app suggestion/hand-off notification
	// to the assigned agent (or silently skips when none is assigned).
	NotifyAgentsSuggestion(ticketID uint, draft Draft) error
}

// AutoResolver subscribes to domain events and drives AI-powered automation:
//   - auto-classification on ticket creation
//   - confidence-gated auto-replies or agent-suggestion on new customer messages
//   - conversation summarization on ticket resolution
type AutoResolver struct {
	assistant *Assistant
	settings  *SettingsStore
	actions   TicketActions
	gen       domain.Generator // direct path for classify / summarize
}

// NewAutoResolver constructs an AutoResolver.  All fields are required.
func NewAutoResolver(assistant *Assistant, settings *SettingsStore, actions TicketActions) *AutoResolver {
	return &AutoResolver{
		assistant: assistant,
		settings:  settings,
		actions:   actions,
		gen:       assistant.gen,
	}
}

// Subscribe wires the three handlers onto the event bus.
func (r *AutoResolver) Subscribe(bus *automation.Bus) {
	bus.Subscribe(automation.EventTicketCreated, r.OnTicketCreated)
	bus.Subscribe(automation.EventMessageCreated, r.OnMessageCreated)
	bus.Subscribe(automation.EventTicketResolved, r.OnTicketResolved)
}

// OnTicketCreated handles new ticket events.  It optionally classifies the
// ticket and then triggers the same auto-reply logic as OnMessageCreated.
func (r *AutoResolver) OnTicketCreated(e automation.Event) {
	set, err := r.settings.Get()
	if err != nil || !set.Enabled {
		return
	}

	ctx := context.Background()

	if set.AutoClassify {
		r.classifyTicket(ctx, e.TicketID)
	}

	// Treat ticket creation like a new customer message so that if auto-reply
	// is on we can respond to the opening description immediately.
	// Synthesise a fake event with empty Source so the guard passes.
	r.handleCustomerMessage(ctx, set, automation.Event{
		Type:     automation.EventMessageCreated,
		TicketID: e.TicketID,
		ActorID:  e.ActorID,
		Source:   "", // human-originated ticket creation
	})
}

// OnMessageCreated handles new-message events.
//
// Loop guard: if e.Source != "" the event was emitted by automation or AI
// (the adapter sets Source="ai" when PostAIReply publishes the event); we
// return immediately to prevent infinite loops.
func (r *AutoResolver) OnMessageCreated(e automation.Event) {
	if e.Source != "" {
		return // AI/automation message — ignore to prevent loops
	}

	set, err := r.settings.Get()
	if err != nil || !set.Enabled || !set.SuggestReplies {
		return
	}

	r.handleCustomerMessage(context.Background(), set, e)
}

// OnTicketResolved handles ticket-resolved events and optionally summarizes.
func (r *AutoResolver) OnTicketResolved(e automation.Event) {
	set, err := r.settings.Get()
	if err != nil || !set.Enabled || !set.AutoSummarizeOnResolve {
		return
	}

	ctx := context.Background()
	in, _, err := r.actions.LoadContext(e.TicketID)
	if err != nil {
		logger.Warn("autoresolve: LoadContext failed for summarize",
			zap.Uint("ticket_id", e.TicketID), zap.Error(err))
		return
	}

	summary, err := r.summarize(ctx, in)
	if err != nil {
		logger.Warn("autoresolve: summarize failed",
			zap.Uint("ticket_id", e.TicketID), zap.Error(err))
		return
	}
	if summary == "" {
		return
	}
	if err := r.actions.SetSummary(e.TicketID, summary); err != nil {
		logger.Warn("autoresolve: SetSummary failed",
			zap.Uint("ticket_id", e.TicketID), zap.Error(err))
	}
}

// handleCustomerMessage contains the shared logic used by both
// OnTicketCreated and OnMessageCreated.
func (r *AutoResolver) handleCustomerMessage(ctx context.Context, set *models.AISettings, e automation.Event) {
	in, customerWaiting, err := r.actions.LoadContext(e.TicketID)
	if err != nil {
		logger.Warn("autoresolve: LoadContext failed",
			zap.Uint("ticket_id", e.TicketID), zap.Error(err))
		return
	}

	// Only proceed when the last message is from the customer (someone is
	// actually waiting).  On ticket creation customerWaiting is always true
	// because the description is treated as the opening message.
	if !customerWaiting {
		return
	}

	draft, err := r.assistant.SuggestReplyStructured(ctx, in)
	if err != nil {
		if IsDisabled(err) || IsNotConfigured(err) {
			return // silently skip — AI not available
		}
		logger.Warn("autoresolve: SuggestReplyStructured failed",
			zap.Uint("ticket_id", e.TicketID), zap.Error(err))
		return
	}

	count, err := r.actions.CountAIReplies(e.TicketID)
	if err != nil {
		logger.Warn("autoresolve: CountAIReplies failed",
			zap.Uint("ticket_id", e.TicketID), zap.Error(err))
		return
	}

	// Cap check: if we've already posted the maximum number of AI replies,
	// hand off to a human rather than auto-replying further.
	if count >= set.MaxAutoRepliesPerTicket {
		if nerr := r.actions.NotifyAgentsSuggestion(e.TicketID, draft); nerr != nil {
			logger.Warn("autoresolve: NotifyAgentsSuggestion failed",
				zap.Uint("ticket_id", e.TicketID), zap.Error(nerr))
		}
		return
	}

	// Decision: auto-post or suggest to agent?
	if set.AutoReplyEnabled &&
		!draft.NeedsClarification &&
		draft.Confidence >= set.AutoReplyConfidence {

		if rerr := r.actions.PostAIReply(e.TicketID, draft.Reply); rerr != nil {
			logger.Warn("autoresolve: PostAIReply failed",
				zap.Uint("ticket_id", e.TicketID), zap.Error(rerr))
		}
		// Note: actual ticket auto-resolution (status→resolved) is deferred to
		// Phase 3's scheduler; here we only post the reply.
	} else {
		if nerr := r.actions.NotifyAgentsSuggestion(e.TicketID, draft); nerr != nil {
			logger.Warn("autoresolve: NotifyAgentsSuggestion failed",
				zap.Uint("ticket_id", e.TicketID), zap.Error(nerr))
		}
	}
}

// ----- LLM helpers ----------------------------------------------------------

// classifyTicket asks the LLM to classify a ticket and updates the DB.
func (r *AutoResolver) classifyTicket(ctx context.Context, ticketID uint) {
	in, _, err := r.actions.LoadContext(ticketID)
	if err != nil {
		logger.Warn("autoresolve: LoadContext failed for classify",
			zap.Uint("ticket_id", ticketID), zap.Error(err))
		return
	}

	cat, prio, tags, err := r.classify(ctx, in)
	if err != nil {
		logger.Warn("autoresolve: classify failed",
			zap.Uint("ticket_id", ticketID), zap.Error(err))
		return
	}
	if cat == "" && prio == "" && len(tags) == 0 {
		return
	}
	if err := r.actions.UpdateClassification(ticketID, cat, prio, tags); err != nil {
		logger.Warn("autoresolve: UpdateClassification failed",
			zap.Uint("ticket_id", ticketID), zap.Error(err))
	}
}

// classifySchema is the JSON schema for the classify structured call.
var classifySchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"category": map[string]interface{}{
			"type":        "string",
			"description": "A short topic category for the ticket, e.g. 'billing', 'login', 'performance'.",
		},
		"priority": map[string]interface{}{
			"type":        "string",
			"enum":        []string{"low", "medium", "high", "critical"},
			"description": "Suggested priority based on urgency and impact.",
		},
		"tags": map[string]interface{}{
			"type":        "array",
			"items":       map[string]interface{}{"type": "string"},
			"description": "Up to 5 descriptive labels for the ticket.",
		},
	},
	"required": []string{"category", "priority", "tags"},
}

// classify calls the LLM to suggest category, priority, and tags for a ticket.
// Returns empty strings/nil when the model returns unusable output.
func (r *AutoResolver) classify(ctx context.Context, in SuggestInput) (category, priority string, tags []string, err error) {
	prompt := buildClassifyPrompt(in)
	result, err := r.gen.GenerateStructured(ctx, prompt, classifySchema, &domain.GenerationOptions{Temperature: 0.2})
	if err != nil {
		return "", "", nil, fmt.Errorf("classify LLM call: %w", err)
	}
	if !result.Valid {
		return "", "", nil, nil
	}

	raw, err := json.Marshal(result.Data)
	if err != nil {
		return "", "", nil, nil
	}
	var out struct {
		Category string   `json:"category"`
		Priority string   `json:"priority"`
		Tags     []string `json:"tags"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", nil, nil
	}

	// Validate priority against the known set; discard unknown values.
	switch out.Priority {
	case "low", "medium", "high", "critical":
		priority = out.Priority
	default:
		priority = "" // unknown value — leave the existing priority
	}
	return out.Category, priority, out.Tags, nil
}

// summarize calls the LLM to produce a plain-text conversation summary.
func (r *AutoResolver) summarize(ctx context.Context, in SuggestInput) (string, error) {
	prompt := buildSummarizePrompt(in)
	out, err := r.gen.Generate(ctx, prompt, &domain.GenerationOptions{Temperature: 0.3})
	if err != nil {
		return "", fmt.Errorf("summarize LLM call: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// buildClassifyPrompt constructs the classification prompt.
func buildClassifyPrompt(in SuggestInput) string {
	var b strings.Builder
	b.WriteString("You are a support ticket classifier. Given the ticket below, output:\n")
	b.WriteString("- category: a short topic label (e.g. billing, login, performance)\n")
	b.WriteString("- priority: one of low / medium / high / critical based on urgency and impact\n")
	b.WriteString("- tags: up to 5 descriptive labels\n\n")
	b.WriteString("Ticket title: " + in.Title + "\n")
	if d := strings.TrimSpace(in.Description); d != "" {
		b.WriteString("Description: " + d + "\n")
	}
	return b.String()
}

// buildSummarizePrompt constructs the summarization prompt.
func buildSummarizePrompt(in SuggestInput) string {
	var b strings.Builder
	b.WriteString("Write a concise 1-3 sentence summary of the following resolved support ticket conversation. ")
	b.WriteString("Focus on: the original problem, steps taken, and how it was resolved.\n\n")
	b.WriteString("Ticket: " + in.Title + "\n")
	if d := strings.TrimSpace(in.Description); d != "" {
		b.WriteString("Description: " + d + "\n")
	}
	if len(in.Conversation) > 0 {
		b.WriteString("\nConversation:\n")
		for _, t := range in.Conversation {
			who := t.Author
			if who == "" {
				if t.IsCustomer {
					who = "Customer"
				} else {
					who = "Agent"
				}
			}
			b.WriteString(who + ": " + t.Content + "\n")
		}
	}
	return b.String()
}
