package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMentions_Single(t *testing.T) {
	out := parseMentions("hey @alice, take a look")
	assert.Equal(t, []string{"alice"}, out)
}

func TestParseMentions_Multiple(t *testing.T) {
	out := parseMentions("@alice and @bob should review this")
	assert.ElementsMatch(t, []string{"alice", "bob"}, out)
}

func TestParseMentions_Dedupe(t *testing.T) {
	out := parseMentions("@alice @alice @bob @alice")
	assert.ElementsMatch(t, []string{"alice", "bob"}, out)
	// Each handle appears exactly once.
	assert.Len(t, out, 2)
}

func TestParseMentions_EmailNotMatched(t *testing.T) {
	// The domain part after @ in an email address must NOT be treated as a mention.
	out := parseMentions("send to alice@example.com for info")
	assert.Empty(t, out)
}

func TestParseMentions_EmailAndMentionMixed(t *testing.T) {
	out := parseMentions("@alice please email bob@example.com")
	assert.Equal(t, []string{"alice"}, out)
}

func TestParseMentions_PunctuationTrailing(t *testing.T) {
	out := parseMentions("hey @bob, how are you?")
	assert.Equal(t, []string{"bob"}, out)
}

func TestParseMentions_NoMentions(t *testing.T) {
	out := parseMentions("this is a plain message with no mentions")
	assert.Empty(t, out)
}

func TestParseMentions_AtStart(t *testing.T) {
	out := parseMentions("@charlie can you help?")
	assert.Equal(t, []string{"charlie"}, out)
}

func TestParseMentions_Lowercase(t *testing.T) {
	out := parseMentions("@Alice and @BOB")
	assert.ElementsMatch(t, []string{"alice", "bob"}, out)
}

func TestParseMentions_HandleWithDotsAndHyphens(t *testing.T) {
	out := parseMentions("@john.doe and @jane-smith")
	assert.ElementsMatch(t, []string{"john.doe", "jane-smith"}, out)
}

func TestParseMentions_HandleWithUnderscore(t *testing.T) {
	out := parseMentions("@super_admin look here")
	assert.Equal(t, []string{"super_admin"}, out)
}

func TestParseMentions_EmptyString(t *testing.T) {
	out := parseMentions("")
	assert.Empty(t, out)
}

func TestParseMentions_OnlyAtSign(t *testing.T) {
	out := parseMentions("@ not a mention")
	// The regex requires at least one word char after @; bare @ is not a match.
	assert.Empty(t, out)
}

func TestParseMentions_MultipleEmailsNoMentions(t *testing.T) {
	out := parseMentions("cc: alice@a.com, bob@b.io, charlie@c.net")
	assert.Empty(t, out)
}

func TestParseMentions_MentionAtEndOfLine(t *testing.T) {
	out := parseMentions("please review @dave")
	assert.Equal(t, []string{"dave"}, out)
}

func TestParseMentions_MultiLine(t *testing.T) {
	out := parseMentions("@alice\ncheck this out\n@bob please respond")
	assert.ElementsMatch(t, []string{"alice", "bob"}, out)
}
