package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ValidateDirectory checks if a directory exists and is accessible.
func ValidateDirectory(path, description string) error {
	if path == "" {
		return fmt.Errorf("%s path cannot be empty", description)
	}

	// Check if directory exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory %s: %w", description, path, err)
		}
		fmt.Printf("Created %s directory: %s\n", description, path)
	} else if err != nil {
		return fmt.Errorf("failed to access %s directory %s: %w", description, path, err)
	} else if !info.IsDir() {
		return fmt.Errorf("%s path %s is not a directory", description, path)
	}

	return nil
}

// ValidateFile checks if a file exists and is accessible.
func ValidateFile(path, description string, required bool) error {
	if path == "" {
		if required {
			return fmt.Errorf("%s file path cannot be empty", description)
		}
		return nil // Optional file not provided
	}

	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		if required {
			return fmt.Errorf("%s file does not exist: %s", description, path)
		}
		return nil // Optional file not found, that's okay
	} else if err != nil {
		return fmt.Errorf("failed to access %s file %s: %w", description, path, err)
	} else if info.IsDir() {
		return fmt.Errorf("%s path %s is a directory, not a file", description, path)
	}

	return nil
}

// ValidateURL checks if a URL is properly formatted.
func ValidateURL(url, description string, required bool) error {
	if url == "" {
		if required {
			return fmt.Errorf("%s URL cannot be empty", description)
		}
		return nil // Optional URL not provided
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("%s URL must start with http:// or https://: %s", description, url)
	}

	return nil
}

// ValidatePort checks if a port number is valid.
func ValidatePort(port int, description string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s port must be between 1 and 65535, got: %d", description, port)
	}
	return nil
}

// ValidateFilePath checks if a file path is safe and valid.
func ValidateFilePath(path, description string, required bool) error {
	if path == "" {
		if required {
			return fmt.Errorf("%s file path cannot be empty", description)
		}
		return nil // Optional path not provided
	}

	// Check for dangerous path components
	dangerous := []string{"..", "~", "/root", "/etc", "/usr/bin", "/bin"}
	for _, d := range dangerous {
		if strings.Contains(path, d) {
			return fmt.Errorf("%s file path contains dangerous component '%s': %s", description, d, path)
		}
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s file path must be absolute: %s", description, path)
	}

	return nil
}

// ValidateLogLevel checks if log level is valid.
func ValidateLogLevel(level string, allowedLevels []string, description string) error {
	if level == "" {
		return fmt.Errorf("%s log level cannot be empty", description)
	}

	for _, allowed := range allowedLevels {
		if level == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid %s log level '%s', must be one of: %v", description, level, allowedLevels)
}

// ValidateJWTSecret checks if JWT secret meets security requirements.
func ValidateJWTSecret(secret string, description string) error {
	if secret == "" {
		return fmt.Errorf("%s cannot be empty", description)
	}

	if len(secret) < 32 {
		return fmt.Errorf("%s must be at least 32 characters long, got %d", description, len(secret))
	}

	// Check for common weak secrets
	weakSecrets := []string{
		"secret",
		"password",
		"123456",
		"admin",
		"default",
		"changeme",
		"password123",
		"qwerty",
	}

	lowerSecret := strings.ToLower(secret)
	for _, weak := range weakSecrets {
		if strings.Contains(lowerSecret, weak) {
			return fmt.Errorf("%s contains weak password '%s'", description, weak)
		}
	}

	return nil
}

// ValidateArrayNotEmpty checks if a string array is not empty.
func ValidateArrayNotEmpty(arr []string, description string) error {
	if len(arr) == 0 {
		return fmt.Errorf("%s cannot be empty", description)
	}
	return nil
}

// ValidateArrayItems checks if array items are valid.
func ValidateArrayItems(items []string, validator func(string) error, description string) error {
	for i, item := range items {
		if err := validator(item); err != nil {
			return fmt.Errorf("invalid item %d in %s: %w", i, description, err)
		}
	}
	return nil
}

// ValidateMapNotEmpty checks if a map is not empty when required.
func ValidateMapNotEmpty(m map[string]interface{}, description string, required bool) error {
	if len(m) == 0 {
		if required {
			return fmt.Errorf("%s cannot be empty", description)
		}
	}
	return nil
}

// ValidateRequiredString checks if a required string is not empty.
func ValidateRequiredString(value, description string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be empty", description)
	}
	return nil
}

// ValidatePositiveDuration checks if a duration is positive.
func ValidatePositiveDuration(duration time.Duration, description string) error {
	if duration <= 0 {
		return fmt.Errorf("%s must be positive, got: %v", description, duration)
	}
	return nil
}

// ValidateRange checks if a value is within the specified range.
func ValidateRange(value, min, max int, description string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d, got: %d", description, min, max, value)
	}
	return nil
}

// ValidateFloatRange checks if a float value is within the specified range.
func ValidateFloatRange(value, min, max float64, description string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %.2f and %.2f, got: %.2f", description, min, max, value)
	}
	return nil
}

// ValidateEnum checks if a value is one of the allowed values.
func ValidateEnum(value string, allowed []string, description string) error {
	for _, allowed := range allowed {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("invalid %s '%s', must be one of: %v", description, value, allowed)
}

// ValidateEmail checks if an email address is valid (basic validation).
func ValidateEmail(email string, description string) error {
	if email == "" {
		return fmt.Errorf("%s cannot be empty", description)
	}

	// Basic email validation - check for @ and domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid %s format: %s", description, email)
	}

	// Check domain format
	domain := parts[1]
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("invalid %s domain: %s", description, domain)
	}

	return nil
}

// ValidateVersion checks if a version string follows semantic versioning.
func ValidateVersion(version string, description string) error {
	if version == "" {
		return fmt.Errorf("%s cannot be empty", description)
	}

	// Basic semantic version validation (major.minor.patch)
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("invalid %s format, expected x.y or x.y.z: %s", description, version)
	}

	return nil
}
