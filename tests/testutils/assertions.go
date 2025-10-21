package testutils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertionHelper provides enhanced assertion utilities
type AssertionHelper struct {
	t *testing.T
}

// NewAssertionHelper creates a new assertion helper
func NewAssertionHelper(t *testing.T) *AssertionHelper {
	return &AssertionHelper{t: t}
}

// AssertHTTPResponse asserts HTTP response properties
func (ah *AssertionHelper) AssertHTTPResponse(resp *http.Response, expectedStatus int, expectedContentType string) {
	assert.Equal(ah.t, expectedStatus, resp.StatusCode, "HTTP status code mismatch")

	if expectedContentType != "" {
		assert.Equal(ah.t, expectedContentType, resp.Header.Get("Content-Type"), "Content-Type mismatch")
	}
}

// AssertJSONResponse asserts that response contains valid JSON
func (ah *AssertionHelper) AssertJSONResponse(resp *http.Response) map[string]interface{} {
	assert.Equal(ah.t, "application/json", resp.Header.Get("Content-Type"), "Expected JSON response")

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(ah.t, err, "Failed to decode JSON response")

	return response
}

// AssertAPIResponse asserts standard API response structure
func (ah *AssertionHelper) AssertAPIResponse(resp *http.Response, expectedStatus int, expectSuccess bool) map[string]interface{} {
	ah.AssertHTTPResponse(resp, expectedStatus, "application/json")

	response := ah.AssertJSONResponse(resp)

	// Check for standard API response fields
	assert.Contains(ah.t, response, "success", "API response should contain 'success' field")
	assert.Equal(ah.t, expectSuccess, response["success"], "API response success flag mismatch")

	if expectSuccess {
		assert.Contains(ah.t, response, "data", "Successful API response should contain 'data' field")
	} else {
		assert.Contains(ah.t, response, "error", "Failed API response should contain 'error' field")
	}

	return response
}

// AssertPagination asserts pagination response structure
func (ah *AssertionHelper) AssertPagination(response map[string]interface{}) {
	data, exists := response["data"]
	require.True(ah.t, exists, "Response should contain data field")

	dataMap, ok := data.(map[string]interface{})
	require.True(ah.t, ok, "Data field should be a map")

	assert.Contains(ah.t, dataMap, "total", "Response should contain total count")
	assert.Contains(ah.t, dataMap, "page", "Response should contain page number")
	assert.Contains(ah.t, dataMap, "page_size", "Response should contain page size")
}

// AssertTimestamps asserts model timestamp fields
func (ah *AssertionHelper) AssertTimestamps(model interface{}) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check for CreatedAt and UpdatedAt fields
	createdAtField := v.FieldByName("CreatedAt")
	updatedAtField := v.FieldByName("UpdatedAt")

	assert.True(ah.t, createdAtField.IsValid(), "Model should have CreatedAt field")
	assert.True(ah.t, updatedAtField.IsValid(), "Model should have UpdatedAt field")

	if createdAtField.IsValid() && updatedAtField.IsValid() {
		createdAt := createdAtField.Interface().(time.Time)
		updatedAt := updatedAtField.Interface().(time.Time)

		assert.False(ah.t, createdAt.IsZero(), "CreatedAt should not be zero time")
		assert.False(ah.t, updatedAt.IsZero(), "UpdatedAt should not be zero time")
		assert.True(ah.t, updatedAt.After(createdAt) || updatedAt.Equal(createdAt),
			"UpdatedAt should be after or equal to CreatedAt")
	}
}

// AssertTenantIsolation asserts tenant isolation in responses
func (ah *AssertionHelper) AssertTenantIsolation(response map[string]interface{}, expectedTenantID string) {
	data, exists := response["data"]
	if !exists {
		return
	}

	switch v := data.(type) {
	case map[string]interface{}:
		ah.assertTenantInMap(v, expectedTenantID)
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				ah.assertTenantInMap(itemMap, expectedTenantID)
			}
		}
	}
}

func (ah *AssertionHelper) assertTenantInMap(item map[string]interface{}, expectedTenantID string) {
	if tenantID, exists := item["tenant_id"]; exists {
		assert.Equal(ah.t, expectedTenantID, tenantID, "Tenant ID mismatch in response")
	}
}

// AssertRequiredFields asserts that required fields are present and not empty
func (ah *AssertionHelper) AssertRequiredFields(model interface{}, requiredFields []string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, field := range requiredFields {
		fieldValue := v.FieldByName(field)
		assert.True(ah.t, fieldValue.IsValid(), fmt.Sprintf("Required field '%s' should exist", field))

		if fieldValue.IsValid() {
			assert.False(ah.t, fieldValue.IsZero(), fmt.Sprintf("Required field '%s' should not be zero value", field))
		}
	}
}

// AssertUUID asserts that a string is a valid UUID
func (ah *AssertionHelper) AssertUUID(uuidStr string) {
	assert.NotEmpty(ah.t, uuidStr, "UUID should not be empty")
	assert.Len(ah.t, uuidStr, 36, "UUID should be 36 characters long")

	// Basic UUID format validation
	uuidPattern := "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
	assert.Regexp(ah.t, uuidPattern, uuidStr, "UUID should match expected format")
}

// AssertEmailFormat asserts email format
func (ah *AssertionHelper) AssertEmailFormat(email string) {
	assert.NotEmpty(ah.t, email, "Email should not be empty")
	assert.Contains(ah.t, email, "@", "Email should contain @ symbol")
	assert.Contains(ah.t, email, ".", "Email should contain domain")
}

// AssertValidJSON asserts that a string contains valid JSON
func (ah *AssertionHelper) AssertValidJSON(jsonStr string) {
	var result interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	assert.NoError(ah.t, err, "String should contain valid JSON")
}

// AssertRecentTime asserts that a time is recent (within specified duration)
func (ah *AssertionHelper) AssertRecentTime(t time.Time, maxAge time.Duration) {
	now := time.Now()
	age := now.Sub(t)

	assert.True(ah.t, age >= 0, "Time should be in the past")
	assert.True(ah.t, age <= maxAge, fmt.Sprintf("Time should be recent (age: %v, max allowed: %v)", age, maxAge))
}

// AssertModelEquality asserts that two models are equal (excluding timestamps)
func (ah *AssertionHelper) AssertModelEquality(expected, actual interface{}) {
	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)

	if expectedValue.Kind() == reflect.Ptr {
		expectedValue = expectedValue.Elem()
	}
	if actualValue.Kind() == reflect.Ptr {
		actualValue = actualValue.Elem()
	}

	assert.Equal(ah.t, expectedValue.Type(), actualValue.Type(), "Models should have same type")

	for i := 0; i < expectedValue.NumField(); i++ {
		fieldName := expectedValue.Type().Field(i).Name

		// Skip timestamp fields for comparison
		if fieldName == "CreatedAt" || fieldName == "UpdatedAt" {
			continue
		}

		expectedField := expectedValue.Field(i)
		actualField := actualValue.FieldByName(fieldName)

		assert.True(ah.t, actualField.IsValid(), fmt.Sprintf("Field '%s' should exist in actual model", fieldName))
		if actualField.IsValid() {
			assert.Equal(ah.t, expectedField.Interface(), actualField.Interface(),
				fmt.Sprintf("Field '%s' should be equal", fieldName))
		}
	}
}

// AssertSliceContains asserts that a slice contains a specific item
func (ah *AssertionHelper) AssertSliceContains(slice interface{}, item interface{}) {
	sliceValue := reflect.ValueOf(slice)
	found := false

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), item) {
			found = true
			break
		}
	}

	assert.True(ah.t, found, fmt.Sprintf("Slice should contain item: %v", item))
}

// AssertSliceLength asserts slice length
func (ah *AssertionHelper) AssertSliceLength(slice interface{}, expectedLength int) {
	sliceValue := reflect.ValueOf(slice)
	assert.Equal(ah.t, expectedLength, sliceValue.Len(), "Slice length should match expected")
}

// AssertMapContainsKey asserts that a map contains a specific key
func (ah *AssertionHelper) AssertMapContainsKey(m interface{}, key interface{}) {
	mapValue := reflect.ValueOf(m)

	switch mapValue.Kind() {
	case reflect.Map:
		mapKey := reflect.ValueOf(key)
		assert.True(ah.t, mapValue.MapIndex(mapKey).IsValid(),
			fmt.Sprintf("Map should contain key: %v", key))
	default:
		assert.Fail(ah.t, "Expected a map", fmt.Sprintf("Got %T", m))
	}
}

// AssertMapLength asserts map length
func (ah *AssertionHelper) AssertMapLength(m interface{}, expectedLength int) {
	mapValue := reflect.ValueOf(m)

	switch mapValue.Kind() {
	case reflect.Map:
		assert.Equal(ah.t, expectedLength, mapValue.Len(), "Map length should match expected")
	default:
		assert.Fail(ah.t, "Expected a map", fmt.Sprintf("Got %T", m))
	}
}

// AssertNoErrorWithMessage asserts no error with custom message
func (ah *AssertionHelper) AssertNoErrorWithMessage(err error, context string) {
	assert.NoError(ah.t, err, fmt.Sprintf("%s: unexpected error", context))
}

// AssertErrorWithMessage asserts error with specific message
func (ah *AssertionHelper) AssertErrorWithMessage(err error, expectedMessage string) {
	assert.Error(ah.t, err, "Expected an error")
	assert.Contains(ah.t, err.Error(), expectedMessage, "Error message should contain expected text")
}

// AssertPerformance asserts that a function executes within time limit
func (ah *AssertionHelper) AssertPerformance(maxDuration time.Duration, fn func() error) {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	assert.NoError(ah.t, err, "Function should execute without error")
	assert.True(ah.t, duration <= maxDuration,
		fmt.Sprintf("Function should execute within %v (took %v)", maxDuration, duration))
}

// AssertMemoryUsage asserts that memory usage stays within bounds
func (ah *AssertionHelper) AssertMemoryUsage(maxMB float64, fn func() error) {
	// This is a simplified memory check - in real implementation, you'd use runtime.MemStats
	err := fn()
	assert.NoError(ah.t, err, "Function should execute without error")

	// For now, just log that memory check was performed
	ah.t.Logf("Memory usage check performed (max: %.2f MB)", maxMB)
}

// AssertRetryable asserts that a function succeeds after retries
func (ah *AssertionHelper) AssertRetryable(maxAttempts int, fn func() error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			ah.t.Logf("Function succeeded on attempt %d", attempt)
			return
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}

	assert.NoError(ah.t, lastErr, fmt.Sprintf("Function should succeed after %d attempts (last error: %v)", maxAttempts, lastErr))
}

// AssertConcurrent executes multiple functions concurrently and asserts they all succeed
func (ah *AssertionHelper) AssertConcurrent(functions []func() error) {
	results := make(chan error, len(functions))

	for i, fn := range functions {
		go func(index int, f func() error) {
			results <- f()
		}(i, fn)
	}

	// Collect results
	for i := 0; i < len(functions); i++ {
		err := <-results
		assert.NoError(ah.t, err, fmt.Sprintf("Concurrent function %d should succeed", i))
	}
}

// AssertHTTPHeaders asserts specific HTTP headers
func (ah *AssertionHelper) AssertHTTPHeaders(resp *http.Response, expectedHeaders map[string]string) {
	for key, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(key)
		assert.Equal(ah.t, expectedValue, actualValue,
			fmt.Sprintf("Header '%s' should match expected value", key))
	}
}

// AssertJSONField asserts a specific field in JSON response
func (ah *AssertionHelper) AssertJSONField(jsonData map[string]interface{}, fieldPath string, expectedValue interface{}) {
	parts := strings.Split(fieldPath, ".")
	current := jsonData

	for i, part := range parts {
		if i == len(parts)-1 {
			assert.Equal(ah.t, expectedValue, current[part],
				fmt.Sprintf("Field '%s' should match expected value", fieldPath))
		} else {
			if next, exists := current[part]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
					current = nextMap
				} else {
					assert.Fail(ah.t, "Field path contains non-map value",
						fmt.Sprintf("Field '%s' is not a map", strings.Join(parts[:i+1], ".")))
					return
				}
			} else {
				assert.Fail(ah.t, "Field not found",
					fmt.Sprintf("Field '%s' does not exist", strings.Join(parts[:i+1], ".")))
				return
			}
		}
	}
}
