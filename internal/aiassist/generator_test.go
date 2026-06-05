package aiassist

import (
	"context"
	"strings"
	"testing"

	"github.com/liliang-cn/agent-go/v2/pkg/domain"

	"github.com/company/smartticket/internal/llm"
)

type fakeChatter struct {
	gotRoles    []string
	gotContents []string
	reply       string
}

func (f *fakeChatter) Chat(_ context.Context, msgs []llm.ChatMessage) (string, error) {
	for _, m := range msgs {
		f.gotRoles = append(f.gotRoles, m.Role)
		f.gotContents = append(f.gotContents, m.Content)
	}
	return f.reply, nil
}

// TestGenerateStructured_ParsesJSON verifies that GenerateStructured correctly
// extracts a JSON object embedded in surrounding text, unmarshals it, and
// returns Valid:true with the parsed fields.
func TestGenerateStructured_ParsesJSON(t *testing.T) {
	f := &fakeChatter{reply: `prefix text {"reply":"hi","confidence":0.9} suffix`}
	g := NewGenerator(f)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"reply":      map[string]interface{}{"type": "string"},
			"confidence": map[string]interface{}{"type": "number"},
		},
	}

	res, err := g.GenerateStructured(context.Background(), "draft a reply", schema, nil)
	if err != nil {
		t.Fatalf("GenerateStructured returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected Valid=true, got false; Raw=%q", res.Raw)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be map[string]interface{}, got %T", res.Data)
	}
	if data["reply"] != "hi" {
		t.Fatalf("expected reply=hi, got %v", data["reply"])
	}
	if data["confidence"] != 0.9 {
		t.Fatalf("expected confidence=0.9, got %v", data["confidence"])
	}
	// Ensure the schema was injected into the prompt sent to the chatter.
	if len(f.gotContents) == 0 {
		t.Fatal("chatter received no messages")
	}
	lastContent := f.gotContents[len(f.gotContents)-1]
	if !strings.Contains(lastContent, "JSON object") {
		t.Fatalf("prompt did not contain JSON instruction; got: %q", lastContent)
	}
}

// TestGenerateStructured_NonJSONReturnsInvalid verifies that a model returning
// free-form prose (no JSON object) yields Valid:false rather than lying.
func TestGenerateStructured_NonJSONReturnsInvalid(t *testing.T) {
	f := &fakeChatter{reply: "sorry, I cannot help with that"}
	g := NewGenerator(f)

	res, err := g.GenerateStructured(context.Background(), "draft", nil, nil)
	if err != nil {
		t.Fatalf("GenerateStructured returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected Valid=false for non-JSON response, got true; Raw=%q", res.Raw)
	}
	if res.Data != nil {
		t.Fatalf("expected Data=nil for invalid response, got %v", res.Data)
	}
}

// TestGenerateStructured_StripsCodeFence verifies that JSON wrapped in
// ```json ... ``` fences is still correctly parsed.
func TestGenerateStructured_StripsCodeFence(t *testing.T) {
	f := &fakeChatter{reply: "```json\n{\"reply\":\"hello\",\"confidence\":0.5}\n```"}
	g := NewGenerator(f)

	res, err := g.GenerateStructured(context.Background(), "draft", nil, nil)
	if err != nil {
		t.Fatalf("GenerateStructured returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected Valid=true after stripping fences; Raw=%q", res.Raw)
	}
	data := res.Data.(map[string]interface{})
	if data["reply"] != "hello" {
		t.Fatalf("expected reply=hello, got %v", data["reply"])
	}
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
