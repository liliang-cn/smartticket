package widget

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

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
	}
	if tktReq.Description == "" {
		tktReq.Description = title
	}

	tktResp, err := s.tkt.CreateTicket(actor, userID, tktReq)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("widget.StartSession: create ticket: %w", err)
	}

	// Stamp the ticket's Channel and ConversationToken fields. GORM's AutoMigrate
	// added these columns; we write them directly so the ticket service does not
	// need to know about widget-specific fields.
	token, err := IssueToken(tktResp.ID, s.secret)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("widget.StartSession: issue token: %w", err)
	}
	if err := s.db.Model(&models.Ticket{}).
		Where("id = ?", tktResp.ID).
		Updates(map[string]interface{}{
			"channel":            "web_widget",
			"conversation_token": token,
		}).Error; err != nil {
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
	anonymous := email == ""

	if anonymous {
		// Generate a unique placeholder identity for anonymous visitors.
		email = fmt.Sprintf("widget-%s@anon.local", randomHex(8))
		if name == "" {
			name = "Website visitor"
		}
	}
	if name == "" {
		// Named email visitor without an explicit display name: use local-part.
		parts := strings.SplitN(email, "@", 2)
		name = parts[0]
	}

	// For anonymous visitors we always create a fresh customer + user so each
	// session is distinct. For named visitors we reuse an existing user if found.
	if !anonymous {
		var existing models.User
		err := s.db.Where("email = ?", email).First(&existing).Error
		if err == nil {
			// Existing user: reuse their customer link (or create one if missing).
			if existing.CustomerID != nil {
				return *existing.CustomerID, existing.ID, name, email, nil
			}
			// User exists but has no customer — create a customer for them.
			cust, cerr := s.createCustomer(name, email)
			if cerr != nil {
				return 0, 0, "", "", cerr
			}
			if err2 := s.db.Model(&existing).Update("customer_id", cust.ID).Error; err2 != nil {
				return 0, 0, "", "", fmt.Errorf("link customer to user: %w", err2)
			}
			return cust.ID, existing.ID, name, email, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, "", "", fmt.Errorf("look up user: %w", err)
		}
	}

	// Create a new Customer (one per widget session for anonymous; one per
	// first-seen email for named visitors).
	cust, err := s.createCustomer(name, email)
	if err != nil {
		return 0, 0, "", "", err
	}

	// Create a placeholder User linked to the customer (no password — widget
	// visitors never log in directly).
	username := strings.ReplaceAll(email, "@", "_at_")
	username = strings.ReplaceAll(username, ".", "_")
	if len(username) > 90 {
		username = username[:90]
	}
	u := models.User{
		Email:        email,
		Username:     username,
		PasswordHash: "-",   // widget users have no password
		FirstName:    name,
		Role:         authz.RoleCustomer,
		IsActive:     true,
		CustomerID:   &cust.ID,
	}
	if err := s.db.Create(&u).Error; err != nil {
		return 0, 0, "", "", fmt.Errorf("create widget user: %w", err)
	}

	return cust.ID, u.ID, name, email, nil
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

	var u models.User
	if err := s.db.Where("email = ?", tkt.RequesterEmail).First(&u).Error; err != nil {
		// If the user is somehow missing, fall back to a zero actor — the ticket
		// service will still allow a customer actor to append messages.
		return &tkt, tkt.CustomerID, 0, nil
	}
	return &tkt, tkt.CustomerID, u.ID, nil
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

// randomHex returns n random hex characters.
func randomHex(n int) string {
	const charset = "0123456789abcdef"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
