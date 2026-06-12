package aiteam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/company/smartticket/internal/aiassist"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
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
	res, err := t.gen.GenerateStructured(ctx, prompt, triageSchema, &domain.GenerationOptions{Temperature: 0.3})
	if err != nil {
		return nil, err
	}

	if !res.Valid || res.Data == nil {
		return &TriageResult{Confidence: 0}, nil
	}

	dataMap, ok := res.Data.(map[string]interface{})
	if !ok {
		return &TriageResult{Confidence: 0}, nil
	}

	var out TriageResult
	if !mapToStruct(dataMap, &out) {
		return &TriageResult{Confidence: 0}, nil
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
	res, err := t.gen.GenerateStructured(ctx, prompt, sentinelSchema, &domain.GenerationOptions{Temperature: 0.3})
	if err != nil {
		return nil, err
	}

	if !res.Valid || res.Data == nil {
		return &SentinelResult{Confidence: 0}, nil
	}

	dataMap, ok := res.Data.(map[string]interface{})
	if !ok {
		return &SentinelResult{Confidence: 0}, nil
	}

	var out SentinelResult
	if !mapToStruct(dataMap, &out) {
		return &SentinelResult{Confidence: 0}, nil
	}
	out.Confidence = clampConfidence(out.Confidence)
	return &out, nil
}
