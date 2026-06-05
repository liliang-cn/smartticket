package aiassist

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/require"
)

// fakeKBSearcher implements KBSearcher returning a fixed list of snippets.
type fakeKBSearcher struct {
	snippets []string
}

func (f *fakeKBSearcher) SnippetsFor(_ context.Context, _ string, _ int) []string {
	return f.snippets
}

// Builds the real agent.Service (with the KB tool registered) on a fake LLM and
// verifies a run completes and returns a draft — proving the agent runtime,
// tool registration and the BYO-LLM adapter integrate end-to-end. Tool-calling
// *behavior* is model-driven and exercised with a live provider, not here.
func TestAssistant_RunsAgentEndToEnd(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		require.NoError(t, db.DB.AutoMigrate(&models.AISettings{}))
		settings := NewSettingsStore(db.DB)

		gen := NewGenerator(&fakeChatter{reply: "Hi Dana — you can reset your password from Settings → Security."})
		a, err := NewAssistant(gen, nil, settings, filepath.Join(t.TempDir(), "agentgo.db"))
		require.NoError(t, err)
		defer a.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		out, err := a.SuggestReply(ctx, SuggestInput{
			Title:       "Cannot login",
			Description: "I forgot my password and can't get in.",
			Conversation: []Turn{
				{Author: "Dana", IsCustomer: true, Content: "I forgot my password, please help."},
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, out, "agent should return a non-empty draft")
	})
}

func TestAssistant_GatedBySettings(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		require.NoError(t, db.DB.AutoMigrate(&models.AISettings{}))
		settings := NewSettingsStore(db.DB)
		off := false
		_, err := settings.Update(UpdateSettings{SuggestReplies: &off})
		require.NoError(t, err)

		a, err := NewAssistant(NewGenerator(&fakeChatter{reply: "x"}), nil, settings, filepath.Join(t.TempDir(), "agentgo.db"))
		require.NoError(t, err)
		defer a.Close()

		_, err = a.SuggestReply(context.Background(), SuggestInput{Title: "x"})
		require.ErrorIs(t, err, ErrDisabled)
	})
}

// TestAssistant_SuggestReplyStructured_ParsesDraft verifies that when the fake
// chatter returns a valid JSON Draft payload, SuggestReplyStructured returns a
// parsed Draft with the correct fields and UsedKB==true (because we supply a
// KB searcher that returns a snippet).
func TestAssistant_SuggestReplyStructured_ParsesDraft(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		require.NoError(t, db.DB.AutoMigrate(&models.AISettings{}))
		settings := NewSettingsStore(db.DB)

		replyJSON := `{"reply":"Please restart the service.","confidence":0.85,"needs_clarification":false,"used_kb":true,"sources":["How to restart"]}`
		chatter := &fakeChatter{reply: replyJSON}
		kb := &fakeKBSearcher{snippets: []string{"How to restart: run service restart"}}

		a, err := NewAssistant(NewGenerator(chatter), kb, settings, filepath.Join(t.TempDir(), "agentgo.db"))
		require.NoError(t, err)
		defer a.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		draft, err := a.SuggestReplyStructured(ctx, SuggestInput{
			Title:       "Service won't start",
			Description: "The service fails on boot.",
		})
		require.NoError(t, err)
		require.Equal(t, "Please restart the service.", draft.Reply)
		require.InDelta(t, 0.85, draft.Confidence, 0.001)
		require.False(t, draft.NeedsClarification)
		require.True(t, draft.UsedKB)
		require.Equal(t, []string{"How to restart"}, draft.Sources)
	})
}

// TestAssistant_SuggestReplyStructured_DisabledGate verifies that when
// SuggestReplies is turned off, SuggestReplyStructured returns ErrDisabled.
func TestAssistant_SuggestReplyStructured_DisabledGate(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		require.NoError(t, db.DB.AutoMigrate(&models.AISettings{}))
		settings := NewSettingsStore(db.DB)
		off := false
		_, err := settings.Update(UpdateSettings{SuggestReplies: &off})
		require.NoError(t, err)

		a, err := NewAssistant(NewGenerator(&fakeChatter{reply: "x"}), nil, settings, filepath.Join(t.TempDir(), "agentgo.db"))
		require.NoError(t, err)
		defer a.Close()

		_, err = a.SuggestReplyStructured(context.Background(), SuggestInput{Title: "x"})
		require.ErrorIs(t, err, ErrDisabled)
	})
}

// TestAssistant_SuggestReplyStructured_Fallback verifies graceful degradation:
// when the model returns non-JSON prose, SuggestReplyStructured still returns a
// Draft (no error) with the raw text as Reply and Confidence==0.
func TestAssistant_SuggestReplyStructured_Fallback(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		require.NoError(t, db.DB.AutoMigrate(&models.AISettings{}))
		settings := NewSettingsStore(db.DB)

		chatter := &fakeChatter{reply: "I am sorry, I cannot help right now."}

		a, err := NewAssistant(NewGenerator(chatter), nil, settings, filepath.Join(t.TempDir(), "agentgo.db"))
		require.NoError(t, err)
		defer a.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		draft, err := a.SuggestReplyStructured(ctx, SuggestInput{
			Title:       "Login issue",
			Description: "Cannot log in.",
		})
		require.NoError(t, err, "fallback should not error")
		require.NotEmpty(t, draft.Reply, "fallback reply should contain raw text")
		require.Equal(t, float64(0), draft.Confidence, "fallback confidence should be 0")
	})
}
