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
