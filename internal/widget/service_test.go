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

func (f *fakeTicketService) CreateTicket(actor authz.Actor, _ uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error) {
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
	// Mirror the real ticket service: a customer actor's CustomerID wins over
	// any req.CustomerID, so widget tickets carry the visitor's customer.
	customerID := req.CustomerID
	if actor.IsCustomer() {
		customerID = actor.CustomerID
	}
	if customerID != nil {
		tkt.CustomerID = customerID
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

	// Customer should exist in DB (domain derived from the contact email).
	var cust models.Customer
	require.NoError(t, db.Where("domain = ?", "example.com").First(&cust).Error)

	// A widget user is created and linked to the customer, but with a synthetic
	// INTERNAL identity — NOT the visitor's typed email (which is unverified).
	var u models.User
	require.NoError(t, db.Where("customer_id = ?", cust.ID).First(&u).Error)
	require.NotNil(t, u.CustomerID)
	require.Equal(t, cust.ID, *u.CustomerID)
	require.NotEqual(t, "alice@example.com", u.Email, "widget user must not adopt the unverified contact email")
	require.Contains(t, u.Email, "@anon.local")

	// The typed contact email lives on the ticket as the requester's contact info.
	var tkt models.Ticket
	require.NoError(t, db.Where("id = ?", resp.TicketID).First(&tkt).Error)
	require.Equal(t, "web_widget", tkt.Channel)
	require.Equal(t, resp.Token, tkt.ConversationToken)
	require.Equal(t, "alice@example.com", tkt.RequesterEmail)
	require.Equal(t, "Alice", tkt.RequesterName)
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

func TestStartSession_SameEmail_DoesNotReuseIdentity(t *testing.T) {
	svc, db := newTestService(t)

	// Two sessions with the same typed email.
	resp1, err := svc.StartSession(StartSessionRequest{Email: "bob@example.com", Message: "first"})
	require.NoError(t, err)
	resp2, err := svc.StartSession(StartSessionRequest{Email: "bob@example.com", Message: "second"})
	require.NoError(t, err)

	// Distinct tickets AND distinct customers — the unverified email is never
	// used to merge sessions into a shared identity.
	require.NotEqual(t, resp1.TicketID, resp2.TicketID)

	var t1, t2 models.Ticket
	require.NoError(t, db.First(&t1, resp1.TicketID).Error)
	require.NoError(t, db.First(&t2, resp2.TicketID).Error)
	require.NotNil(t, t1.CustomerID)
	require.NotNil(t, t2.CustomerID)
	require.NotEqual(t, *t1.CustomerID, *t2.CustomerID, "same email must not collapse into one customer")
}

// TestStartSession_DoesNotInjectIntoExistingPortalUser is the security guarantee:
// a visitor typing a REAL portal user's email must not have their widget ticket
// attached to that user's account.
func TestStartSession_DoesNotInjectIntoExistingPortalUser(t *testing.T) {
	svc, db := newTestService(t)

	// Seed a real portal user + customer (e.g. an existing paying customer).
	realCust := models.Customer{Name: "Real Co", Domain: "victim.com", IsActive: true}
	require.NoError(t, db.Omit("Users", "Tickets").Create(&realCust).Error)
	realUser := models.User{
		Email: "victim@victim.com", Username: "victim", PasswordHash: "x",
		FirstName: "Victim", Role: authz.RoleCustomer, IsActive: true, CustomerID: &realCust.ID,
	}
	require.NoError(t, db.Create(&realUser).Error)

	// An anonymous visitor types the real user's email into the widget.
	resp, err := svc.StartSession(StartSessionRequest{Email: "victim@victim.com", Message: "I'm totally the victim"})
	require.NoError(t, err)

	// The widget ticket must NOT be attached to the real user's customer/user.
	var tkt models.Ticket
	require.NoError(t, db.First(&tkt, resp.TicketID).Error)
	require.NotNil(t, tkt.CustomerID)
	require.NotEqual(t, realCust.ID, *tkt.CustomerID, "widget ticket must not land under the real customer")

	// The real user's identity is untouched: still exactly one user with that email.
	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "victim@victim.com").Count(&count).Error)
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
