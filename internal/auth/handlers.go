package auth

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/logger"
)

// Handlers provides authentication HTTP handlers
type Handlers struct {
	authService *Service
}

// NewHandlers creates new authentication handlers
func NewHandlers(authService *Service) *Handlers {
	return &Handlers{
		authService: authService,
	}
}

// Login handles user login
// @Summary User login
// @Description Authenticate a user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *Handlers) Login(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	tenantID, _ := c.Get("tenant_id")
	requestIDStr, ok := requestID.(string)
	if !ok {
		requestIDStr = ""
	}
	log := logger.GetGlobalLogger().WithRequestID(requestIDStr)

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid request body").
			WithRequestID(requestIDStr).
			WithDetails(err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Trim and normalize email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Get client information
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	tenantIDUint, ok := tenantID.(uint)
	if !ok {
		tenantIDUint = 0
	}
	log.Info("User login attempt",
		zap.String("email", req.Email),
		zap.Uint("tenant_id", tenantIDUint),
		zap.String("client_ip", clientIP),
	)

	response, err := h.authService.Login(&req, clientIP, userAgent)
	if err != nil {
		logger.LogSecurityEvent("auth_login_failed", req.Email, clientIP, userAgent, false)
		appErr := errors.NewUnauthorizedError("Invalid email or password").
			WithRequestID(requestIDStr)
		errors.ErrorHandler(c, appErr)
		return
	}

	logger.LogSecurityEvent("auth_login_success", req.Email, clientIP, userAgent, true)
	log.Info("User login successful",
		zap.String("email", req.Email),
		zap.Uint("user_id", response.User.ID),
	)

	c.JSON(http.StatusOK, response)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} TokenPair
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *Handlers) RefreshToken(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	log := logger.GetGlobalLogger().WithRequestID(requestID.(string))

	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid request body").
			WithRequestID(requestID.(string)).
			WithDetails(err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	tokens, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		log.Warn("Token refresh failed", zap.Error(err))
		appErr := errors.NewUnauthorizedError("Invalid or expired refresh token").
			WithRequestID(requestID.(string))
		errors.ErrorHandler(c, appErr)
		return
	}

	log.Info("Token refreshed successfully")
	c.JSON(http.StatusOK, tokens)
}

// Logout handles user logout
// @Summary User logout
// @Description Logout user and invalidate tokens
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errors.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *Handlers) Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	logger.LogSecurityEvent("auth_logout", strconv.Itoa(int(userID.(uint))), clientIP, userAgent, true)

	// In a real implementation, you might want to:
	// 1. Add the token to a blacklist
	// 2. Revoke the refresh token
	// 3. Log the logout event

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// GetProfile handles getting current user profile
// @Summary Get current user profile
// @Description Get the profile information of the currently authenticated user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Success 200 {object} UserInfo
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/profile [get]
func (h *Handlers) GetProfile(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")

	userInfo, err := h.authService.GetUserInfo(userID.(uint))
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

// ChangePassword handles changing user password
// @Summary Change password
// @Description Change the password of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Param request body ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/change-password [post]
func (h *Handlers) ChangePassword(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	log := logger.GetGlobalLogger().WithRequestID(requestID.(string))

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid request body").
			WithRequestID(requestID.(string)).
			WithDetails(err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	if err := h.authService.ChangePassword(userID.(uint), &req); err != nil {
		log.Warn("Password change failed", zap.Uint("user_id", userID.(uint)), zap.Error(err))

		var appErr *errors.AppError
		if err.Error() == "current password is incorrect" {
			appErr = errors.NewUnauthorizedError("Current password is incorrect").
				WithRequestID(requestID.(string))
		} else {
			appErr = errors.NewInternalError("Failed to change password", err).
				WithRequestID(requestID.(string))
		}
		errors.ErrorHandler(c, appErr)
		return
	}

	log.Info("Password changed successfully", zap.Uint("user_id", userID.(uint)))
	logger.LogSecurityEvent("password_changed", strconv.Itoa(int(userID.(uint))), c.ClientIP(), c.GetHeader("User-Agent"), true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// GetMe handles getting current user information (simplified version)
// @Summary Get current user
// @Description Get basic information about the currently authenticated user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param X-Tenant-ID header string true "Tenant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errors.ErrorResponse
// @Router /api/v1/auth/me [get]
func (h *Handlers) GetMe(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	tenantID, _ := c.Get("tenant_id")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":   userID,
			"role":      userRole,
			"tenant_id": tenantID,
		},
		"request_id": requestID,
	})
}

// ValidateToken handles token validation
// @Summary Validate access token
// @Description Validate if an access token is still valid
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errors.ErrorResponse
// @Router /api/v1/auth/validate [get]
func (h *Handlers) ValidateToken(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"valid":   true,
			"user_id": userID,
			"role":    userRole,
		},
		"request_id": requestID,
	})
}
