package aiteam

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/aiassist"
	"github.com/company/smartticket/internal/models"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- fakes ----

// captureBroadcaster records every Broadcast call for test assertions.
type captureBroadcaster struct {
	mu   sync.Mutex
	msgs []broadcastMsg
}

type broadcastMsg struct {
	room    string
	payload []byte
}

func (c *captureBroadcaster) Broadcast(room string, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, broadcastMsg{room: room, payload: payload})
}

func (c *captureBroadcaster) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.msgs)
}

func (c *captureBroadcaster) last() *broadcastMsg {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.msgs) == 0 {
		return nil
	}
	m := c.msgs[len(c.msgs)-1]
	return &m
}

// successData is a single map that satisfies every specialist schema — fields
// not present in a given schema are simply ignored by that agent's parser.
var successData = map[string]interface{}{
	"priority":             "high",
	"severity":             "major",
	"category":             "account",
	"reasoning":            "test",
	"confidence":           0.75,
	"sentiment":            "neutral",
	"churn_risk":           "none",
	"sla_breach_risk":      false,
	"escalate":             false,
	"suggested_resolution": "test resolution",
	"issues":               []interface{}{},
	"revised_draft":        "",
	"approve":              true,
	"reply":                "test reply",
}

func successText() string {
	b, _ := json.Marshal(successData)
	return string(b)
}

// successGen drives the agent-go task path: every generation method emits the
// combined successData JSON so the StructuredOutput lint validates first try.
type successGen struct{}

func (successGen) Generate(_ context.Context, _ string, _ *domain.GenerationOptions) (string, error) {
	return successText(), nil
}
func (successGen) Stream(_ context.Context, _ string, _ *domain.GenerationOptions, cb func(string)) error {
	cb(successText())
	return nil
}
func (successGen) GenerateWithTools(_ context.Context, _ []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions) (*domain.GenerationResult, error) {
	return &domain.GenerationResult{Content: successText(), Finished: true, FinishReason: "stop"}, nil
}
func (successGen) StreamWithTools(_ context.Context, _ []domain.Message, _ []domain.ToolDefinition, _ *domain.GenerationOptions, cb domain.ToolCallCallback) error {
	return cb(&domain.GenerationResult{Content: successText(), Finished: true, FinishReason: "stop"})
}
func (successGen) GenerateStructured(_ context.Context, _ string, _ interface{}, _ *domain.GenerationOptions) (*domain.StructuredResult, error) {
	return &domain.StructuredResult{Valid: true, Data: successData}, nil
}
func (successGen) RecognizeIntent(_ context.Context, _ string) (*domain.IntentResult, error) {
	return &domain.IntentResult{Intent: domain.IntentAction, Confidence: 0.5}, nil
}

// ---- DB helpers ----

// newOrchDB returns a fresh in-memory gorm.DB with AISettings + AISuggestion
// migrated, ready for orchestrator tests.
func newOrchDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AISettings{}, &models.AISuggestion{}))
	return db
}

// upsertSettings writes the singleton AISettings row using the SettingsStore
// so GORM's default-value handling is bypassed (we save all fields explicitly).
func upsertSettings(t *testing.T, db *gorm.DB, s models.AISettings) {
	t.Helper()
	ss := aiassist.NewSettingsStore(db)
	// Trigger creation of the default row first, then overwrite it.
	existing, err := ss.Get()
	require.NoError(t, err)
	s.ID = existing.ID
	// Use db.Save which writes all fields; GORM respects the struct values.
	require.NoError(t, db.Save(&s).Error)
}

// buildOrchestrator creates a Team (in-memory agent-go store), SuggestionStore,
// and Orchestrator wired with the given generator and DB.
func buildOrchestrator(t *testing.T, gen domain.Generator, db *gorm.DB, bc Broadcaster) (*Orchestrator, *SuggestionStore) {
	t.Helper()
	settings := aiassist.NewSettingsStore(db)
	team, err := NewTeam(filepath.Join(t.TempDir(), "team.db"), gen, nil, settings, db)
	require.NoError(t, err)
	store := NewSuggestionStore(db)
	orch := NewOrchestrator(team, store, bc)
	return orch, store
}

// ---- Tests ----

func TestOrchestrator_MasterDisabled_NoRun(t *testing.T) {
	db := newOrchDB(t)
	upsertSettings(t, db, models.AISettings{
		Enabled:        false,
		TriageEnabled:  true,
		SentinelEnabled: true,
	})

	bc := &captureBroadcaster{}
	orch, store := buildOrchestrator(t, successGen{}, db, bc)

	result, err := orch.Run(context.Background(), "Triage", sampleTC, "")
	require.NoError(t, err)
	require.Nil(t, result, "master disabled: Run must return nil")
	require.Equal(t, 0, bc.count(), "master disabled: no broadcast expected")

	// No row should exist in the store.
	list, err := store.List(sampleTC.TicketID)
	require.NoError(t, err)
	require.Empty(t, list, "master disabled: no suggestion should be persisted")
}

func TestOrchestrator_TriageToggleOff_NoRun(t *testing.T) {
	db := newOrchDB(t)
	upsertSettings(t, db, models.AISettings{
		Enabled:       true,
		TriageEnabled: false,
	})

	bc := &captureBroadcaster{}
	orch, store := buildOrchestrator(t, successGen{}, db, bc)

	result, err := orch.Run(context.Background(), "Triage", sampleTC, "")
	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, 0, bc.count())

	list, _ := store.List(sampleTC.TicketID)
	require.Empty(t, list)
}

func TestOrchestrator_SentinelToggleOff_NoRun(t *testing.T) {
	db := newOrchDB(t)
	upsertSettings(t, db, models.AISettings{
		Enabled:         true,
		SentinelEnabled: false,
	})

	bc := &captureBroadcaster{}
	orch, store := buildOrchestrator(t, successGen{}, db, bc)

	result, err := orch.Run(context.Background(), "Sentinel", sampleTC, "")
	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, 0, bc.count())

	list, _ := store.List(sampleTC.TicketID)
	require.Empty(t, list)
}

func TestOrchestrator_TriageSuccess_PersistsAndBroadcasts(t *testing.T) {
	db := newOrchDB(t)
	upsertSettings(t, db, models.AISettings{
		Enabled:       true,
		TriageEnabled: true,
	})

	bc := &captureBroadcaster{}
	orch, store := buildOrchestrator(t, successGen{}, db, bc)

	tc := TicketContext{TicketID: 99, Title: "Login broken", Description: "Cannot login"}
	result, err := orch.Run(context.Background(), "Triage", tc, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "done", result.Status)
	require.Equal(t, "Triage", result.AgentName)
	require.Greater(t, result.Confidence, float64(0))

	// Persisted in store.
	list, err := store.List(99)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "done", list[0].Status)

	// Broadcast happened.
	require.Equal(t, 1, bc.count())
	last := bc.last()
	require.Equal(t, "ticket:99", last.room)
	require.Contains(t, string(last.payload), "ai_suggestion")
	require.Contains(t, string(last.payload), "Triage")
}

func TestOrchestrator_SentinelThrottle_SecondCallSkipped(t *testing.T) {
	db := newOrchDB(t)
	// Set throttle to 60s so a 2nd immediate call is blocked.
	upsertSettings(t, db, models.AISettings{
		Enabled:             true,
		SentinelEnabled:     true,
		SentinelThrottleSec: 60,
	})

	bc := &captureBroadcaster{}
	orch, store := buildOrchestrator(t, successGen{}, db, bc)

	tc := TicketContext{TicketID: 77, Title: "Escalation risk"}

	// First run: should succeed.
	r1, err := orch.Run(context.Background(), "Sentinel", tc, "")
	require.NoError(t, err)
	require.NotNil(t, r1, "first Sentinel run should produce a suggestion")
	require.Equal(t, 1, bc.count(), "first run: one broadcast")

	// Record the UpdatedAt timestamp after first run.
	firstUpdated := r1.UpdatedAt

	// Second run immediately: should be throttled (within 60s).
	r2, err := orch.Run(context.Background(), "Sentinel", tc, "")
	require.NoError(t, err)
	require.Nil(t, r2, "throttled: second Sentinel run should return nil")
	require.Equal(t, 1, bc.count(), "throttled: broadcaster should not be called a second time")

	// Suggestion in store should still have the original UpdatedAt (not refreshed).
	list, err := store.List(77)
	require.NoError(t, err)
	require.Len(t, list, 1)
	// The row was NOT upserted again — UpdatedAt is unchanged (within 1s tolerance).
	require.WithinDuration(t, firstUpdated, list[0].UpdatedAt, time.Second)
}

func TestOrchestrator_NilBroadcaster_NocrashOnSuccess(t *testing.T) {
	db := newOrchDB(t)
	upsertSettings(t, db, models.AISettings{Enabled: true, TriageEnabled: true})

	// Pass nil broadcaster — must not panic.
	orch, _ := buildOrchestrator(t, successGen{}, db, nil)
	tc := TicketContext{TicketID: 55, Title: "Nil BC test"}

	result, err := orch.Run(context.Background(), "Triage", tc, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "done", result.Status)
}

func TestOrchestrator_NilSettings_RunsUnguarded(t *testing.T) {
	// When settings is nil (e.g. early startup), the orchestrator should still
	// execute the agent (no gating applied).
	db := newOrchDB(t)
	store := NewSuggestionStore(db)
	// Construct team WITHOUT a settings store.
	team, err := NewTeam(filepath.Join(t.TempDir(), "team.db"), successGen{}, nil, nil, db)
	require.NoError(t, err)
	bc := &captureBroadcaster{}
	orch := NewOrchestrator(team, store, bc)

	tc := TicketContext{TicketID: 11, Title: "No settings"}
	result, err := orch.Run(context.Background(), "Triage", tc, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "done", result.Status)
}
