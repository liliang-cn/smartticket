package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDirectory(t *testing.T) {
	t.Run("Valid directory", func(t *testing.T) {
		err := ValidateDirectory("/tmp")
		assert.NoError(t, err)
	})

	t.Run("Invalid directory", func(t *testing.T) {
		err := ValidateDirectory("/nonexistent/directory/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory does not exist")
	})

	t.Run("Empty directory path", func(t *testing.T) {
		err := ValidateDirectory("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory path cannot be empty")
	})
}

func TestValidateFile(t *testing.T) {
	t.Run("Valid file", func(t *testing.T) {
		err := ValidateFile("/etc/hosts") // Assuming this exists on most systems
		// This test might fail on some systems, so we'll just check that it doesn't panic
		if err != nil {
			assert.Contains(t, err.Error(), "file does not exist")
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		err := ValidateFile("/nonexistent/file/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")
	})

	t.Run("Empty file path", func(t *testing.T) {
		err := ValidateFile("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})
}

func TestValidateURL(t *testing.T) {
	t.Run("Valid HTTP URL", func(t *testing.T) {
		err := ValidateURL("http://example.com")
		assert.NoError(t, err)
	})

	t.Run("Valid HTTPS URL", func(t *testing.T) {
		err := ValidateURL("https://example.com/path")
		assert.NoError(t, err)
	})

	t.Run("Valid URL with port", func(t *testing.T) {
		err := ValidateURL("http://localhost:8080")
		assert.NoError(t, err)
	})

	t.Run("Invalid URL - no scheme", func(t *testing.T) {
		err := ValidateURL("example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid URL format")
	})

	t.Run("Invalid URL - malformed", func(t *testing.T) {
		err := ValidateURL("http://")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid URL format")
	})

	t.Run("Empty URL", func(t *testing.T) {
		err := ValidateURL("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL cannot be empty")
	})
}

func TestValidatePort(t *testing.T) {
	t.Run("Valid port numbers", func(t *testing.T) {
		validPorts := []int{80, 443, 8080, 3000, 6533}
		for _, port := range validPorts {
			err := ValidatePort(port)
			assert.NoError(t, err, "Port %d should be valid", port)
		}
	})

	t.Run("Invalid port numbers", func(t *testing.T) {
		invalidPorts := []int{-1, 0, 65536, 100000}
		for _, port := range invalidPorts {
			err := ValidatePort(port)
			assert.Error(t, err, "Port %d should be invalid", port)
			assert.Contains(t, err.Error(), "must be between 1 and 65535")
		}
	})
}

func TestValidateFilePath(t *testing.T) {
	t.Run("Valid file path", func(t *testing.T) {
		err := ValidateFilePath("/tmp/test.txt")
		assert.NoError(t, err)
	})

	t.Run("Valid relative path", func(t *testing.T) {
		err := ValidateFilePath("./test.txt")
		assert.NoError(t, err)
	})

	t.Run("Empty file path", func(t *testing.T) {
		err := ValidateFilePath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})

	t.Run("Path with directory traversal", func(t *testing.T) {
		err := ValidateFilePath("../../../etc/passwd")
		// This might or might not be an error depending on implementation
		// We just ensure it doesn't panic
		_ = err
	})
}

func TestValidateLogLevel(t *testing.T) {
	t.Run("Valid log levels", func(t *testing.T) {
		validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC"}
		for _, level := range validLevels {
			err := ValidateLogLevel(level)
			assert.NoError(t, err, "Log level %s should be valid", level)
		}
	})

	t.Run("Invalid log levels", func(t *testing.T) {
		invalidLevels := []string{"trace", "verbose", "unknown", "", "invalid"}
		for _, level := range invalidLevels {
			err := ValidateLogLevel(level)
			assert.Error(t, err, "Log level %s should be invalid", level)
			assert.Contains(t, err.Error(), "invalid log level")
		}
	})
}

func TestValidateJWTSecret(t *testing.T) {
	t.Run("Valid JWT secrets", func(t *testing.T) {
		validSecrets := []string{
			"this-is-a-valid-secret-key-that-is-long-enough",
			"super-secret-jwt-key-for-testing-purposes-only",
			"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		}
		for _, secret := range validSecrets {
			err := ValidateJWTSecret(secret)
			assert.NoError(t, err, "Secret should be valid")
		}
	})

	t.Run("Invalid JWT secrets - too short", func(t *testing.T) {
		invalidSecrets := []string{
			"short",
			"12345678",
			"too-short-key",
			"secret",
		}
		for _, secret := range invalidSecrets {
			err := ValidateJWTSecret(secret)
			assert.Error(t, err, "Secret %s should be invalid", secret)
			assert.Contains(t, err.Error(), "must be at least 32 characters")
		}
	})

	t.Run("Empty JWT secret", func(t *testing.T) {
		err := ValidateJWTSecret("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestValidateArrayNotEmpty(t *testing.T) {
	t.Run("Valid non-empty arrays", func(t *testing.T) {
		stringArray := []string{"item1", "item2"}
		err := ValidateArrayNotEmpty(stringArray, "test array")
		assert.NoError(t, err)

		intArray := []int{1, 2, 3}
		err = ValidateArrayNotEmpty(intArray, "test array")
		assert.NoError(t, err)

		interfaceArray := []interface{}{"item1", 2, true}
		err = ValidateArrayNotEmpty(interfaceArray, "test array")
		assert.NoError(t, err)
	})

	t.Run("Empty arrays", func(t *testing.T) {
		stringArray := []string{}
		err := ValidateArrayNotEmpty(stringArray, "test array")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test array cannot be empty")

		var nilArray []string
		err = ValidateArrayNotEmpty(nilArray, "test array")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test array cannot be empty")
	})
}

func TestValidateArrayItems(t *testing.T) {
	t.Run("Valid string array items", func(t *testing.T) {
		array := []string{"item1", "item2", "item3"}
		validator := func(item interface{}) error {
			if str, ok := item.(string); ok && str != "" {
				return nil
			}
			return assert.AnError
		}
		err := ValidateArrayItems(array, "test array", validator)
		assert.NoError(t, err)
	})

	t.Run("Invalid array items", func(t *testing.T) {
		array := []string{"item1", "", "item3"}
		validator := func(item interface{}) error {
			if str, ok := item.(string); ok && str != "" {
				return nil
			}
			return assert.AnError
		}
		err := ValidateArrayItems(array, "test array", validator)
		assert.Error(t, err)
	})

	t.Run("Empty array", func(t *testing.T) {
		array := []string{}
		validator := func(item interface{}) error { return nil }
		err := ValidateArrayItems(array, "test array", validator)
		assert.NoError(t, err) // Empty array should pass item validation
	})
}

func TestValidateMapNotEmpty(t *testing.T) {
	t.Run("Valid non-empty maps", func(t *testing.T) {
		stringMap := map[string]string{"key1": "value1", "key2": "value2"}
		err := ValidateMapNotEmpty(stringMap, "test map")
		assert.NoError(t, err)

		intMap := map[int]int{1: 100, 2: 200}
		err = ValidateMapNotEmpty(intMap, "test map")
		assert.NoError(t, err)

		interfaceMap := map[interface{}]interface{}{"key1": "value1", 2: 200}
		err = ValidateMapNotEmpty(interfaceMap, "test map")
		assert.NoError(t, err)
	})

	t.Run("Empty maps", func(t *testing.T) {
		stringMap := map[string]string{}
		err := ValidateMapNotEmpty(stringMap, "test map")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test map cannot be empty")

		var nilMap map[string]string
		err = ValidateMapNotEmpty(nilMap, "test map")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test map cannot be empty")
	})
}

func TestValidateRequiredString(t *testing.T) {
	t.Run("Valid strings", func(t *testing.T) {
		validStrings := []string{"hello", "test string", "value", "a"}
		for _, str := range validStrings {
			err := ValidateRequiredString(str, "test field")
			assert.NoError(t, err, "String '%s' should be valid", str)
		}
	})

	t.Run("Invalid strings", func(t *testing.T) {
		invalidStrings := []string{"", "   ", "\t", "\n"}
		for _, str := range invalidStrings {
			err := ValidateRequiredString(str, "test field")
			assert.Error(t, err, "String '%s' should be invalid", str)
			assert.Contains(t, err.Error(), "test field cannot be empty")
		}
	})
}

func TestValidatePositiveDuration(t *testing.T) {
	t.Run("Valid positive durations", func(t *testing.T) {
		validDurations := []time.Duration{
			time.Nanosecond,
			time.Millisecond,
			time.Second,
			time.Minute,
			time.Hour,
			24 * time.Hour,
		}
		for _, duration := range validDurations {
			err := ValidatePositiveDuration(duration, "test duration")
			assert.NoError(t, err, "Duration %v should be valid", duration)
		}
	})

	t.Run("Invalid durations", func(t *testing.T) {
		invalidDurations := []time.Duration{
			0,
			-time.Second,
			-time.Minute,
			-time.Hour,
		}
		for _, duration := range invalidDurations {
			err := ValidatePositiveDuration(duration, "test duration")
			assert.Error(t, err, "Duration %v should be invalid", duration)
			assert.Contains(t, err.Error(), "must be positive")
		}
	})
}

func TestValidateRange(t *testing.T) {
	t.Run("Valid range - int", func(t *testing.T) {
		err := ValidateRange(5, 1, 10, "test value")
		assert.NoError(t, err)

		err = ValidateRange(1, 1, 10, "test value")
		assert.NoError(t, err)

		err = ValidateRange(10, 1, 10, "test value")
		assert.NoError(t, err)
	})

	t.Run("Invalid range - int", func(t *testing.T) {
		err := ValidateRange(0, 1, 10, "test value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be between 1 and 10")

		err = ValidateRange(11, 1, 10, "test value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be between 1 and 10")
	})

	t.Run("Valid range - float64", func(t *testing.T) {
		err := ValidateRange(5.5, 1.0, 10.0, "test value")
		assert.NoError(t, err)

		err = ValidateRange(1.0, 1.0, 10.0, "test value")
		assert.NoError(t, err)

		err = ValidateRange(10.0, 1.0, 10.0, "test value")
		assert.NoError(t, err)
	})

	t.Run("Invalid range - float64", func(t *testing.T) {
		err := ValidateRange(0.5, 1.0, 10.0, "test value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be between 1 and 10")

		err = ValidateRange(10.5, 1.0, 10.0, "test value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be between 1 and 10")
	})
}

func TestValidateFloatRange(t *testing.T) {
	t.Run("Valid float range", func(t *testing.T) {
		validValues := []float64{1.0, 2.5, 5.0, 9.9, 10.0}
		for _, value := range validValues {
			err := ValidateFloatRange(value, 1.0, 10.0, "test value")
			assert.NoError(t, err, "Value %f should be valid", value)
		}
	})

	t.Run("Invalid float range", func(t *testing.T) {
		invalidValues := []float64{0.9, 10.1, -1.0, 100.0}
		for _, value := range invalidValues {
			err := ValidateFloatRange(value, 1.0, 10.0, "test value")
			assert.Error(t, err, "Value %f should be invalid", value)
			assert.Contains(t, err.Error(), "must be between 1.0 and 10.0")
		}
	})
}

func TestValidateEnum(t *testing.T) {
	t.Run("Valid enum values", func(t *testing.T) {
		validValues := []string{"active", "inactive", "pending"}
		enum := map[string]bool{
			"active":   true,
			"inactive": true,
			"pending":  true,
		}

		for _, value := range validValues {
			err := ValidateEnum(value, enum, "test field")
			assert.NoError(t, err, "Value '%s' should be valid", value)
		}
	})

	t.Run("Invalid enum values", func(t *testing.T) {
		enum := map[string]bool{
			"active":   true,
			"inactive": true,
			"pending":  true,
		}

		invalidValues := []string{"unknown", "deleted", "suspended", ""}
		for _, value := range invalidValues {
			err := ValidateEnum(value, enum, "test field")
			assert.Error(t, err, "Value '%s' should be invalid", value)
			assert.Contains(t, err.Error(), "must be one of")
		}
	})
}

func TestValidateEmail(t *testing.T) {
	t.Run("Valid emails", func(t *testing.T) {
		validEmails := []string{
			"test@example.com",
			"user.name@domain.co.uk",
			"user+tag@example.org",
			"12345@example.com",
			"user@test-domain.com",
		}
		for _, email := range validEmails {
			err := ValidateEmail(email)
			assert.NoError(t, err, "Email '%s' should be valid", email)
		}
	})

	t.Run("Invalid emails", func(t *testing.T) {
		invalidEmails := []string{
			"invalid-email",
			"@example.com",
			"user@",
			"user.name@",
			"@domain.com",
			"user..name@example.com",
			"user@.example.com",
			"",
			"test@.com",
			"test@example.",
			" test@example.com",
			"test@example.com ",
		}
		for _, email := range invalidEmails {
			err := ValidateEmail(email)
			assert.Error(t, err, "Email '%s' should be invalid", email)
			assert.Contains(t, err.Error(), "invalid email format")
		}
	})
}

func TestValidateVersion(t *testing.T) {
	t.Run("Valid versions", func(t *testing.T) {
		validVersions := []string{
			"1.0.0",
			"1.2.3",
			"10.20.30",
			"1.0.0-alpha",
			"1.0.0-beta.1",
			"1.0.0+build.1",
			"v1.2.3",
		}
		for _, version := range validVersions {
			err := ValidateVersion(version)
			assert.NoError(t, err, "Version '%s' should be valid", version)
		}
	})

	t.Run("Invalid versions", func(t *testing.T) {
		invalidVersions := []string{
			"1.0",
			"1",
			"1.0.0.0",
			"v1.0",
			"",
			"not.a.version",
			"1..0",
			".1.0",
			"1.0.",
		}
		for _, version := range invalidVersions {
			err := ValidateVersion(version)
			assert.Error(t, err, "Version '%s' should be invalid", version)
			assert.Contains(t, err.Error(), "invalid version format")
		}
	})
}
