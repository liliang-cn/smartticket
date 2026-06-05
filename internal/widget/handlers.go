package widget

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for the embeddable web-chat widget.
type Handlers struct {
	service *Service
}

// NewHandlers creates widget HTTP handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// StartSession handles POST /widget/session (public, no auth).
//
// Request body (all fields optional):
//
//	{ "email": "...", "name": "...", "message": "..." }
//
// Response:
//
//	{ "success": true, "data": { "token": "...", "ticket_id": 42 } }
func (h *Handlers) StartSession(c *gin.Context) {
	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	resp, err := h.service.StartSession(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// PostMessage handles POST /widget/messages.
//
// The conversation token is resolved from (in order):
//  1. Authorization: Bearer <token> header
//  2. ?token= query param
//  3. JSON body field "token"
//
// Request body: { "message": "...", "token": "..." (optional) }
// Response:     { "success": true, "data": <MessageResponse> }
func (h *Handlers) PostMessage(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "conversation token required"})
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if strings.TrimSpace(body.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "message is required"})
		return
	}

	msg, err := h.service.PostMessage(token, body.Message)
	if err != nil {
		if err == ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": msg})
}

// History handles GET /widget/messages?token=<token>.
//
// Returns all non-internal messages oldest-first.
func (h *Handlers) History(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "conversation token required"})
		return
	}

	msgs, err := h.service.History(token)
	if err != nil {
		if err == ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": msgs})
}

// extractToken reads the conversation token from (in priority order):
//  1. Authorization: Bearer header
//  2. ?token= query param
func extractToken(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if t := c.Query("token"); t != "" {
		return t
	}
	return ""
}
