package ticket

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handlers provides ticket HTTP handlers
type Handlers struct {
	service   *Service
	validator *validator.Validate
}

// NewHandlers creates new ticket handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service:   service,
		validator: validator.New(),
	}
}

// CreateTicket creates a new ticket
func (h *Handlers) CreateTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	// Log ticket creation attempt
	c.Set("security_event", "ticket_creation_attempt")
	c.Set("target_resource", req.Title)

	// Create ticket
	ticket, err := h.service.CreateTicket(tenantID, userID, &req)
	if err != nil {
		c.Set("security_event", "ticket_creation_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful ticket creation
	c.Set("security_event", "ticket_created")
	c.Set("target_resource", ticket.TicketNumber)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    ticket,
	})
}

// GetTicket gets a ticket by ID
func (h *Handlers) GetTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("ticket_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	tenantID := c.GetUint("tenant_id")

	ticket, err := h.service.GetTicket(tenantID, uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    ticket,
	})
}

// ListTickets lists tickets with pagination and filtering
func (h *Handlers) ListTickets(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Build filters
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if priority := c.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if assignedToStr := c.Query("assigned_to"); assignedToStr != "" {
		if assignedTo, err := strconv.ParseUint(assignedToStr, 10, 32); err == nil {
			filters["assigned_to"] = uint(assignedTo)
		}
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	tenantID := c.GetUint("tenant_id")

	tickets, err := h.service.ListTickets(tenantID, page, pageSize, filters)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tickets.Data,
		"meta": gin.H{
			"total":       tickets.Total,
			"page":        tickets.Page,
			"page_size":   tickets.PageSize,
			"total_pages": tickets.TotalPages,
		},
	})
}

// UpdateTicket updates a ticket
func (h *Handlers) UpdateTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("ticket_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	// Log ticket update attempt
	c.Set("security_event", "ticket_update_attempt")
	c.Set("target_resource_id", uint(id))

	// Update ticket
	ticket, err := h.service.UpdateTicket(tenantID, uint(id), userID, &req)
	if err != nil {
		c.Set("security_event", "ticket_update_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful ticket update
	c.Set("security_event", "ticket_updated")
	c.Set("target_resource", ticket.TicketNumber)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    ticket,
	})
}

// DeleteTicket deletes a ticket
func (h *Handlers) DeleteTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("ticket_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	tenantID := c.GetUint("tenant_id")

	// Log ticket deletion attempt
	c.Set("security_event", "ticket_deletion_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.DeleteTicket(tenantID, uint(id)); err != nil {
		c.Set("security_event", "ticket_deletion_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful ticket deletion
	c.Set("security_event", "ticket_deleted")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Ticket deleted successfully",
	})
}

// AssignTicket assigns a ticket to a user
func (h *Handlers) AssignTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("ticket_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	var req struct {
		AssignedTo uint `json:"assigned_to" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	tenantID := c.GetUint("tenant_id")

	// Log ticket assignment attempt
	c.Set("security_event", "ticket_assignment_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.AssignTicket(tenantID, uint(id), req.AssignedTo); err != nil {
		c.Set("security_event", "ticket_assignment_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful ticket assignment
	c.Set("security_event", "ticket_assigned")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Ticket assigned successfully",
	})
}

// GetTicketStats gets ticket statistics
func (h *Handlers) GetTicketStats(c *gin.Context) {
	tenantID := c.GetUint("tenant_id")

	stats, err := h.service.GetTicketStats(tenantID)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetMyTickets gets tickets for the current user
func (h *Handlers) GetMyTickets(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Build filters
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if priority := c.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	// Add filter for assigned to current user
	userID := c.GetUint("user_id")
	filters["assigned_to"] = userID

	tenantID := c.GetUint("tenant_id")

	tickets, err := h.service.ListTickets(tenantID, page, pageSize, filters)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tickets.Data,
		"meta": gin.H{
			"total":       tickets.Total,
			"page":        tickets.Page,
			"page_size":   tickets.PageSize,
			"total_pages": tickets.TotalPages,
		},
	})
}
