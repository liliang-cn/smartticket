package aiassist

import (
	"context"
	"testing"

	"github.com/liliang-cn/agent-go/v2/pkg/domain"

	"github.com/company/smartticket/internal/llm"
)

type fakeChatter struct {
	gotRoles []string
	reply    string
}

func (f *fakeChatter) Chat(_ context.Context, msgs []llm.ChatMessage) (string, error) {
	for _, m := range msgs {
		f.gotRoles = append(f.gotRoles, m.Role)
	}
	return f.reply, nil
}

func TestGenerator_RoutesThroughChatter(t *testing.T) {
	f := &fakeChatter{reply: "Hi, try restarting the service."}
	g := NewGenerator(f)

	out, err := g.Generate(context.Background(), "draft a reply", nil)
	if err != nil || out != f.reply {
		t.Fatalf("Generate = %q, %v", out, err)
	}

	res, err := g.GenerateWithTools(context.Background(), []domain.Message{
		{Role: "system", Content: "you are support"},
		{Role: "user", Content: "printer broken"},
		{Role: "tool", Content: "kb hit"}, // unknown role folds to user
	}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateWithTools err: %v", err)
	}
	if res.Content != f.reply || !res.Finished {
		t.Fatalf("unexpected result: %+v", res)
	}
	// system/user preserved, tool folded to user
	want := []string{"user", "system", "user", "user"}
	if len(f.gotRoles) != len(want) {
		t.Fatalf("roles = %v, want %v", f.gotRoles, want)
	}
	for i := range want {
		if f.gotRoles[i] != want[i] {
			t.Fatalf("role[%d] = %q, want %q", i, f.gotRoles[i], want[i])
		}
	}
}
