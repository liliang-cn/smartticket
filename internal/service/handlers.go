package service

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/auth"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for service management.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new service handlers instance.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// parseServiceID extracts and validates service ID from request parameters.
func (h *Handlers) parseServiceID(c *gin.Context) (uint, error) {
	serviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "Invalid service ID")
		apperrors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(serviceID), nil
}

// getUserInfo extracts user info from context with error handling.
func (h *Handlers) getUserInfo(c *gin.Context) (*auth.UserInfo, error) {
	userInfo, exists := c.Get("user")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("User not authenticated")
		apperrors.ErrorHandler(c, appErr)
		return nil, errors.New("user not authenticated")
	}
	return userInfo.(*auth.UserInfo), nil
}

// logServiceEvent logs a service-related security event.
func (h *Handlers) logServiceEvent(c *gin.Context, event, target string) {
	c.Set("security_event", event)
	c.Set("target_resource", target)
}

// CreateService handles service creation.
func (h *Handlers) CreateService(c *gin.Context) {
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userInfo, exists := c.Get("user")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("User not authenticated")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user := userInfo.(*auth.UserInfo)

	// Log service creation attempt
	c.Set("security_event", "service_creation_attempt")
	c.Set("target_resource", req.Name)

	service, err := h.service.CreateService(user.TenantID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful service creation
	c.Set("security_event", "service_created")
	c.Set("resource_id", service.ID)
	c.Set("resource_name", service.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    service,
		"message": "Service created successfully",
	})
}

// ListServices handles service listing.
func (h *Handlers) ListServices(c *gin.Context) {
	var req ListServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("query_params", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userInfo, exists := c.Get("user")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("User not authenticated")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user := userInfo.(*auth.UserInfo)

	services, total, err := h.service.ListServices(user.TenantID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Calculate pagination
	totalPages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    services,
		"meta": gin.H{
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetService handles getting a single service.
func (h *Handlers) GetService(c *gin.Context) {
	serviceID, err := h.parseServiceID(c)
	if err != nil {
		return
	}

	user, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	service, err := h.service.GetService(user.TenantID, serviceID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    service,
	})
}

// UpdateService handles service update.
func (h *Handlers) UpdateService(c *gin.Context) {
	serviceID, err := h.parseServiceID(c)
	if err != nil {
		return
	}

	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	// Log service update attempt
	h.logServiceEvent(c, "service_update_attempt", strconv.FormatUint(uint64(serviceID), 10))

	service, err := h.service.UpdateService(user.TenantID, serviceID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful service update
	c.Set("security_event", "service_updated")
	c.Set("resource_id", service.ID)
	c.Set("resource_name", service.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    service,
		"message": "Service updated successfully",
	})
}

// DeleteService handles service deletion.
func (h *Handlers) DeleteService(c *gin.Context) {
	serviceID, err := h.parseServiceID(c)
	if err != nil {
		return
	}

	user, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	// Log service deletion attempt
	h.logServiceEvent(c, "service_deletion_attempt", strconv.FormatUint(uint64(serviceID), 10))

	err = h.service.DeleteService(user.TenantID, serviceID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful service deletion
	c.Set("security_event", "service_deleted")
	c.Set("resource_id", serviceID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service deleted successfully",
	})
}

// ActivateService handles service activation.
func (h *Handlers) ActivateService(c *gin.Context) {
	serviceID, err := h.parseServiceID(c)
	if err != nil {
		return
	}

	user, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	// Log service activation attempt
	h.logServiceEvent(c, "service_activation_attempt", strconv.FormatUint(uint64(serviceID), 10))

	err = h.service.ActivateService(user.TenantID, serviceID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful service activation
	c.Set("security_event", "service_activated")
	c.Set("resource_id", serviceID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service activated successfully",
	})
}

// DeactivateService handles service deactivation.
func (h *Handlers) DeactivateService(c *gin.Context) {
	serviceID, err := h.parseServiceID(c)
	if err != nil {
		return
	}

	user, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	// Log service deactivation attempt
	h.logServiceEvent(c, "service_deactivation_attempt", strconv.FormatUint(uint64(serviceID), 10))

	err = h.service.DeactivateService(user.TenantID, serviceID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful service deactivation
	c.Set("security_event", "service_deactivated")
	c.Set("resource_id", serviceID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service deactivated successfully",
	})
}
