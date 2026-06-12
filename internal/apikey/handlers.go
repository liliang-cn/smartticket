package apikey

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

type createReq struct {
	Name      string `json:"name" binding:"required"`
	UserID    uint   `json:"user_id" binding:"required"`
	ExpiresAt *int64 `json:"expires_at"` // unix seconds, optional
}

type keyView struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"key_prefix"`
	UserID     uint   `json:"user_id"`
	IsActive   bool   `json:"is_active"`
	ExpiresAt  *int64 `json:"expires_at"`
	LastUsedAt *int64 `json:"last_used_at"`
	CreatedAt  int64  `json:"created_at"`
}

func toUnix(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	u := t.Unix()
	return &u
}

// Create issues a new API key.
// @Summary Create API key
// @Description Issue a new API key for the given user. The plaintext key is returned ONCE in the response and is not recoverable afterwards.
// @Tags api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createReq true "API key creation parameters"
// @Success 201 {object} map[string]interface{} "key (plaintext, shown once) and api_key object"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/api-keys [post]
func (h *Handlers) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var exp *time.Time
	if req.ExpiresAt != nil {
		tm := time.Unix(*req.ExpiresAt, 0)
		exp = &tm
	}
	createdBy := c.GetUint("user_id")
	plaintext, key, err := h.svc.Create(req.Name, req.UserID, exp, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api key"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"key": plaintext,
		"api_key": keyView{ID: key.ID, Name: key.Name, KeyPrefix: key.KeyPrefix,
			UserID: key.UserID, IsActive: key.IsActive, CreatedAt: key.CreatedAt.Unix()},
	})
}

// List returns all active API keys (key prefix only — no secrets).
// @Summary List API keys
// @Description List all API keys. Returns metadata only; the plaintext key is never included.
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of API key views"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/api-keys [get]
func (h *Handlers) List(c *gin.Context) {
	keys, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list api keys"})
		return
	}
	views := make([]keyView, 0, len(keys))
	for _, k := range keys {
		// KeyPrefix is the first 12 chars (e.g. "stk_live_9ed") — a safe display
		// hint, NOT the secret. The full plaintext is never recoverable from it.
		views = append(views, keyView{ID: k.ID, Name: k.Name, KeyPrefix: k.KeyPrefix, UserID: k.UserID,
			IsActive: k.IsActive, ExpiresAt: toUnix(k.ExpiresAt), LastUsedAt: toUnix(k.LastUsedAt), CreatedAt: k.CreatedAt.Unix()})
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": views})
}

// Revoke deactivates an API key by ID.
// @Summary Revoke API key
// @Description Permanently deactivate an API key. The key will no longer authenticate requests.
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Param id path int true "API key ID"
// @Success 200 {object} map[string]interface{} "revoked: true"
// @Failure 400 {object} map[string]interface{} "Invalid ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/api-keys/{id} [delete]
func (h *Handlers) Revoke(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Revoke(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
