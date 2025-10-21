package tenant

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handlers provides tenant HTTP handlers.
type Handlers struct {
	service   *Service
	validator *validator.Validate
}

// NewHandlers creates new tenant handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service:   service,
		validator: validator.New(),
	}
}

// CreateTenant creates a new tenant.
func (h *Handlers) CreateTenant(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log tenant creation attempt
	c.Set("security_event", "tenant_creation_attempt")
	c.Set("target_resource", req.Name)

	// Create tenant
	tenant, err := h.service.CreateTenant(&req)
	if err != nil {
		c.Set("security_event", "tenant_creation_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful tenant creation
	c.Set("security_event", "tenant_created")
	c.Set("target_resource", tenant.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// GetTenant gets a tenant by ID.
func (h *Handlers) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("tenant_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	tenant, err := h.service.GetTenant(uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// GetTenantBySlug gets a tenant by slug.
func (h *Handlers) GetTenantBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		appErr := errors.NewInvalidInputError("tenant_slug", "")
		errors.ErrorHandler(c, appErr)
		return
	}

	tenant, err := h.service.GetTenantBySlug(slug)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// ListTenants lists all tenants with pagination.
func (h *Handlers) ListTenants(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")

	tenants, err := h.service.ListTenants(page, pageSize, search)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenants.Data,
		"meta": gin.H{
			"total":       tenants.Total,
			"page":        tenants.Page,
			"page_size":   tenants.PageSize,
			"total_pages": tenants.TotalPages,
		},
	})
}

// UpdateTenant updates a tenant.
func (h *Handlers) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("tenant_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log tenant update attempt
	c.Set("security_event", "tenant_update_attempt")
	c.Set("target_resource_id", uint(id))

	// Update tenant
	tenant, err := h.service.UpdateTenant(uint(id), &req)
	if err != nil {
		c.Set("security_event", "tenant_update_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful tenant update
	c.Set("security_event", "tenant_updated")
	c.Set("target_resource", tenant.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// DeleteTenant deletes a tenant.
func (h *Handlers) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("tenant_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log tenant deletion attempt
	c.Set("security_event", "tenant_deletion_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.DeleteTenant(uint(id)); err != nil {
		c.Set("security_event", "tenant_deletion_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful tenant deletion
	c.Set("security_event", "tenant_deleted")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant deleted successfully",
	})
}

// ActivateTenant activates a tenant.
func (h *Handlers) ActivateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("tenant_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log tenant activation attempt
	c.Set("security_event", "tenant_activation_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.ActivateTenant(uint(id)); err != nil {
		c.Set("security_event", "tenant_activation_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful tenant activation
	c.Set("security_event", "tenant_activated")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant activated successfully",
	})
}

// DeactivateTenant deactivates a tenant.
func (h *Handlers) DeactivateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("tenant_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log tenant deactivation attempt
	c.Set("security_event", "tenant_deactivation_attempt")
	c.Set("target_resource_id", uint(id))

	if err := h.service.DeactivateTenant(uint(id)); err != nil {
		c.Set("security_event", "tenant_deactivation_failed")
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful tenant deactivation
	c.Set("security_event", "tenant_deactivated")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant deactivated successfully",
	})
}

// GetTenantStats gets tenant statistics.
func (h *Handlers) GetTenantStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("tenant_id", idStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	stats, err := h.service.GetTenantStats(uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetMyTenantStats gets current user's tenant statistics.
func (h *Handlers) GetMyTenantStats(c *gin.Context) {
	tenantID := c.GetUint("tenant_id")
	if tenantID == 0 {
		appErr := errors.NewUnauthorizedError("Tenant ID not found in context")
		errors.ErrorHandler(c, appErr)
		return
	}

	stats, err := h.service.GetTenantStats(tenantID)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
