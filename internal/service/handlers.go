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
// @Summary Create a new service
// @Description Creates a new service with provided information
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param request body service.CreateServiceRequest true "Service creation data"
// @Success 201 {object} service.ServiceResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services [post]
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
// @Summary List services
// @Description Retrieves a paginated list of services with optional filtering
// @Tags services
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of services per page" default(20) minimum(1) maximum(100)
// @Param search query string false "Search services by name or description"
// @Param type query string false "Filter by service type" Enums(infrastructure,application,support,consulting)
// @Param status query string false "Filter by status" Enums(active,inactive,maintenance)
// @Param product_id query int false "Filter by product ID"
// @Success 200 {object} server.Response
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services [get]
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
// @Summary Get a service by ID
// @Description Retrieves a specific service by its unique identifier
// @Tags services
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param id path int true "Service ID"
// @Success 200 {object} service.ServiceResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services/{id} [get]
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
// @Summary Update a service
// @Description Updates an existing service with new information
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param id path int true "Service ID"
// @Param request body service.UpdateServiceRequest true "Service update data"
// @Success 200 {object} service.ServiceResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services/{id} [put]
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
// @Summary Delete a service
// @Description Soft deletes a service (marks as deleted but preserves data)
// @Tags services
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param id path int true "Service ID"
// @Success 200 {object} server.Response
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services/{id} [delete]
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
// @Summary Activate a service
// @Description Activates an existing service, making it available for use
// @Tags services
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param id path int true "Service ID"
// @Success 200 {object} server.Response
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services/{id}/activate [post]
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
// @Summary Deactivate a service
// @Description Deactivates an existing service, making it unavailable for use
// @Tags services
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param id path int true "Service ID"
// @Success 200 {object} server.Response
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/services/{id}/deactivate [post]
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
