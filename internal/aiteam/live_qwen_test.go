package aiteam

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/company/smartticket/internal/aiassist"
	"github.com/company/smartticket/internal/llm"
	"github.com/stretchr/testify/require"
)

// qwenChatter binds a model to an llm.Client so it satisfies aiassist.Chatter.
type qwenChatter struct {
	c     *llm.Client
	model string
}

func (q qwenChatter) Chat(ctx context.Context, msgs []llm.ChatMessage) (string, error) {
	return q.c.Chat(ctx, q.model, msgs)
}

func (q qwenChatter) ChatJSON(ctx context.Context, msgs []llm.ChatMessage) (string, error) {
	return q.c.ChatJSON(ctx, q.model, msgs)
}

// TestLive_QwenAgents drives all five advisory agents through the real
// TeamManager task queue + native StructuredOutput machinery against a live
// OpenAI-compatible endpoint (e.g. DashScope/qwen). It is skipped unless
// AITEAM_LIVE_KEY is set. Example:
//
//	AITEAM_LIVE_KEY=sk-xxx \
//	AITEAM_LIVE_BASE=https://dashscope.aliyuncs.com/compatible-mode/v1 \
//	AITEAM_LIVE_MODEL=qwen3.7-plus \
//	go test ./internal/aiteam/ -run TestLive_QwenAgents -v -timeout 300s
func TestLive_QwenAgents(t *testing.T) {
	key := os.Getenv("AITEAM_LIVE_KEY")
	if key == "" {
		t.Skip("set AITEAM_LIVE_KEY to run the live qwen integration test")
	}
	base := os.Getenv("AITEAM_LIVE_BASE")
	if base == "" {
		base = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	model := os.Getenv("AITEAM_LIVE_MODEL")
	if model == "" {
		model = "qwen3.7-plus"
	}

	gen := aiassist.NewGenerator(qwenChatter{c: llm.NewClient(base, key), model: model})
	team, err := NewTeam(filepath.Join(t.TempDir(), "team.db"), gen, nil, nil, nil)
	require.NoError(t, err)

	tc := TicketContext{
		TicketID:     1001,
		Title:        "Production API returning 500 after the latest deploy",
		Description:  "Since the 14:00 deploy our checkout endpoint returns HTTP 500 for ~30% of requests. This is blocking customer payments.",
		CustomerName: "Acme Corp",
		SLAState:     "warning: breach in 20m",
		Conversation: "Acme: Checkout is down for many users, we're losing sales.\nAgent: Investigating now.",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	dump := func(label string, v interface{}) {
		b, _ := json.MarshalIndent(v, "", "  ")
		t.Logf("\n--- %s ---\n%s", label, string(b))
	}

	triage, err := team.RunTriage(ctx, tc)
	require.NoError(t, err, "Triage")
	dump("Triage", triage)
	require.NotEmpty(t, triage.Priority, "Triage priority should be populated")
	require.Greater(t, triage.Confidence, 0.0, "Triage confidence should be > 0")

	sentinel, err := team.RunSentinel(ctx, tc)
	require.NoError(t, err, "Sentinel")
	dump("Sentinel", sentinel)
	require.NotEmpty(t, sentinel.Sentiment, "Sentinel sentiment should be populated")

	researcher, err := team.RunResearcher(ctx, tc)
	require.NoError(t, err, "Researcher")
	dump("Researcher", researcher)
	require.NotEmpty(t, researcher.SuggestedResolution, "Researcher resolution should be populated")

	drafter, err := team.RunDrafter(ctx, tc)
	require.NoError(t, err, "Drafter")
	dump("Drafter", drafter)
	require.NotEmpty(t, drafter.Reply, "Drafter reply should be populated")

	reviewer, err := team.RunReviewer(ctx, tc, drafter.Reply)
	require.NoError(t, err, "Reviewer")
	dump("Reviewer", reviewer)
}
