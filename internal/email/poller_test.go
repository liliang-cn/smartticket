package email

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
)

type recordingSink struct{ got []InboundEmail }

func (r *recordingSink) IngestEmail(_ context.Context, in InboundEmail) error {
	r.got = append(r.got, in)
	return nil
}

// Spins up an in-memory IMAP server, appends a fresh unseen message, and
// verifies the poller fetches, parses and ingests it end-to-end.
func TestPoller_FetchesUnseenAndIngests(t *testing.T) {
	be := memory.New()
	srv := server.New(be)
	srv.AllowInsecureAuth = true

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go func() { _ = srv.Serve(l) }()
	time.Sleep(50 * time.Millisecond)

	port := l.Addr().(*net.TCPAddr).Port
	addr := l.Addr().String()

	// Append an unseen message to INBOX.
	raw := "From: Alice Example <alice@example.com>\r\n" +
		"To: support@acme.com\r\n" +
		"Subject: Printer is broken\r\n" +
		"\r\n" +
		"It will not turn on.\r\n"
	c, err := client.Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := c.Login("username", "password"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := c.Append("INBOX", nil, time.Now(), strings.NewReader(raw)); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = c.Logout()

	sink := &recordingSink{}
	p := NewPoller(IMAPOptions{
		Host:     "127.0.0.1",
		Port:     port,
		Username: "username",
		Password: "password",
		Mailbox:  "INBOX",
		TLS:      false,
	}, sink)

	p.pollOnce(context.Background())

	var found *InboundEmail
	for i := range sink.got {
		if sink.got[i].FromEmail == "alice@example.com" {
			found = &sink.got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("poller did not ingest the appended message; got %+v", sink.got)
	}
	if found.Subject != "Printer is broken" {
		t.Errorf("subject = %q, want %q", found.Subject, "Printer is broken")
	}
	if !strings.Contains(found.Text, "will not turn on") {
		t.Errorf("body = %q, want it to contain the message text", found.Text)
	}
}
