package runner

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/test-fleet/test-runner/pkg/models"
)

func TestDetermineType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string type",
			value:    "hello",
			expected: "string",
		},
		{
			name:     "float64 type",
			value:    float64(42.5),
			expected: "number",
		},
		{
			name:     "int type",
			value:    int(42),
			expected: "number",
		},
		{
			name:     "int64 type",
			value:    int64(42),
			expected: "number",
		},
		{
			name:     "int32 type",
			value:    int32(42),
			expected: "number",
		},
		{
			name:     "boolean true",
			value:    true,
			expected: "boolean",
		},
		{
			name:     "boolean false",
			value:    false,
			expected: "boolean",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "null",
		},
		{
			name:     "unknown type defaults to string",
			value:    []string{"test"},
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractHeader(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name        string
		headerName  string
		response    *http.Response
		expected    string
		expectError bool
	}{
		{
			name:       "extract existing header",
			headerName: "Content-Type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			expected:    "application/json",
			expectError: false,
		},
		{
			name:       "extract header case insensitive",
			headerName: "content-type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			expected:    "application/json",
			expectError: false,
		},
		{
			name:       "extract header with mixed case",
			headerName: "CoNtEnT-tYpE",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			expected:    "application/json",
			expectError: false,
		},
		{
			name:       "extract first value from multi-value header",
			headerName: "Set-Cookie",
			response: &http.Response{
				Header: http.Header{
					"Set-Cookie": []string{"cookie1=value1", "cookie2=value2"},
				},
			},
			expected:    "cookie1=value1",
			expectError: false,
		},
		{
			name:       "extract custom header",
			headerName: "X-Request-Id",
			response: &http.Response{
				Header: http.Header{
					"X-Request-Id": []string{"abc123"},
				},
			},
			expected:    "abc123",
			expectError: false,
		},
		{
			name:       "header with empty value",
			headerName: "X-Empty",
			response: &http.Response{
				Header: http.Header{
					"X-Empty": []string{""},
				},
			},
			expected:    "",
			expectError: false,
		},
		{
			name:       "header not found",
			headerName: "X-Missing",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			expected:    "",
			expectError: true,
		},
		{
			name:       "empty headers",
			headerName: "Content-Type",
			response: &http.Response{
				Header: http.Header{},
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.extractHeader(tt.headerName, tt.response)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractJson(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name          string
		path          string
		responseBody  string
		expectedValue interface{}
		expectedType  string
		expectError   bool
	}{
		{
			name:          "extract string field",
			path:          "$.name",
			responseBody:  `{"name": "John Doe", "age": 30}`,
			expectedValue: "John Doe",
			expectedType:  "string",
			expectError:   false,
		},
		{
			name:          "extract number field",
			path:          "$.age",
			responseBody:  `{"name": "John", "age": 30}`,
			expectedValue: float64(30),
			expectedType:  "number",
			expectError:   false,
		},
		{
			name:          "extract boolean field",
			path:          "$.active",
			responseBody:  `{"active": true, "name": "John"}`,
			expectedValue: true,
			expectedType:  "boolean",
			expectError:   false,
		},
		{
			name:          "extract null field",
			path:          "$.data",
			responseBody:  `{"data": null}`,
			expectedValue: nil,
			expectedType:  "null",
			expectError:   false,
		},
		{
			name:          "extract nested field",
			path:          "$.user.email",
			responseBody:  `{"user": {"email": "john@example.com", "id": 1}}`,
			expectedValue: "john@example.com",
			expectedType:  "string",
			expectError:   false,
		},
		{
			name:          "extract deeply nested field",
			path:          "$.user.address.city",
			responseBody:  `{"user": {"address": {"city": "New York", "zip": "10001"}}}`,
			expectedValue: "New York",
			expectedType:  "string",
			expectError:   false,
		},
		{
			name:          "extract array element",
			path:          "$.items[0]",
			responseBody:  `{"items": ["apple", "banana", "orange"]}`,
			expectedValue: "apple",
			expectedType:  "string",
			expectError:   false,
		},
		{
			name:          "extract from array of objects",
			path:          "$.users[0].name",
			responseBody:  `{"users": [{"name": "Alice", "id": 1}, {"name": "Bob", "id": 2}]}`,
			expectedValue: "Alice",
			expectedType:  "string",
			expectError:   false,
		},
		{
			name:          "extract nested number",
			path:          "$.metadata.count",
			responseBody:  `{"metadata": {"count": 42, "status": "ok"}}`,
			expectedValue: float64(42),
			expectedType:  "number",
			expectError:   false,
		},
		{
			name:          "extract root level array element",
			path:          "$[0].id",
			responseBody:  `[{"id": 1, "name": "First"}, {"id": 2, "name": "Second"}]`,
			expectedValue: float64(1),
			expectedType:  "number",
			expectError:   false,
		},
		{
			name:         "invalid JSON",
			path:         "$.name",
			responseBody: `{invalid json}`,
			expectError:  true,
		},
		{
			name:         "path not found",
			path:         "$.nonexistent",
			responseBody: `{"name": "John"}`,
			expectError:  true,
		},
		{
			name:         "invalid path syntax",
			path:         "$..[invalid",
			responseBody: `{"name": "John"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(tt.responseBody)),
			}

			value, valueType, err := e.extractJson(tt.path, response)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
				assert.Equal(t, tt.expectedType, valueType)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name        string
		response    *http.Response
		extractors  []models.Extractors
		expectedMap map[string]models.Variable
		expectError bool
	}{
		{
			name: "extract single JSON variable",
			response: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"userId": 123, "name": "John"}`)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			extractors: []models.Extractors{
				{
					Name:     "USER_ID",
					Type:     "json",
					Source:   "$.userId",
					DataType: "number",
				},
			},
			expectedMap: map[string]models.Variable{
				"USER_ID": {Value: float64(123), Type: "number"},
			},
			expectError: false,
		},
		{
			name: "extract multiple JSON variables",
			response: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"userId": 123, "name": "John", "active": true}`)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			extractors: []models.Extractors{
				{
					Name:     "USER_ID",
					Type:     "json",
					Source:   "$.userId",
					DataType: "number",
				},
				{
					Name:     "USER_NAME",
					Type:     "json",
					Source:   "$.name",
					DataType: "string",
				},
				{
					Name:     "IS_ACTIVE",
					Type:     "json",
					Source:   "$.active",
					DataType: "boolean",
				},
			},
			expectedMap: map[string]models.Variable{
				"USER_ID":   {Value: float64(123), Type: "number"},
				"USER_NAME": {Value: "John", Type: "string"},
				"IS_ACTIVE": {Value: true, Type: "boolean"},
			},
			expectError: false,
		},
		{
			name: "extract header variable",
			response: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{}`)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Request-Id": []string{"req-123"},
				},
			},
			extractors: []models.Extractors{
				{
					Name:     "REQUEST_ID",
					Type:     "header",
					Source:   "X-Request-Id",
					DataType: "string",
				},
			},
			expectedMap: map[string]models.Variable{
				"REQUEST_ID": {Value: "req-123", Type: "string"},
			},
			expectError: false,
		},
		{
			name: "extract both JSON and header variables",
			response: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"userId": 456}`)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Request-Id": []string{"req-456"},
					"X-Rate-Limit": []string{"100"},
				},
			},
			extractors: []models.Extractors{
				{
					Name:     "USER_ID",
					Type:     "json",
					Source:   "$.userId",
					DataType: "number",
				},
				{
					Name:     "REQUEST_ID",
					Type:     "header",
					Source:   "X-Request-Id",
					DataType: "string",
				},
				{
					Name:     "RATE_LIMIT",
					Type:     "header",
					Source:   "X-Rate-Limit",
					DataType: "string",
				},
			},
			expectedMap: map[string]models.Variable{
				"USER_ID":    {Value: float64(456), Type: "number"},
				"REQUEST_ID": {Value: "req-456", Type: "string"},
				"RATE_LIMIT": {Value: "100", Type: "string"},
			},
			expectError: false,
		},
		{
			name: "auto-detect type when DataType not specified",
			response: &http.Response{
				Body:   io.NopCloser(bytes.NewBufferString(`{"count": 42, "name": "test"}`)),
				Header: http.Header{},
			},
			extractors: []models.Extractors{
				{
					Name:   "COUNT",
					Type:   "json",
					Source: "$.count",
				},
				{
					Name:   "NAME",
					Type:   "json",
					Source: "$.name",
				},
			},
			expectedMap: map[string]models.Variable{
				"COUNT": {Value: float64(42), Type: "number"},
				"NAME":  {Value: "test", Type: "string"},
			},
			expectError: false,
		},
		{
			name: "override detected type with DataType",
			response: &http.Response{
				Body:   io.NopCloser(bytes.NewBufferString(`{"id": 123}`)),
				Header: http.Header{},
			},
			extractors: []models.Extractors{
				{
					Name:     "ID",
					Type:     "json",
					Source:   "$.id",
					DataType: "string",
				},
			},
			expectedMap: map[string]models.Variable{
				"ID": {Value: float64(123), Type: "string"},
			},
			expectError: false,
		},
		{
			name: "case insensitive extractor type",
			response: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"name": "test"}`)),
				Header: http.Header{
					"X-Test": []string{"value"},
				},
			},
			extractors: []models.Extractors{
				{
					Name:   "NAME",
					Type:   "JSON",
					Source: "$.name",
				},
				{
					Name:   "TEST_HEADER",
					Type:   "HEADER",
					Source: "X-Test",
				},
			},
			expectedMap: map[string]models.Variable{
				"NAME":        {Value: "test", Type: "string"},
				"TEST_HEADER": {Value: "value", Type: "string"},
			},
			expectError: false,
		},
		{
			name: "unknown extractor type",
			response: &http.Response{
				Body:   io.NopCloser(bytes.NewBufferString(`{}`)),
				Header: http.Header{},
			},
			extractors: []models.Extractors{
				{
					Name:   "TEST",
					Type:   "unknown",
					Source: "test",
				},
			},
			expectError: true,
		},
		{
			name: "JSON extraction error",
			response: &http.Response{
				Body:   io.NopCloser(bytes.NewBufferString(`{invalid}`)),
				Header: http.Header{},
			},
			extractors: []models.Extractors{
				{
					Name:   "NAME",
					Type:   "json",
					Source: "$.name",
				},
			},
			expectError: true,
		},
		{
			name: "header extraction error",
			response: &http.Response{
				Body:   io.NopCloser(bytes.NewBufferString(`{}`)),
				Header: http.Header{},
			},
			extractors: []models.Extractors{
				{
					Name:   "MISSING",
					Type:   "header",
					Source: "X-Missing",
				},
			},
			expectError: true,
		},
		{
			name: "empty extractors list",
			response: &http.Response{
				Body:   io.NopCloser(bytes.NewBufferString(`{}`)),
				Header: http.Header{},
			},
			extractors:  []models.Extractors{},
			expectedMap: map[string]models.Variable{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varMap := make(map[string]models.Variable)
			err := e.extractVariables(tt.response, tt.extractors, varMap)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMap, varMap)
			}
		})
	}
}

func TestExtractVariablesWithExistingMap(t *testing.T) {
	e := &TestRunner{}

	varMap := map[string]models.Variable{
		"EXISTING": {Value: "old-value", Type: "string"},
		"USER_ID":  {Value: float64(999), Type: "number"},
	}

	response := &http.Response{
		Body:   io.NopCloser(bytes.NewBufferString(`{"userId": 123}`)),
		Header: http.Header{},
	}

	extractors := []models.Extractors{
		{
			Name:     "USER_ID",
			Type:     "json",
			Source:   "$.userId",
			DataType: "number",
		},
	}

	err := e.extractVariables(response, extractors, varMap)
	assert.NoError(t, err)

	assert.Equal(t, "old-value", varMap["EXISTING"].Value)
	assert.Equal(t, float64(123), varMap["USER_ID"].Value)
}
