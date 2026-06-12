package ticket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	stderrors "errors"
	"github.com/company/smartticket/internal/aiassist"
	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/realtime"
	"github.com/company/smartticket/internal/sla"
	"gorm.io/gorm"
)

// DeptScoper yields the department IDs a manager oversees (their subtree).
type DeptScoper interface {
	DeptScopeFor(userID uint) ([]uint, error)
}

// SupervisorResolver resolves the manager a user reports to (see department svc).
type SupervisorResolver interface {
	SupervisorOf(userID uint) (*models.User, error)
}

// Service provides ticket management business logic.
type Service struct {
	db                    *gorm.DB
	slaCalculator         *sla.Calculator
	notifier              Notifier           // optional; nil = no-op (see SetNotifier)
	mailer                Mailer             // optional; nil = no-op (see SetMailer)
	suggester             ReplySuggester     // optional; nil = AI suggestions unavailable
	bus                   *automation.Bus    // optional; nil = no domain events
	hub                   *realtime.Hub      // optional; nil = no ws broadcasts
	supervisors           SupervisorResolver // optional; nil = no supervisor notifications
	deptScoper            DeptScoper         // optional; enriches manager actors with dept subtree
	departmentIsolationFn func() bool        // optional; reads live isolation toggle
}

// SetBus wires the domain event bus. Safe to call after NewService.
func (s *Service) SetBus(b *automation.Bus) { s.bus = b }

// SetHub wires the realtime hub for WebSocket broadcasts. Safe to call after NewService.
func (s *Service) SetHub(h *realtime.Hub) { s.hub = h }

// SetSupervisors injects the supervisor resolver used to notify a manager on escalation.
func (s *Service) SetSupervisors(r SupervisorResolver) { s.supervisors = r }

// SetDeptScoper injects the department scoper used to enrich manager actors.
func (s *Service) SetDeptScoper(d DeptScoper) { s.deptScoper = d }

// SetDepartmentIsolation injects a live-reader for the department isolation toggle.
func (s *Service) SetDepartmentIsolation(fn func() bool) { s.departmentIsolationFn = fn }

// departmentIsolation reports the current value of the isolation toggle.
func (s *Service) departmentIsolation() bool {
	if s.departmentIsolationFn == nil {
		return false
	}
	return s.departmentIsolationFn()
}

// scopeToActor restricts a ticket query to the actor's view.
// - Customer actors: scoped to their customer (IDOR-safe).
// - Admin actors: unrestricted.
// - Non-admin team actors (when dept isolation is ON): restricted to tickets
//   assigned to users within the manager's department subtree.
// Manager DeptScope is enriched via deptScoper when not already set.
func (s *Service) scopeToActor(q *gorm.DB, actor authz.Actor) *gorm.DB {
	// Enrich a manager actor with their department subtree (best-effort).
	if s.deptScoper != nil && actor.IsTeam() && !actor.IsAdmin() && len(actor.DeptScope) == 0 {
		if scope, err := s.deptScoper.DeptScopeFor(actor.UserID); err == nil {
			actor.DeptScope = scope
		}
	}
	return authz.Scope(s.db, q, actor, authz.ScopeOptions{
		CustomerColumn:      "customer_id",
		AssigneeColumn:      "assigned_to",
		DepartmentIsolation: s.departmentIsolation(),
	})
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
	Channel        string `json:"channel,omitempty"`                 // e.g. web_widget; empty → default 'web'
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

	// Determine the channel: callers may request a specific channel (e.g.
	// "web_widget"); default to "web" when none is provided.
	channel := req.Channel
	if channel == "" {
		channel = "web"
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
		Channel:        channel,
	}

	if err := s.db.Create(ticket).Error; err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	s.recordEvent(ticket.ID, userID, "created", "created the ticket")

	// Emit domain event (best-effort; never affects the request path).
	if s.bus != nil {
		s.bus.Publish(automation.Event{
			Type:     automation.EventTicketCreated,
			TicketID: ticket.ID,
			ActorID:  actor.UserID,
		})
	}

	return s.ticketToResponse(ticket), nil
}

// GetTicket gets a ticket by ID, scoped to the actor's customer.
func (s *Service) GetTicket(actor authz.Actor, ticketID uint) (*TicketResponse, error) {
	var ticket models.Ticket
	if err := s.scopeToActor(s.db.Where("id = ?", ticketID), actor).
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
	query := s.scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor)

	// Apply filters
	statusFilter, hasStatusFilter := filters["status"].(string)
	if hasStatusFilter && statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	} else {
		// When no explicit status filter is provided, hide merged tickets from
		// the active queue. Callers that want merged tickets must request them
		// explicitly with status=merged.
		query = query.Where("status != ?", "merged")
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

// ListTicketsForDepartment returns tickets whose assignee belongs to the
// manager's department subtree — regardless of the global isolation toggle.
// Non-managers (empty DeptScope after enrichment) get an empty result.
// This is the backing implementation for ?scope=my_department.
func (s *Service) ListTicketsForDepartment(actor authz.Actor, page, pageSize int, filters map[string]interface{}) (*TicketListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Enrich actor with their managed department subtree.
	if s.deptScoper != nil && len(actor.DeptScope) == 0 {
		if scope, err := s.deptScoper.DeptScopeFor(actor.UserID); err == nil {
			actor.DeptScope = scope
		}
	}

	offset := (page - 1) * pageSize

	var tickets []models.Ticket
	var total int64

	// Always apply department isolation regardless of the global toggle.
	query := authz.Scope(s.db, s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor, authz.ScopeOptions{
		CustomerColumn:      "customer_id",
		AssigneeColumn:      "assigned_to",
		DepartmentIsolation: true,
	})

	// Apply the same filters as ListTickets.
	statusFilter, hasStatusFilter := filters["status"].(string)
	if hasStatusFilter && statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	} else {
		query = query.Where("status != ?", "merged")
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

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count department tickets: %w", err)
	}

	if err := query.Preload("AssignedUser").Preload("Customer").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&tickets).Error; err != nil {
		return nil, fmt.Errorf("failed to list department tickets: %w", err)
	}

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
	if err := s.scopeToActor(s.db.Where("id = ?", ticketID), actor).
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

	// Emit domain events for status transitions (best-effort).
	if s.bus != nil && req.Status != "" && req.Status != oldStatus {
		if ticket.Status == "resolved" {
			s.bus.Publish(automation.Event{
				Type:     automation.EventTicketResolved,
				TicketID: ticket.ID,
				ActorID:  userID,
			})
		} else {
			s.bus.Publish(automation.Event{
				Type:     automation.EventTicketUpdated,
				TicketID: ticket.ID,
				ActorID:  userID,
			})
		}
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
	result := s.scopeToActor(s.db.Model(&models.Ticket{}).Where("id = ?", ticketID), actor).
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
// NotFound error (no existence disclosure) when it is outside the actor's scope
// or has been soft-deleted.
func (s *Service) findTicketForActor(actor authz.Actor, ticketID uint) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := s.scopeToActor(s.db.Where("id = ? AND is_deleted = ?", ticketID, false), actor).First(&ticket).Error; err != nil {
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

	// @mention notifications: only for internal notes so that external messages
	// do not expose internal user handles to the customer-visible message stream.
	if isInternal {
		s.notifyMentions(tkt, message.Content, userID)
	}

	if isInternal {
		s.recordEvent(ticketID, userID, "note", "added an internal note")
	} else {
		s.recordEvent(ticketID, userID, "replied", "replied to the ticket")
	}

	// Load the author so the response carries the author's name/role.
	_ = s.db.Preload("User").First(message, message.ID).Error

	// Best-effort: email the requester when an agent posts a public reply, so
	// the conversation reaches them outside the app. Internal notes never leave.
	if s.mailer != nil && actor.IsTeam() && !isInternal && tkt != nil && strings.TrimSpace(tkt.RequesterEmail) != "" {
		author := "Support"
		if message.User != nil {
			if n := strings.TrimSpace(message.User.FirstName + " " + message.User.LastName); n != "" {
				author = n
			}
		}
		go s.mailer.SendTicketReply(context.Background(), tkt.RequesterEmail, tkt.TicketNumber, tkt.Title, tkt.ID, message.Content, author)
	}

	resp := messageToResponse(message)

	// Emit domain event and broadcast over WebSocket (best-effort).
	if s.bus != nil {
		s.bus.Publish(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: ticketID,
			ActorID:  actor.UserID,
		})
	}
	if s.hub != nil {
		if payload, err := json.Marshal(resp); err == nil {
			// Broadcast to the ticket room so connected agents see the new message.
			s.hub.Broadcast(fmt.Sprintf("ticket:%d", ticketID), payload)
			// Also broadcast to the widget room when the ticket originates from a
			// web_widget conversation so the embedded chat widget receives the reply.
			if tkt != nil && tkt.Channel == "web_widget" {
				s.hub.Broadcast(fmt.Sprintf("widget:%d", ticketID), payload)
			}
		}
	}

	return &resp, nil
}

// PostAIMessage appends a public AI-authored reply to the ticket.
// It bypasses the human notification/mailer pipeline and publishes the
// EventMessageCreated event with Source:"ai" so the AutoResolver's loop-guard
// fires and the message is not re-processed.
func (s *Service) PostAIMessage(ticketID uint, body string) error {
	var tkt models.Ticket
	if err := s.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("ticket")
		}
		return fmt.Errorf("failed to load ticket for AI message: %w", err)
	}

	msg := &models.Message{
		TicketID:    ticketID,
		UserID:      0, // system / no human author
		Content:     strings.TrimSpace(body),
		ContentType: "text",
		IsInternal:  false,
		IsFromAI:    true,
	}
	if err := s.db.Create(msg).Error; err != nil {
		return fmt.Errorf("failed to persist AI message: %w", err)
	}

	s.recordEvent(ticketID, 0, "replied", "AI replied to the ticket")

	resp := messageToResponse(msg)

	// Publish with Source:"ai" — this is the key that prevents the orchestrator
	// from treating this event as a new customer message (loop guard).
	if s.bus != nil {
		s.bus.Publish(automation.Event{
			Type:     automation.EventMessageCreated,
			TicketID: ticketID,
			ActorID:  0,
			Source:   "ai",
		})
	}
	if s.hub != nil {
		if payload, err := json.Marshal(resp); err == nil {
			s.hub.Broadcast(fmt.Sprintf("ticket:%d", ticketID), payload)
			if tkt.Channel == "web_widget" {
				s.hub.Broadcast(fmt.Sprintf("widget:%d", ticketID), payload)
			}
		}
	}

	return nil
}

// CountAIMessages returns the number of AI-authored messages on a ticket.
func (s *Service) CountAIMessages(ticketID uint) (int, error) {
	var count int64
	if err := s.db.Model(&models.Message{}).
		Where("ticket_id = ? AND is_from_ai = ?", ticketID, true).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count AI messages: %w", err)
	}
	return int(count), nil
}

// UpdateTicketClassification sets category, priority, and tags on a ticket
// without triggering the full UpdateTicket lifecycle (no SLA recalc, no
// activity log for silent AI updates).
func (s *Service) UpdateTicketClassification(ticketID uint, category, priority string, tags []string) error {
	updates := map[string]interface{}{}
	if category != "" {
		updates["category"] = category
	}
	if priority != "" {
		updates["priority"] = priority
	}
	if len(tags) > 0 {
		raw, _ := json.Marshal(tags)
		updates["tags"] = string(raw)
	}
	if len(updates) == 0 {
		return nil
	}
	result := s.db.Model(&models.Ticket{}).Where("id = ?", ticketID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update ticket classification: %w", result.Error)
	}
	return nil
}

// SetTicketSummary stores an AI-generated summary on the ticket.
func (s *Service) SetTicketSummary(ticketID uint, summary string) error {
	result := s.db.Model(&models.Ticket{}).Where("id = ?", ticketID).Update("summary", summary)
	if result.Error != nil {
		return fmt.Errorf("set ticket summary: %w", result.Error)
	}
	return nil
}

// LoadAISuggestInput assembles the SuggestInput for a ticket and reports
// whether the last message in the conversation comes from a customer (i.e.
// someone is waiting for a reply).  On a brand-new ticket with no messages
// the description itself counts as the customer's opening, so customerWaiting
// is true.
func (s *Service) LoadAISuggestInput(ticketID uint) (aiassist.SuggestInput, bool, error) {
	var tkt models.Ticket
	if err := s.db.Where("id = ? AND is_deleted = ?", ticketID, false).
		Preload("Customer").First(&tkt).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return aiassist.SuggestInput{}, false, errors.NewNotFoundError("ticket")
		}
		return aiassist.SuggestInput{}, false, fmt.Errorf("load ticket for AI: %w", err)
	}

	conv, _ := s.buildConversation(ticketID)

	in := aiassist.SuggestInput{
		Title:        tkt.Title,
		Description:  tkt.Description,
		CustomerName: tkt.RequesterName,
		Conversation: conv,
	}

	// Determine whether the last conversational turn is from a customer.
	// If there are no messages the ticket description is the opening statement
	// from the requester, so we treat that as customer-waiting.
	customerWaiting := true
	if len(conv) > 0 {
		customerWaiting = conv[len(conv)-1].IsCustomer
	}

	return in, customerWaiting, nil
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
		Merged        int64 `json:"merged"`
		OverdueCount  int64 `json:"overdue_count"`
		CriticalCount int64 `json:"critical_count"`
		HighCount     int64 `json:"high_count"`
		MediumCount   int64 `json:"medium_count"`
		LowCount      int64 `json:"low_count"`
	}

	// Get basic status counts
	if err := s.scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
		Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total tickets: %w", err)
	}

	// Get status breakdown
	rows, err := s.scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
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
		case "merged":
			stats.Merged = count
		}
	}

	// Get priority breakdown
	priorityRows, err := s.scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
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
	if err := s.scopeToActor(s.db.Model(&models.Ticket{}).Where("is_deleted = ?", false), actor).
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
		"merged_tickets":      stats.Merged,
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
			"merged":      stats.Merged,
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

// SetFieldAutomation updates a single field (priority|status|severity) on a ticket
// and publishes the resulting domain event with Source:"automation" so the engine's
// recursion guard fires and does not re-process the event.
func (s *Service) SetFieldAutomation(ticketID uint, field, value string) error {
	var tkt models.Ticket
	if err := s.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		return fmt.Errorf("SetFieldAutomation: load ticket: %w", err)
	}

	updates := map[string]interface{}{field: value}
	if field == "status" && value == "resolved" {
		now := time.Now()
		updates["resolved_at"] = now
		updates["resolution_time"] = now
	}

	if err := s.db.Model(&models.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error; err != nil {
		return fmt.Errorf("SetFieldAutomation: update %s: %w", field, err)
	}

	if s.bus != nil {
		s.bus.Publish(automation.Event{
			Type:     automation.EventTicketUpdated,
			TicketID: ticketID,
			ActorID:  0,
			Source:   "automation",
		})
	}
	return nil
}

// AddTagAutomation appends tag to the ticket's JSON tag list (no-op if already present).
// Publishes EventTicketUpdated with Source:"automation".
func (s *Service) AddTagAutomation(ticketID uint, tag string) error {
	var tkt models.Ticket
	if err := s.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		return fmt.Errorf("AddTagAutomation: load ticket: %w", err)
	}

	var tags []string
	if tkt.Tags != "" {
		_ = json.Unmarshal([]byte(tkt.Tags), &tags)
	}
	for _, t := range tags {
		if t == tag {
			return nil // already present
		}
	}
	tags = append(tags, tag)
	raw, _ := json.Marshal(tags)

	if err := s.db.Model(&models.Ticket{}).Where("id = ?", ticketID).
		Update("tags", string(raw)).Error; err != nil {
		return fmt.Errorf("AddTagAutomation: save tags: %w", err)
	}

	if s.bus != nil {
		s.bus.Publish(automation.Event{
			Type:     automation.EventTicketUpdated,
			TicketID: ticketID,
			Source:   "automation",
		})
	}
	return nil
}

// AssignAutomation sets assigned_to and/or assigned_team_id on a ticket.
// Publishes EventTicketUpdated with Source:"automation".
func (s *Service) AssignAutomation(ticketID uint, userID, teamID *uint) error {
	updates := map[string]interface{}{}
	if userID != nil {
		updates["assigned_to"] = userID
	}
	if teamID != nil {
		updates["assigned_team_id"] = teamID
	}
	if len(updates) == 0 {
		return nil
	}
	if err := s.db.Model(&models.Ticket{}).Where("id = ? AND is_deleted = ?", ticketID, false).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("AssignAutomation: %w", err)
	}
	if s.bus != nil {
		s.bus.Publish(automation.Event{
			Type:     automation.EventTicketUpdated,
			TicketID: ticketID,
			Source:   "automation",
		})
	}
	return nil
}

// EscalateAutomation bumps the ticket priority one rank (e.g. medium → high) and
// best-effort notifies the assignee's supervisor so escalation reaches a person.
// Publishes EventTicketUpdated with Source:"automation".
func (s *Service) EscalateAutomation(ticketID uint) error {
	escalate := map[string]string{
		"low":    "medium",
		"medium": "high",
		"high":   "critical",
	}
	var tkt models.Ticket
	if err := s.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		return fmt.Errorf("EscalateAutomation: load: %w", err)
	}
	if next, ok := escalate[tkt.Priority]; ok {
		if err := s.SetFieldAutomation(ticketID, "priority", next); err != nil {
			return err
		}
	}
	// Best-effort: notify the assignee's supervisor so escalation reaches a person.
	if s.supervisors != nil && tkt.AssignedTo != nil && s.notifier != nil {
		if sup, err := s.supervisors.SupervisorOf(*tkt.AssignedTo); err == nil && sup != nil {
			body := fmt.Sprintf("Ticket %s was escalated and needs your attention.", tkt.TicketNumber)
			s.notifier.Notify(context.Background(), []uint{sup.ID}, "ticket_escalated",
				"Ticket escalated", body, "ticket", tkt.ID)
		}
	}
	return nil
}
