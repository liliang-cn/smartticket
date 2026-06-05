package automation_test

import (
	"testing"

	"github.com/company/smartticket/internal/automation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEffector records every call made to it.
type fakeEffector struct {
	assigns    []assignCall
	tags       []tagCall
	fields     []fieldCall
	notifs     []notifCall
	emails     []emailCall
	escalates  []uint
	aiSuggests []uint
	aiReplies  []uint
	closes     []uint
}

type assignCall struct {
	ticketID uint
	userID   *uint
	teamID   *uint
}
type tagCall struct {
	ticketID uint
	tag      string
}
type fieldCall struct {
	ticketID uint
	field    string
	value    string
}
type notifCall struct {
	ticketID uint
	message  string
}
type emailCall struct {
	ticketID uint
	subject  string
	body     string
}

func (f *fakeEffector) Assign(ticketID uint, userID, teamID *uint) error {
	f.assigns = append(f.assigns, assignCall{ticketID, userID, teamID})
	return nil
}
func (f *fakeEffector) AddTag(ticketID uint, tag string) error {
	f.tags = append(f.tags, tagCall{ticketID, tag})
	return nil
}
func (f *fakeEffector) SetField(ticketID uint, field, value string) error {
	f.fields = append(f.fields, fieldCall{ticketID, field, value})
	return nil
}
func (f *fakeEffector) Notify(ticketID uint, message string) error {
	f.notifs = append(f.notifs, notifCall{ticketID, message})
	return nil
}
func (f *fakeEffector) SendEmail(ticketID uint, subject, body string) error {
	f.emails = append(f.emails, emailCall{ticketID, subject, body})
	return nil
}
func (f *fakeEffector) Escalate(ticketID uint) error {
	f.escalates = append(f.escalates, ticketID)
	return nil
}
func (f *fakeEffector) AISuggest(ticketID uint) error {
	f.aiSuggests = append(f.aiSuggests, ticketID)
	return nil
}
func (f *fakeEffector) AIAutoReply(ticketID uint) error {
	f.aiReplies = append(f.aiReplies, ticketID)
	return nil
}
func (f *fakeEffector) Close(ticketID uint) error {
	f.closes = append(f.closes, ticketID)
	return nil
}

func uid(v uint) *uint { return &v }

func TestExecutor_Assign(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(42, []automation.Action{
		{Type: "assign", Params: map[string]any{"user_id": float64(7)}},
	})
	require.NoError(t, err)
	require.Len(t, eff.assigns, 1)
	assert.Equal(t, uint(42), eff.assigns[0].ticketID)
	assert.Equal(t, uid(7), eff.assigns[0].userID)
	assert.Nil(t, eff.assigns[0].teamID)
}

func TestExecutor_AddTag(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(1, []automation.Action{
		{Type: "add_tag", Params: map[string]any{"tag": "urgent"}},
	})
	require.NoError(t, err)
	require.Len(t, eff.tags, 1)
	assert.Equal(t, "urgent", eff.tags[0].tag)
}

func TestExecutor_SetPriority(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(1, []automation.Action{
		{Type: "set_priority", Params: map[string]any{"value": "critical"}},
	})
	require.NoError(t, err)
	require.Len(t, eff.fields, 1)
	assert.Equal(t, "priority", eff.fields[0].field)
	assert.Equal(t, "critical", eff.fields[0].value)
}

func TestExecutor_SetStatus(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(1, []automation.Action{
		{Type: "set_status", Params: map[string]any{"value": "in_progress"}},
	})
	require.NoError(t, err)
	require.Len(t, eff.fields, 1)
	assert.Equal(t, "status", eff.fields[0].field)
}

func TestExecutor_SetSeverity(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(1, []automation.Action{
		{Type: "set_severity", Params: map[string]any{"value": "critical"}},
	})
	require.NoError(t, err)
	require.Len(t, eff.fields, 1)
	assert.Equal(t, "severity", eff.fields[0].field)
}

func TestExecutor_Notify(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(5, []automation.Action{
		{Type: "notify", Params: map[string]any{"message": "hello"}},
	})
	require.NoError(t, err)
	require.Len(t, eff.notifs, 1)
	assert.Equal(t, "hello", eff.notifs[0].message)
}

func TestExecutor_SendEmail(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(5, []automation.Action{
		{Type: "send_email", Params: map[string]any{"subject": "subj", "body": "bd"}},
	})
	require.NoError(t, err)
	require.Len(t, eff.emails, 1)
	assert.Equal(t, "subj", eff.emails[0].subject)
}

func TestExecutor_Escalate(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(9, []automation.Action{{Type: "escalate"}})
	require.NoError(t, err)
	assert.Equal(t, []uint{9}, eff.escalates)
}

func TestExecutor_Close(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(9, []automation.Action{{Type: "close"}})
	require.NoError(t, err)
	assert.Equal(t, []uint{9}, eff.closes)
}

func TestExecutor_AISuggest(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(9, []automation.Action{{Type: "ai_suggest"}})
	require.NoError(t, err)
	assert.Equal(t, []uint{9}, eff.aiSuggests)
}

func TestExecutor_AIAutoReply(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(9, []automation.Action{{Type: "ai_auto_reply"}})
	require.NoError(t, err)
	assert.Equal(t, []uint{9}, eff.aiReplies)
}

func TestExecutor_UnknownTypeSkipped(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	// unknown type should NOT error — just skip + log
	err := x.Run(1, []automation.Action{
		{Type: "unknown_future_action"},
		{Type: "close"}, // should still run
	})
	require.NoError(t, err)
	assert.Equal(t, []uint{1}, eff.closes)
}

func TestExecutor_MultipleActions(t *testing.T) {
	eff := &fakeEffector{}
	x := automation.NewExecutor(eff)
	err := x.Run(10, []automation.Action{
		{Type: "add_tag", Params: map[string]any{"tag": "vip"}},
		{Type: "set_priority", Params: map[string]any{"value": "high"}},
		{Type: "notify", Params: map[string]any{"message": "escalating"}},
	})
	require.NoError(t, err)
	assert.Len(t, eff.tags, 1)
	assert.Len(t, eff.fields, 1)
	assert.Len(t, eff.notifs, 1)
}
