package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/logger"
)

// rateLimiter represents a rate limiter for client IPs.
type rateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

// newRateLimiter creates a new rate limiter.
func newRateLimiter(rps int, burst int) *rateLimiter {
	return &rateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   rate.Limit(rps),
		b:   burst,
	}
}

// Allow checks if a request from the given IP is allowed.
func (rl *rateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.ips[ip] = limiter
	}

	return limiter.Allow()
}

// isPublicWidgetPath reports whether the request path belongs to the public
// widget endpoints (bundle, demo page, REST API, WebSocket). These routes are
// embedded on arbitrary customer domains, so they must allow any origin.
func isPublicWidgetPath(path string) bool {
	return path == "/widget.js" ||
		strings.HasPrefix(path, "/widget/")
}

// setupCORS configures CORS middleware.
func (s *Server) setupCORS() {
	s.router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Public widget endpoints must allow any origin because they are embedded
		// on arbitrary customer domains. For all other routes, check the configured
		// allow-list.
		if isPublicWidgetPath(c.Request.URL.Path) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.CORS.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Set other CORS headers
		c.Header("Access-Control-Allow-Methods", strings.Join(s.config.CORS.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", strings.Join(s.config.CORS.ExposedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", strconv.FormatBool(s.config.CORS.AllowCredentials))
		c.Header("Access-Control-Max-Age", strconv.Itoa(s.config.CORS.MaxAge))

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}

// setupRateLimiting configures rate limiting middleware.
func (s *Server) setupRateLimiting() {
	limiter := newRateLimiter(s.config.RateLimit.RequestsPerSecond, s.config.RateLimit.Burst)

	s.router.Use(func(c *gin.Context) {
		clientIP := c.ClientIP()
		requestID, exists := c.Get("request_id")
		var log *logger.Logger
		if !exists {
			log = logger.GetGlobalLogger()
		} else {
			_ = requestID // Could extract requestID for logging if needed
			log = logger.GetGlobalLogger()
		}

		if !limiter.Allow(clientIP) {
			log.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			var requestIDStr string
			if requestIDVal, ok := requestID.(string); ok {
				requestIDStr = requestIDVal
			}
			appErr := errors.NewRateLimitError("Rate limit exceeded. Please try again later.").
				WithRequestID(requestIDStr).
				WithContext("client_ip", clientIP).
				WithContext("path", c.Request.URL.Path).
				WithContext("method", c.Request.Method)
			errors.ErrorHandler(c, appErr)
			return
		}

		c.Next()
	})
}

// setupSecurityHeaders configures security headers middleware.
func (s *Server) setupSecurityHeaders() {
	s.router.Use(func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// Hide server information
		c.Header("Server", "SmartTicket")

		// Content Security Policy (basic)
		if s.config.IsProduction() {
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		}

		c.Next()
	})
}

// requestIDMiddleware adds a unique request ID to each request.
func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// authMiddleware validates JWT tokens.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		var log *logger.Logger
		if !exists {
			log = logger.GetGlobalLogger()
		} else {
			_ = requestID // Could extract requestID for logging if needed
			log = logger.GetGlobalLogger()
		}
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		// Skip auth for development mode if needed
		if s.config.IsDevelopment() && c.GetHeader("X-Skip-Auth") == "true" {
			log.Debug("Skipping authentication for development")
			// Set a mock user for development
			c.Set("user_id", uint(1))
			c.Set("user_role", "admin")
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			appErr := errors.NewUnauthorizedError("Authorization header is required").
				WithRequestID(c.GetString("request_id"))
			logger.LogSecurityEvent("auth_missing_header", "", clientIP, userAgent, false)
			errors.ErrorHandler(c, appErr)
			return
		}

		// Validate Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			appErr := errors.NewValidationError("Authorization header must be in format 'Bearer {token}'").
				WithRequestID(c.GetString("request_id"))
			logger.LogSecurityEvent("auth_invalid_format", "", clientIP, userAgent, false)
			errors.ErrorHandler(c, appErr)
			return
		}

		token := parts[1]

		// Validate JWT token with auth service
		userID, userRole, customerID, err := s.validateJWTToken(token)
		if err != nil {
			appErr := errors.NewUnauthorizedError("Invalid or expired token").
				WithRequestID(c.GetString("request_id"))
			logger.LogSecurityEvent("auth_invalid_token", "", clientIP, userAgent, false)
			errors.ErrorHandler(c, appErr)
			return
		}

		// Log successful authentication
		logger.LogSecurityEvent("auth_success", fmt.Sprintf("%d", userID), clientIP, userAgent, true)

		// Set user information in context
		c.Set("user_id", userID)
		c.Set("user_role", userRole)
		if customerID != nil {
			c.Set("user_customer_id", *customerID)
		}
		c.Next()
	}
}

// adminMiddleware checks if user has admin privileges.
func (s *Server) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			userID, userIDExists := c.Get("user_id")
			logger.Debug("adminMiddleware: user_role not found",
				zap.Bool("userID_exists", userIDExists),
				zap.Any("userID", userID))

			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User authentication required",
				},
			})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok || role != "admin" {
			logger.Debug("adminMiddleware: user_role found but not admin",
				zap.String("role", role),
				zap.Bool("ok", ok))
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "Admin privileges required",
				},
			})
			c.Abort()
			return
		}

		logger.Debug("adminMiddleware: admin check passed",
			zap.String("role", role))
		c.Next()
	}
}

// checkUserPermission validates specific permission for the current user.
func (s *Server) checkUserPermission(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
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

		role, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid user role format",
				},
			})
			c.Abort()
			return
		}

		// Check if user has required role or is admin (admin bypasses role check)
		if role != requiredRole && role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": fmt.Sprintf("Requires role: %s or admin", requiredRole),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// wsActorFromContext builds an authz.Actor from the gin context values set by
// the auth middleware. Mirrors ticket.actorFromContext without requiring an import
// of the ticket package from the server package.
func wsActorFromContext(c *gin.Context) authz.Actor {
	a := authz.Actor{
		UserID: c.GetUint("user_id"),
		Role:   c.GetString("user_role"),
	}
	if v, ok := c.Get("user_customer_id"); ok {
		if cid, ok := v.(uint); ok {
			a.CustomerID = &cid
		}
	}
	return a
}

// Helper functions

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// validateJWTToken validates a JWT token and returns user information.
func (s *Server) validateJWTToken(token string) (uint, string, *uint, error) {
	if s.authService == nil {
		return 0, "", nil, fmt.Errorf("auth service not available")
	}

	claims, err := s.authService.ValidateToken(token)
	if err != nil {
		return 0, "", nil, err
	}

	return claims.UserID, claims.Role, claims.CustomerID, nil
}
