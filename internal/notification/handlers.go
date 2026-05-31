package notification

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handlers exposes in-app notification REST endpoints. Every endpoint operates
// only on the calling user's notifications.
type Handlers struct {
	svc *Service
}

// NewHandlers builds notification handlers.
func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

// List returns the current user's notifications, newest-first, paginated.
// Query params: unread=true, page, page_size.
func (h *Handlers) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	unreadOnly := c.Query("unread") == "true"
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	items, total, err := h.svc.List(userID, unreadOnly, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"meta":    gin.H{"total": total},
	})
}

// UnreadCount returns the current user's unread notification count.
func (h *Handlers) UnreadCount(c *gin.Context) {
	userID := c.GetUint("user_id")
	count, err := h.svc.UnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"count": count}})
}

// MarkRead marks one of the current user's notifications read.
func (h *Handlers) MarkRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid notification id"}})
		return
	}
	if err := h.svc.MarkRead(userID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkAllRead marks all of the current user's notifications read.
func (h *Handlers) MarkAllRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	if err := h.svc.MarkAllRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
