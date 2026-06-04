package email

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"go.uber.org/zap"

	_ "github.com/emersion/go-message/charset" // register non-UTF8 charsets

	"github.com/company/smartticket/internal/logger"
)

// IMAPOptions configures the mailbox poller.
type IMAPOptions struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Mailbox      string
	TLS          bool
	PollInterval time.Duration
}

// Poller periodically fetches unseen mail over IMAP and ingests it as tickets —
// the fully self-hosted inbound path (no webhook, no DNS). Processed messages
// are flagged \Seen so they are not handled twice.
type Poller struct {
	opt  IMAPOptions
	sink TicketSink
}

// NewPoller builds an IMAP poller.
func NewPoller(opt IMAPOptions, sink TicketSink) *Poller {
	if opt.Mailbox == "" {
		opt.Mailbox = "INBOX"
	}
	if opt.PollInterval <= 0 {
		opt.PollInterval = 60 * time.Second
	}
	return &Poller{opt: opt, sink: sink}
}

// Run polls until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	logger.Info("email: IMAP poller started",
		zap.String("host", p.opt.Host), zap.String("mailbox", p.opt.Mailbox),
		zap.Duration("interval", p.opt.PollInterval))
	ticker := time.NewTicker(p.opt.PollInterval)
	defer ticker.Stop()
	p.pollOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Info("email: IMAP poller stopped")
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *Poller) pollOnce(ctx context.Context) {
	c, err := p.connect()
	if err != nil {
		logger.Error("email: IMAP connect failed", zap.Error(err))
		return
	}
	defer func() { _ = c.Logout() }()

	if _, err := c.Select(p.opt.Mailbox, false); err != nil {
		logger.Error("email: IMAP select failed", zap.String("mailbox", p.opt.Mailbox), zap.Error(err))
		return
	}

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		logger.Error("email: IMAP search failed", zap.Error(err))
		return
	}
	if len(ids) == 0 {
		return
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}

	messages := make(chan *imap.Message, 16)
	fetchDone := make(chan error, 1)
	go func() { fetchDone <- c.Fetch(seqset, items, messages) }()

	var ingested []uint32
	for msg := range messages {
		in, ok := p.parse(msg, section)
		if !ok || in.FromEmail == "" {
			continue
		}
		if err := p.sink.IngestEmail(ctx, in); err != nil {
			logger.Error("email: IMAP ingest failed", zap.String("from", in.FromEmail), zap.Error(err))
			continue
		}
		ingested = append(ingested, msg.SeqNum)
		logger.Info("email: IMAP message ingested",
			zap.String("from", in.FromEmail), zap.String("subject", in.Subject))
	}
	if err := <-fetchDone; err != nil {
		logger.Error("email: IMAP fetch failed", zap.Error(err))
	}

	if len(ingested) > 0 {
		markSet := new(imap.SeqSet)
		markSet.AddNum(ingested...)
		op := imap.FormatFlagsOp(imap.AddFlags, true)
		if err := c.Store(markSet, op, []interface{}{imap.SeenFlag}, nil); err != nil {
			logger.Error("email: IMAP mark-seen failed", zap.Error(err))
		}
	}
}

func (p *Poller) connect() (*client.Client, error) {
	addr := fmt.Sprintf("%s:%d", p.opt.Host, p.opt.Port)
	var (
		c   *client.Client
		err error
	)
	if p.opt.TLS {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}
	if err != nil {
		return nil, err
	}
	if err := c.Login(p.opt.Username, p.opt.Password); err != nil {
		_ = c.Logout()
		return nil, err
	}
	return c, nil
}

// parse extracts a normalized InboundEmail from an IMAP message, preferring the
// full MIME body and falling back to the envelope.
func (p *Poller) parse(msg *imap.Message, section *imap.BodySectionName) (InboundEmail, bool) {
	if msg == nil {
		return InboundEmail{}, false
	}
	body := msg.GetBody(section)
	if body == nil {
		return p.fromEnvelope(msg), true
	}

	mr, err := mail.CreateReader(body)
	if err != nil {
		return p.fromEnvelope(msg), true
	}

	in := InboundEmail{}
	if subj, err := mr.Header.Subject(); err == nil {
		in.Subject = subj
	}
	if addrs, err := mr.Header.AddressList("From"); err == nil && len(addrs) > 0 {
		in.FromName = addrs[0].Name
		in.FromEmail = addrs[0].Address
	}
	if mid, err := mr.Header.MessageID(); err == nil {
		in.MessageID = mid
	}

	var textBody, htmlBody string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if h, ok := part.Header.(*mail.InlineHeader); ok {
			ct, _, _ := h.ContentType()
			data, _ := io.ReadAll(part.Body)
			switch {
			case strings.HasPrefix(ct, "text/plain") && textBody == "":
				textBody = string(data)
			case strings.HasPrefix(ct, "text/html") && htmlBody == "":
				htmlBody = string(data)
			}
		}
	}

	in.Text = strings.TrimSpace(textBody)
	if in.Text == "" && htmlBody != "" {
		in.Text = stripHTML(htmlBody)
	}
	in.FromEmail = strings.ToLower(strings.TrimSpace(in.FromEmail))
	in.Subject = strings.TrimSpace(in.Subject)
	if in.FromEmail == "" {
		return p.fromEnvelope(msg), true
	}
	return in, true
}

func (p *Poller) fromEnvelope(msg *imap.Message) InboundEmail {
	in := InboundEmail{}
	if msg.Envelope == nil {
		return in
	}
	in.Subject = strings.TrimSpace(msg.Envelope.Subject)
	if len(msg.Envelope.From) > 0 {
		a := msg.Envelope.From[0]
		in.FromName = a.PersonalName
		in.FromEmail = strings.ToLower(strings.TrimSpace(a.MailboxName + "@" + a.HostName))
	}
	return in
}
