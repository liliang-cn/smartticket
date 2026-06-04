package aiassist

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "github.com/company/smartticket/internal/errors"
)

// IsDisabled reports whether err is the AI-feature-disabled sentinel.
func IsDisabled(err error) bool { return errors.Is(err, ErrDisabled) }

// IsNotConfigured reports whether err is the no-provider sentinel.
func IsNotConfigured(err error) bool { return errors.Is(err, ErrNotConfigured) }

// SettingsHandlers exposes the AI feature toggles over HTTP.
type SettingsHandlers struct {
	store *SettingsStore
}

// NewSettingsHandlers builds the handlers.
func NewSettingsHandlers(store *SettingsStore) *SettingsHandlers {
	return &SettingsHandlers{store: store}
}

// Get returns the current AI settings (any authenticated user — the agent UI
// reads it to know which AI affordances to show).
func (h *SettingsHandlers) Get(c *gin.Context) {
	s, err := h.store.Get()
	if err != nil {
		apperr.ErrorHandler(c, apperr.NewDatabaseError("ai settings", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
}

// Update applies AI setting changes (admin only — gated at the route).
func (h *SettingsHandlers) Update(c *gin.Context) {
	var req UpdateSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.ErrorHandler(c, apperr.NewInvalidInputError("request_body", err.Error()))
		return
	}
	s, err := h.store.Update(req)
	if err != nil {
		apperr.ErrorHandler(c, apperr.NewDatabaseError("ai settings", err))
		return
	}
	c.Set("security_event", "ai_settings_updated")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
}
