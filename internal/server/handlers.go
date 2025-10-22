package server

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/logger"
)

// Response represents a standard API response.
type Response struct {
	Success bool              `json:"success"`
	Data    interface{}       `json:"data,omitempty"`
	Error   *errors.ErrorInfo `json:"error,omitempty"`
	Meta    *MetaInfo         `json:"meta,omitempty"`
}

// MetaInfo represents metadata in paginated responses.
type MetaInfo struct {
	Total      int `json:"total,omitempty"`
	Page       int `json:"page,omitempty"`
	PageSize   int `json:"page_size,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Health check handlers

// healthCheck handles health check requests.
// @Summary Health check
// @Description Checks the health status of the application and its dependencies
// @Tags system
// @Produce json
// @Success 200 {object} server.Response
// @Failure 503 {object} server.Response
// @Router /api/v1/health [get]
func (s *Server) healthCheck(c *gin.Context) {
	requestID, exists := c.Get("request_id")
	log := logger.GetGlobalLogger()
	if exists {
		if requestIDStr, ok := requestID.(string); ok {
			// Could use requestIDStr for logging if needed
			_ = requestIDStr
		}
	}

	// Check database health
	dbHealthy := s.db.IsHealthy()
	log.Debug("Database health check", zap.Bool("healthy", dbHealthy))

	// Check overall health
	healthy := dbHealthy

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
		log.Warn("Service health check failed", zap.Bool("database_healthy", dbHealthy))
	} else {
		log.Debug("Service health check passed")
	}

	response := Response{
		Success: healthy,
		Data: map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"checks": map[string]interface{}{
				"database": map[string]interface{}{
					"status": func() string {
						if dbHealthy {
							return "ok"
						}
						return "error"
					}(),
				},
			},
		},
	}

	if !healthy {
		response.Error = &errors.ErrorInfo{
			Code:    "HEALTH_CHECK_FAILED",
			Message: "Service is not healthy",
		}
	}

	c.JSON(status, response)
}

// versionInfo handles version information requests.
// @Summary Get version information
// @Description Returns version and build information about the application
// @Tags system
// @Produce json
// @Success 200 {object} server.Response
// @Router /api/v1/version [get]
func (s *Server) versionInfo(c *gin.Context) {
	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"version":     "1.0.0",
			"build_time":  "unknown",
			"go_version":  runtime.Version(),
			"environment": s.config.Environment,
		},
	}

	c.JSON(http.StatusOK, response)
}

// appInfo handles application information requests.
// @Summary Get application information
// @Description Returns general information about the SmartTicket application
// @Tags system
// @Produce json
// @Success 200 {object} server.Response
// @Router /api/v1/info [get]
func (s *Server) appInfo(c *gin.Context) {
	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"name":        "SmartTicket",
			"description": "Self-hosted multi-tenant ticketing platform",
			"version":     "1.0.0",
			"features": []string{
				"Multi-tenant support",
				"Ticket management",
				"Knowledge base",
				"AI integration",
				"Data import/export",
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

// User handlers - these are implemented in separate handler packages

// Ticket handlers - these are implemented in separate handler packages

// Knowledge base handlers - these are implemented in separate handler packages

// Admin handlers - these are implemented in separate handler packages
