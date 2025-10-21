package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/services"
)

// PermissionHandler handles permission-related API endpoints.
type PermissionHandler struct {
	permissionService *services.PermissionService
	responseHelper    *ResponseHelper
}

// NewPermissionHandler creates a new permission handler.
func NewPermissionHandler(permissionService *services.PermissionService) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		responseHelper:    NewResponseHelper(),
	}
}

// GetAllPermissions returns all permissions.
func (h *PermissionHandler) GetAllPermissions(c *gin.Context) {
	permissions, err := h.permissionService.GetAllPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get permissions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    permissions,
	})
}

// GetPermissionByID returns a permission by ID.
func (h *PermissionHandler) GetPermissionByID(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	permission, err := h.permissionService.GetPermissionByID(c.Request.Context(), id)
	if h.responseHelper.HandleGormError(c, err, "permission") {
		return
	}

	h.responseHelper.SendSuccess(c, permission)
}

// CreatePermission creates a new permission.
func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	var req models.Permission

	if !h.responseHelper.BindJSON(c, &req) {
		return
	}

	// Permissions are global, not tenant-specific

	err := h.permissionService.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to create permission")
		return
	}

	h.responseHelper.SendSuccessWithStatus(c, http.StatusCreated, req)
}

// UpdatePermission updates an existing permission.
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	var req models.Permission
	if !h.responseHelper.BindJSON(c, &req) {
		return
	}

	req.ID = id

	err := h.permissionService.UpdatePermission(c.Request.Context(), &req)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to update permission")
		return
	}

	h.responseHelper.SendSuccess(c, req)
}

// DeletePermission deletes a permission.
func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	err := h.permissionService.DeletePermission(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "cannot delete system permission" {
			h.responseHelper.SendForbiddenError(c, "Cannot delete system permission")
			return
		}

		h.responseHelper.SendInternalError(c, "Failed to delete permission")
		return
	}

	h.responseHelper.SendSuccess(c, gin.H{"message": "Permission deleted successfully"})
}

// GetUserPermissions returns all permissions for a user.
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	userID, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	tenantID := h.responseHelper.GetTenantIDFromContext(c)

	permissions, err := h.permissionService.GetUserPermissions(c.Request.Context(), userID, tenantID)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to get user permissions")
		return
	}

	h.responseHelper.SendSuccess(c, permissions)
}

// AssignPermissionToUser assigns a permission directly to a user.
func (h *PermissionHandler) AssignPermissionToUser(c *gin.Context) {
	userID, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		PermissionID uint `json:"permission_id" binding:"required"`
	}

	if !h.responseHelper.BindJSON(c, &req) {
		return
	}

	tenantID := h.responseHelper.GetTenantIDFromContext(c)

	err := h.permissionService.AssignPermissionToUser(c.Request.Context(), userID, req.PermissionID, tenantID)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to assign permission to user")
		return
	}

	h.responseHelper.SendSuccess(c, gin.H{"message": "Permission assigned successfully"})
}

// RemovePermissionFromUser removes a permission directly from a user.
func (h *PermissionHandler) RemovePermissionFromUser(c *gin.Context) {
	userID, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	permissionID, ok := h.responseHelper.ParseIDParam(c, "permissionId")
	if !ok {
		return
	}

	tenantID := h.responseHelper.GetTenantIDFromContext(c)

	err := h.permissionService.RemovePermissionFromUser(c.Request.Context(), userID, permissionID, tenantID)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to remove permission from user")
		return
	}

	h.responseHelper.SendSuccess(c, gin.H{"message": "Permission removed successfully"})
}
