package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("Test error")

	assert.Equal(t, ErrCodeValidation, err.Code)
	assert.Equal(t, "Test error", err.Message)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.False(t, err.Timestamp.IsZero())
}

func TestNewUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("Access denied")

	assert.Equal(t, ErrCodeUnauthorized, err.Code)
	assert.Equal(t, "Access denied", err.Message)
	assert.Equal(t, SeverityMedium, err.Severity)
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("User")

	assert.Equal(t, ErrCodeNotFound, err.Code)
	assert.Equal(t, "User not found", err.Message)
	assert.Equal(t, SeverityLow, err.Severity)
}

func TestNewInternalError(t *testing.T) {
	originalErr := errors.New("database connection failed")
	err := NewInternalError("Database error", originalErr)

	assert.Equal(t, ErrCodeInternal, err.Code)
	assert.Equal(t, "Database error", err.Message)
	assert.Equal(t, SeverityHigh, err.Severity)
	assert.Equal(t, originalErr, err.Cause)
}

func TestErrorWithMethods(t *testing.T) {
	originalErr := errors.New("test error")
	err := NewValidationError("Test error").
		WithRequestID("req-123").
		WithUserID(456).
		WithCause(originalErr).
		WithContext("field", "email").
		WithStackTrace()

	assert.Equal(t, "req-123", err.RequestID)
	assert.Equal(t, uint(456), err.UserID)
	assert.Equal(t, originalErr, err.Cause)
	assert.Equal(t, "email", err.Context["field"])
	assert.NotEmpty(t, err.StackTrace)
}

func TestErrorWithDetails(t *testing.T) {
	err := NewValidationError("Test error").WithDetails("field must be required")

	assert.Equal(t, "field must be required", err.Details)
}

func TestIsAppError(t *testing.T) {
	appErr := NewValidationError("Test error")
	plainErr := errors.New("plain error")

	isAppErr, ok := IsAppError(appErr)
	assert.True(t, ok)
	assert.NotNil(t, isAppErr)

	_, ok = IsAppError(plainErr)
	assert.False(t, ok)
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("test error")
	wrapped := WrapError(originalErr, ErrCodeInternal, "Wrapped error")

	assert.Equal(t, ErrCodeInternal, wrapped.Code)
	assert.Equal(t, "Wrapped error", wrapped.Message)
	assert.Equal(t, originalErr, wrapped.Cause)
}

func TestHandleError(t *testing.T) {
	// Test with AppError
	appErr := NewValidationError("Test error")
	handled := HandleError(appErr, "req-123")

	assert.Equal(t, appErr, handled)
	assert.Equal(t, "req-123", handled.RequestID)

	// Test with plain error
	plainErr := errors.New("plain error")
	handled = HandleError(plainErr, "req-456")

	assert.Equal(t, ErrCodeInternal, handled.Code)
	assert.Equal(t, "An unexpected error occurred", handled.Message)
	assert.Equal(t, "req-456", handled.RequestID)
}

func TestErrorToHTTPStatus(t *testing.T) {
	testCases := []struct {
		error    *AppError
		expected int
	}{
		{NewValidationError("test"), 400},
		{NewUnauthorizedError("test"), 401},
		{NewNotFoundError("test"), 404},
		{NewInternalError("test", nil), 500},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, tc.error.ToHTTPStatus())
	}
}

func TestErrorString(t *testing.T) {
	err := NewValidationError("Test error").WithDetails("field required")

	str := err.Error()
	assert.Contains(t, str, "VALIDATION_ERROR")
	assert.Contains(t, str, "Test error")
	assert.Contains(t, str, "field required")
}

func TestErrorLog(t *testing.T) {
	// This test just verifies that Log() doesn't panic
	err := NewValidationError("Test error")
	err.Log() // Should not panic
}

func TestErrorCreationChain(t *testing.T) {
	// Test that we can chain error creation methods
	originalErr := errors.New("test error")

	err := NewValidationError("Invalid input").
		WithDetails("Field 'email' is required").
		WithRequestID("req-123").
		WithUserID(456).
		WithCause(originalErr).
		WithContext("field", "email").
		WithStackTrace()

	assert.Equal(t, ErrCodeValidation, err.Code)
	assert.Equal(t, "Invalid input", err.Message)
	assert.Equal(t, "Field 'email' is required", err.Details)
	assert.Equal(t, "req-123", err.RequestID)
	assert.Equal(t, uint(456), err.UserID)
	assert.Equal(t, originalErr, err.Cause)
	assert.Equal(t, "email", err.Context["field"])
	assert.NotEmpty(t, err.StackTrace)
}
