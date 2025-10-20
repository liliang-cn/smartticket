package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.validate)
}

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()

	// Test with valid struct
	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"min=18"`
	}

	validTest := &TestStruct{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	err := validator.Validate(validTest)
	assert.NoError(t, err)

	// Test with invalid struct
	invalidTest := &TestStruct{
		Name:  "", // required field missing
		Email: "invalid-email",
		Age:   15, // below minimum
	}

	err = validator.Validate(invalidTest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Validation failed")
}

func TestValidator_ValidateVar(t *testing.T) {
	validator := NewValidator()

	// Test valid email
	err := validator.ValidateVar("test@example.com", "required,email")
	assert.NoError(t, err)

	// Test invalid email
	err = validator.ValidateVar("invalid-email", "required,email")
	assert.Error(t, err)
}

func TestValidateEmail(t *testing.T) {
	// Test valid emails
	assert.NoError(t, ValidateEmail("test@example.com"))
	assert.NoError(t, ValidateEmail("user.name+tag@domain.co.uk"))

	// Test invalid emails
	assert.Error(t, ValidateEmail(""))
	assert.Error(t, ValidateEmail("invalid-email"))
	assert.Error(t, ValidateEmail("@domain.com"))
}

func TestValidatePassword(t *testing.T) {
	// Test valid passwords
	assert.NoError(t, ValidatePassword("StrongPass123!"))
	assert.NoError(t, ValidatePassword("MySecureP@ssw0rd"))

	// Test invalid passwords
	assert.Error(t, ValidatePassword("weak"))
	assert.Error(t, ValidatePassword("12345678"))
	assert.Error(t, ValidatePassword("password"))
	assert.Error(t, ValidatePassword("PASSWORD"))
}

func TestValidateSlug(t *testing.T) {
	// Test valid slugs
	assert.NoError(t, ValidateSlug("valid-slug"))
	assert.NoError(t, ValidateSlug("test-slug-123"))
	assert.NoError(t, ValidateSlug("single"))

	// Test invalid slugs
	assert.Error(t, ValidateSlug("invalid slug"))
	assert.Error(t, ValidateSlug("slug-with--double-dash"))
	assert.Error(t, ValidateSlug("slug_with_underscore"))
	assert.NoError(t, ValidateSlug("")) // Empty slug is valid
}

func TestValidateRequired(t *testing.T) {
	// Test valid values
	assert.NoError(t, ValidateRequired("test", "field"))
	assert.NoError(t, ValidateRequired(123, "field"))

	// Test invalid values
	assert.Error(t, ValidateRequired("", "field"))
	assert.Error(t, ValidateRequired(nil, "field"))
}

func TestValidateStringLength(t *testing.T) {
	// Test valid length
	assert.NoError(t, ValidateStringLength("test", "field", 1, 10))
	assert.NoError(t, ValidateStringLength("length", "field", 3, 10))

	// Test invalid length
	assert.Error(t, ValidateStringLength("", "field", 1, 10))                 // Too short
	assert.Error(t, ValidateStringLength("this is too long", "field", 1, 10)) // Too long
}

func TestValidateRange(t *testing.T) {
	// Test valid range
	assert.NoError(t, ValidateRange(5, "field", 1, 10))
	assert.NoError(t, ValidateRange(1, "field", 1, 10))
	assert.NoError(t, ValidateRange(10, "field", 1, 10))

	// Test invalid range
	assert.Error(t, ValidateRange(0, "field", 1, 10))  // Too small
	assert.Error(t, ValidateRange(15, "field", 1, 10)) // Too large
}

func TestValidateEnum(t *testing.T) {
	allowed := []string{"low", "medium", "high"}

	// Test valid enum values
	assert.NoError(t, ValidateEnum("low", "field", allowed))
	assert.NoError(t, ValidateEnum("high", "field", allowed))

	// Test invalid enum value
	assert.Error(t, ValidateEnum("critical", "field", allowed))
}

func TestValidateUUID(t *testing.T) {
	// Test valid UUIDs
	assert.NoError(t, ValidateUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.NoError(t, ValidateUUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8"))

	// Test invalid UUIDs
	assert.Error(t, ValidateUUID(""))
	assert.Error(t, ValidateUUID("invalid-uuid"))
	assert.Error(t, ValidateUUID("550e8400-e29b-41d4-a716"))
}

func TestValidateDate(t *testing.T) {
	// Test valid dates
	assert.NoError(t, ValidateDate("2023-12-25"))
	assert.NoError(t, ValidateDate("2024-01-01"))

	// Test invalid dates
	assert.Error(t, ValidateDate("2023-13-01")) // Invalid month
	assert.Error(t, ValidateDate("2023-12-32")) // Invalid day
	assert.Error(t, ValidateDate("25-12-2023")) // Wrong format
	assert.NoError(t, ValidateDate(""))         // Empty date is valid (optional)
}

func TestValidatePaginationParams(t *testing.T) {
	// Test valid parameters
	assert.NoError(t, ValidatePaginationParams(1, 20))
	assert.NoError(t, ValidatePaginationParams(10, 50))

	// Test invalid parameters
	assert.Error(t, ValidatePaginationParams(0, 20))  // Invalid page
	assert.Error(t, ValidatePaginationParams(1, 0))   // Invalid page size
	assert.Error(t, ValidatePaginationParams(1, 101)) // Page size too large
}

func TestValidateSort(t *testing.T) {
	// Test valid sort
	assert.NoError(t, ValidateSort("created_at:asc", "created_at"))
	assert.NoError(t, ValidateSort("name:desc", "name"))
	assert.NoError(t, ValidateSort("", "field")) // Empty sort is valid

	// Test invalid sort
	assert.Error(t, ValidateSort("created_at", "field"))           // Missing direction
	assert.Error(t, ValidateSort("invalid_format", "field"))       // Invalid format
	assert.Error(t, ValidateSort("created_at:invalid", "field"))   // Invalid direction
	assert.Error(t, ValidateSort("wrong_field:asc", "created_at")) // Wrong field
}
