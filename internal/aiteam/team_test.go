package aiteam

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/liliang-cn/agent-go/v2/pkg/domain"
	"github.com/stretchr/testify/require"
)

// fakeGen is a no-op domain.Generator for tests that do not need LLM output.
type fakeGen struct{}

func (fakeGen) Generate(_ context.Context, _ string, _ *domain.GenerationOptions) (string, error) {
	return "", nil
}

func (fakeGen) Stream(_ context.Context, _ string, _ *domain.GenerationOptions, cb func(string)) error {
	cb("")
	return nil
}

func (fakeGen) GenerateWithTools(_ context.Context, _ []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions) (*domain.GenerationResult, error) {
	return &domain.GenerationResult{Content: "", Finished: true, FinishReason: "stop"}, nil
}

func (fakeGen) StreamWithTools(_ context.Context, _ []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions, cb domain.ToolCallCallback) error {
	return cb(&domain.GenerationResult{Content: "", Finished: true, FinishReason: "stop"})
}

func (fakeGen) GenerateStructured(_ context.Context, _ string, _ interface{}, _ *domain.GenerationOptions) (*domain.StructuredResult, error) {
	return &domain.StructuredResult{Data: nil, Raw: "", Valid: false}, nil
}

func (fakeGen) RecognizeIntent(_ context.Context, _ string) (*domain.IntentResult, error) {
	return &domain.IntentResult{Intent: domain.IntentAction, Confidence: 0.5}, nil
}

func TestTeamRegistersSpecialists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "team.db")
	team, err := NewTeam(dbPath, fakeGen{}, nil, nil, nil)
	require.NoError(t, err)

	members, err := team.Members()
	require.NoError(t, err)

	names := map[string]bool{}
	for _, m := range members {
		names[m.Name] = true
	}

	require.True(t, names["Triage"], "expected Triage specialist to be registered")
	require.True(t, names["Sentinel"], "expected Sentinel specialist to be registered")
	require.True(t, names["Researcher"], "expected Researcher specialist to be registered")
	require.True(t, names["Reviewer"], "expected Reviewer specialist to be registered")
	require.True(t, names["Drafter"], "expected Drafter specialist to be registered")
	require.GreaterOrEqual(t, len(members), 5, "expected at least 5 members")

	// Idempotent: building again must not duplicate.
	team2, err := NewTeam(dbPath, fakeGen{}, nil, nil, nil)
	require.NoError(t, err)
	members2, err := team2.Members()
	require.NoError(t, err)
	require.Equal(t, len(members), len(members2), "second NewTeam call must not duplicate members")
}
