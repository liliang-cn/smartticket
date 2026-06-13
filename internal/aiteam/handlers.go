package aiteam

import (
	"fmt"
	"net/http"

	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for on-demand AI advisory team endpoints.
type Handlers struct {
	orch     *Orchestrator
	store    *SuggestionStore
	buildCtx func(ticketID uint) TicketContext
}

// NewHandlers creates Handlers. orch may be nil (AI not configured);
// in that case all Run-based endpoints return 503.
func NewHandlers(orch *Orchestrator, store *SuggestionStore, buildCtx func(uint) TicketContext) *Handlers {
	return &Handlers{orch: orch, store: store, buildCtx: buildCtx}
}

// parseTicketID extracts the ":id" route param and returns 400 on failure.
func parseTicketID(c *gin.Context) (uint, bool) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
		return 0, false
	}
	return id, true
}

// parseSuggestionID extracts the ":sid" route param and returns 400 on failure.
func parseSuggestionID(c *gin.Context) (uint, bool) {
	var sid uint
	if _, err := fmt.Sscanf(c.Param("sid"), "%d", &sid); err != nil || sid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid suggestion id"})
		return 0, false
	}
	return sid, true
}

// Research dispatches the Researcher advisory agent on demand.
// @Summary Run AI Researcher on a ticket
// @Tags ai
// @Security BearerAuth
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 {object} models.AISuggestion
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/tickets/{id}/ai/research [post]
func (h *Handlers) Research(c *gin.Context) {
	if h.orch == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI not configured"})
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	// Async: return immediately with a pending suggestion; the result streams to
	// the Copilot panel over the realtime hub when the agent finishes.
	sug := h.orch.RunAsync("Researcher", h.buildCtx(id), "")
	if sug == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI disabled"})
		return
	}
	c.JSON(http.StatusAccepted, sug)
}

// reviewRequest is the body for the Review endpoint.
type reviewRequest struct {
	Draft string `json:"draft"`
}

// Review dispatches the Reviewer advisory agent on demand with a draft.
// @Summary Run AI Reviewer on a ticket draft
// @Tags ai
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Ticket ID"
// @Param request body reviewRequest true "Draft to review"
// @Success 200 {object} models.AISuggestion
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/tickets/{id}/ai/review [post]
func (h *Handlers) Review(c *gin.Context) {
	if h.orch == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI not configured"})
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	var body reviewRequest
	// Ignore bind error — draft is optional; an empty string is valid for the Reviewer.
	_ = c.ShouldBindJSON(&body)

	sug := h.orch.RunAsync("Reviewer", h.buildCtx(id), body.Draft)
	if sug == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI disabled"})
		return
	}
	c.JSON(http.StatusAccepted, sug)
}

// Draft dispatches the Drafter advisory agent on demand.
// @Summary Run AI Drafter on a ticket
// @Tags ai
// @Security BearerAuth
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 {object} models.AISuggestion
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/tickets/{id}/ai/draft [post]
func (h *Handlers) Draft(c *gin.Context) {
	if h.orch == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI not configured"})
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	sug := h.orch.RunAsync("Drafter", h.buildCtx(id), "")
	if sug == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI disabled"})
		return
	}
	c.JSON(http.StatusAccepted, sug)
}

// List returns all AI suggestions for a ticket.
// @Summary List AI suggestions for a ticket
// @Tags ai
// @Security BearerAuth
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tickets/{id}/ai/suggestions [get]
func (h *Handlers) List(c *gin.Context) {
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	sugs, err := h.store.List(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sugs == nil {
		sugs = []models.AISuggestion{} // ensure JSON sends [] not null
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": sugs})
}

// Adopt marks an AI suggestion as adopted by the current user.
// The frontend applies the actual change via existing ticket APIs.
// @Summary Adopt an AI suggestion
// @Tags ai
// @Security BearerAuth
// @Produce json
// @Param id path int true "Ticket ID"
// @Param sid path int true "Suggestion ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tickets/{id}/ai/suggestions/{sid}/adopt [post]
func (h *Handlers) Adopt(c *gin.Context) {
	ticketID, ok := parseTicketID(c)
	if !ok {
		return
	}
	sid, ok := parseSuggestionID(c)
	if !ok {
		return
	}
	if !h.suggestionBelongsToTicket(c, sid, ticketID) {
		return
	}
	userID := c.GetUint("user_id")
	if err := h.store.Adopt(sid, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"adopted": true})
}

// Dismiss marks an AI suggestion as dismissed.
// @Summary Dismiss an AI suggestion
// @Tags ai
// @Security BearerAuth
// @Produce json
// @Param id path int true "Ticket ID"
// @Param sid path int true "Suggestion ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tickets/{id}/ai/suggestions/{sid}/dismiss [post]
func (h *Handlers) Dismiss(c *gin.Context) {
	ticketID, ok := parseTicketID(c)
	if !ok {
		return
	}
	sid, ok := parseSuggestionID(c)
	if !ok {
		return
	}
	if !h.suggestionBelongsToTicket(c, sid, ticketID) {
		return
	}
	if err := h.store.Dismiss(sid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dismissed": true})
}

// suggestionBelongsToTicket verifies suggestion sid is attached to ticketID,
// closing an IDOR where a caller authorized for one ticket could adopt/dismiss
// a suggestion belonging to another. Writes the error response on failure.
func (h *Handlers) suggestionBelongsToTicket(c *gin.Context, sid, ticketID uint) bool {
	sug, err := h.store.Get(sid)
	if err != nil || sug == nil || sug.TicketID != ticketID {
		c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
		return false
	}
	return true
}
