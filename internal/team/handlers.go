package team

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for team management.
type Handlers struct {
	service *Service
}

// NewHandlers creates new team handlers.
func NewHandlers(service *Service) *Handlers { return &Handlers{service: service} }

// parseID parses a uint route parameter. On failure it writes a 400 response
// and returns (0, false).
func parseID(c *gin.Context, param string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || v == 0 {
		errors.ErrorHandler(c, errors.NewInvalidInputError(param, "must be a positive integer"))
		return 0, false
	}
	return uint(v), true
}

// ListTeams returns all teams.
// GET /teams
func (h *Handlers) ListTeams(c *gin.Context) {
	teams, err := h.service.ListTeams()
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": teams})
}

// CreateTeam creates a new team (admin-only).
// POST /teams
func (h *Handlers) CreateTeam(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	t, err := h.service.CreateTeam(&req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": t})
}

// GetTeam returns a single team by ID.
// GET /teams/:id
func (h *Handlers) GetTeam(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	t, err := h.service.GetTeam(id)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": t})
}

// UpdateTeam patches a team (admin-only).
// PUT /teams/:id
func (h *Handlers) UpdateTeam(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	t, err := h.service.UpdateTeam(id, &req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": t})
}

// DeleteTeam deletes a team (admin-only).
// DELETE /teams/:id
func (h *Handlers) DeleteTeam(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteTeam(id); err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListMembers returns the members of a team.
// GET /teams/:id/members
func (h *Handlers) ListMembers(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	members, err := h.service.ListMembers(id)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": members})
}

// AddMember adds a user to a team (admin-only). Idempotent.
// POST /teams/:id/members  body: {user_id}
func (h *Handlers) AddMember(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	if err := h.service.AddMember(id, req.UserID); err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RemoveMember removes a user from a team (admin-only).
// DELETE /teams/:id/members/:userId
func (h *Handlers) RemoveMember(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	uid, ok := parseID(c, "userId")
	if !ok {
		return
	}
	if err := h.service.RemoveMember(id, uid); err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
