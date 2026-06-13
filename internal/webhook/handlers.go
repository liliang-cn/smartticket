package webhook

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

type createReq struct {
	Name   string   `json:"name" binding:"required"`
	URL    string   `json:"url" binding:"required,url"`
	Events []string `json:"events" binding:"required"`
}

type whView struct {
	ID     uint     `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

// Create registers a new webhook endpoint.
// @Summary Create webhook
// @Description Register a new outbound webhook. Returns the webhook object and a one-time HMAC signing secret.
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createReq true "Webhook creation parameters"
// @Success 201 {object} map[string]interface{} "webhook object and HMAC secret (shown once)"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/webhooks [post]
func (h *Handlers) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wh, err := h.svc.Create(CreateInput{Name: req.Name, URL: req.URL, Events: req.Events}, c.GetUint("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create webhook"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"webhook": whView{ID: wh.ID, Name: wh.Name, URL: wh.URL, Events: req.Events, Active: wh.Active},
		"secret":  wh.Secret,
	})
}

// List returns all registered webhooks.
// @Summary List webhooks
// @Description List all registered outbound webhook endpoints.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of webhook views"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/webhooks [get]
func (h *Handlers) List(c *gin.Context) {
	whs, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list"})
		return
	}
	out := make([]whView, 0, len(whs))
	for _, wh := range whs {
		var events []string
		_ = jsonUnmarshal(wh.Events, &events)
		out = append(out, whView{ID: wh.ID, Name: wh.Name, URL: wh.URL, Events: events, Active: wh.Active})
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": out})
}

// Delete removes a webhook by ID.
// @Summary Delete webhook
// @Description Permanently delete a registered webhook endpoint. Pending deliveries are abandoned.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Webhook ID"
// @Success 200 {object} map[string]interface{} "deleted: true"
// @Failure 400 {object} map[string]interface{} "Invalid ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/webhooks/{id} [delete]
func (h *Handlers) Delete(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// Deliveries returns recent delivery attempts for a webhook.
// @Summary List webhook deliveries
// @Description Return the most recent delivery attempts (up to 100) for a given webhook, including status codes and response bodies.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Webhook ID"
// @Success 200 {object} map[string]interface{} "List of delivery records"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/webhooks/{id}/deliveries [get]
func (h *Handlers) Deliveries(c *gin.Context) {
	id := parseID(c.Param("id"))
	ds, err := h.svc.Deliveries(id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deliveries": ds})
}

// Test enqueues a synthetic ping delivery so the admin can verify connectivity.
// @Summary Test webhook
// @Description Enqueue a synthetic "ping" event to the webhook so the admin can verify the endpoint is reachable.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Webhook ID"
// @Success 200 {object} map[string]interface{} "queued: true"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/webhooks/{id}/test [post]
func (h *Handlers) Test(c *gin.Context) {
	id := parseID(c.Param("id"))
	if err := h.svc.EnqueueTo(id, "ping", `{"event":"ping"}`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queued": true})
}

func parseID(s string) uint {
	var id uint
	_, _ = fmt.Sscanf(s, "%d", &id)
	return id
}
