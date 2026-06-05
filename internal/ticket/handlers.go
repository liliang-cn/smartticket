package ticket

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// actorFromContext builds the authorization Actor from the values the auth
// middleware places in the gin context.
func actorFromContext(c *gin.Context) authz.Actor {
	a := authz.Actor{
		UserID: c.GetUint("user_id"),
		Role:   c.GetString("user_role"),
	}
	if v, ok := c.Get("user_customer_id"); ok {
		if cid, ok := v.(uint); ok {
			a.CustomerID = &cid
		}
	}
	return a
}

// Handlers provides ticket HTTP handlers.
type Handlers struct {
	service   *Service
	validator *validator.Validate
}

// NewHandlers creates new ticket handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service:   service,
		validator: validator.New(),
	}
}

// CreateTicket creates a new ticket.
// @Summary Create a new ticket
// @Description Creates a new support ticket with the provided details
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param request body ticket.CreateTicketRequest true "Ticket creation data"
// @Success 201 {object} ticket.TicketResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets [post]
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

	// Log ticket creation attempt
	c.Set("security_event", "ticket_creation_attempt")
	c.Set("target_resource", req.Title)

	// Create ticket
	ticket, err := h.service.CreateTicket(actorFromContext(c), userID, &req)
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

// GetTicket gets a ticket by ID.
// @Summary Get a ticket by ID
// @Description Retrieves a specific ticket by its unique identifier
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Ticket ID"
// @Success 200 {object} ticket.TicketResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id} [get]
func (h *Handlers) GetTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("ticket_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	ticket, err := h.service.GetTicket(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    ticket,
	})
}

// GetTicketSLA returns the SLA policy governing a ticket (matched rule +
// template, targets, due date and status).
// @Summary Get a ticket's SLA policy
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tickets/{id}/sla [get]
func (h *Handlers) GetTicketSLA(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}
	sla, err := h.service.GetTicketSLA(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sla})
}

// GetTicketEvents returns a ticket's activity log (creation, status/priority
// changes, assignments, replies). Customer-isolated; internal-note events are
// hidden from customer actors.
// @Summary Get a ticket's activity log
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tickets/{id}/events [get]
func (h *Handlers) GetTicketEvents(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}
	events, err := h.service.ListTicketEvents(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": events})
}

// ListTickets lists tickets with pagination and filtering.
// @Summary List tickets
// @Description Retrieves a paginated list of tickets with optional filtering
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of tickets per page" default(20) minimum(1) maximum(100)
// @Param status query string false "Filter by ticket status" Enums(open,in_progress,resolved,closed,cancelled)
// @Param priority query string false "Filter by priority" Enums(low,medium,high,critical)
// @Param category query string false "Filter by category"
// @Param assigned_to query int false "Filter by assigned user ID"
// @Param search query string false "Search tickets by title, description, or requester"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets [get]
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

	tickets, err := h.service.ListTickets(actorFromContext(c), page, pageSize, filters)
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

// UpdateTicket updates a ticket.
// @Summary Update a ticket
// @Description Updates an existing ticket with new information
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Ticket ID"
// @Param request body ticket.UpdateTicketRequest true "Ticket update data"
// @Success 200 {object} ticket.TicketResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id} [put]
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

	// Log ticket update attempt
	c.Set("security_event", "ticket_update_attempt")
	c.Set("target_resource_id", uint(id))

	// Update ticket
	ticket, err := h.service.UpdateTicket(actorFromContext(c), uint(id), userID, &req)
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

// DeleteTicket deletes a ticket.
// @Summary Delete a ticket
// @Description Soft deletes a ticket (marks as deleted but preserves data)
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Ticket ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id} [delete]
func (h *Handlers) DeleteTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("ticket_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log ticket deletion attempt
	c.Set("security_event", "ticket_deletion_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.DeleteTicket(actorFromContext(c), uint(id)); err != nil {
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

// AssignTicket assigns a ticket to a user.
// @Summary Assign a ticket
// @Description Assigns a ticket to a specific user for handling
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Ticket ID"
// @Param request body object{assigned_to:int} true "Assignment data"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id}/assign [post]
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

	// Log ticket assignment attempt
	c.Set("security_event", "ticket_assignment_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.AssignTicket(actorFromContext(c), uint(id), req.AssignedTo); err != nil {
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

// GetTicketStats gets ticket statistics.
// @Summary Get ticket statistics
// @Description Retrieves statistical information about tickets
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/stats [get]
func (h *Handlers) GetTicketStats(c *gin.Context) {
	stats, err := h.service.GetTicketStats(actorFromContext(c))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetMyTickets gets tickets for the current user.
// @Summary Get my tickets
// @Description Retrieves tickets assigned to the currently authenticated user
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of tickets per page" default(20) minimum(1) maximum(100)
// @Param status query string false "Filter by ticket status" Enums(open,in_progress,resolved,closed,cancelled)
// @Param priority query string false "Filter by priority" Enums(low,medium,high,critical)
// @Param search query string false "Search tickets by title, description, or requester"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/my [get]
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

	tickets, err := h.service.ListTickets(actorFromContext(c), page, pageSize, filters)
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

// GetTicketMessages lists a ticket's messages.
// @Summary List ticket messages
// @Description Lists messages on a ticket. Customers see only their own customer's tickets and never internal notes.
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id}/messages [get]
func (h *Handlers) GetTicketMessages(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	messages, err := h.service.ListMessages(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": messages})
}

// CreateTicketMessage adds a message to a ticket.
// @Summary Add ticket message
// @Description Adds a message to a ticket. Customers cannot create internal notes and can only post to their own customer's tickets.
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Param request body ticket.CreateMessageRequest true "Message content"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id}/messages [post]
func (h *Handlers) CreateTicketMessage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}

	message, err := h.service.CreateMessage(actorFromContext(c), uint(id), c.GetUint("user_id"), &req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": message})
}

// MergeTicket merges the ticket at :id into the target specified in the JSON body.
// Body: {"into": <targetID>}
// Team-only.
func (h *Handlers) MergeTicket(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	var body struct {
		Into uint `json:"into" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}

	if err := h.service.Merge(actorFromContext(c), uint(id), body.Into); err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Ticket merged successfully"})
}

// CreateTicketLink creates a link between the ticket at :id and a target ticket.
// Body: {"target_id": <uint>, "type": "related|duplicate|blocks"}
// Team-only.
func (h *Handlers) CreateTicketLink(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	var body struct {
		TargetID uint   `json:"target_id" binding:"required"`
		Type     string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}

	link, err := h.service.LinkTickets(actorFromContext(c), uint(id), body.TargetID, body.Type)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": link})
}

// ListTicketLinks returns all links for the ticket at :id.
func (h *Handlers) ListTicketLinks(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	links, err := h.service.ListLinks(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": links})
}

// UnlinkTicket deletes the link identified by :linkId.
// The link must be associated with the ticket at :id.
// Team-only.
func (h *Handlers) UnlinkTicket(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	linkID, err := strconv.ParseUint(c.Param("linkId"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("link_id", c.Param("linkId")))
		return
	}

	if err := h.service.Unlink(actorFromContext(c), uint(ticketID), uint(linkID)); err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Link removed successfully"})
}

// SuggestReply returns an AI-drafted reply for the ticket (team-only). The
// service maps AI unavailability (disabled / no provider) to a clear error.
// The response includes structured fields (confidence, needs_clarification,
// used_kb, sources) in addition to the reply text. The "reply" key is
// preserved so the existing frontend (which reads .data.reply) continues to
// work unchanged.
func (h *Handlers) SuggestReply(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}
	draft, err := h.service.SuggestReplyDraft(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"reply":               draft.Reply,
			"confidence":          draft.Confidence,
			"needs_clarification": draft.NeedsClarification,
			"used_kb":             draft.UsedKB,
			"sources":             draft.Sources,
		},
	})
}
