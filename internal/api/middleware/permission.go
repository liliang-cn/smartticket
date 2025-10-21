package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PermissionServiceInterface defines the interface for permission services.
type PermissionServiceInterface interface {
	GetUserPermissions(ctx context.Context, userID uint, tenantID string) ([]models.Permission, error)
	GetUserRoles(ctx context.Context, userID uint, tenantID string) ([]models.Role, error)
	GetDatabase() *gorm.DB
	HasPermission(ctx context.Context, userID uint, tenantID string, permissionCode string) (bool, error)
}

// PermissionMiddleware provides permission checking functionality.
type PermissionMiddleware struct {
	permissionService PermissionServiceInterface
}

// NewPermissionMiddleware creates a new permission middleware.
func NewPermissionMiddleware(permissionService PermissionServiceInterface) *PermissionMiddleware {
	return &PermissionMiddleware{
		permissionService: permissionService,
	}
}

// RequirePermission creates a middleware that requires a specific permission.
func (pm *PermissionMiddleware) RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (should be set by auth middleware)
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		// Check if user has the required permission
		permission := user.(*models.User)
		tenantID := c.GetString("tenant_id")

		// Get user's permissions
		userPermissions, err := pm.permissionService.GetUserPermissions(c.Request.Context(), permission.ID, tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to check permissions",
				},
			})
			c.Abort()
			return
		}

		// Check if user has the required permission
		hasPermission := false
		for _, p := range userPermissions {
			if p.Code == permissionCode {
				hasPermission = true
				break
			}
		}

		// Also check user's role permissions
		roles, err := pm.permissionService.GetUserRoles(c.Request.Context(), permission.ID, tenantID)
		if err == nil {
			for _, role := range roles {
				for _, perm := range role.Permissions {
					if perm.Code == permissionCode {
						hasPermission = true
						break
					}
				}
				if hasPermission {
					break
				}
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":     "FORBIDDEN",
					"message":  "Insufficient permissions",
					"required": permissionCode,
				},
			})
			c.Abort()
			return
		}

		// Set permissions in context for downstream handlers
		permissions := make([]string, len(userPermissions))
		for i, p := range userPermissions {
			permissions[i] = p.Code
		}
		c.Set("user_permissions", permissions)

		// Also include role permissions
		for _, role := range roles {
			for _, perm := range role.Permissions {
				if perm.Code != "" {
					permissions = append(permissions, perm.Code)
				}
			}
		}
		c.Set("all_permissions", permissions)

		c.Next()
	}
}

// RequireAnyPermission creates a middleware that requires any of the specified permissions.
func (pm *PermissionMiddleware) RequireAnyPermission(permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		permission := user.(*models.User)
		tenantID := c.GetString("tenant_id")

		// Get user's permissions
		userPermissions, err := pm.permissionService.GetUserPermissions(c.Request.Context(), permission.ID, tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to check permissions",
				},
			})
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		userPermCodes := make(map[string]bool)
		for _, p := range userPermissions {
			userPermCodes[p.Code] = true
		}

		// Also check role permissions
		roles, err := pm.permissionService.GetUserRoles(c.Request.Context(), permission.ID, tenantID)
		if err == nil {
			for _, role := range roles {
				for _, perm := range role.Permissions {
					userPermCodes[perm.Code] = true
				}
			}
		}

		// Check if any required permission is available
		hasPermission := false
		for _, requiredPerm := range permissionCodes {
			if userPermCodes[requiredPerm] {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":         "FORBIDDEN",
					"message":      "Insufficient permissions",
					"required_any": permissionCodes,
				},
			})
			c.Abort()
			return
		}

		// Set permissions in context
		permissions := make([]string, 0, len(userPermissions))
		for p := range userPermCodes {
			permissions = append(permissions, p)
		}
		c.Set("all_permissions", permissions)

		c.Next()
	}
}

// RequireOwnership creates a middleware that requires user to own the resource.
func (pm *PermissionMiddleware) RequireOwnership(resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		// Extract resource ID from path
		resourceID := c.Param("id")
		if resourceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Resource ID is required",
				},
			})
			c.Abort()
			return
		}

		// Check ownership based on resource type
		permission := user.(*models.User)
		tenantID := c.GetString("tenant_id")
		ownsResource := false

		switch strings.ToLower(resourceType) {
		case "ticket":
			var ticket models.Ticket
			userIDStr := strconv.FormatUint(uint64(permission.ID), 10)
			if err := pm.permissionService.GetDatabase().Where("id = ? AND tenant_id = ? AND created_by = ?",
				resourceID, tenantID, userIDStr).First(&ticket).Error; err == nil {
				ownsResource = true
			}
		case "message":
			var message models.Message
			// Messages don't have tenant_id directly, they belong to tickets which belong to tenants
			if err := pm.permissionService.GetDatabase().Where("id = ? AND user_id = ?",
				resourceID, permission.ID).First(&message).Error; err == nil {
				// Also verify the associated ticket belongs to the tenant
				var ticket models.Ticket
				// Convert tenantID string to int for comparison
				tenantIDInt := 0
				if id, err := strconv.Atoi(tenantID); err == nil {
					tenantIDInt = id
				}
				if err := pm.permissionService.GetDatabase().Where("id = ? AND tenant_id = ?",
					message.TicketID, tenantIDInt).First(&ticket).Error; err == nil {
					ownsResource = true
				}
			}
		case "knowledge":
			var article models.KnowledgeArticle
			if err := pm.permissionService.GetDatabase().Where("id = ? AND tenant_id = ? AND author_id = ?",
				resourceID, tenantID, permission.ID).First(&article).Error; err == nil {
				ownsResource = true
			}
		case "user":
			// Users can only edit their own profile unless they have admin permissions
			if resourceID == strconv.FormatUint(uint64(permission.ID), 10) {
				ownsResource = true
			}
		}

		// Also check if user has admin permissions
		hasAdminPermission := false
		userPermissions, err := pm.permissionService.GetUserPermissions(c.Request.Context(), permission.ID, tenantID)
		if err == nil {
			for _, p := range userPermissions {
				if p.Code == "admin:system" {
					hasAdminPermission = true
					break
				}
			}
		}

		if !ownsResource && !hasAdminPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":     "FORBIDDEN",
					"message":  "You can only access your own resources",
					"resource": resourceType,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TenantMiddleware ensures the request has a valid tenant ID.
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Tenant ID is required",
				},
			})
			c.Abort()
			return
		}

		// Validate tenant exists and is active
		// This would be implemented with a tenant service
		// For now, we'll just set the tenant ID in context
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}
