package llm

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/company/smartticket/internal/models"
)

// CortexProbe runs an embed->store->recall round-trip with a sample vector.
type CortexProbe func(ctx context.Context, vec []float32) error

// Handlers exposes LLM provider REST endpoints.
type Handlers struct {
	svc   *Service
	probe CortexProbe
}

// NewHandlers builds handlers.
func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

// SetCortexProbe injects the CortexDB round-trip probe (set by the server after
// the store opens; keeps this package free of a knowledgebase import).
func (h *Handlers) SetCortexProbe(fn CortexProbe) { h.probe = fn }

// providerView is the masked, client-safe representation.
type providerView struct {
	models.LLMProvider
	APIKeyMasked string `json:"api_key_masked"`
}

func (h *Handlers) view(p models.LLMProvider) providerView {
	masked := ""
	if p.APIKey != "" {
		masked = "********" // ciphertext on disk; show only that a key exists
	}
	return providerView{LLMProvider: p, APIKeyMasked: masked}
}

func (h *Handlers) List(c *gin.Context) {
	ps, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	views := make([]providerView, len(ps))
	for i, p := range ps {
		views[i] = h.view(p)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": views})
}

func (h *Handlers) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "provider not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.view(*p)})
}

func (h *Handlers) Create(c *gin.Context) {
	var in CreateProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	p, err := h.svc.Create(in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": h.view(*p)})
}

func (h *Handlers) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in CreateProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	p, err := h.svc.Update(uint(id), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.view(*p)})
}

func (h *Handlers) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Test runs the provider self-test (chat + embedding + optional CortexDB round-trip).
func (h *Handlers) Test(c *gin.Context) {
	res := h.svc.Test(c.Request.Context(), h.probe)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}
