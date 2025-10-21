package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ResponseHelper provides common response functions for handlers
type ResponseHelper struct{}

// NewResponseHelper creates a new response helper
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// SendSuccess sends a successful response
func (r *ResponseHelper) SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// SendSuccessWithStatus sends a successful response with custom status code
func (r *ResponseHelper) SendSuccessWithStatus(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
	})
}

// SendError sends an error response
func (r *ResponseHelper) SendError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// SendValidationError sends a validation error response
func (r *ResponseHelper) SendValidationError(c *gin.Context, message string) {
	r.SendError(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

// SendNotFoundError sends a not found error response
func (r *ResponseHelper) SendNotFoundError(c *gin.Context, resource string) {
	r.SendError(c, http.StatusNotFound, "NOT_FOUND", resource+" not found")
}

// SendInternalError sends an internal server error response
func (r *ResponseHelper) SendInternalError(c *gin.Context, message string) {
	r.SendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// SendForbiddenError sends a forbidden error response
func (r *ResponseHelper) SendForbiddenError(c *gin.Context, message string) {
	r.SendError(c, http.StatusForbidden, "FORBIDDEN", message)
}

// ParseIDParam parses an ID parameter from the request
func (r *ResponseHelper) ParseIDParam(c *gin.Context, paramName string) (uint, bool) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		r.SendValidationError(c, "Invalid "+paramName)
		return 0, false
	}
	return uint(id), true
}

// HandleGormError handles common GORM errors
func (r *ResponseHelper) HandleGormError(c *gin.Context, err error, resource string) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.SendNotFoundError(c, resource)
		return true
	}

	r.SendInternalError(c, "Failed to get "+resource)
	return true
}

// BindJSON safely binds JSON request
func (r *ResponseHelper) BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		r.SendValidationError(c, err.Error())
		return false
	}
	return true
}

// GetTenantIDFromContext gets tenant ID from context
func (r *ResponseHelper) GetTenantIDFromContext(c *gin.Context) string {
	return c.GetString("tenant_id")
}