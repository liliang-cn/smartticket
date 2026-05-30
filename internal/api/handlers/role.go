package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/services"
)

// RoleHandler handles role-related API endpoints.
type RoleHandler struct {
	permissionService *services.PermissionService
	responseHelper    *ResponseHelper
}

// NewRoleHandler creates a new role handler.
func NewRoleHandler(permissionService *services.PermissionService) *RoleHandler {
	return &RoleHandler{
		permissionService: permissionService,
		responseHelper:    NewResponseHelper(),
	}
}

// GetAllRoles returns all roles.
// @Summary Get all roles
// @Description Retrieves a list of all roles
// @Tags roles
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {array} models.Role
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/admin/roles [get]
func (h *RoleHandler) GetAllRoles(c *gin.Context) {
	roles, err := h.permissionService.GetAllRoles(c.Request.Context())
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to get roles")
		return
	}

	h.responseHelper.SendSuccess(c, roles)
}

// GetRoleByID returns a role by ID.
func (h *RoleHandler) GetRoleByID(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	role, err := h.permissionService.GetRoleByID(c.Request.Context(), id)
	if h.responseHelper.HandleGormError(c, err, "role") {
		return
	}

	h.responseHelper.SendSuccess(c, role)
}

// CreateRole creates a new role.
// @Summary Create a new role
// @Description Creates a new role with provided details and permissions
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param request body models.Role true "Role creation data"
// @Success 201 {object} models.Role
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/admin/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req models.Role

	if !h.responseHelper.BindJSON(c, &req) {
		return
	}

	err := h.permissionService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to create role")
		return
	}

	h.responseHelper.SendSuccessWithStatus(c, http.StatusCreated, req)
}

// UpdateRole updates an existing role.
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	var req models.Role
	if !h.responseHelper.BindJSON(c, &req) {
		return
	}

	req.ID = id

	err := h.permissionService.UpdateRole(c.Request.Context(), &req)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to update role")
		return
	}

	h.responseHelper.SendSuccess(c, req)
}

// DeleteRole deletes a role.
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	err := h.permissionService.DeleteRole(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "cannot delete system role" {
			h.responseHelper.SendForbiddenError(c, "Cannot delete system role")
			return
		}

		h.responseHelper.SendInternalError(c, "Failed to delete role")
		return
	}

	h.responseHelper.SendSuccess(c, gin.H{"message": "Role deleted successfully"})
}

// GetRolePermissions returns all permissions for a role.
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
	id, ok := h.responseHelper.ParseIDParam(c, "id")
	if !ok {
		return
	}

	permissions, err := h.permissionService.GetRolePermissions(c.Request.Context(), id)
	if err != nil {
		h.responseHelper.SendInternalError(c, "Failed to get role permissions")
		return
	}

	h.responseHelper.SendSuccess(c, permissions)
}

// AssignPermissionToRole assigns a permission to a role.
func (h *RoleHandler) AssignPermissionToRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid role ID",
			},
		})
		return
	}

	var req struct {
		PermissionID uint `json:"permission_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	err = h.permissionService.AssignPermissionToRole(c.Request.Context(), uint(id), req.PermissionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to assign permission to role",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Permission assigned to role successfully"},
	})
}

// RemovePermissionFromRole removes a permission from a role.
func (h *RoleHandler) RemovePermissionFromRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid role ID",
			},
		})
		return
	}

	permissionIDStr := c.Param("permissionId")
	permissionID, err := strconv.ParseUint(permissionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid permission ID",
			},
		})
		return
	}

	err = h.permissionService.RemovePermissionFromRole(c.Request.Context(), uint(id), uint(permissionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to remove permission from role",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Permission removed from role successfully"},
	})
}

// AssignRoleToUser assigns a role to a user.
func (h *RoleHandler) AssignRoleToUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid user ID",
			},
		})
		return
	}

	var req struct {
		RoleID uint `json:"role_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	err = h.permissionService.AssignRoleToUser(c.Request.Context(), uint(userID), req.RoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to assign role to user",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Role assigned to user successfully"},
	})
}

// RemoveRoleFromUser removes a role from a user.
func (h *RoleHandler) RemoveRoleFromUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid user ID",
			},
		})
		return
	}

	roleIDStr := c.Param("roleId")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid role ID",
			},
		})
		return
	}

	err = h.permissionService.RemoveRoleFromUser(c.Request.Context(), uint(userID), uint(roleID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to remove role from user",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Role removed from user successfully"},
	})
}

// GetUserRoles returns all roles for a user.
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid user ID",
			},
		})
		return
	}

	roles, err := h.permissionService.GetUserRoles(c.Request.Context(), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get user roles",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    roles,
	})
}
