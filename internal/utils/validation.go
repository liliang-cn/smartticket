package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	stderrors "errors"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/logger"
)

// Validator wraps the validator instance.
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new validator instance.
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validators
	_ = v.RegisterValidation("slug", validateSlug)
	_ = v.RegisterValidation("strong_password", validateStrongPassword)
	_ = v.RegisterValidation("ticket_number", validateTicketNumber)
	_ = v.RegisterValidation("version", validateVersion)
	_ = v.RegisterValidation("api_key_prefix", validateAPIKeyPrefix)

	return &Validator{validate: v}
}

// Validate validates a struct and returns detailed errors.
func (v *Validator) Validate(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		var validationErrors validator.ValidationErrors
		if stderrors.As(err, &validationErrors) {
			return v.formatValidationErrors(validationErrors)
		}
		return errors.NewValidationError("Validation failed").WithCause(err)
	}
	return nil
}

// ValidateVar validates a single field.
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	if err := v.validate.Var(field, tag); err != nil {
		var validationErrors validator.ValidationErrors
		if stderrors.As(err, &validationErrors) {
			return v.formatValidationErrors(validationErrors)
		}
		return errors.NewValidationError("Field validation failed").WithCause(err)
	}
	return nil
}

// formatValidationErrors converts validator errors to a user-friendly format.
func (v *Validator) formatValidationErrors(errs validator.ValidationErrors) error {
	var errorMessages []string

	for _, e := range errs {
		fieldName := e.Field()
		tag := e.Tag()
		param := e.Param()

		// Get field name from struct tag if available
		if e.StructNamespace() != "" {
			fieldName = v.getFieldName(e.StructNamespace(), e.Field())
		}

		message := v.getErrorMessage(fieldName, tag, param)
		errorMessages = append(errorMessages, message)
	}

	return errors.NewValidationError("Validation failed").WithDetails(strings.Join(errorMessages, "; "))
}

// getFieldName extracts the field name from struct tags.
func (v *Validator) getFieldName(structNamespace, fieldName string) string {
	// For now, return the field name as-is
	// In a real implementation, you could use reflection to get json tags
	return fieldName
}

// getErrorMessage generates user-friendly error messages.
func (v *Validator) getErrorMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "slug":
		return fmt.Sprintf("%s must contain only letters, numbers, and hyphens", field)
	case "strong_password":
		return fmt.Sprintf("%s must be at least 8 characters with uppercase, lowercase, numbers, and special characters", field)
	case "ticket_number":
		return fmt.Sprintf("%s must be in format TKT-XXXXX where X is a digit", field)
	case "version":
		return fmt.Sprintf("%s must be in format x.y or x.y.z", field)
	case "api_key_prefix":
		return fmt.Sprintf("%s must start with 'sk_'", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, param)
	case "unique":
		return fmt.Sprintf("%s must be unique", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// Custom validation functions

// validateSlug validates slug format (letters, numbers, hyphens).
func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if slug == "" {
		return true
	}

	// Slug regex: lowercase letters, numbers, hyphens, no spaces, no consecutive hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, slug)
	return matched
}

// validateStrongPassword validates password strength.
func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// validateTicketNumber validates ticket number format.
func validateTicketNumber(fl validator.FieldLevel) bool {
	ticketNumber := fl.Field().String()
	if ticketNumber == "" {
		return true
	}

	matched, _ := regexp.MatchString(`^TKT-\d{5}$`, ticketNumber)
	return matched
}

// validateVersion validates semantic version format.
func validateVersion(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	if version == "" {
		return true
	}

	matched, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
	return matched
}

// validateAPIKeyPrefix validates API key prefix.
func validateAPIKeyPrefix(fl validator.FieldLevel) bool {
	apiKey := fl.Field().String()
	if apiKey == "" {
		return true
	}

	return strings.HasPrefix(apiKey, "sk_")
}

// Validation helper functions

// ValidateEmail validates email format.
func ValidateEmail(email string) error {
	v := NewValidator()
	return v.ValidateVar(email, "required,email")
}

// ValidateURL validates URL format.
func ValidateURL(url string) error {
	v := NewValidator()
	return v.ValidateVar(url, "url")
}

// ValidateSlug validates slug format.
func ValidateSlug(slug string) error {
	v := NewValidator()
	return v.ValidateVar(slug, "slug")
}

// ValidatePassword validates password strength.
func ValidatePassword(password string) error {
	v := NewValidator()
	return v.ValidateVar(password, "required,strong_password,min=8")
}

// ValidateRequired validates required fields.
func ValidateRequired(value interface{}, fieldName string) error {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.String && value.(string) == "") {
		return errors.NewValidationError(fmt.Sprintf("%s is required", fieldName))
	}
	return nil
}

// ValidateStringLength validates string length.
func ValidateStringLength(value string, fieldName string, min, max int) error {
	length := len(value)
	if length < min {
		return errors.NewValidationError(fmt.Sprintf("%s must be at least %d characters", fieldName, min))
	}
	if max > 0 && length > max {
		return errors.NewValidationError(fmt.Sprintf("%s must be at most %d characters", fieldName, max))
	}
	return nil
}

// ValidateRange validates numeric range.
func ValidateRange(value int, fieldName string, min, max int) error {
	if value < min {
		return errors.NewValidationError(fmt.Sprintf("%s must be at least %d", fieldName, min))
	}
	if max > 0 && value > max {
		return errors.NewValidationError(fmt.Sprintf("%s must be at most %d", fieldName, max))
	}
	return nil
}

// ValidateEnum validates that value is in allowed values.
func ValidateEnum(value string, fieldName string, allowed []string) error {
	for _, allowed := range allowed {
		if value == allowed {
			return nil
		}
	}
	return errors.NewValidationError(fmt.Sprintf("%s must be one of: %s", fieldName, strings.Join(allowed, ", ")))
}

// ValidateUUID validates UUID format.
func ValidateUUID(uuid string) error {
	if uuid == "" {
		return errors.NewValidationError("UUID is required")
	}

	// UUID regex for basic validation
	matched, _ := regexp.MatchString(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`, uuid)
	if !matched {
		return errors.NewValidationError("Invalid UUID format")
	}

	return nil
}

// ValidatePhoneNumber validates phone number format.
func ValidatePhoneNumber(phone string) error {
	if phone == "" {
		return nil // Optional field
	}

	// Basic phone number validation (international format)
	matched, _ := regexp.MatchString(`^\+?[1-9]\d{1,14}$`, phone)
	if !matched {
		return errors.NewValidationError("Invalid phone number format")
	}

	return nil
}

// ValidateIPAddress validates IP address format.
func ValidateIPAddress(ip string) error {
	if ip == "" {
		return nil // Optional field
	}

	// Basic IPv4 validation
	matched, _ := regexp.MatchString(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`, ip)
	if !matched {
		return errors.NewValidationError("Invalid IP address format")
	}

	return nil
}

// ValidateJSON validates JSON format.
func ValidateJSON(jsonStr string) error {
	if jsonStr == "" {
		return nil // Optional field
	}

	// Simple JSON validation - check for basic structure
	trimmed := strings.TrimSpace(jsonStr)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return errors.NewValidationError("Invalid JSON format")
	}

	// In a real implementation, you'd use json.Unmarshal to validate
	return nil
}

// ValidatePaginationParams validates pagination parameters (renamed to avoid conflict).
func ValidatePaginationParams(page, pageSize int) error {
	if page < 1 {
		return errors.NewValidationError("Page must be greater than 0")
	}
	if pageSize < 1 || pageSize > 100 {
		return errors.NewValidationError("Page size must be between 1 and 100")
	}
	return nil
}

// ValidateDate validates date format (YYYY-MM-DD).
func ValidateDate(dateStr string) error {
	if dateStr == "" {
		return nil // Optional field
	}

	// Basic date format validation
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, dateStr)
	if !matched {
		return errors.NewValidationError("Date must be in YYYY-MM-DD format")
	}

	// Try to parse the date to validate it's a real date
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return errors.NewValidationError("Invalid date")
	}

	return nil
}

// ValidateSort validates sort parameters.
func ValidateSort(sort, allowedField string) error {
	if sort == "" {
		return nil // Optional field
	}

	// Sort format: field:asc or field:desc
	parts := strings.Split(sort, ":")
	if len(parts) != 2 {
		return errors.NewValidationError("Sort must be in format 'field:direction'")
	}

	field := parts[0]
	direction := parts[1]

	// Validate field name if allowedField is provided
	if allowedField != "" && field != allowedField {
		return errors.NewValidationError(fmt.Sprintf("Sort field must be '%s'", allowedField))
	}

	// Validate direction
	if direction != "asc" && direction != "desc" {
		return errors.NewValidationError("Sort direction must be 'asc' or 'desc'")
	}

	return nil
}

// ValidateFilter validates filter parameters.
func ValidateFilter(filter map[string]interface{}, allowedFilters map[string]string) error {
	for key, value := range filter {
		// Check if filter is allowed
		if allowedFilter, exists := allowedFilters[key]; exists {
			// Validate filter value type
			switch allowedFilter {
			case "string":
				if _, ok := value.(string); !ok {
					return errors.NewValidationError(fmt.Sprintf("Filter '%s' must be a string", key))
				}
			case "int":
				if _, ok := value.(int); !ok {
					return errors.NewValidationError(fmt.Sprintf("Filter '%s' must be an integer", key))
				}
			case "bool":
				if _, ok := value.(bool); !ok {
					return errors.NewValidationError(fmt.Sprintf("Filter '%s' must be a boolean", key))
				}
			case "array":
				if _, ok := value.([]interface{}); !ok {
					return errors.NewValidationError(fmt.Sprintf("Filter '%s' must be an array", key))
				}
			}
		} else {
			return errors.NewValidationError(fmt.Sprintf("Filter '%s' is not allowed", key))
		}
	}
	return nil
}

// Global validator instance.
var globalValidator *Validator

// GetValidator returns the global validator instance.
func GetValidator() *Validator {
	if globalValidator == nil {
		globalValidator = NewValidator()
	}
	return globalValidator
}

// ValidateStruct validates a struct using the global validator.
func ValidateStruct(s interface{}) error {
	return GetValidator().Validate(s)
}

// ValidateField validates a single field using the global validator.
func ValidateField(field interface{}, tag string) error {
	return GetValidator().ValidateVar(field, tag)
}

// LogValidationError logs validation errors with context.
func LogValidationError(err error, context map[string]interface{}) {
	if appErr, ok := errors.IsAppError(err); ok && appErr.Code == errors.ErrCodeValidation {
		logger.Debug("Validation error",
			zap.String("error_code", string(appErr.Code)),
			zap.String("error_message", appErr.Message),
			zap.Any("context", context),
		)
	}
}
