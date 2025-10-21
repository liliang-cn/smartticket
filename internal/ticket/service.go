package ticket

import (
	"fmt"
	"strings"
	"time"

	stderrors "errors"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"gorm.io/gorm"
)

// Service provides ticket management business logic.
type Service struct {
	db            *gorm.DB
	slaCalculator *sla.Calculator
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

// CreateTicket creates a new ticket.
func (s *Service) CreateTicket(tenantID uint, userID uint, req *CreateTicketRequest) (*TicketResponse, error) {
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
	ticketNumber, err := s.generateTicketNumber(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ticket number: %w", err)
	}

	// Calculate SLA using the SLA calculator
	slaDueDate, err := s.slaCalculator.CalculateSLADueDates(tenantID, req.Priority, req.Severity, req.ProductID, req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate SLA: %w", err)
	}

	// Create ticket
	ticket := &models.Ticket{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: nil, // Temporarily nil to avoid FK constraints
			UpdatedBy: nil,
		},
		TenantID:       tenantID,
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

	return s.ticketToResponse(ticket), nil
}

// GetTicket gets a ticket by ID.
func (s *Service) GetTicket(tenantID uint, ticketID uint) (*TicketResponse, error) {
	var ticket models.Ticket
	if err := s.db.Where("id = ? AND tenant_id = ?", ticketID, tenantID).
		Preload("AssignedUser").
		First(&ticket).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("ticket")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return s.ticketToResponse(&ticket), nil
}

// ListTickets lists tickets with pagination and filtering.
func (s *Service) ListTickets(tenantID uint, page, pageSize int, filters map[string]interface{}) (*TicketListResponse, error) {
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

	query := s.db.Model(&models.Ticket{}).
		Where("tenant_id = ?", tenantID)

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
	if err := query.Preload("AssignedUser").
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

// UpdateTicket updates a ticket.
func (s *Service) UpdateTicket(tenantID uint, ticketID uint, userID uint, req *UpdateTicketRequest) (*TicketResponse, error) {
	var ticket models.Ticket
	if err := s.db.Where("id = ? AND tenant_id = ?", ticketID, tenantID).
		First(&ticket).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("ticket")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

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
	if req.AssignedTo != nil {
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
		slaDueDate, err := s.slaCalculator.CalculateSLADueDates(tenantID, ticket.Priority, ticket.Severity, ticket.ProductID, ticket.ServiceID)
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

	// Reload with associations
	if err := s.db.Preload("AssignedUser").
		First(&ticket, ticket.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload ticket: %w", err)
	}

	return s.ticketToResponse(&ticket), nil
}

// DeleteTicket soft deletes a ticket.
func (s *Service) DeleteTicket(tenantID uint, ticketID uint) error {
	result := s.db.Model(&models.Ticket{}).
		Where("id = ? AND tenant_id = ?", ticketID, tenantID).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("failed to delete ticket: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("ticket")
	}

	return nil
}

// AssignTicket assigns a ticket to a user.
func (s *Service) AssignTicket(tenantID uint, ticketID uint, assignedTo uint) error {
	result := s.db.Model(&models.Ticket{}).
		Where("id = ? AND tenant_id = ?", ticketID, tenantID).
		Update("assigned_to", &assignedTo)

	if result.Error != nil {
		return fmt.Errorf("failed to assign ticket: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("ticket")
	}

	return nil
}

// GetTicketStats gets ticket statistics.
func (s *Service) GetTicketStats(tenantID uint) (map[string]interface{}, error) {
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
	if err := s.db.Model(&models.Ticket{}).
		Where("tenant_id = ?", tenantID).
		Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total tickets: %w", err)
	}

	// Get status breakdown
	rows, err := s.db.Model(&models.Ticket{}).
		Where("tenant_id = ?", tenantID).
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
	priorityRows, err := s.db.Model(&models.Ticket{}).
		Where("tenant_id = ? AND status IN ?", tenantID, []string{"open", "in_progress"}).
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
	if err := s.db.Model(&models.Ticket{}).
		Where("tenant_id = ? AND status IN ? AND due_date < ?",
			tenantID, []string{"open", "in_progress"}, now).
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

func (s *Service) generateTicketNumber(tenantID uint) (string, error) {
	var count int64
	if err := s.db.Model(&models.Ticket{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return "", err
	}

	// Format: TK-{tenant_id}-{sequence_number}
	return fmt.Sprintf("TK-%d-%d", tenantID, count+1), nil
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

	// Add assigned user info
	if ticket.AssignedUser != nil {
		response.AssignedUser = &UserInfo{
			ID:        ticket.AssignedUser.ID,
			Email:     ticket.AssignedUser.Email,
			Username:  ticket.AssignedUser.Username,
			FirstName: ticket.AssignedUser.FirstName,
			LastName:  ticket.AssignedUser.LastName,
			Role:      ticket.AssignedUser.Role,
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
