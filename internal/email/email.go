// Package email provides bidirectional email: outbound ticket replies (over the
// Resend HTTP API or any SMTP server) and inbound email-to-ticket ingestion via
// a signed webhook. It is decoupled from the ticket package — outbound is used
// through a small interface the ticket package owns, and inbound calls back into
// the ticket package through the TicketSink interface defined here.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// Options configures the outbound mailer.
type Options struct {
	Provider     string // "resend" (default) or "smtp"
	FromName     string
	FromAddress  string
	ResendAPIKey string
	SMTP         SMTPOptions
}

// SMTPOptions configures the SMTP transport (used when Provider == "smtp").
type SMTPOptions struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      string // "starttls" (default), "tls" (implicit) or "none"
}

// Service sends outbound mail. It implements the ticket package's Mailer
// interface (SendTicketReply).
type Service struct {
	opt    Options
	client *http.Client
}

// NewService builds an outbound mail service.
func NewService(opt Options) *Service {
	if opt.Provider == "" {
		opt.Provider = "resend"
	}
	return &Service{opt: opt, client: &http.Client{Timeout: 20 * time.Second}}
}

// SendTicketReply emails a customer when an agent posts a public reply. The
// ticket number is embedded in the subject (e.g. "[TK-12] ...") so the
// customer's reply can be routed back to the same ticket. Best-effort: failures
// are logged, never returned to the request path.
func (s *Service) SendTicketReply(ctx context.Context, to, ticketNumber, ticketTitle string, ticketID uint, body, authorName string) {
	to = strings.TrimSpace(to)
	if to == "" || s.opt.FromAddress == "" {
		return
	}
	subject := strings.TrimSpace(fmt.Sprintf("[%s] %s", ticketNumber, ticketTitle))
	text := body
	if n := strings.TrimSpace(authorName); n != "" {
		text = body + "\n\n— " + n
	}

	var err error
	switch strings.ToLower(s.opt.Provider) {
	case "smtp":
		err = s.sendSMTP(to, subject, text)
	default:
		err = s.sendResend(ctx, to, subject, text)
	}
	if err != nil {
		logger.Error("email: ticket reply not sent",
			zap.String("to", to), zap.String("ticket", ticketNumber), zap.Error(err))
		return
	}
	logger.Info("email: ticket reply sent", zap.String("to", to), zap.String("ticket", ticketNumber))
}

// fromHeader renders `Name <addr>` (or just the address when no name).
func (s *Service) fromHeader() string {
	if name := strings.TrimSpace(s.opt.FromName); name != "" {
		return fmt.Sprintf("%s <%s>", name, s.opt.FromAddress)
	}
	return s.opt.FromAddress
}

// --- Resend (HTTP API) ---

func (s *Service) sendResend(ctx context.Context, to, subject, text string) error {
	if s.opt.ResendAPIKey == "" {
		return fmt.Errorf("resend api key is not configured")
	}
	payload := map[string]any{
		"from":    s.fromHeader(),
		"to":      []string{to},
		"subject": subject,
		"text":    text,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.opt.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("resend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// --- SMTP ---

func (s *Service) sendSMTP(to, subject, text string) error {
	if s.opt.SMTP.Host == "" {
		return fmt.Errorf("smtp host is not configured")
	}
	msg := buildMIME(s.fromHeader(), to, subject, text)
	addr := net.JoinHostPort(s.opt.SMTP.Host, fmt.Sprint(s.opt.SMTP.Port))
	var auth smtp.Auth
	if s.opt.SMTP.Username != "" {
		auth = smtp.PlainAuth("", s.opt.SMTP.Username, s.opt.SMTP.Password, s.opt.SMTP.Host)
	}

	switch strings.ToLower(s.opt.SMTP.TLS) {
	case "tls": // implicit TLS (usually port 465)
		return s.sendSMTPImplicitTLS(addr, auth, to, msg)
	default: // starttls / none — net/smtp upgrades opportunistically
		return smtp.SendMail(addr, auth, s.opt.FromAddress, []string{to}, msg)
	}
}

func (s *Service) sendSMTPImplicitTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.opt.SMTP.Host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, s.opt.SMTP.Host)
	if err != nil {
		return err
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(s.opt.FromAddress); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// buildMIME assembles a minimal RFC 5322 text/plain message. The Subject and
// From name are RFC 2047 encoded so non-ASCII (e.g. Chinese) is preserved.
func buildMIME(from, to, subject, text string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(text, "\n", "\r\n"))
	return []byte(b.String())
}
