package errors

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// ErrorInfo represents error information in API responses
type ErrorInfo struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// ErrorMiddleware handles errors consistently across the application
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last().Err

			// Handle the error
			appErr := HandleError(err, getRequestID(c))

			// Create error response
			errorResponse := ErrorResponse{
				Success: false,
				Error: &ErrorInfo{
					Code:      string(appErr.Code),
					Message:   appErr.Message,
					Details:   appErr.Details,
					RequestID: appErr.RequestID,
					Timestamp: appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
					Context:   appErr.Context,
				},
			}

			// Set response status and return JSON
			c.JSON(appErr.ToHTTPStatus(), errorResponse)
			c.Abort()
		}
	}
}

// RecoveryMiddleware recovers from panics and converts them to errors
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := getRequestID(c)

		// Log the panic
		logger.WithRequestID(requestID).Error("Panic recovered",
			zap.Any("panic", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
		)

		// Create internal server error
		var message string
		if recoveredErr, ok := recovered.(error); ok {
			message = recoveredErr.Error()
		} else {
			message = "An unexpected error occurred"
		}

		appErr := NewInternalError(message, nil).
			WithRequestID(requestID).
			WithStackTrace().
			WithContext("method", c.Request.Method).
			WithContext("path", c.Request.URL.Path).
			WithContext("client_ip", c.ClientIP())

		// Log the error
		appErr.Log()

		// Create error response
		errorResponse := ErrorResponse{
			Success: false,
			Error: &ErrorInfo{
				Code:      string(appErr.Code),
				Message:   appErr.Message,
				RequestID: appErr.RequestID,
				Timestamp: appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			},
		}

		c.JSON(http.StatusInternalServerError, errorResponse)
		c.Abort()
	})
}

// ValidationMiddleware handles validation errors from request binding
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for binding errors
		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				if ginErr.Type == gin.ErrorTypeBind {
					// Convert binding error to validation error
					appErr := NewValidationError("Invalid request data").
						WithRequestID(getRequestID(c)).
						WithDetails(ginErr.Error())

					// Log validation error
					appErr.Log()

					// Create error response
					errorResponse := ErrorResponse{
						Success: false,
						Error: &ErrorInfo{
							Code:      string(appErr.Code),
							Message:   appErr.Message,
							Details:   appErr.Details,
							RequestID: appErr.RequestID,
							Timestamp: appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
						},
					}

					c.JSON(http.StatusBadRequest, errorResponse)
					c.Abort()
					return
				}
			}
		}
	}
}

// ErrorHandler provides a convenient way to handle errors in handlers
func ErrorHandler(c *gin.Context, err error) {
	if err == nil {
		return
	}

	appErr := HandleError(err, getRequestID(c))

	// Create error response
	errorResponse := ErrorResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:      string(appErr.Code),
			Message:   appErr.Message,
			Details:   appErr.Details,
			RequestID: appErr.RequestID,
			Timestamp: appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Context:   appErr.Context,
		},
	}

	c.JSON(appErr.ToHTTPStatus(), errorResponse)
	c.Abort()
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(c *gin.Context) {
	appErr := NewNotFoundError("endpoint").
		WithRequestID(getRequestID(c)).
		WithContext("method", c.Request.Method).
		WithContext("path", c.Request.URL.Path)

	// Log the error
	appErr.Log()

	errorResponse := ErrorResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:      string(appErr.Code),
			Message:   appErr.Message,
			RequestID: appErr.RequestID,
			Timestamp: appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Context:   appErr.Context,
		},
	}

	c.JSON(http.StatusNotFound, errorResponse)
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler(c *gin.Context) {
	appErr := NewBusinessRuleError("method_not_allowed", "Method not allowed for this endpoint").
		WithRequestID(getRequestID(c)).
		WithContext("method", c.Request.Method).
		WithContext("path", c.Request.URL.Path).
		WithContext("allowed_methods", c.Writer.Header().Get("Allow"))

	// Log the error
	appErr.Log()

	errorResponse := ErrorResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:      string(appErr.Code),
			Message:   appErr.Message,
			RequestID: appErr.RequestID,
			Timestamp: appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Context:   appErr.Context,
		},
	}

	c.JSON(http.StatusMethodNotAllowed, errorResponse)
}

// Helper functions

// getRequestID extracts request ID from context
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// WithDetails adds details to an AppError
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithTimestamp sets a custom timestamp
func (e *AppError) WithTimestamp(timestamp time.Time) *AppError {
	e.Timestamp = timestamp
	return e
}
