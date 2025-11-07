package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/logger"
)

// Handlers provides user management HTTP handlers.
type Handlers struct {
	userService *Service
}

// NewHandlers creates new user management handlers.
func NewHandlers(userService *Service) *Handlers {
	return &Handlers{
		userService: userService,
	}
}

// CreateUser handles user creation
// @Summary Create a new user
// @Description Create a new user in the current tenant
// @Tags users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User creation data"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 201 {object} internal_auth.UserInfo
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 409 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users [post].
func (h *Handlers) CreateUser(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	log := logger.GetGlobalLogger().WithRequestID(requestID.(string))

	// Check permissions - only admin can create users
	if userRole != "admin" {
		appErr := errors.NewForbiddenError("Only administrators can create users").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid request body").
			WithRequestID(requestID.(string)).
			WithDetails(err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	log.Info("User creation attempt",
		zap.String("creator_id", strconv.Itoa(int(userID.(uint)))),
		zap.String("email", req.Email),
		zap.String("role", req.Role),
	)

	userInfo, err := h.userService.CreateUser(&req)
	if err != nil {
		log.Warn("User creation failed", zap.Error(err))

		var appErr *errors.AppError
		if err.Error() == "email already exists" || err.Error() == "username already exists" {
			appErr = errors.NewConflictError(err.Error()).
				WithRequestID(requestID.(string))
		} else {
			appErr = errors.NewInternalError("Failed to create user", err).
				WithRequestID(requestID.(string))
		}
		errors.ErrorHandler(c, appErr)
		return
	}

	logger.LogSecurityEvent("user_created", req.Email, c.ClientIP(), c.GetHeader("User-Agent"), true)
	log.Info("User created successfully",
		zap.String("email", req.Email),
		zap.Uint("user_id", userInfo.ID),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    userInfo,
	})
}

// GetUser handles getting a specific user
// @Summary Get user by ID
// @Description Get detailed information about a specific user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} internal_auth.UserInfo
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users/{id} [get].
func (h *Handlers) GetUser(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	// Parse user ID from path
	targetUserID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewValidationError("Invalid user ID").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Check permissions - users can only view their own profile, admins can view any
	if userRole != "admin" && uint(targetUserID) != userID.(uint) {
		appErr := errors.NewForbiddenError("You can only view your own profile").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	userInfo, err := h.userService.GetUser(uint(targetUserID))
	if err != nil {
		appErr := errors.NewNotFoundError("User").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userInfo,
	})
}

// UpdateUser handles updating user information
// @Summary Update user
// @Description Update user information
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserRequest true "User update data"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} internal_auth.UserInfo
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 409 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users/{id} [put].
func (h *Handlers) UpdateUser(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	log := logger.GetGlobalLogger().WithRequestID(requestID.(string))

	// Parse user ID from path
	targetUserID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewValidationError("Invalid user ID").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Check permissions - users can only update their own profile, admins can update any
	if userRole != "admin" && uint(targetUserID) != userID.(uint) {
		appErr := errors.NewForbiddenError("You can only update your own profile").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid request body").
			WithRequestID(requestID.(string)).
			WithDetails(err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Non-admin users cannot change their role
	if userRole != "admin" && req.Role != "" {
		appErr := errors.NewForbiddenError("Only administrators can change user roles").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	log.Info("User update attempt",
		zap.String("updater_id", strconv.Itoa(int(userID.(uint)))),
		zap.Uint("target_user_id", uint(targetUserID)),
	)

	userInfo, err := h.userService.UpdateUser(uint(targetUserID), &req)
	if err != nil {
		log.Warn("User update failed", zap.Error(err))

		var appErr *errors.AppError
		if err.Error() == "email already exists" || err.Error() == "username already exists" {
			appErr = errors.NewConflictError(err.Error()).
				WithRequestID(requestID.(string))
		} else if err.Error() == "user not found" {
			appErr = errors.NewNotFoundError("User").
				WithRequestID(requestID.(string))
		} else {
			appErr = errors.NewInternalError("Failed to update user", err).
				WithRequestID(requestID.(string))
		}
		errors.ErrorHandler(c, appErr)
		return
	}

	logger.LogSecurityEvent("user_updated", userInfo.Email, c.ClientIP(), c.GetHeader("User-Agent"), true)
	log.Info("User updated successfully",
		zap.Uint("user_id", userInfo.ID),
		zap.String("email", userInfo.Email),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userInfo,
	})
}

// DeleteUser handles deleting a user
// @Summary Delete user
// @Description Soft delete a user account
// @Tags users
// @Param id path string true "User ID"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users/{id} [delete].
func (h *Handlers) DeleteUser(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	log := logger.GetGlobalLogger().WithRequestID(requestID.(string))

	// Only admins can delete users
	if userRole != "admin" {
		appErr := errors.NewForbiddenError("Only administrators can delete users").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Parse user ID from path
	targetUserID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewValidationError("Invalid user ID").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Prevent self-deletion
	if uint(targetUserID) == userID.(uint) {
		appErr := errors.NewForbiddenError("You cannot delete your own account").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	log.Info("User deletion attempt",
		zap.String("deleter_id", strconv.Itoa(int(userID.(uint)))),
		zap.Uint("target_user_id", uint(targetUserID)),
	)

	if err := h.userService.DeleteUser(uint(targetUserID)); err != nil {
		log.Warn("User deletion failed", zap.Error(err))

		var appErr *errors.AppError
		if err.Error() == "user not found" {
			appErr = errors.NewNotFoundError("User").
				WithRequestID(requestID.(string))
		} else {
			appErr = errors.NewInternalError("Failed to delete user", err).
				WithRequestID(requestID.(string))
		}
		errors.ErrorHandler(c, appErr)
		return
	}

	logger.LogSecurityEvent("user_deleted", strconv.Itoa(int(targetUserID)), c.ClientIP(), c.GetHeader("User-Agent"), true)
	log.Info("User deleted successfully",
		zap.Uint("target_user_id", uint(targetUserID)),
	)

	c.Status(http.StatusNoContent)
}

// ListUsers handles listing users with pagination and filters
// @Summary List users
// @Description Get a paginated list of users with optional filters
// @Tags users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param search query string false "Search term"
// @Param role query string false "Filter by role"
// @Param is_active query bool false "Filter by active status"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} UserListResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users [get].
func (h *Handlers) ListUsers(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userRole, _ := c.Get("user_role")

	// Only admins can list all users
	if userRole != "admin" {
		appErr := errors.NewForbiddenError("Only administrators can list users").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	var req UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := errors.NewValidationError("Invalid query parameters").
			WithRequestID(requestID.(string)).
			WithDetails(err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	response, err := h.userService.ListUsers(&req)
	if err != nil {
		appErr := errors.NewInternalError("Failed to list users", err).
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	c.JSON(http.StatusOK, response)
}

// ActivateUser handles activating a user account
// @Summary Activate user
// @Description Activate a user account
// @Tags users
// @Param id path string true "User ID"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users/{id}/activate [post].
func (h *Handlers) ActivateUser(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userRole, _ := c.Get("user_role")

	// Only admins can activate users
	if userRole != "admin" {
		appErr := errors.NewForbiddenError("Only administrators can activate users").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Parse user ID from path
	targetUserID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewValidationError("Invalid user ID").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	if err := h.userService.ActivateUser(uint(targetUserID)); err != nil {
		appErr := errors.NewInternalError("Failed to activate user", err).
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	logger.LogSecurityEvent("user_activated", strconv.Itoa(int(targetUserID)), c.ClientIP(), c.GetHeader("User-Agent"), true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User activated successfully",
	})
}

// DeactivateUser handles deactivating a user account
// @Summary Deactivate user
// @Description Deactivate a user account
// @Tags users
// @Param id path string true "User ID"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users/{id}/deactivate [post].
func (h *Handlers) DeactivateUser(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	// Only admins can deactivate users
	if userRole != "admin" {
		appErr := errors.NewForbiddenError("Only administrators can deactivate users").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Parse user ID from path
	targetUserID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewValidationError("Invalid user ID").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Prevent self-deactivation
	if uint(targetUserID) == userID.(uint) {
		appErr := errors.NewForbiddenError("You cannot deactivate your own account").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	if err := h.userService.DeactivateUser(uint(targetUserID)); err != nil {
		appErr := errors.NewInternalError("Failed to deactivate user", err).
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	logger.LogSecurityEvent("user_deactivated", strconv.Itoa(int(targetUserID)), c.ClientIP(), c.GetHeader("User-Agent"), true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deactivated successfully",
	})
}

// GetUserStats handles getting user statistics
// @Summary Get user statistics
// @Description Get user statistics for the current tenant
// @Tags users
// @Produce json
// @Param X-Tenant-ID header string true "Tenant ID"
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/users/stats [get].
func (h *Handlers) GetUserStats(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userRole, _ := c.Get("user_role")

	// Only admins can view user statistics
	if userRole != "admin" {
		appErr := errors.NewForbiddenError("Only administrators can view user statistics").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	stats, err := h.userService.GetUserStats()
	if err != nil {
		appErr := errors.NewInternalError("Failed to get user statistics", err).
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
