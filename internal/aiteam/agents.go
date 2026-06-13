package aiteam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/company/smartticket/internal/aiassist"
)

// memberInstructions maps specialist name → system prompt text. Built at init
// time from the same memberDefs slice so there is a single source of truth.
var memberInstructions = func() map[string]string {
	m := make(map[string]string, len(memberDefs))
	for _, d := range memberDefs {
		m[d.name] = d.instructions
	}
	return m
}()

// TicketContext is the input rendered into an agent prompt.
type TicketContext struct {
	TicketID     uint
	Title        string
	Description  string
	Conversation string // flattened messages, oldest→newest
	CustomerName string
	SLAState     string // e.g. "warning: breach in 30m" — empty if none
}

// TriageResult holds the structured output of the Triage agent.
type TriageResult struct {
	Priority        string  `json:"priority"`
	Severity        string  `json:"severity"`
	Category        string  `json:"category"`
	SuggestedTeamID *uint   `json:"suggested_team_id"`
	Reasoning       string  `json:"reasoning"`
	Confidence      float64 `json:"confidence"`
}

// SentinelResult holds the structured output of the Sentinel agent.
type SentinelResult struct {
	Sentiment     string  `json:"sentiment"`
	ChurnRisk     string  `json:"churn_risk"`
	SLABreachRisk bool    `json:"sla_breach_risk"`
	Escalate      bool    `json:"escalate"`
	Reasoning     string  `json:"reasoning"`
	Confidence    float64 `json:"confidence"`
}

// triageSchema is the JSON schema for TriageResult, mirroring aiassist.draftSchema.
var triageSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"priority": map[string]interface{}{
			"type":        "string",
			"description": "Ticket priority: critical, high, medium, or low.",
		},
		"severity": map[string]interface{}{
			"type":        "string",
			"description": "Ticket severity: blocker, major, minor, or trivial.",
		},
		"category": map[string]interface{}{
			"type":        "string",
			"description": "Short category label for the ticket type, e.g. 'billing', 'technical', 'account'.",
		},
		"suggested_team_id": map[string]interface{}{
			"type":        "integer",
			"description": "Numeric team ID to route to, if obvious. Omit or set null when unknown.",
		},
		"reasoning": map[string]interface{}{
			"type":        "string",
			"description": "Brief reasoning behind the priority/severity/category judgement.",
		},
		"confidence": map[string]interface{}{
			"type":        "number",
			"description": "0 to 1. How confident you are in this triage. Use LOW (< 0.5) when context is insufficient.",
		},
	},
	"required": []string{"priority", "severity", "category", "reasoning", "confidence"},
}

// sentinelSchema is the JSON schema for SentinelResult.
var sentinelSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"sentiment": map[string]interface{}{
			"type":        "string",
			"description": "Customer sentiment: positive, neutral, frustrated, or angry.",
		},
		"churn_risk": map[string]interface{}{
			"type":        "string",
			"description": "Churn risk level: none, low, medium, or high.",
		},
		"sla_breach_risk": map[string]interface{}{
			"type":        "boolean",
			"description": "true if SLA breach is imminent or already in breach.",
		},
		"escalate": map[string]interface{}{
			"type":        "boolean",
			"description": "true if the ticket should be escalated to a manager immediately.",
		},
		"reasoning": map[string]interface{}{
			"type":        "string",
			"description": "Brief reasoning behind the risk and escalation assessment.",
		},
		"confidence": map[string]interface{}{
			"type":        "number",
			"description": "0 to 1. How confident you are in this assessment.",
		},
	},
	"required": []string{"sentiment", "churn_risk", "sla_breach_risk", "escalate", "reasoning", "confidence"},
}

// renderContext formats a TicketContext into human-readable prompt text.
func renderContext(tc TicketContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ticket #%d\n", tc.TicketID)
	fmt.Fprintf(&b, "Title: %s\n", tc.Title)
	if d := strings.TrimSpace(tc.Description); d != "" {
		fmt.Fprintf(&b, "Description: %s\n", d)
	}
	if tc.CustomerName != "" {
		fmt.Fprintf(&b, "Customer: %s\n", tc.CustomerName)
	}
	if tc.SLAState != "" {
		fmt.Fprintf(&b, "SLA Status: %s\n", tc.SLAState)
	}
	if c := strings.TrimSpace(tc.Conversation); c != "" {
		b.WriteString("\nConversation (oldest→newest):\n")
		b.WriteString(c)
		b.WriteString("\n")
	}
	return b.String()
}

// clampConfidence clamps a float64 into [0, 1].
func clampConfidence(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// mapToStruct marshals a map[string]interface{} into a typed struct via JSON
// round-trip. Returns false if marshalling or unmarshalling fails.
func mapToStruct(src map[string]interface{}, dst interface{}) bool {
	b, err := json.Marshal(src)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, dst) == nil
}

// RunTriage runs the Triage specialist for the given ticket context. Returns
// aiassist.ErrNotConfigured when no generator is wired, aiassist.ErrDisabled
// when the AI feature toggle is off. On an unparseable model response it
// returns a zero-value TriageResult with Confidence 0 (graceful degradation).
func (t *Team) RunTriage(ctx context.Context, tc TicketContext) (*TriageResult, error) {
	if t.gen == nil {
		return nil, aiassist.ErrNotConfigured
	}
	if t.settings != nil {
		set, err := t.settings.Get()
		if err != nil {
			return nil, err
		}
		if !set.Enabled {
			return nil, aiassist.ErrDisabled
		}
	}

	prompt := memberInstructions["Triage"] + "\n\n" + renderContext(tc)
	dataMap, valid, err := t.structured(ctx, "Triage", prompt, triageSpec)
	if err != nil {
		return nil, err
	}
	if !valid || dataMap == nil {
		return &TriageResult{Confidence: 0}, nil
	}

	var out TriageResult
	if !mapToStruct(dataMap, &out) {
		return &TriageResult{Confidence: 0}, nil
	}
	out.Confidence = clampConfidence(out.Confidence)
	return &out, nil
}

// Snippet is a knowledge-base snippet returned as a citation.
type Snippet struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// SimilarTicket is a past ticket found by the SimilarTicketSearcher.
type SimilarTicket struct {
	ID             uint    `json:"id"`
	Title          string  `json:"title"`
	Resolution     string  `json:"resolution"`
	MergeCandidate bool    `json:"merge_candidate"`
	Score          float64 `json:"score"`
}

// ResearcherResult holds the structured output of the Researcher agent.
type ResearcherResult struct {
	KBCitations         []Snippet       `json:"kb_citations"`
	SimilarTickets      []SimilarTicket `json:"similar_tickets"`
	SuggestedResolution string          `json:"suggested_resolution"`
	Confidence          float64         `json:"confidence"`
}

// ReviewIssue is a single issue flagged by the Reviewer agent.
type ReviewIssue struct {
	Type     string `json:"type"`     // tone|accuracy|policy|missing_info
	Severity string `json:"severity"` // low|medium|high
	Note     string `json:"note"`
}

// ReviewerResult holds the structured output of the Reviewer agent.
type ReviewerResult struct {
	Issues       []ReviewIssue `json:"issues"`
	RevisedDraft string        `json:"revised_draft"`
	Approve      bool          `json:"approve"`
	Confidence   float64       `json:"confidence"`
}

// DrafterResult holds the structured output of the Drafter agent.
type DrafterResult struct {
	Reply      string  `json:"reply"`
	Confidence float64 `json:"confidence"`
}

// researcherSchema is the JSON schema for the LLM-generated part of ResearcherResult.
// kb_citations and similar_tickets are attached in Go code, not produced by the LLM.
var researcherSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"suggested_resolution": map[string]interface{}{
			"type":        "string",
			"description": "Proposed resolution for the ticket based on provided KB snippets and similar tickets context.",
		},
		"confidence": map[string]interface{}{
			"type":        "number",
			"description": "0 to 1. How confident you are in the proposed resolution.",
		},
	},
	"required": []string{"suggested_resolution", "confidence"},
}

// reviewerSchema is the JSON schema for ReviewerResult.
var reviewerSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"issues": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Issue type: tone, accuracy, policy, or missing_info.",
					},
					"severity": map[string]interface{}{
						"type":        "string",
						"description": "Issue severity: low, medium, or high.",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Brief explanation of the issue.",
					},
				},
				"required": []string{"type", "severity", "note"},
			},
			"description": "List of issues found in the draft reply.",
		},
		"revised_draft": map[string]interface{}{
			"type":        "string",
			"description": "Revised version of the draft with issues addressed. Empty if no revision needed.",
		},
		"approve": map[string]interface{}{
			"type":        "boolean",
			"description": "true if the draft is acceptable to send as-is (or after minor edits).",
		},
		"confidence": map[string]interface{}{
			"type":        "number",
			"description": "0 to 1. How confident you are in this review.",
		},
	},
	"required": []string{"issues", "revised_draft", "approve", "confidence"},
}

// drafterSchema is the JSON schema for DrafterResult.
var drafterSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"reply": map[string]interface{}{
			"type":        "string",
			"description": "The drafted reply body to send to the customer. Friendly, professional, no placeholders, never invent facts.",
		},
		"confidence": map[string]interface{}{
			"type":        "number",
			"description": "0 to 1. How confident you are this reply fully resolves the ticket.",
		},
	},
	"required": []string{"reply", "confidence"},
}

// RunResearcher runs the Researcher specialist for the given ticket context. It
// gathers KB snippets and similar tickets, attaches them to the result in Go,
// and asks the LLM to propose a resolution. Returns aiassist.ErrNotConfigured
// when no generator is wired, aiassist.ErrDisabled when the AI feature toggle is
// off. Gracefully handles nil KB / nil SimilarTicketSearcher (returns empty lists).
func (t *Team) RunResearcher(ctx context.Context, tc TicketContext) (*ResearcherResult, error) {
	if t.gen == nil {
		return nil, aiassist.ErrNotConfigured
	}
	if t.settings != nil {
		set, err := t.settings.Get()
		if err != nil {
			return nil, err
		}
		if !set.Enabled {
			return nil, aiassist.ErrDisabled
		}
	}

	// Gather KB snippets (nil-safe).
	var kbSnippets []Snippet
	if t.kb != nil {
		raw := t.kb.SnippetsFor(ctx, tc.Title+" "+tc.Description, 4)
		for _, s := range raw {
			kbSnippets = append(kbSnippets, Snippet{Title: s, Snippet: s})
		}
	}
	if kbSnippets == nil {
		kbSnippets = []Snippet{}
	}

	// Gather similar tickets (nil-safe).
	var similarTickets []SimilarTicket
	if t.similar != nil {
		found, err := t.similar.SearchSimilar(ctx, tc.Title+" "+tc.Description, 5)
		if err == nil && len(found) > 0 {
			similarTickets = found
		}
	}
	if similarTickets == nil {
		similarTickets = []SimilarTicket{}
	}

	// Build prompt with gathered context.
	var b strings.Builder
	b.WriteString(memberInstructions["Researcher"])
	b.WriteString("\n\n")
	b.WriteString(renderContext(tc))
	b.WriteString("\nKnowledge base snippets:\n")
	if len(kbSnippets) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, s := range kbSnippets {
			b.WriteString("- " + s.Snippet + "\n")
		}
	}
	b.WriteString("\nSimilar past tickets:\n")
	if len(similarTickets) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, st := range similarTickets {
			fmt.Fprintf(&b, "- #%d %s: %s\n", st.ID, st.Title, st.Resolution)
		}
	}

	dataMap, valid, err := t.structured(ctx, "Researcher", b.String(), researcherSpec)
	if err != nil {
		return nil, err
	}

	out := &ResearcherResult{
		KBCitations:    kbSnippets,
		SimilarTickets: similarTickets,
	}

	if !valid || dataMap == nil {
		return out, nil
	}

	if !mapToStruct(dataMap, out) {
		// mapToStruct overwrites out; restore the gathered data.
		out.KBCitations = kbSnippets
		out.SimilarTickets = similarTickets
		return out, nil
	}
	// mapToStruct may zero out the gathered slices since they're not in the LLM
	// schema — re-attach them.
	out.KBCitations = kbSnippets
	out.SimilarTickets = similarTickets
	out.Confidence = clampConfidence(out.Confidence)
	return out, nil
}

// RunReviewer runs the Reviewer specialist for the given ticket context and a
// draft reply. Returns aiassist.ErrNotConfigured when no generator is wired,
// aiassist.ErrDisabled when the AI feature toggle is off.
func (t *Team) RunReviewer(ctx context.Context, tc TicketContext, draft string) (*ReviewerResult, error) {
	if t.gen == nil {
		return nil, aiassist.ErrNotConfigured
	}
	// Load settings once: gate on Enabled and reuse for ReplyInstructions.
	var replyGuidelines string
	if t.settings != nil {
		set, err := t.settings.Get()
		if err != nil {
			return nil, err
		}
		if !set.Enabled {
			return nil, aiassist.ErrDisabled
		}
		replyGuidelines = strings.TrimSpace(set.ReplyInstructions)
	}

	var b strings.Builder
	b.WriteString(memberInstructions["Reviewer"])
	b.WriteString("\n\n")
	b.WriteString(renderContext(tc))
	b.WriteString("\nDraft reply to review:\n")
	b.WriteString(draft)
	b.WriteString("\n")

	if replyGuidelines != "" {
		b.WriteString("\nTeam reply guidelines:\n")
		b.WriteString(replyGuidelines)
		b.WriteString("\n")
	}

	dataMap, valid, err := t.structured(ctx, "Reviewer", b.String(), reviewerSpec)
	if err != nil {
		return nil, err
	}
	if !valid || dataMap == nil {
		return &ReviewerResult{Issues: []ReviewIssue{}, Confidence: 0}, nil
	}

	var out ReviewerResult
	if !mapToStruct(dataMap, &out) {
		return &ReviewerResult{Issues: []ReviewIssue{}, Confidence: 0}, nil
	}
	if out.Issues == nil {
		out.Issues = []ReviewIssue{}
	}
	out.Confidence = clampConfidence(out.Confidence)
	return &out, nil
}

// RunDrafter runs the Drafter specialist for the given ticket context. It
// gathers KB snippets to provide context and asks the LLM to draft a reply.
// Returns aiassist.ErrNotConfigured when no generator is wired,
// aiassist.ErrDisabled when the AI feature toggle is off.
func (t *Team) RunDrafter(ctx context.Context, tc TicketContext) (*DrafterResult, error) {
	if t.gen == nil {
		return nil, aiassist.ErrNotConfigured
	}
	if t.settings != nil {
		set, err := t.settings.Get()
		if err != nil {
			return nil, err
		}
		if !set.Enabled {
			return nil, aiassist.ErrDisabled
		}
	}

	// Gather KB snippets (nil-safe).
	var kbSnippets []string
	if t.kb != nil {
		kbSnippets = t.kb.SnippetsFor(ctx, tc.Title+" "+tc.Description, 4)
	}

	var b strings.Builder
	b.WriteString(memberInstructions["Drafter"])
	b.WriteString("\n\n")
	b.WriteString(renderContext(tc))
	b.WriteString("\nKnowledge base context:\n")
	if len(kbSnippets) == 0 {
		b.WriteString("(no relevant articles)\n")
	} else {
		for _, s := range kbSnippets {
			b.WriteString("- " + s + "\n")
		}
	}

	dataMap, valid, err := t.structured(ctx, "Drafter", b.String(), drafterSpec)
	if err != nil {
		return nil, err
	}
	if !valid || dataMap == nil {
		return &DrafterResult{Confidence: 0}, nil
	}

	var out DrafterResult
	if !mapToStruct(dataMap, &out) {
		return &DrafterResult{Confidence: 0}, nil
	}
	out.Confidence = clampConfidence(out.Confidence)
	return &out, nil
}

// RunSentinel runs the Sentinel specialist for the given ticket context. Returns
// aiassist.ErrNotConfigured when no generator is wired, aiassist.ErrDisabled
// when the AI feature toggle is off. On an unparseable model response it
// returns a zero-value SentinelResult with Confidence 0 (graceful degradation).
func (t *Team) RunSentinel(ctx context.Context, tc TicketContext) (*SentinelResult, error) {
	if t.gen == nil {
		return nil, aiassist.ErrNotConfigured
	}
	if t.settings != nil {
		set, err := t.settings.Get()
		if err != nil {
			return nil, err
		}
		if !set.Enabled {
			return nil, aiassist.ErrDisabled
		}
	}

	prompt := memberInstructions["Sentinel"] + "\n\n" + renderContext(tc)
	dataMap, valid, err := t.structured(ctx, "Sentinel", prompt, sentinelSpec)
	if err != nil {
		return nil, err
	}
	if !valid || dataMap == nil {
		return &SentinelResult{Confidence: 0}, nil
	}

	var out SentinelResult
	if !mapToStruct(dataMap, &out) {
		return &SentinelResult{Confidence: 0}, nil
	}
	out.Confidence = clampConfidence(out.Confidence)
	return &out, nil
}
