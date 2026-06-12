package apikey

import (
	"fmt"
	"net/http"
	"strings"
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

// Create issues a key. The plaintext is in the response ONCE; it is never
// retrievable again.
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

// safePrefix strips the well-known label from a stored key prefix so that the
// List response never contains a usable key fragment. The DB stores the first
// 12 chars (e.g. "stk_live_9ed"); we return only the unique trailing portion
// (e.g. "9ed") as a visual hint for the user to identify their keys.
func safePrefix(p string) string {
	const label = keyPrefixLabel + "_" // "stk_live_"
	return strings.TrimPrefix(p, label)
}

func (h *Handlers) List(c *gin.Context) {
	keys, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list api keys"})
		return
	}
	views := make([]keyView, 0, len(keys))
	for _, k := range keys {
		views = append(views, keyView{ID: k.ID, Name: k.Name, KeyPrefix: safePrefix(k.KeyPrefix), UserID: k.UserID,
			IsActive: k.IsActive, ExpiresAt: toUnix(k.ExpiresAt), LastUsedAt: toUnix(k.LastUsedAt), CreatedAt: k.CreatedAt.Unix()})
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": views})
}

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
