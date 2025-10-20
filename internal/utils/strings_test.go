package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	// Test different lengths
	lengths := []int{0, 1, 8, 16, 32, 64}

	for _, length := range lengths {
		str := GenerateRandomString(length)
		assert.Len(t, str, length, "Generated string should have correct length")

		// Check that all characters are from the expected charset
		for _, char := range str {
			assert.True(t, strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", char),
				"Generated string should only contain alphanumeric characters")
		}
	}
}

func TestGenerateRandomStringWithCharset(t *testing.T) {
	customCharset := "abc123"
	str := GenerateRandomStringWithCharset(10, customCharset)
	assert.Len(t, str, 10)

	// Check that all characters are from the custom charset
	for _, char := range str {
		assert.True(t, strings.ContainsRune(customCharset, char),
			"Generated string should only contain characters from custom charset")
	}
}

func TestSlugify(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"This is a Test", "this-is-a-test"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#$%^&*()Characters", "specialcharacters"},
		{"Already-slugified", "already-slugified"},
		{"", ""},
		{"---leading-and-trailing---", "leading-and-trailing"},
		{"CamelCaseString", "camelcasestring"},
		{"UPPERCASE", "uppercase"},
		{"123 Numbers 456", "123-numbers-456"},
	}

	for _, tc := range testCases {
		result := Slugify(tc.input)
		assert.Equal(t, tc.expected, result, "Slugify should work correctly for input: %s", tc.input)
	}
}

func TestTruncateString(t *testing.T) {
	testCases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello World", 5, "He..."},
		{"Short", 10, "Short"},
		{"Exact", 5, "Exact"},
		{"", 10, ""},
		{"Test", 0, ""},
		{"Hello World", 11, "Hello World"},
		{"Hello World", 3, "Hel"},
	}

	for _, tc := range testCases {
		result := TruncateString(tc.input, tc.maxLen)
		assert.Equal(t, tc.expected, result, "TruncateString should work correctly")
	}
}

func TestCapitalize(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "Hello world"},
		{"HELLO WORLD", "HELLO WORLD"},
		{"hELLO wORLD", "HELLO wORLD"},
		{"", ""},
		{"a", "A"},
		{"123 abc", "123 abc"},
		{"  leading space", "  leading space"},
	}

	for _, tc := range testCases {
		result := Capitalize(tc.input)
		assert.Equal(t, tc.expected, result, "Capitalize should work correctly")
	}
}

func TestCamelCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "helloWorld"},
		{"hello_world", "helloWorld"},
		{"hello-world", "helloWorld"},
		{"Hello World", "helloWorld"},
		{"HELLO_WORLD", "helloWorld"},
		{"single", "single"},
		{"", ""},
		{"alreadyCamelCase", "alreadycamelcase"},
		{"Multiple   Spaces", "multipleSpaces"},
	}

	for _, tc := range testCases {
		result := CamelCase(tc.input)
		assert.Equal(t, tc.expected, result, "CamelCase should work correctly for input: %s", tc.input)
	}
}

func TestPascalCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "HelloWorld"},
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"Hello World", "HelloWorld"},
		{"HELLO_WORLD", "HelloWorld"},
		{"single", "Single"},
		{"", ""},
		{"alreadyPascalCase", "Alreadypascalcase"},
		{"Multiple   Spaces", "MultipleSpaces"},
	}

	for _, tc := range testCases {
		result := PascalCase(tc.input)
		assert.Equal(t, tc.expected, result, "PascalCase should work correctly for input: %s", tc.input)
	}
}

func TestSnakeCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"helloWorld", "hello_world"},
		{"HelloWorld", "hello_world"},
		{"hello-world", "hello-world"},
		{"HELLO_WORLD", "hello_world"},
		{"single", "single"},
		{"", ""},
		{"already_snake_case", "already_snake_case"},
		{"Multiple   Spaces", "multiple   spaces"},
	}

	for _, tc := range testCases {
		result := SnakeCase(tc.input)
		assert.Equal(t, tc.expected, result, "SnakeCase should work correctly for input: %s", tc.input)
	}
}

func TestKebabCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"helloWorld", "hello-world"},
		{"HelloWorld", "hello-world"},
		{"hello_world", "hello_world"},
		{"HELLO-WORLD", "hello-world"},
		{"single", "single"},
		{"", ""},
		{"already-kebab-case", "already-kebab-case"},
		{"Multiple   Spaces", "multiple   spaces"},
	}

	for _, tc := range testCases {
		result := KebabCase(tc.input)
		assert.Equal(t, tc.expected, result, "KebabCase should work correctly for input: %s", tc.input)
	}
}

func TestContainsAny(t *testing.T) {
	testCases := []struct {
		str      string
		slice    []string
		expected bool
	}{
		{"apple banana orange", []string{"banana", "grape"}, true},
		{"apple orange", []string{"banana", "grape"}, false},
		{"", []string{"apple"}, false},
		{"test string", []string{"", "test"}, true},
		{"lower", []string{"UPPER", "lower"}, true},
		{"lower", []string{"UPPER", "LOWER"}, false},
	}

	for _, tc := range testCases {
		result := ContainsAny(tc.str, tc.slice)
		assert.Equal(t, tc.expected, result, "ContainsAny should work correctly")
	}
}

func TestContainsAll(t *testing.T) {
	testCases := []struct {
		slice    []string
		expected bool
	}{
		{[]string{"hello", "world"}, true},
		{[]string{"hello", "test"}, false},
		{[]string{"hello"}, true},
		{[]string{}, true},
	}

	for _, tc := range testCases {
		result := ContainsAll("hello world", tc.slice)
		assert.Equal(t, tc.expected, result, "ContainsAll should work correctly")
	}
}

func TestStartsWithAny(t *testing.T) {
	testCases := []struct {
		prefixes []string
		expected bool
	}{
		{[]string{"hello", "hi", "hey"}, true},
		{[]string{"bye", "goodbye"}, false},
		{[]string{}, false},
		{[]string{""}, true},
	}

	for _, tc := range testCases {
		result := StartsWithAny("hello world", tc.prefixes)
		assert.Equal(t, tc.expected, result, "StartsWithAny should work correctly")
	}
}

func TestEndsWithAny(t *testing.T) {
	testCases := []struct {
		suffixes []string
		expected bool
	}{
		{[]string{"world", "earth", "planet"}, true},
		{[]string{"hello", "hi"}, false},
		{[]string{}, false},
		{[]string{""}, true},
	}

	for _, tc := range testCases {
		result := EndsWithAny("hello world", tc.suffixes)
		assert.Equal(t, tc.expected, result, "EndsWithAny should work correctly")
	}
}

func TestStripWhitespace(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "helloworld"},
		{"  hello   world  ", "helloworld"},
		{"hello\t\nworld", "helloworld"},
		{"", ""},
		{"    ", ""},
	}

	for _, tc := range testCases {
		result := StripWhitespace(tc.input)
		assert.Equal(t, tc.expected, result, "StripWhitespace should work correctly")
	}
}

func TestIsEmptyString(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"\t\n\r", true},
		{"hello", false},
		{"  hello  ", false},
		{"0", false},
	}

	for _, tc := range testCases {
		result := IsEmptyString(tc.input)
		assert.Equal(t, tc.expected, result, "IsEmptyString should work correctly")
	}
}

func TestIsNotEmpty(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"   ", false},
		{"\t\n\r", false},
		{"hello", true},
		{"  hello  ", true},
		{"0", true},
	}

	for _, tc := range testCases {
		result := IsNotEmpty(tc.input)
		assert.Equal(t, tc.expected, result, "IsNotEmpty should work correctly")
	}
}

func TestMaskString(t *testing.T) {
	// Test email masking
	email := "user@example.com"
	masked := MaskString(email, 2, 2, "*")
	assert.Equal(t, "us************om", masked)

	// Test phone masking
	phone := "1234567890"
	maskedPhone := MaskString(phone, 3, 3, "*")
	assert.Equal(t, "123****890", maskedPhone)

	// Test string shorter than keepStart + keepEnd
	short := "abc"
	maskedShort := MaskString(short, 2, 2, "*")
	assert.Equal(t, short, maskedShort)

	// Test empty string
	maskedEmpty := MaskString("", 2, 2, "*")
	assert.Equal(t, "", maskedEmpty)

	// Test custom mask character
	customMask := MaskString("1234567890", 2, 2, "#")
	assert.Equal(t, "12######90", customMask)
}

func TestMaskEmail(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "us*r@example.com"},
		{"a@domain.com", "a@domain.com"},
		{"very.long.username@domain.com", "ve***************e@domain.com"},
		{"invalid-email", "in***********"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := MaskEmail(tc.input)
		assert.Equal(t, tc.expected, result, "MaskEmail should work correctly for input: %s", tc.input)
	}
}

func TestExtractDomain(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "example.com"},
		{"test@sub.domain.co.uk", "sub.domain.co.uk"},
		{"invalid-email", ""},
		{"@", ""},
		{"@domain.com", "domain.com"},
		{"user@", ""},
	}

	for _, tc := range testCases {
		result := ExtractDomain(tc.input)
		assert.Equal(t, tc.expected, result, "ExtractDomain should work correctly for input: %s", tc.input)
	}
}

func TestIsValidURLString(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"https://www.example.com", true},
		{"http://example.com", true},
		{"ftp://example.com", false},
		{"www.example.com", false},
		{"example.com", false},
		{"", false},
		{"https://", true},
	}

	for _, tc := range testCases {
		result := IsValidURLString(tc.input)
		assert.Equal(t, tc.expected, result, "IsValidURLString should work correctly")
	}
}

func TestGenerateTicketNumber(t *testing.T) {
	// Test ticket number generation
	ticket := GenerateTicketNumber()
	assert.Len(t, ticket, 9) // TKT-XXXXX format
	assert.True(t, strings.HasPrefix(ticket, "TKT-"))

	// Test that ticket numbers are different
	ticket2 := GenerateTicketNumber()
	assert.NotEqual(t, ticket, ticket2)

	// Test format
	assert.Regexp(t, `^TKT-\d{5}$`, ticket)
}

func TestExtractInitials(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"John Doe", "JD"},
		{"john doe", "JD"},
		{"Mary Jane Watson", "MJW"},
		{" single ", "S"},
		{"", ""},
		{"  ", ""},
		{"A", "A"},
	}

	for _, tc := range testCases {
		result := ExtractInitials(tc.input)
		assert.Equal(t, tc.expected, result, "ExtractInitials should work correctly")
	}
}

func TestGenerateRandomColor(t *testing.T) {
	// Test color generation
	color := GenerateRandomColor()
	assert.Len(t, color, 7) // #RRGGBB format
	assert.True(t, strings.HasPrefix(color, "#"))

	// Test format
	assert.Regexp(t, `^#[0-9a-fA-F]{6}$`, color)

	// Test that colors are different
	color2 := GenerateRandomColor()
	assert.NotEqual(t, color, color2)
}

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range testCases {
		result := FormatBytes(tc.bytes)
		assert.Equal(t, tc.expected, result, "FormatBytes should work correctly")
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		seconds  int64
		expected string
	}{
		{30, "30 seconds"},
		{60, "1 minutes"},
		{90, "1 minutes 30 seconds"},
		{3600, "1 hours"},
		{3660, "1 hours 1 minutes"},
		{7200, "2 hours"},
	}

	for _, tc := range testCases {
		result := FormatDuration(tc.seconds)
		assert.Equal(t, tc.expected, result, "FormatDuration should work correctly")
	}
}

func TestRandomChoice(t *testing.T) {
	items := []string{"apple", "banana", "orange", "grape"}

	// Test that RandomChoice returns an item from the slice
	choice := RandomChoice(items)
	assert.Contains(t, items, choice)

	// Test with empty slice
	var emptySlice []string
	emptyChoice := RandomChoice(emptySlice)
	assert.Equal(t, "", emptyChoice)
}

func TestShuffleString(t *testing.T) {
	input := "hello"
	shuffled := ShuffleString(input)

	// Check that shuffled string has same length and characters
	assert.Len(t, shuffled, len(input))
	assert.Equal(t, len(input), len(shuffled))

	// Check that all characters are present
	for _, char := range input {
		assert.Contains(t, shuffled, string(char))
	}

	// Test with empty string
	emptyShuffled := ShuffleString("")
	assert.Equal(t, "", emptyShuffled)
}

func TestGenerateOTP(t *testing.T) {
	// Test OTP generation
	otp := GenerateOTP(6)
	assert.Len(t, otp, 6)
	assert.Regexp(t, `^\d{6}$`, otp)

	// Test different lengths
	otp8 := GenerateOTP(8)
	assert.Len(t, otp8, 8)
	assert.Regexp(t, `^\d{8}$`, otp8)

	// Test that OTPs are different
	otp2 := GenerateOTP(6)
	assert.NotEqual(t, otp, otp2)
}

func TestFormatCurrency(t *testing.T) {
	testCases := []struct {
		amount   float64
		currency string
		expected string
	}{
		{123.45, "USD", "$123.45"},
		{123.45, "EUR", "€123.45"},
		{123.45, "GBP", "£123.45"},
		{123.45, "JPY", "123.45 JPY"},
	}

	for _, tc := range testCases {
		result := FormatCurrency(tc.amount, tc.currency)
		assert.Equal(t, tc.expected, result, "FormatCurrency should work correctly")
	}
}
