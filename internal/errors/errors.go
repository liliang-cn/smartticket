package errors

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// ErrorCode represents different types of errors in the system.
type ErrorCode string

const (
	// Validation errors.
	ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField  ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidFormat ErrorCode = "INVALID_FORMAT"
	ErrCodeValueTooLarge ErrorCode = "VALUE_TOO_LARGE"
	ErrCodeValueTooSmall ErrorCode = "VALUE_TOO_SMALL"

	// Authentication/Authorization errors.
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrCodeExpiredToken       ErrorCode = "EXPIRED_TOKEN"
	ErrCodeMissingAuth        ErrorCode = "MISSING_AUTH"
	ErrCodeInvalidPassword    ErrorCode = "INVALID_PASSWORD"
	ErrCodeAccountLocked      ErrorCode = "ACCOUNT_LOCKED"
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"

	// Resource errors.
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists  ErrorCode = "ALREADY_EXISTS"
	ErrCodeResourceLocked ErrorCode = "RESOURCE_LOCKED"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeStaleResource  ErrorCode = "STALE_RESOURCE"

	// Business logic errors.
	ErrCodeBusinessRule        ErrorCode = "BUSINESS_RULE_VIOLATION"
	ErrCodeQuotaExceeded       ErrorCode = "QUOTA_EXCEEDED"
	ErrCodeLimitReached        ErrorCode = "LIMIT_REACHED"
	ErrCodeInvalidState        ErrorCode = "INVALID_STATE"
	ErrCodeOperationNotAllowed ErrorCode = "OPERATION_NOT_ALLOWED"

	// System errors.
	ErrCodeInternal           ErrorCode = "INTERNAL_ERROR"
	ErrCodeDatabase           ErrorCode = "DATABASE_ERROR"
	ErrCodeNetwork            ErrorCode = "NETWORK_ERROR"
	ErrCodeTimeout            ErrorCode = "TIMEOUT"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeExternalService    ErrorCode = "EXTERNAL_SERVICE_ERROR"
	ErrCodeDependencyError    ErrorCode = "DEPENDENCY_ERROR"

	// Rate limiting errors.
	ErrCodeRateLimit       ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeTooManyRequests ErrorCode = "TOO_MANY_REQUESTS"

	// File errors.
	ErrCodeFileNotFound    ErrorCode = "FILE_NOT_FOUND"
	ErrCodeFileTooLarge    ErrorCode = "FILE_TOO_LARGE"
	ErrCodeInvalidFileType ErrorCode = "INVALID_FILE_TYPE"
	ErrCodeUploadFailed    ErrorCode = "UPLOAD_FAILED"

	// Configuration errors.
	ErrCodeConfigError   ErrorCode = "CONFIGURATION_ERROR"
	ErrCodeMissingConfig ErrorCode = "MISSING_CONFIG"
	ErrCodeInvalidConfig ErrorCode = "INVALID_CONFIG"
)

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// AppError represents a structured application error.
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Severity   ErrorSeverity          `json:"severity"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     uint                   `json:"user_id,omitempty"`
	TenantID   uint                   `json:"tenant_id,omitempty"`
	Cause      error                  `json:"-"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the error.
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithContext adds context to the error.
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRequestID adds request ID to the error.
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithUserID adds user ID to the error.
func (e *AppError) WithUserID(userID uint) *AppError {
	e.UserID = userID
	return e
}


// WithStackTrace adds stack trace to the error.
func (e *AppError) WithStackTrace() *AppError {
	e.StackTrace = getStackTrace()
	return e
}

// ToHTTPStatus returns the appropriate HTTP status code for the error.
func (e *AppError) ToHTTPStatus() int {
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}

	switch e.Code {
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeMissingField, ErrCodeInvalidFormat, ErrCodeValueTooLarge, ErrCodeValueTooSmall:
		return http.StatusBadRequest
	case ErrCodeUnauthorized, ErrCodeInvalidToken, ErrCodeExpiredToken, ErrCodeMissingAuth, ErrCodeInvalidPassword, ErrCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrCodeForbidden, ErrCodeAccountLocked, ErrCodeOperationNotAllowed:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeAlreadyExists, ErrCodeConflict, ErrCodeStaleResource:
		return http.StatusConflict
	case ErrCodeRateLimit, ErrCodeTooManyRequests:
		return http.StatusTooManyRequests
	case ErrCodeFileTooLarge:
		return http.StatusRequestEntityTooLarge
	case ErrCodeBusinessRule, ErrCodeQuotaExceeded, ErrCodeLimitReached, ErrCodeInvalidState:
		return http.StatusUnprocessableEntity
	case ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeInternal, ErrCodeDatabase, ErrCodeNetwork, ErrCodeDependencyError, ErrCodeConfigError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Log logs the error with appropriate level based on severity.
func (e *AppError) Log() {
	fields := []zap.Field{
		zap.String("error_code", string(e.Code)),
		zap.String("error_message", e.Message),
		zap.String("severity", string(e.Severity)),
		zap.Time("timestamp", e.Timestamp),
		zap.Int("http_status", e.ToHTTPStatus()),
	}

	if e.Details != "" {
		fields = append(fields, zap.String("details", e.Details))
	}
	if e.RequestID != "" {
		fields = append(fields, zap.String("request_id", e.RequestID))
	}
	if e.UserID != 0 {
		fields = append(fields, zap.Uint("user_id", e.UserID))
	}
		if e.Cause != nil {
		fields = append(fields, zap.Error(e.Cause))
	}
	if e.StackTrace != "" {
		fields = append(fields, zap.String("stack_trace", e.StackTrace))
	}

	// Add context fields
	for key, value := range e.Context {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case uint:
			fields = append(fields, zap.Uint(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}

	// Log with appropriate level based on severity
	switch e.Severity {
	case SeverityLow:
		logger.Debug("Application error", fields...)
	case SeverityMedium:
		logger.Info("Application error", fields...)
	case SeverityHigh:
		logger.Warn("Application error", fields...)
	case SeverityCritical:
		logger.Error("Application error", fields...)
	default:
		logger.Info("Application error", fields...)
	}
}

// Error builder functions

// NewValidationError creates a new validation error.
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Severity:   SeverityLow,
		Timestamp:  time.Now(),
	}
}

// NewInvalidInputError creates a new invalid input error.
func NewInvalidInputError(field, value string) *AppError {
	return &AppError{
		Code:       ErrCodeInvalidInput,
		Message:    fmt.Sprintf("Invalid value for field '%s'", field),
		Details:    fmt.Sprintf("Value: %s", value),
		HTTPStatus: http.StatusBadRequest,
		Severity:   SeverityLow,
		Timestamp:  time.Now(),
	}
}

// NewUnauthorizedError creates a new unauthorized error.
func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	return &AppError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
	}
}

// NewForbiddenError creates a new forbidden error.
func NewForbiddenError(message string) *AppError {
	if message == "" {
		message = "Insufficient permissions"
	}
	return &AppError{
		Code:       ErrCodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
	}
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(resource string) *AppError {
	// Convert to title case manually to avoid deprecated strings.Title
	title := resource
	if len(resource) > 0 {
		title = strings.ToUpper(resource[:1]) + strings.ToLower(resource[1:])
	}
	message := fmt.Sprintf("%s not found", title)
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    message,
		HTTPStatus: http.StatusNotFound,
		Severity:   SeverityLow,
		Timestamp:  time.Now(),
		Context:    map[string]interface{}{"resource": resource},
	}
}

// NewConflictError creates a new conflict error.
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
	}
}

// NewBusinessRuleError creates a new business rule violation error.
func NewBusinessRuleError(rule string, message string) *AppError {
	return &AppError{
		Code:       ErrCodeBusinessRule,
		Message:    message,
		Details:    fmt.Sprintf("Rule: %s", rule),
		HTTPStatus: http.StatusUnprocessableEntity,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
		Context:    map[string]interface{}{"rule": rule},
	}
}

// NewRateLimitError creates a new rate limit error.
func NewRateLimitError(message string) *AppError {
	if message == "" {
		message = "Rate limit exceeded"
	}
	return &AppError{
		Code:       ErrCodeRateLimit,
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
	}
}

// NewInternalError creates a new internal server error.
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		Details:    "An internal error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Severity:   SeverityHigh,
		Timestamp:  time.Now(),
		Cause:      cause,
	}
}

// NewDatabaseError creates a new database error.
func NewDatabaseError(operation string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeDatabase,
		Message:    fmt.Sprintf("Database operation failed: %s", operation),
		HTTPStatus: http.StatusInternalServerError,
		Severity:   SeverityHigh,
		Timestamp:  time.Now(),
		Cause:      cause,
		Context:    map[string]interface{}{"operation": operation},
	}
}

// NewServiceUnavailableError creates a new service unavailable error.
func NewServiceUnavailableError(service string) *AppError {
	message := fmt.Sprintf("Service '%s' is currently unavailable", service)
	return &AppError{
		Code:       ErrCodeServiceUnavailable,
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
		Severity:   SeverityHigh,
		Timestamp:  time.Now(),
		Context:    map[string]interface{}{"service": service},
	}
}

// NewExternalServiceError creates a new external service error.
func NewExternalServiceError(message string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeExternalService,
		Message:    message,
		HTTPStatus: http.StatusBadGateway,
		Severity:   SeverityHigh,
		Timestamp:  time.Now(),
		Cause:      cause,
	}
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(operation string, timeout time.Duration) *AppError {
	message := fmt.Sprintf("Operation '%s' timed out after %v", operation, timeout)
	return &AppError{
		Code:       ErrCodeTimeout,
		Message:    message,
		HTTPStatus: http.StatusRequestTimeout,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
		Context: map[string]interface{}{
			"operation": operation,
			"timeout":   timeout.String(),
		},
	}
}

// NewFileError creates a new file-related error.
func NewFileError(operation, filename string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeUploadFailed,
		Message:    fmt.Sprintf("File %s failed: %s", operation, filename),
		HTTPStatus: http.StatusInternalServerError,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
		Cause:      cause,
		Context:    map[string]interface{}{"operation": operation, "filename": filename},
	}
}

// Helper functions

// getStackTrace captures the current stack trace.
func getStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// IsAppError checks if an error is an AppError.
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if ok := errors.As(err, &appErr); ok {
		return appErr, true
	}
	return nil, false
}

// WrapError wraps any error into an AppError.
func WrapError(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, return it
	if appErr, ok := IsAppError(err); ok {
		return appErr
	}

	return &AppError{
		Code:       code,
		Message:    message,
		Details:    err.Error(),
		HTTPStatus: http.StatusInternalServerError,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
		Cause:      err,
	}
}

// HandleError logs an error and returns an appropriate response.
func HandleError(err error, requestID string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, add request ID and log it
	if appErr, ok := IsAppError(err); ok {
		appErr.WithRequestID(requestID).Log()
		return appErr
	}

	// Wrap unknown errors
	appErr := WrapError(err, ErrCodeInternal, "An unexpected error occurred")
	appErr.WithRequestID(requestID).WithStackTrace().Log()
	return appErr
}
