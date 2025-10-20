package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/company/smartticket/internal/errors"
)

// JSON utilities

// ToJSON converts any value to JSON string with error handling
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", errors.NewInternalError("Failed to marshal JSON", err)
	}
	return string(data), nil
}

// ToJSONPretty converts any value to pretty-printed JSON string
func ToJSONPretty(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", errors.NewInternalError("Failed to marshal pretty JSON", err)
	}
	return string(data), nil
}

// FromJSON parses JSON string into the provided interface
func FromJSON(jsonStr string, v interface{}) error {
	if err := json.Unmarshal([]byte(jsonStr), v); err != nil {
		return errors.NewValidationError("Failed to parse JSON").WithCause(err)
	}
	return nil
}

// FromJSONBytes parses JSON bytes into the provided interface
func FromJSONBytes(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return errors.NewValidationError("Failed to parse JSON bytes").WithCause(err)
	}
	return nil
}

// IsValidJSON checks if a string is valid JSON
func IsValidJSON(jsonStr string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}

// JSONGet extracts a value from JSON using dot notation
func JSONGet(jsonStr, path string) (interface{}, error) {
	var data interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return nil, err
	}

	return getNestedValue(data, strings.Split(path, "."))
}

// JSONSet sets a value in JSON using dot notation
func JSONSet(jsonStr, path string, value interface{}) (string, error) {
	var data interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	if err := setNestedValue(&data, strings.Split(path, "."), value); err != nil {
		return "", err
	}

	return ToJSON(data)
}

// JSONMerge merges two JSON objects
func JSONMerge(jsonStr1, jsonStr2 string) (string, error) {
	var data1, data2 map[string]interface{}

	if err := FromJSON(jsonStr1, &data1); err != nil {
		return "", err
	}

	if err := FromJSON(jsonStr2, &data2); err != nil {
		return "", err
	}

	merged := mergeMaps(data1, data2)
	return ToJSON(merged)
}

// JSONRemove removes a field from JSON using dot notation
func JSONRemove(jsonStr, path string) (string, error) {
	var data interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	if err := removeNestedValue(&data, strings.Split(path, ".")); err != nil {
		return "", err
	}

	return ToJSON(data)
}

// JSONGetKeys extracts all keys from a JSON object
func JSONGetKeys(jsonStr string) ([]string, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	return keys, nil
}

// JSONFlatten flattens a nested JSON object
func JSONFlatten(jsonStr string, separator string) (map[string]interface{}, error) {
	var data interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return nil, err
	}

	flattened := make(map[string]interface{})
	flattenValue(data, "", separator, flattened)
	return flattened, nil
}

// JSONUnflatten unflattens a flattened JSON object
func JSONUnflatten(flat map[string]interface{}, separator string) (map[string]interface{}, error) {
	var result interface{} = make(map[string]interface{})

	for key, value := range flat {
		if err := setNestedValue(&result, strings.Split(key, separator), value); err != nil {
			return nil, err
		}
	}

	return result.(map[string]interface{}), nil
}

// Conversion utilities

// ToString converts any value to string
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ToInt converts any value to int
func ToInt(v interface{}) (int, error) {
	if v == nil {
		return 0, nil
	}

	switch val := v.(type) {
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case uint:
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, errors.NewValidationError("Cannot convert string to int").WithCause(err)
		}
		return i, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, errors.NewValidationError("Cannot convert to int")
	}
}

// ToFloat64 converts any value to float64
func ToFloat64(v interface{}) (float64, error) {
	if v == nil {
		return 0.0, nil
	}

	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, errors.NewValidationError("Cannot convert string to float64").WithCause(err)
		}
		return f, nil
	case bool:
		if val {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, errors.NewValidationError("Cannot convert to float64")
	}
}

// ToBool converts any value to bool
func ToBool(v interface{}) (bool, error) {
	if v == nil {
		return false, nil
	}

	switch val := v.(type) {
	case bool:
		return val, nil
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(val).Int() != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(val).Uint() != 0, nil
	case float32, float64:
		return reflect.ValueOf(val).Float() != 0, nil
	case string:
		switch strings.ToLower(val) {
		case "true", "1", "yes", "on", "enabled":
			return true, nil
		case "false", "0", "no", "off", "disabled":
			return false, nil
		default:
			return false, errors.NewValidationError("Cannot convert string to bool")
		}
	default:
		return false, errors.NewValidationError("Cannot convert to bool")
	}
}

// ToSlice converts any value to slice
func ToSlice(v interface{}) ([]interface{}, error) {
	if v == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, errors.NewValidationError("Value is not a slice")
	}

	result := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result[i] = rv.Index(i).Interface()
	}

	return result, nil
}

// ToMap converts any value to map
func ToMap(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, errors.NewValidationError("Value is not a map")
	}

	result := make(map[string]interface{})
	for _, key := range rv.MapKeys() {
		result[ToString(key.Interface())] = rv.MapIndex(key).Interface()
	}

	return result, nil
}

// Type checking utilities

// IsEmpty checks if a value is empty
func IsEmpty(v interface{}) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return rv.IsNil()
	default:
		return false
	}
}

// IsNumeric checks if a value is numeric
func IsNumeric(v interface{}) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	case string:
		_, err := strconv.ParseFloat(v.(string), 64)
		return err == nil
	default:
		return false
	}
}

// IsInteger checks if a value is an integer
func IsInteger(v interface{}) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		_, err := strconv.ParseInt(v.(string), 10, 64)
		return err == nil
	default:
		return false
	}
}

// IsFloat checks if a value is a float
func IsFloat(v interface{}) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case float32, float64:
		return true
	case string:
		_, err := strconv.ParseFloat(v.(string), 64)
		return err == nil && strings.Contains(v.(string), ".")
	default:
		return false
	}
}

// IsArray checks if a value is an array/slice
func IsArray(v interface{}) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
}

// IsObject checks if a value is an object/map
func IsObject(v interface{}) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Map
}

// Deep copy utilities

// DeepCopy creates a deep copy of any value
func DeepCopy(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.NewInternalError("Failed to marshal for deep copy", err)
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.NewInternalError("Failed to unmarshal for deep copy", err)
	}

	return result, nil
}

// DeepEqual checks if two values are deeply equal
func DeepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// Helper functions for JSON operations

func getNestedValue(data interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return data, nil
	}

	current := data
	for _, key := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			if next, exists := v[key]; exists {
				current = next
			} else {
				return nil, errors.NewValidationError("Path not found in JSON")
			}
		case []interface{}:
			index, err := strconv.Atoi(key)
			if err != nil {
				return nil, errors.NewValidationError("Invalid array index in path")
			}
			if index < 0 || index >= len(v) {
				return nil, errors.NewValidationError("Array index out of bounds")
			}
			current = v[index]
		default:
			return nil, errors.NewValidationError("Invalid path for JSON structure")
		}
	}

	return current, nil
}

func setNestedValue(data *interface{}, path []string, value interface{}) error {
	if len(path) == 0 {
		*data = value
		return nil
	}

	key := path[0]
	remainingPath := path[1:]

	switch v := (*data).(type) {
	case map[string]interface{}:
		if len(remainingPath) == 0 {
			v[key] = value
		} else {
			if next, exists := v[key]; exists {
				if err := setNestedValue(&next, remainingPath, value); err != nil {
					return err
				}
				v[key] = next
			} else {
				var newValue interface{} = make(map[string]interface{})
				if err := setNestedValue(&newValue, remainingPath, value); err != nil {
					return err
				}
				v[key] = newValue
			}
		}
	case nil:
		var newMap interface{} = make(map[string]interface{})
		if err := setNestedValue(&newMap, path, value); err != nil {
			return err
		}
		*data = newMap
	default:
		return errors.NewValidationError("Invalid path for JSON structure")
	}

	return nil
}

func removeNestedValue(data *interface{}, path []string) error {
	if len(path) == 0 {
		return errors.NewValidationError("Cannot remove root element")
	}

	key := path[0]
	remainingPath := path[1:]

	switch v := (*data).(type) {
	case map[string]interface{}:
		if len(remainingPath) == 0 {
			delete(v, key)
		} else {
			if next, exists := v[key]; exists {
				if err := removeNestedValue(&next, remainingPath); err != nil {
					return err
				}
				if IsEmpty(next) {
					delete(v, key)
				} else {
					v[key] = next
				}
			}
		}
	default:
		return errors.NewValidationError("Invalid path for JSON structure")
	}

	return nil
}

func mergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy first map
	for k, v := range m1 {
		result[k] = v
	}

	// Merge second map
	for k, v := range m2 {
		if existing, exists := result[k]; exists {
			// If both are maps, merge them recursively
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if vMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeMaps(existingMap, vMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

func flattenValue(data interface{}, prefix, separator string, result map[string]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + separator + key
			}
			flattenValue(value, newPrefix, separator, result)
		}
	case []interface{}:
		for i, value := range v {
			newPrefix := fmt.Sprintf("%s%d", prefix, i)
			flattenValue(value, newPrefix, separator, result)
		}
	default:
		result[prefix] = v
	}
}
