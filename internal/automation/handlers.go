package automation

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	apperrors "github.com/company/smartticket/internal/errors"
)

// Handlers provides HTTP handlers for the automation rules API.
type Handlers struct {
	svc *Service
}

// NewHandlers creates a new Handlers backed by svc.
func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

// ListRules handles GET /automations
func (h *Handlers) ListRules(c *gin.Context) {
	rules, err := h.svc.ListRules()
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rules})
}

// CreateRule handles POST /automations
func (h *Handlers) CreateRule(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.ErrorHandler(c, apperrors.NewValidationError(err.Error()))
		return
	}
	rule, err := h.svc.CreateRule(&req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": rule})
}

// GetRule handles GET /automations/:id
func (h *Handlers) GetRule(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	rule, err := h.svc.GetRule(id)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rule})
}

// UpdateRule handles PUT /automations/:id
func (h *Handlers) UpdateRule(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.ErrorHandler(c, apperrors.NewValidationError(err.Error()))
		return
	}
	rule, err := h.svc.UpdateRule(id, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rule})
}

// DeleteRule handles DELETE /automations/:id
func (h *Handlers) DeleteRule(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteRule(id); err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ReorderRules handles POST /automations/reorder
func (h *Handlers) ReorderRules(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.ErrorHandler(c, apperrors.NewValidationError(err.Error()))
		return
	}
	if err := h.svc.ReorderRules(req.IDs); err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func parseID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "invalid automation rule ID")
		apperrors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(id64), nil
}
