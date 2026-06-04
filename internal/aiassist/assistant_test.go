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
