package email

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInboundNormalize_FlatStringFrom(t *testing.T) {
	p := inboundPayload{
		From:    json.RawMessage(`"Jane Roe <jane@acme.com>"`),
		Subject: "Cannot log in",
		Text:    "  It says invalid password.  ",
	}
	in := p.normalize()
	assert.Equal(t, "jane@acme.com", in.FromEmail)
	assert.Equal(t, "Jane Roe", in.FromName)
	assert.Equal(t, "Cannot log in", in.Subject)
	assert.Equal(t, "It says invalid password.", in.Text)
}

func TestInboundNormalize_ResendEnvelopeObjectFromHTMLFallback(t *testing.T) {
	p := inboundPayload{
		Data: &inboundPayload{
			From:    json.RawMessage(`{"name":"Bob","email":"bob@example.com"}`),
			Subject: "Re: [TK-7] Help",
			HTML:    "<p>Still <b>broken</b></p>",
			Headers: map[string]string{"in-reply-to": "<abc@mail>"},
		},
	}
	in := p.normalize()
	assert.Equal(t, "bob@example.com", in.FromEmail)
	assert.Equal(t, "Bob", in.FromName)
	assert.Equal(t, "Re: [TK-7] Help", in.Subject)
	assert.Equal(t, "Still broken", in.Text) // HTML stripped for the text fallback
	assert.Equal(t, "<abc@mail>", in.InReplyTo)
}

func TestInboundNormalize_PlainAddressString(t *testing.T) {
	p := inboundPayload{From: json.RawMessage(`"solo@x.io"`), Subject: "Hi"}
	in := p.normalize()
	assert.Equal(t, "solo@x.io", in.FromEmail)
	assert.Equal(t, "", in.FromName)
}
