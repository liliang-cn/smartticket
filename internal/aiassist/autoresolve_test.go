package aiassist

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// fakeActions is a test double for TicketActions.
// ----------------------------------------------------------------------------

type fakeActions struct {
	postedReplies   []string
	notifiedDrafts  []Draft
	classifications []classifyCall
	summaries       map[uint]string

	aiReplyCount int
	loadCtxFn    func(ticketID uint) (SuggestInput, bool, error)
	countAIFn    func(ticketID uint) (int, error)
}

type classifyCall struct {
	ticketID uint
	category string
	priority string
	tags     []string
}

func newFakeActions() *fakeActions {
	return &fakeActions{
		summaries: make(map[uint]string),
	}
}

func (f *fakeActions) PostAIReply(_ uint, body string) error {
	f.postedReplies = append(f.postedReplies, body)
	return nil
}

func (f *fakeActions) CountAIReplies(ticketID uint) (int, error) {
	if f.countAIFn != nil {
		return f.countAIFn(ticketID)
	}
	return f.aiReplyCount, nil
}

func (f *fakeActions) UpdateClassification(ticketID uint, category, priority string, tags []string) error {
	f.classifications = append(f.classifications, classifyCall{ticketID, category, priority, tags})
	return nil
}

func (f *fakeActions) SetSummary(ticketID uint, summary string) error {
	f.summaries[ticketID] = summary
	return nil
}

func (f *fakeActions) LoadContext(ticketID uint) (SuggestInput, bool, error) {
	if f.loadCtxFn != nil {
		return f.loadCtxFn(ticketID)
	}
	return SuggestInput{
		Title:       "Cannot log in",
		Description: "I forgot my password.",
		Conversation: []Turn{
			{Author: "Alice", IsCustomer: true, Content: "I forgot my password, please help."},
		},
	}, true, nil
}

func (f *fakeActions) NotifyAgentsSuggestion(_ uint, draft Draft) error {
	f.notifiedDrafts = append(f.notifiedDrafts, draft)
	return nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

// buildResolver builds an AutoResolver with the given chatter reply and settings
// modifier applied against an in-memory DB.
func buildResolver(t *testing.T, db *database.Database, reply string, mutateFn func(*UpdateSettings)) (*AutoResolver, *fakeActions) {
	t.Helper()
	require.NoError(t, db.DB.AutoMigrate(&models.AISettings{}))
	store := NewSettingsStore(db.DB)

	// Start from a fresh default and apply mutations.
	upd := UpdateSettings{}
	if mutateFn != nil {
		mutateFn(&upd)
	}
	_, err := store.Update(upd)
	require.NoError(t, err)

	gen := NewGenerator(&fakeChatter{reply: reply})
	a, err := NewAssistant(gen, nil, store, filepath.Join(t.TempDir(), "agentgo.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = a.Close() })

	actions := newFakeActions()
	return NewAutoResolver(a, store, actions), actions
}

// highConfidenceJSON is a canned structured reply the fakeChatter can return
// to make SuggestReplyStructured produce a draft with Confidence=0.95.
const highConfidenceJSON = `{"reply":"Please reset your password at Settings > Security.","confidence":0.95,"needs_clarification":false,"used_kb":false,"sources":[]}`

// lowConfidenceJSON produces a draft with Confidence=0.30.
const lowConfidenceJSON = `{"reply":"I need more details to help you.","confidence":0.30,"needs_clarification":false,"used_kb":false,"sources":[]}`

// clarifyJSON produces a draft where NeedsClarification=true.
const clarifyJSON = `{"reply":"Could you tell me which product you are using?","confidence":0.80,"needs_clarification":true,"used_kb":false,"sources":[]}`

// ----------------------------------------------------------------------------
// OnMessageCreated tests
// ----------------------------------------------------------------------------

// TestOnMessageCreated_AutoReply verifies that when confidence >= threshold
// and AutoReplyEnabled is true and under the cap, PostAIReply is called once.
func TestOnMessageCreated_AutoReply(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "", // human origin
		})

		require.Len(t, actions.postedReplies, 1, "expected one AI reply")
		assert.Contains(t, actions.postedReplies[0], "reset your password")
		assert.Empty(t, actions.notifiedDrafts, "no agent notification expected when auto-posting")
	})
}

// TestOnMessageCreated_LowConfidenceSuggests verifies that when confidence <
// threshold the orchestrator notifies agents instead of auto-posting.
func TestOnMessageCreated_LowConfidenceSuggests(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, lowConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75 // threshold higher than 0.30
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "",
		})

		assert.Empty(t, actions.postedReplies, "PostAIReply must NOT be called")
		require.Len(t, actions.notifiedDrafts, 1, "expected agent suggestion notification")
	})
}

// TestOnMessageCreated_AutoReplyDisabledSuggests verifies that when
// AutoReplyEnabled=false we always fall back to agent suggestion.
func TestOnMessageCreated_AutoReplyDisabledSuggests(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// AutoReplyEnabled defaults to false; high-confidence reply should still
		// yield a suggestion (not an auto-post).
		resolver, actions := buildResolver(t, db, highConfidenceJSON, nil)

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "",
		})

		assert.Empty(t, actions.postedReplies)
		require.Len(t, actions.notifiedDrafts, 1)
	})
}

// TestOnMessageCreated_AtCapHandsToHuman verifies that when CountAIReplies
// returns >= MaxAutoRepliesPerTicket, we notify agents and do NOT post.
func TestOnMessageCreated_AtCapHandsToHuman(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75
			cap := 2
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
			u.MaxAutoRepliesPerTicket = &cap
		})
		// Simulate that 2 AI replies already exist (== cap).
		actions.aiReplyCount = 2

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "",
		})

		assert.Empty(t, actions.postedReplies, "must not post when at cap")
		require.Len(t, actions.notifiedDrafts, 1, "expected hand-to-human notification")
	})
}

// TestOnMessageCreated_GloballyDisabled verifies that when Enabled=false
// nothing is called.
func TestOnMessageCreated_GloballyDisabled(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			off := false
			u.Enabled = &off
		})

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "",
		})

		assert.Empty(t, actions.postedReplies)
		assert.Empty(t, actions.notifiedDrafts)
	})
}

// TestOnMessageCreated_LoopGuard verifies that events with Source!="" are
// silently ignored (prevents AI → AI message loops).
func TestOnMessageCreated_LoopGuard(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.5
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})

		// Simulate an AI-originated event.
		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "ai",
		})

		assert.Empty(t, actions.postedReplies, "loop guard: AI events must be ignored")
		assert.Empty(t, actions.notifiedDrafts)
	})
}

// TestOnMessageCreated_NeedsClarificationSuggests verifies that even with
// high confidence, a clarification-needed draft goes to agent suggestion.
func TestOnMessageCreated_NeedsClarificationSuggests(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, clarifyJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "",
		})

		assert.Empty(t, actions.postedReplies, "clarification drafts must not be auto-posted")
		require.Len(t, actions.notifiedDrafts, 1)
	})
}

// TestOnMessageCreated_NotCustomerWaiting verifies that when LoadContext
// returns customerWaiting=false the orchestrator takes no action.
func TestOnMessageCreated_NotCustomerWaiting(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})
		// Override LoadContext to say no customer is waiting.
		actions.loadCtxFn = func(_ uint) (SuggestInput, bool, error) {
			return SuggestInput{Title: "test"}, false, nil
		}

		resolver.OnMessageCreated(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 1,
			Source:   "",
		})

		assert.Empty(t, actions.postedReplies)
		assert.Empty(t, actions.notifiedDrafts)
	})
}

// ----------------------------------------------------------------------------
// OnTicketResolved tests
// ----------------------------------------------------------------------------

// TestOnTicketResolved_Summarize verifies that AutoSummarizeOnResolve=true
// triggers SetSummary with a non-empty string.
func TestOnTicketResolved_Summarize(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, "This ticket was resolved by resetting the password.", func(u *UpdateSettings) {
			on := true
			u.AutoSummarizeOnResolve = &on
		})

		resolver.OnTicketResolved(automation.Event{
			Type:     automation.EventTicketResolved,
			TicketID: 42,
		})

		sum, ok := actions.summaries[42]
		require.True(t, ok, "SetSummary should have been called for ticket 42")
		assert.NotEmpty(t, sum)
	})
}

// TestOnTicketResolved_SummarizeDisabled verifies no summary is written when
// AutoSummarizeOnResolve=false (the default).
func TestOnTicketResolved_SummarizeDisabled(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, "some reply", nil)

		resolver.OnTicketResolved(automation.Event{
			Type:     automation.EventTicketResolved,
			TicketID: 7,
		})

		_, ok := actions.summaries[7]
		assert.False(t, ok, "SetSummary must not be called when disabled")
	})
}

// ----------------------------------------------------------------------------
// OnTicketCreated + AutoClassify tests
// ----------------------------------------------------------------------------

// TestOnTicketCreated_Classify verifies that AutoClassify=true triggers
// UpdateClassification.
func TestOnTicketCreated_Classify(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		classifyReply := `{"category":"login","priority":"high","tags":["auth","password"]}`
		resolver, actions := buildResolver(t, db, classifyReply, func(u *UpdateSettings) {
			on := true
			u.AutoClassify = &on
		})

		resolver.OnTicketCreated(automation.Event{
			Type:     automation.EventTicketCreated,
			TicketID: 10,
		})

		require.Len(t, actions.classifications, 1)
		assert.Equal(t, "login", actions.classifications[0].category)
		assert.Equal(t, "high", actions.classifications[0].priority)
	})
}

// TestOnTicketCreated_LoopSourceEmpty verifies that OnTicketCreated with
// a human-originated event (Source="") flows through normally.
func TestOnTicketCreated_LoopSourceEmpty(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})

		resolver.OnTicketCreated(automation.Event{
			Type:     automation.EventTicketCreated,
			TicketID: 5,
			Source:   "",
		})

		// Should auto-post because confidence is high and auto-reply is on.
		require.Len(t, actions.postedReplies, 1)
	})
}

// ----------------------------------------------------------------------------
// Subscribe integration smoke test
// ----------------------------------------------------------------------------

func TestAutoResolver_Subscribe_WiresHandlers(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		resolver, actions := buildResolver(t, db, highConfidenceJSON, func(u *UpdateSettings) {
			on := true
			conf := 0.75
			u.AutoReplyEnabled = &on
			u.AutoReplyConfidence = &conf
		})

		bus := automation.NewBus()
		resolver.Subscribe(bus)

		bus.Publish(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: 99,
			Source:   "",
		})
		require.Len(t, actions.postedReplies, 1)
	})
}

// TestClassify_InvalidPriorityIgnored verifies that an unknown priority
// returned by the model is discarded (empty string stored instead).
func TestClassify_InvalidPriorityIgnored(t *testing.T) {
	ctx := context.Background()
	f := &fakeChatter{reply: `{"category":"billing","priority":"urgent","tags":[]}`}
	g := NewGenerator(f)
	ar := &AutoResolver{gen: g}

	cat, prio, tags, err := ar.classify(ctx, SuggestInput{Title: "Overcharged", Description: "I was billed twice."})
	require.NoError(t, err)
	assert.Equal(t, "billing", cat)
	assert.Equal(t, "", prio, "unknown priority 'urgent' should be discarded")
	assert.Empty(t, tags)
}
