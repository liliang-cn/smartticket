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

// ---- fakeKBSearcher ----

type fakeKBSearcher struct {
	snippets []string
}

func (f *fakeKBSearcher) SnippetsFor(_ context.Context, _ string, _ int) []string {
	return f.snippets
}

// ---- fakeSimilarSearcher ----

type fakeSimilarSearcher struct {
	tickets []SimilarTicket
	err     error
}

func (f *fakeSimilarSearcher) SearchSimilar(_ context.Context, _ string, _ int) ([]SimilarTicket, error) {
	return f.tickets, f.err
}

// ---- Researcher tests ----

func TestRunResearcher_AttachesKBAndSimilarTickets(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"suggested_resolution": "Reset the user's session token.",
				"confidence":           0.75,
			},
		},
	}
	kb := &fakeKBSearcher{snippets: []string{"How to reset password", "Account recovery steps"}}
	sim := &fakeSimilarSearcher{
		tickets: []SimilarTicket{
			{ID: 10, Title: "Login broken", Resolution: "Cache cleared", MergeCandidate: false, Score: 0.9},
		},
	}
	team := &Team{gen: gen, kb: kb, similar: sim}

	out, err := team.RunResearcher(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "Reset the user's session token.", out.SuggestedResolution)
	require.InDelta(t, 0.75, out.Confidence, 0.001)
	// KB citations attached by Go code
	require.Len(t, out.KBCitations, 2)
	require.Equal(t, "How to reset password", out.KBCitations[0].Title)
	// Similar tickets attached by Go code
	require.Len(t, out.SimilarTickets, 1)
	require.Equal(t, uint(10), out.SimilarTickets[0].ID)
}

func TestRunResearcher_NilKBAndSimilar_EmptyLists(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"suggested_resolution": "Check documentation.",
				"confidence":           0.5,
			},
		},
	}
	team := &Team{gen: gen, kb: nil, similar: nil}

	out, err := team.RunResearcher(context.Background(), sampleTC)
	require.NoError(t, err)
	require.NotNil(t, out.KBCitations)
	require.Len(t, out.KBCitations, 0)
	require.NotNil(t, out.SimilarTickets)
	require.Len(t, out.SimilarTickets, 0)
}

func TestRunResearcher_InvalidResult_GracefulZero(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{Valid: false, Data: nil, Raw: "not json"},
	}
	team := &Team{gen: gen}

	out, err := team.RunResearcher(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "", out.SuggestedResolution)
	require.Equal(t, float64(0), out.Confidence)
	require.NotNil(t, out.KBCitations)
	require.NotNil(t, out.SimilarTickets)
}

func TestRunResearcher_NilGen_ReturnsErrNotConfigured(t *testing.T) {
	team := &Team{gen: nil}
	_, err := team.RunResearcher(context.Background(), sampleTC)
	require.ErrorIs(t, err, aiassist.ErrNotConfigured)
}

func TestRunResearcher_GeneratorError_Propagates(t *testing.T) {
	sentinelErr := errors.New("llm down")
	gen := cannedGen{err: sentinelErr}
	team := &Team{gen: gen}
	_, err := team.RunResearcher(context.Background(), sampleTC)
	require.ErrorIs(t, err, sentinelErr)
}

func TestRunResearcher_ConfidenceClamped(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"suggested_resolution": "Try again.",
				"confidence":           1.8, // over 1 — must clamp
			},
		},
	}
	team := &Team{gen: gen}
	out, err := team.RunResearcher(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, float64(1), out.Confidence)
}

// ---- Reviewer tests ----

func TestRunReviewer_ParsesCannedData(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"issues": []interface{}{
					map[string]interface{}{
						"type":     "tone",
						"severity": "medium",
						"note":     "Reply sounds curt.",
					},
				},
				"revised_draft": "Thank you for reaching out. We will resolve this shortly.",
				"approve":       false,
				"confidence":    0.8,
			},
		},
	}
	team := &Team{gen: gen}

	out, err := team.RunReviewer(context.Background(), sampleTC, "We'll fix it.")
	require.NoError(t, err)
	require.False(t, out.Approve)
	require.Len(t, out.Issues, 1)
	require.Equal(t, "tone", out.Issues[0].Type)
	require.Equal(t, "medium", out.Issues[0].Severity)
	require.Equal(t, "Thank you for reaching out. We will resolve this shortly.", out.RevisedDraft)
	require.InDelta(t, 0.8, out.Confidence, 0.001)
}

func TestRunReviewer_InvalidResult_GracefulZero(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{Valid: false, Data: nil, Raw: "bad"},
	}
	team := &Team{gen: gen}

	out, err := team.RunReviewer(context.Background(), sampleTC, "draft")
	require.NoError(t, err)
	require.Equal(t, float64(0), out.Confidence)
	require.NotNil(t, out.Issues)
}

func TestRunReviewer_NilGen_ReturnsErrNotConfigured(t *testing.T) {
	team := &Team{gen: nil}
	_, err := team.RunReviewer(context.Background(), sampleTC, "draft")
	require.ErrorIs(t, err, aiassist.ErrNotConfigured)
}

func TestRunReviewer_GeneratorError_Propagates(t *testing.T) {
	sentinelErr := errors.New("timeout")
	gen := cannedGen{err: sentinelErr}
	team := &Team{gen: gen}
	_, err := team.RunReviewer(context.Background(), sampleTC, "draft")
	require.ErrorIs(t, err, sentinelErr)
}

func TestRunReviewer_ConfidenceClamped(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"issues":        []interface{}{},
				"revised_draft": "",
				"approve":       true,
				"confidence":    -0.2, // below 0 — must clamp
			},
		},
	}
	team := &Team{gen: gen}
	out, err := team.RunReviewer(context.Background(), sampleTC, "Looks good!")
	require.NoError(t, err)
	require.Equal(t, float64(0), out.Confidence)
}

// ---- Drafter tests ----

func TestRunDrafter_ParsesCannedData(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"reply":      "Hello, we have resolved your issue.",
				"confidence": 0.9,
			},
		},
	}
	team := &Team{gen: gen}

	out, err := team.RunDrafter(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "Hello, we have resolved your issue.", out.Reply)
	require.InDelta(t, 0.9, out.Confidence, 0.001)
}

func TestRunDrafter_WithKBSnippets(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"reply":      "Please follow the KB steps.",
				"confidence": 0.85,
			},
		},
	}
	kb := &fakeKBSearcher{snippets: []string{"Step 1: clear cache", "Step 2: re-login"}}
	team := &Team{gen: gen, kb: kb}

	out, err := team.RunDrafter(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "Please follow the KB steps.", out.Reply)
}

func TestRunDrafter_InvalidResult_GracefulZero(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{Valid: false, Data: nil, Raw: "not json"},
	}
	team := &Team{gen: gen}

	out, err := team.RunDrafter(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, "", out.Reply)
	require.Equal(t, float64(0), out.Confidence)
}

func TestRunDrafter_NilGen_ReturnsErrNotConfigured(t *testing.T) {
	team := &Team{gen: nil}
	_, err := team.RunDrafter(context.Background(), sampleTC)
	require.ErrorIs(t, err, aiassist.ErrNotConfigured)
}

func TestRunDrafter_GeneratorError_Propagates(t *testing.T) {
	sentinelErr := errors.New("provider error")
	gen := cannedGen{err: sentinelErr}
	team := &Team{gen: gen}
	_, err := team.RunDrafter(context.Background(), sampleTC)
	require.ErrorIs(t, err, sentinelErr)
}

func TestRunDrafter_ConfidenceClamped(t *testing.T) {
	gen := cannedGen{
		result: &domain.StructuredResult{
			Valid: true,
			Data: map[string]interface{}{
				"reply":      "Hi there.",
				"confidence": 1.5, // over 1 — must clamp
			},
		},
	}
	team := &Team{gen: gen}
	out, err := team.RunDrafter(context.Background(), sampleTC)
	require.NoError(t, err)
	require.Equal(t, float64(1), out.Confidence)
}
