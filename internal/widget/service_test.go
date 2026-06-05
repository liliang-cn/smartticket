package widget

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/ticket"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------
// test helpers
// -------------------------------------------------------------------

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Customer{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.SLATemplate{},
		&models.SLARule{},
	))
	return db
}

const testSecret = "widget-test-secret-2024"

// fakeTicketService implements TicketCreator in-memory for isolated tests.
type fakeTicketService struct {
	db      *gorm.DB
	counter uint
}

func (f *fakeTicketService) CreateTicket(_ authz.Actor, _ uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error) {
	f.counter++
	channel := req.Channel
	if channel == "" {
		channel = "web"
	}
	tkt := &models.Ticket{
		BaseModel:      models.BaseModel{},
		TicketNumber:   fmt.Sprintf("TK-%d", f.counter),
		Title:          req.Title,
		Description:    req.Description,
		Status:         "open",
		Priority:       req.Priority,
		Severity:       req.Severity,
		RequesterName:  req.RequesterName,
		RequesterEmail: req.RequesterEmail,
		Channel:        channel,
	}
	if req.CustomerID != nil {
		tkt.CustomerID = req.CustomerID
	}
	if err := f.db.Create(tkt).Error; err != nil {
		return nil, err
	}
	return &ticket.TicketResponse{
		ID:             tkt.ID,
		TicketNumber:   tkt.TicketNumber,
		Title:          tkt.Title,
		RequesterName:  tkt.RequesterName,
		RequesterEmail: tkt.RequesterEmail,
	}, nil
}

func (f *fakeTicketService) CreateMessage(_ authz.Actor, ticketID, _ uint, req *ticket.CreateMessageRequest) (*ticket.MessageResponse, error) {
	msg := &models.Message{
		TicketID:    ticketID,
		Content:     req.Content,
		ContentType: req.ContentType,
		IsInternal:  false,
	}
	if err := f.db.Create(msg).Error; err != nil {
		return nil, err
	}
	r := ticket.MessageResponse{
		ID:          msg.ID,
		TicketID:    ticketID,
		Content:     msg.Content,
		ContentType: msg.ContentType,
	}
	return &r, nil
}

func (f *fakeTicketService) ListMessages(_ authz.Actor, ticketID uint) ([]ticket.MessageResponse, error) {
	var msgs []models.Message
	if err := f.db.Where("ticket_id = ? AND is_internal = ?", ticketID, false).
		Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	out := make([]ticket.MessageResponse, len(msgs))
	for i, m := range msgs {
		out[i] = ticket.MessageResponse{
			ID:          m.ID,
			TicketID:    m.TicketID,
			Content:     m.Content,
			ContentType: m.ContentType,
			IsInternal:  m.IsInternal,
		}
	}
	return out, nil
}

func newTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	fake := &fakeTicketService{db: db}
	svc := NewService(db, fake, testSecret)
	return svc, db
}

// -------------------------------------------------------------------
// StartSession tests
// -------------------------------------------------------------------

func TestStartSession_WithEmail_CreatesCustomerTicketToken(t *testing.T) {
	svc, db := newTestService(t)

	resp, err := svc.StartSession(StartSessionRequest{
		Email:   "alice@example.com",
		Name:    "Alice",
		Message: "Hello, I need help",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	require.NotZero(t, resp.TicketID)

	// Token must decode back to the ticket ID.
	gotID, err := ParseToken(resp.Token, testSecret)
	require.NoError(t, err)
	require.Equal(t, resp.TicketID, gotID)

	// Customer should exist in DB.
	var cust models.Customer
	require.NoError(t, db.Where("domain = ?", "example.com").First(&cust).Error)

	// User should exist and be linked to the customer.
	var u models.User
	require.NoError(t, db.Where("email = ?", "alice@example.com").First(&u).Error)
	require.NotNil(t, u.CustomerID)
	require.Equal(t, cust.ID, *u.CustomerID)

	// Ticket row should carry channel=web_widget and the token.
	var tkt models.Ticket
	require.NoError(t, db.Where("id = ?", resp.TicketID).First(&tkt).Error)
	require.Equal(t, "web_widget", tkt.Channel)
	require.Equal(t, resp.Token, tkt.ConversationToken)
}

func TestStartSession_Anonymous_Works(t *testing.T) {
	svc, db := newTestService(t)

	resp, err := svc.StartSession(StartSessionRequest{
		Message: "I have a question",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	require.NotZero(t, resp.TicketID)

	// Anonymous customer should be stored with the anon domain.
	var cust models.Customer
	require.NoError(t, db.Where("domain = ?", "anon.local").First(&cust).Error)
	require.True(t, cust.IsActive)
}

func TestStartSession_ReusesExistingCustomerForSameEmail(t *testing.T) {
	svc, db := newTestService(t)

	// First session.
	resp1, err := svc.StartSession(StartSessionRequest{Email: "bob@example.com", Message: "first"})
	require.NoError(t, err)

	// Second session with same email should reuse the same user/customer.
	resp2, err := svc.StartSession(StartSessionRequest{Email: "bob@example.com", Message: "second"})
	require.NoError(t, err)

	// Both sessions produce distinct tickets.
	require.NotEqual(t, resp1.TicketID, resp2.TicketID)

	// Only one user record should exist for this email.
	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "bob@example.com").Count(&count).Error)
	require.EqualValues(t, 1, count)
}

// -------------------------------------------------------------------
// PostMessage tests
// -------------------------------------------------------------------

func TestPostMessage_AppendsMessage(t *testing.T) {
	svc, db := newTestService(t)

	resp, err := svc.StartSession(StartSessionRequest{Email: "carol@test.com", Message: "hi"})
	require.NoError(t, err)

	msg, err := svc.PostMessage(resp.Token, "follow-up question")
	require.NoError(t, err)
	require.Equal(t, resp.TicketID, msg.TicketID)
	require.Equal(t, "follow-up question", msg.Content)

	// Verify it's in the DB.
	var dbMsg models.Message
	require.NoError(t, db.Where("id = ?", msg.ID).First(&dbMsg).Error)
	require.Equal(t, "follow-up question", dbMsg.Content)
}

func TestPostMessage_BadToken_ReturnsErrInvalidToken(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.PostMessage("not-a-valid-token", "hello")
	require.ErrorIs(t, err, ErrInvalidToken)
}

// -------------------------------------------------------------------
// History tests
// -------------------------------------------------------------------

func TestHistory_ExcludesInternalMessages(t *testing.T) {
	svc, db := newTestService(t)

	resp, err := svc.StartSession(StartSessionRequest{Email: "dave@test.com", Message: "public msg"})
	require.NoError(t, err)

	// Inject an internal note directly.
	require.NoError(t, db.Create(&models.Message{
		TicketID:    resp.TicketID,
		Content:     "internal agent note",
		ContentType: "text",
		IsInternal:  true,
	}).Error)

	msgs, err := svc.History(resp.Token)
	require.NoError(t, err)

	for _, m := range msgs {
		require.False(t, m.IsInternal, "internal messages must not appear in widget history")
	}
	// The initial public message should appear (created by StartSession).
	found := false
	for _, m := range msgs {
		if m.Content == "public msg" {
			found = true
		}
	}
	require.True(t, found, "expected initial public message in history")
}

func TestHistory_BadToken_ReturnsErrInvalidToken(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.History("bad-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}
