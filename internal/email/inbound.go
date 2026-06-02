package email

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// InboundEmail is a normalized inbound message handed to the ticket system.
type InboundEmail struct {
	FromName  string
	FromEmail string
	Subject   string
	Text      string
	MessageID string
	InReplyTo string
}

// TicketSink routes an inbound email into the ticket system (append to an
// existing ticket, or open a new one). Implemented by ticket.Service.
type TicketSink interface {
	IngestEmail(ctx context.Context, in InboundEmail) error
}

// InboundHandler is the public webhook that turns inbound email into tickets.
// It is not behind JWT, so it is authenticated by a shared secret presented as
// the `X-Webhook-Secret` header or `?secret=` query parameter.
type InboundHandler struct {
	sink   TicketSink
	secret string
}

// NewInboundHandler builds the webhook handler.
func NewInboundHandler(sink TicketSink, secret string) *InboundHandler {
	return &InboundHandler{sink: sink, secret: secret}
}

// Handle accepts a Resend Inbound payload (or any compatible JSON) and ingests it.
func (h *InboundHandler) Handle(c *gin.Context) {
	if !h.authorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "unauthorized"}})
		return
	}

	var p inboundPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid payload"}})
		return
	}
	in := p.normalize()
	if in.FromEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "missing sender"}})
		return
	}

	if err := h.sink.IngestEmail(c.Request.Context(), in); err != nil {
		logger.Error("email: inbound ingest failed", zap.String("from", in.FromEmail), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "ingest failed"}})
		return
	}
	logger.Info("email: inbound ingested", zap.String("from", in.FromEmail), zap.String("subject", in.Subject))
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *InboundHandler) authorized(c *gin.Context) bool {
	if h.secret == "" {
		return false
	}
	got := c.GetHeader("X-Webhook-Secret")
	if got == "" {
		got = c.Query("secret")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) == 1
}

// inboundPayload tolerates both a flat shape and Resend's `{type, data:{...}}`
// envelope, and accepts `from` as either a string or an object.
type inboundPayload struct {
	From    json.RawMessage   `json:"from"`
	Subject string            `json:"subject"`
	Text    string            `json:"text"`
	HTML    string            `json:"html"`
	Headers map[string]string `json:"headers"`

	// Resend-style envelope.
	Data *inboundPayload `json:"data"`
}

func (p inboundPayload) normalize() InboundEmail {
	// Unwrap the Resend envelope: prefer `data` when present.
	if p.Data != nil {
		merged := *p.Data
		if merged.Subject == "" {
			merged.Subject = p.Subject
		}
		return merged.normalize()
	}

	name, email := parseFrom(p.From)
	text := strings.TrimSpace(p.Text)
	if text == "" && p.HTML != "" {
		text = stripHTML(p.HTML)
	}
	return InboundEmail{
		FromName:  name,
		FromEmail: strings.ToLower(strings.TrimSpace(email)),
		Subject:   strings.TrimSpace(p.Subject),
		Text:      text,
		MessageID: p.Headers["message-id"],
		InReplyTo: p.Headers["in-reply-to"],
	}
}

// parseFrom extracts a display name + address from `from`, which may be a JSON
// string ("a@b.com" / "Name <a@b.com>") or an object ({name,email}/{address}).
func parseFrom(raw json.RawMessage) (name, email string) {
	if len(raw) == 0 {
		return "", ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if addr, err := mail.ParseAddress(s); err == nil {
			return addr.Name, addr.Address
		}
		return "", strings.TrimSpace(s)
	}
	var obj struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Address string `json:"address"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		e := obj.Email
		if e == "" {
			e = obj.Address
		}
		return obj.Name, e
	}
	return "", ""
}

// stripHTML is a crude tag remover for the text fallback when only HTML is sent.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
