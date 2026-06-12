package aiteam

import (
	"context"
	"errors"
	"testing"

	"github.com/company/smartticket/internal/aiassist"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
	"github.com/stretchr/testify/require"
)

// cannedGen extends fakeGen with configurable canned structured results.
type cannedGen struct {
	result *domain.StructuredResult
	err    error
}

func (c cannedGen) Generate(_ context.Context, _ string, _ *domain.GenerationOptions) (string, error) {
	return "", nil
}

func (c cannedGen) Stream(_ context.Context, _ string, _ *domain.GenerationOptions, cb func(string)) error {
	cb("")
	return nil
}

func (c cannedGen) GenerateWithTools(_ context.Context, _ []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions) (*domain.GenerationResult, error) {
	return &domain.GenerationResult{Content: "", Finished: true, FinishReason: "stop"}, nil
}

func (c cannedGen) StreamWithTools(_ context.Context, _ []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions, cb domain.ToolCallCallback) error {
	return cb(&domain.GenerationResult{Content: "", Finished: true, FinishReason: "stop"})
}

func (c cannedGen) GenerateStructured(_ context.Context, _ string, _ interface{}, _ *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if c.err != nil {
		return nil, c.err
	}
	if c.result != nil {
		return c.result, nil
	}
	return &domain.StructuredResult{Data: nil, Raw: "", Valid: false}, nil
}

func (c cannedGen) RecognizeIntent(_ context.Context, _ string) (*domain.IntentResult, error) {
	return &domain.IntentResult{Intent: domain.IntentAction, Confidence: 0.5}, nil
}

// sampleTC is a minimal TicketContext used across tests.
var sampleTC = TicketContext{
	TicketID:     42,
	Title:        "Cannot login",
	Description:  "Login page shows error after password reset.",
	CustomerName: "Alice",
	SLAState:     "warning: breach in 30m",
	Conversation: "Alice: My password reset didn't work.\nAgent: I'll look into it.",
}

// ---- Triage tests ----

func TestRunTriage_ParsesCannedData(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"priority":   "high",
				"severity":   "major",
				"category":   "account",
				"reasoning":  "Login is business-critical",
				"confidence": 0.8,
			},
		},
	}
	team := &Team{gen: gen}

	out, err := team.RunTriage(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "high", out.Priority)
	require.Equal(t, "major", out.Severity)
	require.Equal(t, "account", out.Category)
	require.InDelta(t, 0.8, out.Confidence, 0.001)
}

func TestRunTriage_InvalidResult_GracefulZero(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{Valid: false, Data: nil, Raw: "not json"},
	}
	team := &Team{gen: gen}

	out, err := team.RunTriage(context.Background(), sampleTC)
	require.NoError(t, err, "invalid result must not error")
	require.Equal(t, float64(0), out.Confidence)
	require.Equal(t, "", out.Priority)
}

func TestRunTriage_NilGen_ReturnsErrNotConfigured(t *testing.T) {
	team := &Team{gen: nil}

	_, err := team.RunTriage(context.Background(), sampleTC)
	require.ErrorIs(t, err, aiassist.ErrNotConfigured)
}

func TestRunTriage_GeneratorError_Propagates(t *testing.T) {
	sentinelErr := errors.New("llm unavailable")
	gen := cannedGen{err: sentinelErr}
	team := &Team{gen: gen}

	_, err := team.RunTriage(context.Background(), sampleTC)
	require.ErrorIs(t, err, sentinelErr)
}

func TestRunTriage_ConfidenceClamped(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"priority":   "low",
				"severity":   "trivial",
				"category":   "billing",
				"reasoning":  "low priority",
				"confidence": 1.5, // over 1 — must clamp
			},
		},
	}
	team := &Team{gen: gen}

	out, err := team.RunTriage(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, float64(1), out.Confidence)
}

// ---- Sentinel tests ----

func TestRunSentinel_ParsesCannedData(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"sentiment":       "angry",
				"churn_risk":      "high",
				"sla_breach_risk": true,
				"escalate":        true,
				"reasoning":       "Customer is frustrated and SLA is at risk",
				"confidence":      0.7,
			},
		},
	}
	team := &Team{gen: gen}

	out, err := team.RunSentinel(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "angry", out.Sentiment)
	require.Equal(t, "high", out.ChurnRisk)
	require.True(t, out.SLABreachRisk)
	require.True(t, out.Escalate)
	require.InDelta(t, 0.7, out.Confidence, 0.001)
}

func TestRunSentinel_InvalidResult_GracefulZero(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{Valid: false, Data: nil, Raw: "not json"},
	}
	team := &Team{gen: gen}

	out, err := team.RunSentinel(context.Background(), sampleTC)
	require.NoError(t, err, "invalid result must not error")
	require.Equal(t, float64(0), out.Confidence)
	require.False(t, out.Escalate)
}

func TestRunSentinel_NilGen_ReturnsErrNotConfigured(t *testing.T) {
	team := &Team{gen: nil}

	_, err := team.RunSentinel(context.Background(), sampleTC)
	require.ErrorIs(t, err, aiassist.ErrNotConfigured)
}

func TestRunSentinel_GeneratorError_Propagates(t *testing.T) {
	sentinelErr := errors.New("timeout")
	gen := cannedGen{err: sentinelErr}
	team := &Team{gen: gen}

	_, err := team.RunSentinel(context.Background(), sampleTC)
	require.ErrorIs(t, err, sentinelErr)
}

func TestRunSentinel_ConfidenceClamped(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"sentiment":       "neutral",
				"churn_risk":      "none",
				"sla_breach_risk": false,
				"escalate":        false,
				"reasoning":       "all ok",
				"confidence":      -0.5, // below 0 — must clamp
			},
		},
	}
	team := &Team{gen: gen}

	out, err := team.RunSentinel(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, float64(0), out.Confidence)
}

// ---- renderContext helper test ----

func TestRenderContext_IncludesFields(t *testing.T) {
	tc := TicketContext{
		TicketID:     7,
		Title:        "Test ticket",
		Description:  "A description",
		CustomerName: "Bob",
		SLAState:     "critical: breached",
		Conversation: "Bob: help\nAgent: on it",
	}
	out := renderContext(tc)
	require.Contains(t, out, "Ticket #7")
	require.Contains(t, out, "Test ticket")
	require.Contains(t, out, "A description")
	require.Contains(t, out, "Bob")
	require.Contains(t, out, "critical: breached")
	require.Contains(t, out, "Bob: help")
}

func TestRenderContext_EmptyOptionals(t *testing.T) {
	tc := TicketContext{TicketID: 1, Title: "Simple"}
	out := renderContext(tc)
	require.Contains(t, out, "Ticket #1")
	require.Contains(t, out, "Simple")
	require.NotContains(t, out, "Description:")
	require.NotContains(t, out, "Customer:")
	require.NotContains(t, out, "SLA Status:")
}

// ---- memberInstructions sanity check ----

func TestMemberInstructions_ContainsAllSpecialists(t *testing.T) {
	for _, name := range []string{"Triage", "Sentinel", "Researcher", "Reviewer", "Drafter"} {
		require.NotEmpty(t, memberInstructions[name], "missing instructions for %s", name)
	}
}
