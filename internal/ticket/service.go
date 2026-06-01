package ticket

import (
	"context"
	"fmt"
	"strings"
	"time"

	stderrors "errors"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"gorm.io/gorm"
)

// scopeToActor restricts a ticket query to the actor's customer when the actor
// is a customer-role user. Team actors (admin/engineer) are unrestricted.
func scopeToActor(q *gorm.DB, actor authz.Actor) *gorm.DB {
	if actor.IsCustomer() && actor.CustomerID != nil {
		return q.Where("customer_id = ?", *actor.CustomerID)
	}
	return q
}

// Service provides ticket management business logic.
type Service struct {
	db            *gorm.DB
	slaCalculator *sla.Calculator
	notifier      Notifier // optional; nil = no-op (see SetNotifier)
}

// NewService creates a new ticket service.
func NewService(db *gorm.DB, slaCalculator *sla.Calculator) *Service {
	return &Service{
		db:            db,
		slaCalculator: slaCalculator,
	}
}

// CreateTicketRequest represents a ticket creation request.
type CreateTicketRequest struct {
	Title          string `json:"title" binding:"required,min=1,max=255"`
	Description    string `json:"description" binding:"required,min=1"`
	Priority       string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Severity       string `json:"severity" binding:"omitempty,oneof=trivial minor major critical"`
	Category       string `json:"category" binding:"omitempty,max=100"`
	Type           string `json:"type" binding:"omitempty,max=50"`
	ProductID      *uint  `json:"product_id" binding:"omitempty"`
	ServiceID      *uint  `json:"service_id" binding:"omitempty"`
	CustomerID     *uint  `json:"customer_id" binding:"omitempty"` // team may file on behalf of a customer; ignored for customer actors
	RequesterName  string `json:"requester_name" binding:"required,min=1,max=255"`
	RequesterEmail string `json:"requester_email" binding:"required,email,max=255"`
	Tags           string `json:"tags" binding:"omitempty"`          // JSON array
	CustomFields   string `json:"custom_fields" binding:"omitempty"` // JSON object
}

// UpdateTicketRequest represents a ticket update request.
type UpdateTicketRequest struct {
	Title          string `json:"title" binding:"omitempty,min=1,max=255"`
	Description    string `json:"description" binding:"omitempty,min=1"`
	Status         string `json:"status" binding:"omitempty,oneof=open in_progress resolved closed cancelled"`
	Priority       string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Severity       string `json:"severity" binding:"omitempty,oneof=trivial minor major critical"`
	Category       string `json:"category" binding:"omitempty,max=100"`
	Type           string `json:"type" binding:"omitempty,max=50"`
	ProductID      *uint  `json:"product_id" binding:"omitempty"`
	ServiceID      *uint  `json:"service_id" binding:"omitempty"`
	AssignedTo     *uint  `json:"assigned_to"`
	RequesterName  string `json:"requester_name" binding:"omitempty,max=255"`
	RequesterEmail string `json:"requester_email" binding:"omitempty,email,max=255"`
	Tags           string `json:"tags" binding:"omitempty"`          // JSON array
	CustomFields   string `json:"custom_fields" binding:"omitempty"` // JSON object
}

// TicketResponse represents a ticket response.
type TicketResponse struct {
	ID              uint                   `json:"id"`
	TicketNumber    string                 `json:"ticket_number"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"`
	Priority        string                 `json:"priority"`
	Severity        string                 `json:"severity"`
	Category        string                 `json:"category"`
	Type            string                 `json:"type"`
	ProductID       *uint                  `json:"product_id"`
	ServiceID       *uint                  `json:"service_id"`
	CustomerID      *uint                  `json:"customer_id"`
	CustomerName    string                 `json:"customer_name,omitempty"`
	AssignedTo      *uint                  `json:"assigned_to"`
	AssignedUser    *UserInfo              `json:"assigned_user,omitempty"`
	RequesterName   string                 `json:"requester_name"`
	RequesterEmail  string                 `json:"requester_email"`
	Tags            []string               `json:"tags"`
	CustomFields    map[string]interface{} `json:"custom_fields"`
	IsDeleted       bool                   `json:"is_deleted"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	ResolutionTime  *time.Time             `json:"resolution_time"`
	ResolvedAt      *time.Time             `json:"resolved_at"`
	DueDate         *time.Time             `json:"due_date"`
	SLAStatus       string                 `json:"sla_status"`
	MessageCount    int64                  `json:"message_count"`
	AttachmentCount int64                  `json:"attachment_count"`
}

// TicketListResponse represents a paginated ticket list.
type TicketListResponse struct {
	Data       []TicketResponse `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// UserInfo represents basic user information in ticket responses.
type UserInfo struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// CreateTicket creates a new ticket. A customer actor's ticket is forced to the
// actor's own customer; a team actor may optionally set req.CustomerID.
func (s *Service) CreateTicket(actor authz.Actor, userID uint, req *CreateTicketRequest) (*TicketResponse, error) {
	// Normalize and validate input
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.RequesterName = strings.TrimSpace(req.RequesterName)
	req.RequesterEmail = strings.ToLower(strings.TrimSpace(req.RequesterEmail))

	// Set defaults
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Severity == "" {
		req.Severity = "minor"
	}

	// Generate ticket number
	ticketNumber, err := s.generateTicketNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ticket number: %w", err)
	}

	// Calculate SLA using the SLA calculator
	slaDueDate, err := s.slaCalculator.CalculateSLADueDates(req.Priority, req.Severity, req.ProductID, req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate SLA: %w", err)
	}

	// Determine the owning customer: a customer actor always files for their own
	// customer; a team actor may set one explicitly via the request.
	customerID := req.CustomerID
	if actor.IsCustomer() {
		customerID = actor.CustomerID
	}

	// Create ticket
	ticket := &models.Ticket{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: nil, // Temporarily nil to avoid FK constraints
			UpdatedBy: nil,
		},
		TicketNumber:   ticketNumber,
		Title:          req.Title,
		Description:    req.Description,
		Status:         "open",
		Priority:       req.Priority,
		Severity:       req.Severity,
		Category:       req.Category,
		Type:           req.Type,
		ProductID:      req.ProductID,
		ServiceID:      req.ServiceID,
		CustomerID:     customerID,
		AssignedTo:     nil, // Default to nil (unassigned)
		RequesterName:  req.RequesterName,
		RequesterEmail: req.RequesterEmail,
		Tags:           req.Tags,
		CustomFields:   req.CustomFields,
		DueDate:        &slaDueDate.ResponseDueDate,
		SLAStatus:      "within", // Default SLA status
	}

	if err := s.db.Create(ticket).Error; err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	s.recordEvent(ticket.ID, userID, "created", "created the ticket")

	return s.ticketToResponse(ticket), nil
}

// GetTicket gets a ticket by ID, scoped to the actor's customer.
func (s *Service) GetTicket(actor authz.Actor, ticketID uint) (*TicketResponse, error) {
	var ticket models.Ticket
	if err := scopeToActor(s.db.Where("id = ?", ticketID), actor).
		Preload("AssignedUser").Preload("Customer").
		First(&ticket).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("ticket")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return s.ticketToResponse(&ticket), nil
}

// ListTickets lists tickets with pagination and filtering, scoped to the actor's customer.
func (s *Service) ListTickets(actor authz.Actor, page, pageSize int, filters map[string]interface{}) (*TicketListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var tickets []models.Ticket
	var total int64

	// Exclude soft-deleted tickets from listings.
	query := scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor)

	// Apply filters
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if category, ok := filters["category"].(string); ok && category != "" {
		query = query.Where("category = ?", category)
	}
	if assignedTo, ok := filters["assigned_to"].(uint); ok && assignedTo > 0 {
		query = query.Where("assigned_to = ?", assignedTo)
	}
	if search, ok := filters["search"].(string); ok && search != "" {
		search = strings.TrimSpace(search)
		query = query.Where("title ILIKE ? OR description ILIKE ? OR requester_name ILIKE ? OR requester_email ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	// Get paginated results with associations
	if err := query.Preload("AssignedUser").Preload("Customer").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&tickets).Error; err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	// Convert to response
	responses := make([]TicketResponse, len(tickets))
	for i, ticket := range tickets {
		responses[i] = *s.ticketToResponse(&ticket)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &TicketListResponse{
		Data:       responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateTicket updates a ticket, scoped to the actor's customer.
func (s *Service) UpdateTicket(actor authz.Actor, ticketID uint, userID uint, req *UpdateTicketRequest) (*TicketResponse, error) {
	var ticket models.Ticket
	if err := scopeToActor(s.db.Where("id = ?", ticketID), actor).
		First(&ticket).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("ticket")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// Capture pre-update values for the activity log.
	oldPriority := ticket.Priority
	oldSeverity := ticket.Severity
	oldAssigned := ticket.AssignedTo

	// Normalize input
	if req.Title != "" {
		req.Title = strings.TrimSpace(req.Title)
	}
	if req.Description != "" {
		req.Description = strings.TrimSpace(req.Description)
	}
	if req.RequesterName != "" {
		req.RequesterName = strings.TrimSpace(req.RequesterName)
	}
	if req.RequesterEmail != "" {
		req.RequesterEmail = strings.ToLower(strings.TrimSpace(req.RequesterEmail))
	}

	// Update fields
	if req.Title != "" {
		ticket.Title = req.Title
	}
	if req.Description != "" {
		ticket.Description = req.Description
	}
	oldStatus := ticket.Status
	if req.Status != "" {
		ticket.Status = req.Status
		// Handle status transitions
		if req.Status == "resolved" && ticket.ResolvedAt == nil {
			now := time.Now()
			ticket.ResolvedAt = &now
			ticket.ResolutionTime = &now
		} else if req.Status != "resolved" && ticket.ResolvedAt != nil {
			ticket.ResolvedAt = nil
			ticket.ResolutionTime = nil
		}
	}

	// Track if priority or severity changed for SLA recalculation
	priorityChanged := false
	severityChanged := false

	if req.Priority != "" {
		if req.Priority != ticket.Priority {
			priorityChanged = true
		}
		ticket.Priority = req.Priority
	}
	if req.Severity != "" {
		if req.Severity != ticket.Severity {
			severityChanged = true
		}
		ticket.Severity = req.Severity
	}
	if req.Category != "" {
		ticket.Category = req.Category
	}
	if req.Type != "" {
		ticket.Type = req.Type
	}
	if req.ProductID != nil {
		ticket.ProductID = req.ProductID
	}
	if req.ServiceID != nil {
		ticket.ServiceID = req.ServiceID
	}
	if req.AssignedTo != nil && actor.IsTeam() {
		// Only team members may (re)assign a ticket.
		ticket.AssignedTo = req.AssignedTo
	}
	if req.RequesterName != "" {
		ticket.RequesterName = req.RequesterName
	}
	if req.RequesterEmail != "" {
		ticket.RequesterEmail = req.RequesterEmail
	}
	if req.Tags != "" {
		ticket.Tags = req.Tags
	}
	if req.CustomFields != "" {
		ticket.CustomFields = req.CustomFields
	}

	// Recalculate SLA if priority, severity, product, or service changed
	if priorityChanged || severityChanged || req.ProductID != nil || req.ServiceID != nil {
		slaDueDate, err := s.slaCalculator.CalculateSLADueDates(ticket.Priority, ticket.Severity, ticket.ProductID, ticket.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to recalculate SLA: %w", err)
		}
		ticket.DueDate = &slaDueDate.ResponseDueDate
		// SLA status will be updated by updateSLAStatus
	}

	// Update SLA status
	s.updateSLAStatus(&ticket)

	if err := s.db.Save(&ticket).Error; err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Activity log: record the meaningful changes.
	if req.Status != "" && ticket.Status != oldStatus {
		s.recordEvent(ticket.ID, userID, "status", fmt.Sprintf("changed status: %s → %s", oldStatus, ticket.Status))
	}
	if priorityChanged {
		s.recordEvent(ticket.ID, userID, "priority", fmt.Sprintf("changed priority: %s → %s", oldPriority, ticket.Priority))
	}
	if severityChanged {
		s.recordEvent(ticket.ID, userID, "severity", fmt.Sprintf("changed severity: %s → %s", oldSeverity, ticket.Severity))
	}
	if req.AssignedTo != nil && actor.IsTeam() && !sameUintPtr(oldAssigned, ticket.AssignedTo) {
		s.recordEvent(ticket.ID, userID, "assigned", s.assignSummary(ticket.AssignedTo))
	}

	// Best-effort: on an actual status change, notify the ticket's customer-side
	// users (skip if no owning customer). Author is excluded from recipients.
	if s.notifier != nil && req.Status != "" && req.Status != oldStatus && ticket.CustomerID != nil {
		recipients := s.customerRecipients(*ticket.CustomerID, userID)
		s.notifier.Notify(context.Background(), recipients, "ticket_status",
			fmt.Sprintf("Ticket #%d is now %s", ticket.ID, ticket.Status), "", "ticket", ticket.ID)
	}

	// Reload with associations
	if err := s.db.Preload("AssignedUser").Preload("Customer").
		First(&ticket, ticket.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload ticket: %w", err)
	}

	return s.ticketToResponse(&ticket), nil
}

// DeleteTicket soft deletes a ticket, scoped to the actor's customer.
func (s *Service) DeleteTicket(actor authz.Actor, ticketID uint) error {
	result := scopeToActor(s.db.Model(&models.Ticket{}).Where("id = ?", ticketID), actor).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("failed to delete ticket: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("ticket")
	}

	return nil
}

// AssignTicket assigns a ticket to a user. Team-only.
func (s *Service) AssignTicket(actor authz.Actor, ticketID uint, assignedTo uint) error {
	if !actor.IsTeam() {
		return errors.NewForbiddenError("only team members can assign tickets")
	}
	result := s.db.Model(&models.Ticket{}).
		Where("id = ?", ticketID).
		Update("assigned_to", &assignedTo)

	if result.Error != nil {
		return fmt.Errorf("failed to assign ticket: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("ticket")
	}

	// Best-effort: tell the newly assigned agent (never the actor themselves).
	if s.notifier != nil && assignedTo != 0 && assignedTo != actor.UserID {
		s.notifier.Notify(context.Background(), []uint{assignedTo}, "ticket_assigned",
			fmt.Sprintf("You were assigned ticket #%d", ticketID), "", "ticket", ticketID)
	}

	s.recordEvent(ticketID, actor.UserID, "assigned", s.assignSummary(&assignedTo))

	return nil
}

// MessageResponse is a flat, schema-safe view of a ticket message.
type MessageResponse struct {
	ID          uint      `json:"id"`
	TicketID    uint      `json:"ticket_id"`
	UserID      uint      `json:"user_id"`
	AuthorName  string    `json:"author_name,omitempty"`
	AuthorRole  string    `json:"author_role,omitempty"`
	Content     string    `json:"content"`
	ContentType string    `json:"content_type"`
	IsInternal  bool      `json:"is_internal"`
	IsFromAI    bool      `json:"is_from_ai"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateMessageRequest is the payload for adding a message to a ticket.
type CreateMessageRequest struct {
	Content     string `json:"content" binding:"required,min=1"`
	ContentType string `json:"content_type" binding:"omitempty,oneof=text html markdown"`
	IsInternal  bool   `json:"is_internal"`
}

// findTicketForActor loads a ticket scoped to the actor's customer, returning a
// NotFound error (no existence disclosure) when it is outside the actor's scope.
func (s *Service) findTicketForActor(actor authz.Actor, ticketID uint) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := scopeToActor(s.db.Where("id = ?", ticketID), actor).First(&ticket).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("ticket")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	return &ticket, nil
}

// ListMessages returns a ticket's messages, scoped to the actor's customer.
// Customer actors never see internal notes (IsInternal).
func (s *Service) ListMessages(actor authz.Actor, ticketID uint) ([]MessageResponse, error) {
	if _, err := s.findTicketForActor(actor, ticketID); err != nil {
		return nil, err
	}

	query := s.db.Preload("User").Where("ticket_id = ?", ticketID)
	if actor.IsCustomer() {
		query = query.Where("is_internal = ?", false)
	}

	var messages []models.Message
	if err := query.Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	out := make([]MessageResponse, len(messages))
	for i := range messages {
		out[i] = messageToResponse(&messages[i])
	}
	return out, nil
}

// CreateMessage adds a message to a ticket, scoped to the actor's customer.
// Customer actors cannot create internal notes (IsInternal is forced false).
func (s *Service) CreateMessage(actor authz.Actor, ticketID, userID uint, req *CreateMessageRequest) (*MessageResponse, error) {
	tkt, err := s.findTicketForActor(actor, ticketID)
	if err != nil {
		return nil, err
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "text"
	}
	isInternal := req.IsInternal
	if actor.IsCustomer() {
		isInternal = false
	}

	message := &models.Message{
		TicketID:    ticketID,
		UserID:      userID,
		Content:     strings.TrimSpace(req.Content),
		ContentType: contentType,
		IsInternal:  isInternal,
	}
	if err := s.db.Create(message).Error; err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Best-effort in-app notification (after the message is committed; never
	// affects the request path). Internal notes are NEVER surfaced to customers.
	s.notifyMessage(actor, tkt, message, userID)

	if isInternal {
		s.recordEvent(ticketID, userID, "note", "added an internal note")
	} else {
		s.recordEvent(ticketID, userID, "replied", "replied to the ticket")
	}

	// Load the author so the response carries the author's name/role.
	_ = s.db.Preload("User").First(message, message.ID).Error

	resp := messageToResponse(message)
	return &resp, nil
}

// notifyMessage emits a ticket_reply notification for a freshly created message.
// - Team author + public reply: notify the ticket's customer-side users.
// - Team author + internal note: notify no one (customers must NOT learn of it).
// - Customer author: notify the assigned agent (if any).
// The author is always excluded from recipients.
func (s *Service) notifyMessage(actor authz.Actor, tkt *models.Ticket, msg *models.Message, authorID uint) {
	if s.notifier == nil {
		return
	}
	snippet := msg.Content
	if len(snippet) > 140 {
		snippet = snippet[:140]
	}
	title := fmt.Sprintf("New reply on ticket #%d", tkt.ID)

	switch {
	case actor.IsTeam():
		if msg.IsInternal || tkt.CustomerID == nil {
			return // internal note: never notify customers; unowned: nobody to notify
		}
		recipients := s.customerRecipients(*tkt.CustomerID, authorID)
		s.notifier.Notify(context.Background(), recipients, "ticket_reply", title, snippet, "ticket", tkt.ID)
	case actor.IsCustomer():
		if tkt.AssignedTo == nil || *tkt.AssignedTo == authorID {
			return
		}
		s.notifier.Notify(context.Background(), []uint{*tkt.AssignedTo}, "ticket_reply", title, snippet, "ticket", tkt.ID)
	}
}

func messageToResponse(m *models.Message) MessageResponse {
	r := MessageResponse{
		ID:          m.ID,
		TicketID:    m.TicketID,
		UserID:      m.UserID,
		Content:     m.Content,
		ContentType: m.ContentType,
		IsInternal:  m.IsInternal,
		IsFromAI:    m.IsFromAI,
		CreatedAt:   m.CreatedAt,
	}
	if m.User != nil {
		r.AuthorName = displayName(m.User)
		r.AuthorRole = m.User.Role
	}
	return r
}

// displayName returns a human-friendly name for a user, falling back to
// username then email when names are unset.
func displayName(u *models.User) string {
	full := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if full != "" {
		return full
	}
	if u.Username != "" {
		return u.Username
	}
	return u.Email
}

// GetTicketStats gets ticket statistics, scoped to the actor's customer.
func (s *Service) GetTicketStats(actor authz.Actor) (map[string]interface{}, error) {
	var stats struct {
		Total         int64 `json:"total"`
		Open          int64 `json:"open"`
		InProgress    int64 `json:"in_progress"`
		Resolved      int64 `json:"resolved"`
		Closed        int64 `json:"closed"`
		OverdueCount  int64 `json:"overdue_count"`
		CriticalCount int64 `json:"critical_count"`
		HighCount     int64 `json:"high_count"`
		MediumCount   int64 `json:"medium_count"`
		LowCount      int64 `json:"low_count"`
	}

	// Get basic status counts
	if err := scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
		Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total tickets: %w", err)
	}

	// Get status breakdown
	rows, err := scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
		Select("status, COUNT(*) as count").
		Group("status").
		Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		switch status {
		case "open":
			stats.Open = count
		case "in_progress":
			stats.InProgress = count
		case "resolved":
			stats.Resolved = count
		case "closed":
			stats.Closed = count
		}
	}

	// Get priority breakdown
	priorityRows, err := scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
		Where("status IN ?", []string{"open", "in_progress"}).
		Select("priority, COUNT(*) as count").
		Group("priority").
		Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get priority breakdown: %w", err)
	}
	defer func() { _ = priorityRows.Close() }()

	for priorityRows.Next() {
		var priority string
		var count int64
		if err := priorityRows.Scan(&priority, &count); err != nil {
			continue
		}
		switch priority {
		case "critical":
			stats.CriticalCount = count
		case "high":
			stats.HighCount = count
		case "medium":
			stats.MediumCount = count
		case "low":
			stats.LowCount = count
		}
	}

	// Get overdue count
	now := time.Now()
	if err := scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
		Where("status IN ? AND due_date < ?",
			[]string{"open", "in_progress"}, now).
		Count(&stats.OverdueCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count overdue tickets: %w", err)
	}

	result := map[string]interface{}{
		"total_tickets":       stats.Total,
		"open_tickets":        stats.Open,
		"in_progress_tickets": stats.InProgress,
		"resolved_tickets":    stats.Resolved,
		"closed_tickets":      stats.Closed,
		"overdue_tickets":     stats.OverdueCount,
		"priority_breakdown": map[string]int64{
			"critical": stats.CriticalCount,
			"high":     stats.HighCount,
			"medium":   stats.MediumCount,
			"low":      stats.LowCount,
		},
		"status_breakdown": map[string]int64{
			"open":        stats.Open,
			"in_progress": stats.InProgress,
			"resolved":    stats.Resolved,
			"closed":      stats.Closed,
		},
	}

	return result, nil
}

// Helper methods

func (s *Service) generateTicketNumber() (string, error) {
	var count int64
	if err := s.db.Model(&models.Ticket{}).
		Count(&count).Error; err != nil {
		return "", err
	}

	// Format: TK-{sequence_number}
	return fmt.Sprintf("TK-%d", count+1), nil
}

func (s *Service) calculateDueDate(priority string) *time.Time {
	now := time.Now()
	var hours int

	switch priority {
	case "critical":
		hours = 4 // 4 hours
	case "high":
		hours = 8 // 8 hours (1 business day)
	case "medium":
		hours = 24 // 24 hours (3 business days)
	case "low":
		hours = 72 // 72 hours (5 business days)
	default:
		hours = 24
	}

	dueDate := now.Add(time.Duration(hours) * time.Hour)
	return &dueDate
}

func (s *Service) updateSLAStatus(ticket *models.Ticket) {
	if ticket.DueDate == nil || ticket.Status == "resolved" || ticket.Status == "closed" {
		if ticket.Status == "resolved" || ticket.Status == "closed" {
			ticket.SLAStatus = "met"
		}
		return
	}

	now := time.Now()
	if now.After(*ticket.DueDate) {
		ticket.SLAStatus = "breached"
	} else {
		// Check if we're within 24 hours of due date
		threshold := ticket.DueDate.Add(-24 * time.Hour)
		if now.After(threshold) {
			ticket.SLAStatus = "warning"
		} else {
			ticket.SLAStatus = "within"
		}
	}
}

func (s *Service) ticketToResponse(ticket *models.Ticket) *TicketResponse {
	response := &TicketResponse{
		ID:             ticket.ID,
		TicketNumber:   ticket.TicketNumber,
		Title:          ticket.Title,
		Description:    ticket.Description,
		Status:         ticket.Status,
		Priority:       ticket.Priority,
		Severity:       ticket.Severity,
		Category:       ticket.Category,
		Type:           ticket.Type,
		ProductID:      ticket.ProductID,
		ServiceID:      ticket.ServiceID,
		CustomerID:     ticket.CustomerID,
		AssignedTo:     ticket.AssignedTo,
		RequesterName:  ticket.RequesterName,
		RequesterEmail: ticket.RequesterEmail,
		IsDeleted:      ticket.IsDeleted,
		CreatedAt:      ticket.CreatedAt,
		UpdatedAt:      ticket.UpdatedAt,
		ResolutionTime: ticket.ResolutionTime,
		ResolvedAt:     ticket.ResolvedAt,
		DueDate:        ticket.DueDate,
		SLAStatus:      ticket.SLAStatus,
	}

	// Add the owning customer organization's name when the relation is loaded.
	if ticket.Customer != nil {
		response.CustomerName = ticket.Customer.Name
	}

	// Add assigned user info (role would be determined by auth service)
	if ticket.AssignedUser != nil {
		response.AssignedUser = &UserInfo{
			ID:        ticket.AssignedUser.ID,
			Email:     ticket.AssignedUser.Email,
			Username:  ticket.AssignedUser.Username,
			FirstName: ticket.AssignedUser.FirstName,
			LastName:  ticket.AssignedUser.LastName,
			Role:      "user", // Role would be determined by auth service in actual implementation
		}
	}

	// Parse JSON fields
	if ticket.Tags != "" {
		// This would require JSON parsing - for now, return empty array
		response.Tags = []string{}
	}

	if ticket.CustomFields != "" {
		// This would require JSON parsing - for now, return empty map
		response.CustomFields = make(map[string]interface{})
	}

	// Get message count
	s.db.Model(&models.Message{}).Where("ticket_id = ?", ticket.ID).Count(&response.MessageCount)

	// Get attachment count
	s.db.Model(&models.Attachment{}).Where("ticket_id = ?", ticket.ID).Count(&response.AttachmentCount)

	return response
}

// TicketSLAResponse describes the SLA policy governing a ticket: which rule
// (and template) matched, the response/resolution targets, and the ticket's
// current due date and SLA status.
type TicketSLAResponse struct {
	TicketID          uint       `json:"ticket_id"`
	Priority          string     `json:"priority"`
	Severity          string     `json:"severity"`
	Source            string     `json:"source"`      // "rule" or "default"
	PolicyName        string     `json:"policy_name"` // SLA template name, rule label, or "Default policy"
	ResponseMinutes   int        `json:"response_minutes"`
	ResolutionMinutes int        `json:"resolution_minutes"`
	BusinessOnly      bool       `json:"business_only"`
	DueDate           *time.Time `json:"due_date"`
	SLAStatus         string     `json:"sla_status"`
}

// GetTicketSLA resolves the SLA policy that governs a ticket — the same rule the
// due-date calculator matches (by priority/severity/product/service), with its
// template name — so the UI can show exactly which SLA applies. Access is
// customer-isolated via GetTicket.
func (s *Service) GetTicketSLA(actor authz.Actor, ticketID uint) (*TicketSLAResponse, error) {
	tr, err := s.GetTicket(actor, ticketID)
	if err != nil {
		return nil, err
	}

	out := &TicketSLAResponse{
		TicketID:  tr.ID,
		Priority:  tr.Priority,
		Severity:  tr.Severity,
		DueDate:   tr.DueDate,
		SLAStatus: tr.SLAStatus,
	}

	// Targets (minutes) come from the calculator, which falls back to the
	// built-in defaults when no rule matches.
	if due, derr := s.slaCalculator.CalculateSLADueDates(tr.Priority, tr.Severity, tr.ProductID, tr.ServiceID); derr == nil && due != nil {
		out.ResponseMinutes = due.ResponseMinutes
		out.ResolutionMinutes = due.ResolutionMinutes
		out.BusinessOnly = due.BusinessOnly
	}

	// The policy name + source come from the matched rule (or the default).
	if rule, ok := s.slaCalculator.MatchRule(tr.Priority, tr.Severity, tr.ProductID, tr.ServiceID); ok {
		out.Source = "rule"
		if rule.SLATemplate.Name != "" {
			out.PolicyName = rule.SLATemplate.Name
		} else {
			out.PolicyName = fmt.Sprintf("SLA rule #%d", rule.ID)
		}
	} else {
		out.Source = "default"
		out.PolicyName = "Default policy (by priority)"
	}

	return out, nil
}

// sameUintPtr reports whether two *uint hold the same value (both nil counts as equal).
func sameUintPtr(a, b *uint) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// userDisplay resolves a user's display name by ID for activity summaries.
func (s *Service) userDisplay(id uint) string {
	if id == 0 {
		return ""
	}
	var u models.User
	if err := s.db.First(&u, id).Error; err == nil {
		return displayName(&u)
	}
	return fmt.Sprintf("user #%d", id)
}

// assignSummary renders an assignment activity summary.
func (s *Service) assignSummary(assignedTo *uint) string {
	if assignedTo == nil || *assignedTo == 0 {
		return "unassigned the ticket"
	}
	return "assigned the ticket to " + s.userDisplay(*assignedTo)
}

// recordEvent appends an entry to a ticket's activity log. Best-effort: a
// failure here never affects the originating operation.
func (s *Service) recordEvent(ticketID, userID uint, action, summary string) {
	_ = s.db.Create(&models.TicketEvent{
		TicketID: ticketID,
		UserID:   userID,
		Action:   action,
		Summary:  summary,
	}).Error
}

// TicketEventResponse is a flat, schema-safe view of a ticket activity entry.
type TicketEventResponse struct {
	ID        uint      `json:"id"`
	Action    string    `json:"action"`
	Summary   string    `json:"summary"`
	ActorName string    `json:"actor_name,omitempty"`
	ActorRole string    `json:"actor_role,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ListTicketEvents returns a ticket's activity log (oldest first), scoped to the
// actor's customer. Internal-note events are hidden from customer actors.
func (s *Service) ListTicketEvents(actor authz.Actor, ticketID uint) ([]TicketEventResponse, error) {
	if _, err := s.findTicketForActor(actor, ticketID); err != nil {
		return nil, err
	}

	var events []models.TicketEvent
	if err := s.db.Preload("User").Where("ticket_id = ?", ticketID).
		Order("created_at ASC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to list ticket events: %w", err)
	}

	out := make([]TicketEventResponse, 0, len(events))
	for i := range events {
		e := &events[i]
		if actor.IsCustomer() && e.Action == "note" {
			continue // never reveal internal notes to customers
		}
		r := TicketEventResponse{
			ID:        e.ID,
			Action:    e.Action,
			Summary:   e.Summary,
			CreatedAt: e.CreatedAt,
		}
		if e.User != nil {
			r.ActorName = displayName(e.User)
			r.ActorRole = e.User.Role
		} else {
			r.ActorName = "System"
		}
		out = append(out, r)
	}
	return out, nil
}
