package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"regexp"
	"strings"
)

// String utilities

// GenerateRandomString generates a random string of specified length.
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	func() { _, _ = rand.Read(bytes) }()
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}

// GenerateRandomStringWithCharset generates a random string using custom charset.
func GenerateRandomStringWithCharset(length int, charset string) string {
	bytes := make([]byte, length)
	func() { _, _ = rand.Read(bytes) }()
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}

// GenerateAPIKey generates a new API key with prefix.
func GenerateAPIKey(prefix string, length int) string {
	token, err := GenerateSecureCryptoToken(length)
	if err != nil {
		// Fallback to less secure method
		token = GenerateRandomString(length)
	}
	return fmt.Sprintf("%s_%s", prefix, token)
}

// GenerateUUID generates a UUID v4.
func GenerateUUID() string {
	b := make([]byte, 16)
	func() { _, _ = rand.Read(b) }()
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Slugify converts a string to a URL-friendly slug.
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove special characters except hyphens
	re := regexp.MustCompile(`[^a-z0-9-]`)
	s = re.ReplaceAllString(s, "")

	// Remove consecutive hyphens
	re = regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// SanitizeString removes potentially dangerous characters.
func SanitizeString(s string) string {
	// Remove SQL injection characters
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, ";", "")
	s = strings.ReplaceAll(s, "--", "")

	// Remove script tags
	re := regexp.MustCompile(`(?i)<script.*?>.*?</script>`)
	s = re.ReplaceAllString(s, "")

	// Remove HTML tags
	re = regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	return strings.TrimSpace(s)
}

// TruncateString truncates a string to specified length with ellipsis.
func TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

// Pluralize returns the plural form of a word based on count.
func Pluralize(word string, count int) string {
	if count == 1 {
		return word
	}

	// Simple English pluralization rules
	if strings.HasSuffix(word, "y") {
		return strings.TrimSuffix(word, "y") + "ies"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "sh") || strings.HasSuffix(word, "ch") {
		return word + "es"
	}
	if strings.HasSuffix(word, "f") {
		return strings.TrimSuffix(word, "f") + "ves"
	}
	if strings.HasSuffix(word, "fe") {
		return strings.TrimSuffix(word, "fe") + "ves"
	}

	return word + "s"
}

// Capitalize capitalizes the first letter of a string.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// CamelCase converts a string to camelCase.
func CamelCase(s string) string {
	// Split by spaces, hyphens, underscores
	parts := regexp.MustCompile(`[\s\-_]+`).Split(s, -1)

	for i, part := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(part)
		} else {
			parts[i] = Capitalize(strings.ToLower(part))
		}
	}

	return strings.Join(parts, "")
}

// PascalCase converts a string to PascalCase.
func PascalCase(s string) string {
	// Split by spaces, hyphens, underscores
	parts := regexp.MustCompile(`[\s\-_]+`).Split(s, -1)

	for i, part := range parts {
		parts[i] = Capitalize(strings.ToLower(part))
	}

	return strings.Join(parts, "")
}

// SnakeCase converts a string to snake_case.
func SnakeCase(s string) string {
	// Insert underscore before uppercase letters
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	s = re.ReplaceAllString(s, "${1}_${2}")

	// Convert to lowercase
	return strings.ToLower(s)
}

// KebabCase converts a string to kebab-case.
func KebabCase(s string) string {
	// Insert hyphen before uppercase letters
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	s = re.ReplaceAllString(s, "${1}-${2}")

	// Convert to lowercase
	return strings.ToLower(s)
}

// ContainsAny checks if string contains any of the substrings.
func ContainsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ContainsAll checks if string contains all of the substrings.
func ContainsAll(s string, substrings []string) bool {
	for _, substr := range substrings {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}

// StartsWithAny checks if string starts with any of the prefixes.
func StartsWithAny(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// EndsWithAny checks if string ends with any of the suffixes.
func EndsWithAny(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// StripWhitespace removes all whitespace from string.
func StripWhitespace(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// IsEmptyString checks if string is empty or contains only whitespace (renamed to avoid conflict).
func IsEmptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// IsNotEmpty checks if string is not empty and contains non-whitespace characters.
func IsNotEmpty(s string) bool {
	return !IsEmptyString(s)
}

// MaskString masks a string for display (e.g., for sensitive data).
func MaskString(s string, showFirst, showLast int, maskChar string) string {
	if len(s) <= showFirst+showLast {
		return s
	}

	if maskChar == "" {
		maskChar = "*"
	}

	first := s[:showFirst]
	last := s[len(s)-showLast:]
	mask := strings.Repeat(maskChar, len(s)-showFirst-showLast)

	return first + mask + last
}

// MaskEmail masks an email address for display.
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return MaskString(email, 2, 0, "*")
	}

	username := parts[0]
	domain := parts[1]

	maskedUsername := MaskString(username, 2, 1, "*")
	return fmt.Sprintf("%s@%s", maskedUsername, domain)
}

// ExtractDomain extracts domain from email address.
func ExtractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// IsValidURLString checks if string is a valid URL (basic validation) (renamed to avoid conflict).
func IsValidURLString(url string) bool {
	if url == "" {
		return false
	}

	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	// More comprehensive validation could be added here
	return true
}

// GenerateTicketNumber generates a ticket number in format TKT-XXXXX.
func GenerateTicketNumber() string {
	// Generate 5 random digits
	digits := make([]byte, 5)
	func() { _, _ = rand.Read(digits) }()
	for i, b := range digits {
		digits[i] = '0' + (b % 10)
	}

	return fmt.Sprintf("TKT-%s", string(digits))
}

// ExtractInitials extracts initials from a name.
func ExtractInitials(name string) string {
	words := strings.Fields(name)
	initials := make([]string, 0, len(words))

	for _, word := range words {
		if len(word) > 0 {
			initials = append(initials, strings.ToUpper(string(word[0])))
		}
	}

	return strings.Join(initials, "")
}

// GenerateRandomColor generates a random hex color.
func GenerateRandomColor() string {
	color := make([]byte, 3)
	func() { _, _ = rand.Read(color) }()
	return fmt.Sprintf("#%02x%02x%02x", color[0], color[1], color[2])
}

// FormatBytes formats bytes as human readable string.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration as human readable string.
func FormatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	if seconds < 3600 {
		minutes := seconds / 60
		remSeconds := seconds % 60
		if remSeconds == 0 {
			return fmt.Sprintf("%d minutes", minutes)
		}
		return fmt.Sprintf("%d minutes %d seconds", minutes, remSeconds)
	}
	hours := seconds / 3600
	remMinutes := (seconds % 3600) / 60
	if remMinutes == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, remMinutes)
}

// RandomChoice randomly selects an item from a slice.
func RandomChoice[T any](items []T) T {
	if len(items) == 0 {
		var zero T
		return zero
	}

	// Generate cryptographically secure random number
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	if err != nil {
		// Fallback to less secure method
		return items[0] // Just return first item
	}

	return items[n.Int64()]
}

// ShuffleString shuffles characters in a string.
func ShuffleString(s string) string {
	runes := []rune(s)
	mathrand.Shuffle(len(runes), func(i, j int) {
		runes[i], runes[j] = runes[j], runes[i]
	})
	return string(runes)
}

// GenerateOTP generates a one-time password.
func GenerateOTP(length int) string {
	digits := make([]byte, length)
	func() { _, _ = rand.Read(digits) }()
	for i, b := range digits {
		digits[i] = '0' + (b % 10)
	}
	return string(digits)
}

// FormatCurrency formats amount as currency string.
func FormatCurrency(amount float64, currency string) string {
	switch currency {
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	case "GBP":
		return fmt.Sprintf("£%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}
