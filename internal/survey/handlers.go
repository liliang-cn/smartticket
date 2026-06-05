package survey

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handlers provides HTTP handlers for CSAT surveys.
type Handlers struct {
	service *Service
}

// NewHandlers creates survey HTTP handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// surveyPublicView is the public JSON representation returned by the GET
// handler. It deliberately omits internal data (ticket title, requester
// details, etc.) to expose only what is needed to render the rating form.
type surveyPublicView struct {
	TicketID  uint `json:"ticket_id"`
	Rating    int  `json:"rating"`
	Responded bool `json:"responded"`
}

// GetSurvey handles GET /api/v1/survey/:token (public, no auth).
// Returns enough data for the frontend to render the survey form without
// leaking ticket internals.
func (h *Handlers) GetSurvey(c *gin.Context) {
	token := c.Param("token")
	s, err := h.service.GetByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "survey not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": surveyPublicView{
			TicketID:  s.TicketID,
			Rating:    s.Rating,
			Responded: s.RespondedAt != nil,
		},
	})
}

// submitRequest is the body for POST /api/v1/survey/:token.
type submitRequest struct {
	Rating  int    `json:"rating"  binding:"required"`
	Comment string `json:"comment"`
}

// SubmitSurvey handles POST /api/v1/survey/:token (public, no auth).
func (h *Handlers) SubmitSurvey(c *gin.Context) {
	token := c.Param("token")

	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	err := h.service.Submit(token, req.Rating, req.Comment)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "survey not found"})
			return
		}
		msg := err.Error()
		// Validation errors → 400; already responded → 409; anything else → 500.
		switch {
		case containsAny(msg, "between 1 and 5"):
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": msg})
		case containsAny(msg, "already responded"):
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": msg})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStats handles GET /api/v1/survey/stats (protected — team/admin auth).
func (h *Handlers) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
