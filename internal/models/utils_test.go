package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_Value(t *testing.T) {
	tests := []struct {
		name     string
		input    JSON
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "nil_json",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "empty_json",
			input:    JSON{},
			expected: []byte("{}"),
			wantErr:  false,
		},
		{
			name: "simple_json",
			input: JSON{
				"key1": "value1",
				"key2": 42,
			},
			expected: []byte(`{"key1":"value1","key2":42}`),
			wantErr:  false,
		},
		{
			name: "nested_json",
			input: JSON{
				"user": map[string]interface{}{
					"name": "John",
					"age":  30,
				},
				"active": true,
			},
			// Note: order may vary in map serialization
			wantErr: false,
		},
		{
			name: "json_with_array",
			input: JSON{
				"items": []interface{}{"item1", "item2", "item3"},
				"count": 3,
			},
			wantErr: false,
		},
		{
			name: "json_with_null_values",
			input: JSON{
				"nullValue":   nil,
				"emptyString": "",
				"zeroNumber":  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.Value()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.input == nil {
					assert.Nil(t, result)
				} else {
					// For non-nil cases, verify that the result is valid JSON
					resultBytes, ok := result.([]byte)
					require.True(t, ok, "Result should be []byte")

					// Verify it's valid JSON by unmarshaling
					var decoded map[string]interface{}
					err := json.Unmarshal(resultBytes, &decoded)
					assert.NoError(t, err, "Result should be valid JSON")

					// For specific expected values, compare directly
					if tt.expected != nil && len(tt.expected.([]byte)) > 0 {
						var expectedDecoded map[string]interface{}
						err := json.Unmarshal(tt.expected.([]byte), &expectedDecoded)
						if err == nil {
							assert.Equal(t, expectedDecoded, decoded)
						}
					}
				}
			}
		})
	}
}

func TestJSON_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected JSON
		wantErr  bool
	}{
		{
			name:     "nil_input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "empty_byte_slice",
			input:    []byte{},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "empty_string",
			input:    "",
			expected: nil,
			wantErr:  false,
		},
		{
			name:  "valid_json_bytes",
			input: []byte(`{"key1":"value1","key2":42}`),
			expected: JSON{
				"key1": "value1",
				"key2": float64(42), // JSON numbers are float64 by default
			},
			wantErr: false,
		},
		{
			name:  "valid_json_string",
			input: `{"name":"John","age":30}`,
			expected: JSON{
				"name": "John",
				"age":  float64(30),
			},
			wantErr: false,
		},
		{
			name:  "nested_json",
			input: []byte(`{"user":{"name":"Alice","details":{"age":25,"city":"NYC"}},"active":true}`),
			expected: JSON{
				"user": map[string]interface{}{
					"name": "Alice",
					"details": map[string]interface{}{
						"age":  float64(25),
						"city": "NYC",
					},
				},
				"active": true,
			},
			wantErr: false,
		},
		{
			name:  "json_with_array",
			input: `{"items":["a","b","c"],"count":3}`,
			expected: JSON{
				"items": []interface{}{"a", "b", "c"},
				"count": float64(3),
			},
			wantErr: false,
		},
		{
			name:  "json_with_null",
			input: `{"value":null,"empty":"","zero":0}`,
			expected: JSON{
				"value": nil,
				"empty": "",
				"zero":  float64(0),
			},
			wantErr: false,
		},
		{
			name:    "invalid_json_bytes",
			input:   []byte(`{invalid json}`),
			wantErr: true,
		},
		{
			name:    "invalid_json_string",
			input:   `{missing quotes}`,
			wantErr: true,
		},
		{
			name:    "unsupported_type",
			input:   123, // int is not supported
			wantErr: true,
		},
		{
			name:    "unsupported_struct",
			input:   struct{ Name string }{Name: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j JSON
			err := j.Scan(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, j)
			}
		})
	}
}

func TestJSON_String(t *testing.T) {
	tests := []struct {
		name     string
		input    JSON
		expected string
	}{
		{
			name:     "nil_json",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty_json",
			input:    JSON{},
			expected: "{}",
		},
		{
			name: "simple_json",
			input: JSON{
				"key": "value",
			},
			expected: `{"key":"value"}`,
		},
		{
			name: "multiple_fields",
			input: JSON{
				"name":   "John",
				"age":    30,
				"active": true,
			},
			// Note: map iteration order is not guaranteed, so we'll verify it's valid JSON
			expected: "", // Will be checked as valid JSON in the test logic
		},
		{
			name: "nested_json",
			input: JSON{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  25,
				},
				"settings": map[string]interface{}{
					"theme": "dark",
				},
			},
			expected: "", // Will be checked as valid JSON in the test logic
		},
		{
			name: "json_with_array",
			input: JSON{
				"items": []interface{}{"item1", "item2"},
				"count": 2,
			},
			expected: "", // Will be checked as valid JSON in the test logic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()

			if tt.input == nil {
				assert.Equal(t, "", result)
			} else if tt.expected == "" && len(tt.input) > 1 {
				// For complex cases with empty expected (multiple fields), verify it's valid JSON with correct content
				var decoded map[string]interface{}
				err := json.Unmarshal([]byte(result), &decoded)
				assert.NoError(t, err, "Result should be valid JSON")

				// Verify all expected keys and values are present
				// Note: numbers in JSON become float64 when unmarshaled
				for key, expectedValue := range tt.input {
					actualValue := decoded[key]

					// Handle type conversion for numbers
					if expectedInt, ok := expectedValue.(int); ok {
						if actualFloat, ok := actualValue.(float64); ok {
							assert.Equal(t, float64(expectedInt), actualFloat)
						} else {
							assert.Equal(t, expectedValue, actualValue)
						}
					} else if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
						actualMap, ok := actualValue.(map[string]interface{})
						assert.True(t, ok, "Expected map type")

						// Recursively check nested maps
						for nestedKey, nestedExpected := range expectedMap {
							nestedActual := actualMap[nestedKey]
							if nestedInt, ok := nestedExpected.(int); ok {
								if nestedFloat, ok := nestedActual.(float64); ok {
									assert.Equal(t, float64(nestedInt), nestedFloat)
								} else {
									assert.Equal(t, nestedExpected, nestedActual)
								}
							} else {
								assert.Equal(t, nestedExpected, nestedActual)
							}
						}
					} else {
						assert.Equal(t, expectedValue, actualValue)
					}
				}
			} else if len(tt.expected) > 0 {
				// For specific expected values, verify exact match (for simple cases)
				assert.Equal(t, tt.expected, result)
			} else {
				// For empty JSON or other cases, verify exact match
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestJSON_String_ErrorHandling(t *testing.T) {
	// Test with a value that can't be marshaled to JSON
	j := JSON{
		"invalid": func() {}, // functions can't be marshaled to JSON
	}

	result := j.String()
	assert.Equal(t, "", result, "Should return empty string when JSON marshal fails")
}

func TestJSON_RoundTrip(t *testing.T) {
	// Test Value() -> Scan() round trip
	original := JSON{
		"string":  "test",
		"number":  42.5,
		"boolean": true,
		"null":    nil,
		"array":   []interface{}{"a", "b", "c"},
		"object": map[string]interface{}{
			"nested": "value",
			"count":  3,
		},
	}

	// Convert to driver.Value
	value, err := original.Value()
	require.NoError(t, err)
	require.NotNil(t, value)

	// Scan back to JSON
	var scanned JSON
	err = scanned.Scan(value)
	require.NoError(t, err)

	// Compare the results
	assert.Equal(t, original["string"], scanned["string"])
	assert.Equal(t, original["boolean"], scanned["boolean"])
	assert.Equal(t, original["null"], scanned["null"])

	// Numbers might change precision during JSON encoding/decoding
	assert.Equal(t, float64(42.5), scanned["number"])

	// Arrays should match
	assert.Equal(t, original["array"], scanned["array"])

	// Objects should match (but nested numbers become float64)
	originalObject := original["object"].(map[string]interface{})
	scannedObject := scanned["object"].(map[string]interface{})
	assert.Equal(t, originalObject["nested"], scannedObject["nested"])
	assert.Equal(t, float64(3), scannedObject["count"])
}

func TestJSON_DatabaseCompatibility(t *testing.T) {
	// Test that JSON implements the required interfaces
	var j JSON

	// Test that it implements driver.Valuer
	_, ok := interface{}(j).(driver.Valuer)
	assert.True(t, ok, "JSON should implement driver.Valuer")

	// Test that it implements sql.Scanner (by testing the Scan method exists)
	assert.NotPanics(t, func() {
		j.Scan([]byte(`{"test": "value"}`))
	}, "JSON should implement sql.Scanner")
}

func TestJSON_EdgeCases(t *testing.T) {
	t.Run("very_large_json", func(t *testing.T) {
		// Test with a large JSON object
		large := JSON{}
		for i := 0; i < 1000; i++ {
			large[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}

		value, err := large.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		var scanned JSON
		err = scanned.Scan(value)
		assert.NoError(t, err)
		assert.Len(t, scanned, 1000)
	})

	t.Run("unicode_content", func(t *testing.T) {
		unicode := JSON{
			"emoji":   "🚀💡",
			"chinese": "你好世界",
			"special": "Special chars: àáâãäåæçèéêë",
		}

		value, err := unicode.Value()
		assert.NoError(t, err)

		var scanned JSON
		err = scanned.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, unicode["emoji"], scanned["emoji"])
		assert.Equal(t, unicode["chinese"], scanned["chinese"])
		assert.Equal(t, unicode["special"], scanned["special"])
	})

	t.Run("deeply_nested", func(t *testing.T) {
		nested := JSON{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"level4": map[string]interface{}{
							"value": "deep",
						},
					},
				},
			},
		}

		value, err := nested.Value()
		assert.NoError(t, err)

		var scanned JSON
		err = scanned.Scan(value)
		assert.NoError(t, err)

		// Navigate to deep value
		level1 := scanned["level1"].(map[string]interface{})
		level2 := level1["level2"].(map[string]interface{})
		level3 := level2["level3"].(map[string]interface{})
		level4 := level3["level4"].(map[string]interface{})
		assert.Equal(t, "deep", level4["value"])
	})
}
