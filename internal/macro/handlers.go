package macro

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for the macro endpoints.
type Handlers struct {
	service *Service
}

// NewHandlers creates macro HTTP handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// callerID extracts the acting user's ID from the gin context (set by authMiddleware).
func callerID(c *gin.Context) uint {
	return c.GetUint("user_id")
}

// List returns all macros visible to the acting user.
// GET /macros
func (h *Handlers) List(c *gin.Context) {
	macros, err := h.service.List(callerID(c))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": macros})
}

// Create creates a new macro owned by the acting user.
// POST /macros
func (h *Handlers) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	m, err := h.service.Create(callerID(c), req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": m})
}

// Get retrieves a single macro by ID (visibility-checked).
// GET /macros/:id
func (h *Handlers) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		errors.ErrorHandler(c, errors.NewValidationError("invalid macro id"))
		return
	}
	m, serr := h.service.Get(callerID(c), id)
	if serr != nil {
		errors.ErrorHandler(c, serr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": m})
}

// Update patches a macro.
// PUT /macros/:id
func (h *Handlers) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		errors.ErrorHandler(c, errors.NewValidationError("invalid macro id"))
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	m, serr := h.service.Update(callerID(c), id, req)
	if serr != nil {
		errors.ErrorHandler(c, serr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": m})
}

// Delete removes a macro (visibility + ownership checked).
// DELETE /macros/:id
func (h *Handlers) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		errors.ErrorHandler(c, errors.NewValidationError("invalid macro id"))
		return
	}
	if serr := h.service.Delete(callerID(c), id); serr != nil {
		errors.ErrorHandler(c, serr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ApplyRequest is the body for POST /macros/:id/apply.
// The server layer (server.go) sets a "macro_render_ctx" key in the gin context
// when the acting user's name and ticket details are resolved from the ticket
// service. If not set, the handler falls back to the JSON body fields.
type ApplyRequest struct {
	CustomerName  string `json:"customer_name"`
	AgentName     string `json:"agent_name"`
	TicketID      string `json:"ticket_id"`
	TicketSubject string `json:"ticket_subject"`
}

// Apply renders the macro body and returns {rendered, actions}.
// POST /macros/:id/apply
func (h *Handlers) Apply(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		errors.ErrorHandler(c, errors.NewValidationError("invalid macro id"))
		return
	}

	// Prefer context-injected RenderContext (assembled by server.go from ticket
	// service + user service) over a manually-supplied body.
	var rctx RenderContext
	if v, ok := c.Get("macro_render_ctx"); ok {
		if rc, ok := v.(RenderContext); ok {
			rctx = rc
		}
	} else {
		var req ApplyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
			return
		}
		rctx = RenderContext{
			CustomerName:  req.CustomerName,
			AgentName:     req.AgentName,
			TicketID:      req.TicketID,
			TicketSubject: req.TicketSubject,
		}
	}

	rendered, actions, serr := h.service.Apply(id, callerID(c), rctx)
	if serr != nil {
		errors.ErrorHandler(c, serr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"rendered": rendered,
		"actions":  actions,
	}})
}

// parseID parses the :id path parameter as a uint.
func parseID(c *gin.Context) (uint, error) {
	v, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
