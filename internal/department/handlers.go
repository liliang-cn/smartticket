package department

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handlers holds HTTP handlers for the department admin API.
type Handlers struct{ svc *Service }

// NewHandlers constructs a Handlers wired to the given Service.
func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

// parseID extracts a non-zero uint ID from the ":id" path parameter.
func parseID(c *gin.Context) (uint, bool) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

type createReq struct {
	Name      string `json:"name" binding:"required"`
	ParentID  *uint  `json:"parent_id"`
	ManagerID *uint  `json:"manager_id"`
}

// Create handles POST /admin/departments.
func (h *Handlers) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dept, err := h.svc.Create(CreateInput{
		Name:      req.Name,
		ParentID:  req.ParentID,
		ManagerID: req.ManagerID,
	})
	if err != nil {
		if errors.Is(err, ErrCycle) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cycle"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create department"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"department": dept})
}

// List handles GET /admin/departments.
func (h *Handlers) List(c *gin.Context) {
	depts, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"departments": depts})
}

type updateReq struct {
	Name      *string `json:"name"`
	ParentID  *uint   `json:"parent_id"`
	ManagerID *uint   `json:"manager_id"`
}

// Update handles PUT /admin/departments/:id.
func (h *Handlers) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.Update(id, UpdateInput{
		Name:      req.Name,
		ParentID:  req.ParentID,
		ManagerID: req.ManagerID,
	})
	if err != nil {
		if errors.Is(err, ErrCycle) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cycle"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update department"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

// Delete handles DELETE /admin/departments/:id.
func (h *Handlers) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete department"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
