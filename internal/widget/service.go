package widget

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/ticket"
	"gorm.io/gorm"
)

// TicketCreator is the subset of ticket.Service that the widget needs for
// creating tickets and messages. A narrow interface prevents import cycles and
// makes the service fully testable with fakes.
type TicketCreator interface {
	CreateTicket(actor authz.Actor, userID uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error)
	CreateMessage(actor authz.Actor, ticketID, userID uint, req *ticket.CreateMessageRequest) (*ticket.MessageResponse, error)
	ListMessages(actor authz.Actor, ticketID uint) ([]ticket.MessageResponse, error)
}

// Service implements the widget session and messaging operations.
type Service struct {
	db     *gorm.DB
	tkt    TicketCreator
	secret string
}

// NewService constructs a widget service. secret must be the application JWT
// secret (cfg.JWT.Secret) so the conversation tokens the widget issues are
// validated by the same key as everything else.
func NewService(db *gorm.DB, tkt TicketCreator, secret string) *Service {
	return &Service{db: db, tkt: tkt, secret: secret}
}

// StartSessionRequest is the payload for POST /widget/session.
type StartSessionRequest struct {
	// Email is optional; if empty an anonymous session is started.
	Email string `json:"email"`
	// Name is optional display name for the visitor.
	Name string `json:"name"`
	// Message is the first message body. If empty, the ticket is created without
	// an initial customer message (the visitor can post messages later).
	Message string `json:"message"`
}

// SessionResponse is returned by StartSession.
type SessionResponse struct {
	// Token is the signed conversation token the widget client stores and sends
	// on subsequent requests (Authorization: Bearer <token> or ?token=).
	Token    string `json:"token"`
	TicketID uint   `json:"ticket_id"`
}

// StartSession creates a customer (or reuses one by email) and opens a new
// web_widget ticket, returning a signed conversation token.
func (s *Service) StartSession(req StartSessionRequest) (SessionResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	req.Message = strings.TrimSpace(req.Message)

	// Resolve or create the visitor's customer record and a representative user.
	custID, userID, requesterName, requesterEmail, err := s.resolveVisitor(req.Email, req.Name)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("widget.StartSession: resolve visitor: %w", err)
	}

	// Build the ticket title from the message, or fall back to a default.
	title := "Website chat"
	if req.Message != "" {
		title = firstN(req.Message, 100)
	}

	// Create the ticket as the visitor (customer actor).
	actor := authz.Actor{
		UserID:     userID,
		Role:       authz.RoleCustomer,
		CustomerID: &custID,
	}
	tktReq := &ticket.CreateTicketRequest{
		Title:          title,
		Description:    req.Message,
		Priority:       "medium",
		Severity:       "minor",
		RequesterName:  requesterName,
		RequesterEmail: requesterEmail,
		Channel:        "web_widget",
	}
	if tktReq.Description == "" {
		tktReq.Description = title
	}

	tktResp, err := s.tkt.CreateTicket(actor, userID, tktReq)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("widget.StartSession: create ticket: %w", err)
	}

	// The conversation token encodes the ticket ID so it must be issued after
	// the ticket is persisted. A failure here is non-critical (the ticket already
	// exists with the correct channel) but we surface the error so callers know
	// the session is unusable.
	token, err := IssueToken(tktResp.ID, s.secret)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("widget.StartSession: issue token: %w", err)
	}
	// Stamp the conversation_token only (channel is already set atomically at
	// creation time via the ticket service).
	if err := s.db.Model(&models.Ticket{}).
		Where("id = ?", tktResp.ID).
		Update("conversation_token", token).Error; err != nil {
		return SessionResponse{}, fmt.Errorf("widget.StartSession: stamp token: %w", err)
	}

	// If the caller supplied an initial message, create it now (after the token
	// is stored so PostMessage's broadcast fires with a consistent room key).
	if req.Message != "" {
		_, err = s.tkt.CreateMessage(actor, tktResp.ID, userID, &ticket.CreateMessageRequest{
			Content:     req.Message,
			ContentType: "text",
		})
		if err != nil {
			// Best-effort; the ticket already exists — don't roll back.
			_ = err
		}
	}

	return SessionResponse{Token: token, TicketID: tktResp.ID}, nil
}

// PostMessage appends a customer message to the conversation identified by the
// conversation token. Fires the same ticket-service path so domain events and
// hub broadcasts happen automatically.
func (s *Service) PostMessage(token, body string) (*ticket.MessageResponse, error) {
	ticketID, err := ParseToken(token, s.secret)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tkt, custID, userID, err := s.ticketVisitorIDs(ticketID)
	if err != nil {
		return nil, fmt.Errorf("widget.PostMessage: %w", err)
	}
	if tkt == nil {
		return nil, ErrInvalidToken
	}

	actor := authz.Actor{
		UserID:     userID,
		Role:       authz.RoleCustomer,
		CustomerID: custID,
	}
	msg, err := s.tkt.CreateMessage(actor, ticketID, userID, &ticket.CreateMessageRequest{
		Content:     strings.TrimSpace(body),
		ContentType: "text",
	})
	if err != nil {
		return nil, fmt.Errorf("widget.PostMessage: %w", err)
	}
	return msg, nil
}

// History returns all non-internal messages on the conversation, oldest first.
func (s *Service) History(token string) ([]ticket.MessageResponse, error) {
	ticketID, err := ParseToken(token, s.secret)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tkt, custID, userID, err := s.ticketVisitorIDs(ticketID)
	if err != nil {
		return nil, fmt.Errorf("widget.History: %w", err)
	}
	if tkt == nil {
		return nil, ErrInvalidToken
	}

	actor := authz.Actor{
		UserID:     userID,
		Role:       authz.RoleCustomer,
		CustomerID: custID,
	}
	return s.tkt.ListMessages(actor, ticketID)
}

// -------------------------------------------------------------------
// internal helpers
// -------------------------------------------------------------------

// resolveVisitor finds or creates a Customer + User for the widget visitor.
// Returns (customerID, userID, requesterName, requesterEmail, error).
func (s *Service) resolveVisitor(email, name string) (uint, uint, string, string, error) {
	// SECURITY: the visitor-supplied email is UNVERIFIED contact info. Widget
	// sessions must NEVER look up or reuse an existing portal user/customer by it,
	// otherwise a visitor could type a real user's address and have their ticket
	// injected into that account (mis-attribution + a shared conversation thread).
	// Every session creates an independent, widget-scoped User+Customer with a
	// unique INTERNAL identity that can't collide with a real portal user; the
	// typed email/name ride along on the ticket as the requester's contact info,
	// and an agent can merge/link to a real customer later once verified.
	contactEmail := strings.TrimSpace(email) // caller already lowercased
	contactName := strings.TrimSpace(name)
	if contactName == "" {
		if contactEmail != "" {
			contactName = strings.SplitN(contactEmail, "@", 2)[0]
		} else {
			contactName = "Website visitor"
		}
	}

	cust, err := s.createCustomer(contactName, contactEmail)
	if err != nil {
		return 0, 0, "", "", err
	}

	// Unique internal identity — never reuses or collides with a portal user.
	internalEmail := fmt.Sprintf("widget-%s@anon.local", randomHex(8))
	u := models.User{
		Email:        internalEmail,
		Username:     "widget_" + randomHex(8),
		PasswordHash: "-", // widget visitors have no password / never log in
		FirstName:    contactName,
		Role:         authz.RoleCustomer,
		IsActive:     true,
		CustomerID:   &cust.ID,
	}
	if err := s.db.Create(&u).Error; err != nil {
		return 0, 0, "", "", fmt.Errorf("create widget user: %w", err)
	}

	return cust.ID, u.ID, contactName, contactEmail, nil
}

// createCustomer inserts a minimal Customer row for the widget visitor.
func (s *Service) createCustomer(name, email string) (*models.Customer, error) {
	cust := &models.Customer{
		Name:     name,
		Domain:   domainOf(email),
		IsActive: true,
	}
	if err := s.db.Omit("Users", "Tickets").Create(cust).Error; err != nil {
		return nil, fmt.Errorf("create widget customer: %w", err)
	}
	return cust, nil
}

// ticketVisitorIDs loads the ticket and derives the actor credentials from the
// ticket's requester email (the widget user we created at session start).
func (s *Service) ticketVisitorIDs(ticketID uint) (*models.Ticket, *uint, uint, error) {
	var tkt models.Ticket
	if err := s.db.Where("id = ?", ticketID).First(&tkt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, 0, nil
		}
		return nil, nil, 0, fmt.Errorf("load ticket %d: %w", ticketID, err)
	}

	// Derive the visitor's user from the ticket's customer (1:1 for widget
	// sessions). This is decoupled from RequesterEmail on purpose: RequesterEmail
	// now holds the visitor's UNVERIFIED contact email, not the internal user
	// identity, so it must not be used to resolve the acting user.
	if tkt.CustomerID != nil {
		var u models.User
		if err := s.db.Where("customer_id = ?", *tkt.CustomerID).First(&u).Error; err == nil {
			return &tkt, tkt.CustomerID, u.ID, nil
		}
	}
	// Fall back to a zero user — the ticket service still allows a customer actor
	// (scoped by CustomerID) to append messages on its own ticket.
	return &tkt, tkt.CustomerID, 0, nil
}

// firstN returns up to n runes of s (UTF-8 safe).
func firstN(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}

// domainOf extracts the domain part of an email, or returns "anon.local".
func domainOf(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return "anon.local"
}

// randomHex returns n random hex characters using a cryptographically secure
// source, preventing collision when multiple anonymous sessions start in the
// same nanosecond.
func randomHex(n int) string {
	// n hex chars require ceil(n/2) random bytes.
	bytes := make([]byte, (n+1)/2)
	if _, err := cryptorand.Read(bytes); err != nil {
		// crypto/rand failure is extremely rare (e.g. OS entropy exhausted).
		// Panic rather than silently producing a predictable value.
		panic(fmt.Sprintf("widget.randomHex: crypto/rand.Read: %v", err))
	}
	return hex.EncodeToString(bytes)[:n]
}
