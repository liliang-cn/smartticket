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
