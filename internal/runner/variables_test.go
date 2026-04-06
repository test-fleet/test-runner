package runner

import (
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/test-fleet/test-runner/pkg/models"
)

func TestFindVariableRefs(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single variable",
			input:    "${user_id}",
			expected: []string{"user_id"},
		},
		{
			name:     "multiple variables",
			input:    "${token} and ${user_id}",
			expected: []string{"token", "user_id"},
		},
		{
			name:     "no variables",
			input:    "plain text without variables",
			expected: []string{},
		},
		{
			name:     "variables in URL",
			input:    "https://api.example.com/${version}/users/${user_id}",
			expected: []string{"version", "user_id"},
		},
		{
			name:     "variables with numbers and underscores",
			input:    "${var_1} ${VAR_2} ${var123}",
			expected: []string{"var_1", "VAR_2", "var123"},
		},
		{
			name:     "duplicate variables",
			input:    "${token} and ${token} again",
			expected: []string{"token", "token"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "malformed variables",
			input:    "$token {user_id} ${} ${}",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.findVariableRefs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceVars(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name     string
		text     string
		varMap   map[string]models.Variable
		expected string
	}{
		{
			name: "replace string variable",
			text: "Hello ${name}",
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: "Hello John",
		},
		{
			name: "replace number variable (float64)",
			text: "Price: $${price}",
			varMap: map[string]models.Variable{
				"price": {Value: 99.99, Type: "number"},
			},
			expected: "Price: $99.99",
		},
		{
			name: "replace number variable (int)",
			text: "Count: ${count}",
			varMap: map[string]models.Variable{
				"count": {Value: 42, Type: "number"},
			},
			expected: "Count: 42",
		},
		{
			name: "replace boolean variable",
			text: "Active: ${is_active}",
			varMap: map[string]models.Variable{
				"is_active": {Value: true, Type: "boolean"},
			},
			expected: "Active: true",
		},
		{
			name: "replace nil variable",
			text: "Value: ${null_var}",
			varMap: map[string]models.Variable{
				"null_var": {Value: nil, Type: "null"},
			},
			expected: "Value: ",
		},
		{
			name: "replace multiple variables",
			text: "${greeting} ${name}, you have ${count} messages",
			varMap: map[string]models.Variable{
				"greeting": {Value: "Hello", Type: "string"},
				"name":     {Value: "Alice", Type: "string"},
				"count":    {Value: 5, Type: "number"},
			},
			expected: "Hello Alice, you have 5 messages",
		},
		{
			name: "variable not in map",
			text: "Hello ${unknown}",
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: "Hello ${unknown}",
		},
		{
			name:     "empty text",
			text:     "",
			varMap:   map[string]models.Variable{},
			expected: "",
		},
		{
			name: "no variables in text",
			text: "Plain text",
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: "Plain text",
		},
		{
			name: "same variable multiple times",
			text: "${name} ${name} ${name}",
			varMap: map[string]models.Variable{
				"name": {Value: "Bob", Type: "string"},
			},
			expected: "Bob Bob Bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.replaceVars(tt.text, tt.varMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceUrlVars(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name     string
		urlStr   string
		varMap   map[string]models.Variable
		expected string
	}{
		{
			name:   "replace path variable",
			urlStr: "https://api.example.com/users/${user_id}",
			varMap: map[string]models.Variable{
				"user_id": {Value: "123", Type: "string"},
			},
			expected: "https://api.example.com/users/123",
		},
		{
			name:   "replace multiple path variables",
			urlStr: "https://api.example.com/${version}/users/${user_id}",
			varMap: map[string]models.Variable{
				"version": {Value: "v1", Type: "string"},
				"user_id": {Value: "456", Type: "string"},
			},
			expected: "https://api.example.com/v1/users/456",
		},
		{
			name:   "clean up empty path segments",
			urlStr: "https://api.example.com//users//${user_id}//data",
			varMap: map[string]models.Variable{
				"user_id": {Value: "789", Type: "string"},
			},
			expected: "https://api.example.com/users/789/data",
		},
		{
			name:   "variable results in empty segment",
			urlStr: "https://api.example.com/users/${empty}/data",
			varMap: map[string]models.Variable{
				"empty": {Value: "", Type: "string"},
			},
			expected: "https://api.example.com/users/data",
		},
		{
			name:   "empty URL",
			urlStr: "",
			varMap: map[string]models.Variable{
				"user_id": {Value: "123", Type: "string"},
			},
			expected: "",
		},
		{
			name:   "URL with query parameters",
			urlStr: "https://api.example.com/users?id=${user_id}&active=${is_active}",
			varMap: map[string]models.Variable{
				"user_id":   {Value: "123", Type: "string"},
				"is_active": {Value: true, Type: "boolean"},
			},
			expected: "https://api.example.com/users?id=123&active=true",
		},
		{
			name:   "URL with spaces gets encoded",
			urlStr: "not a valid url ${var}",
			varMap: map[string]models.Variable{
				"var": {Value: "test", Type: "string"},
			},
			expected: "/not%20a%20valid%20url%20test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.replaceUrlVars(tt.urlStr, tt.varMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceHeaderVars(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name     string
		headers  map[string]string
		varMap   map[string]models.Variable
		expected map[string]string
	}{
		{
			name: "replace authorization header",
			headers: map[string]string{
				"Authorization": "Bearer ${token}",
				"Content-Type":  "application/json",
			},
			varMap: map[string]models.Variable{
				"token": {Value: "abc123xyz", Type: "string"},
			},
			expected: map[string]string{
				"Authorization": "Bearer abc123xyz",
				"Content-Type":  "application/json",
			},
		},
		{
			name: "replace multiple header values",
			headers: map[string]string{
				"X-User-ID":   "${user_id}",
				"X-Client-ID": "${client_id}",
				"X-Version":   "${version}",
			},
			varMap: map[string]models.Variable{
				"user_id":   {Value: "user123", Type: "string"},
				"client_id": {Value: "client456", Type: "string"},
				"version":   {Value: "v2", Type: "string"},
			},
			expected: map[string]string{
				"X-User-ID":   "user123",
				"X-Client-ID": "client456",
				"X-Version":   "v2",
			},
		},
		{
			name: "no variables in headers",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			varMap: map[string]models.Variable{
				"token": {Value: "abc123", Type: "string"},
			},
			expected: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
		},
		{
			name:    "empty headers",
			headers: map[string]string{},
			varMap: map[string]models.Variable{
				"token": {Value: "abc123", Type: "string"},
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.ReplaceHeaderVars(tt.headers, tt.varMap)
			assert.Equal(t, tt.expected, tt.headers)
		})
	}
}

func TestReplaceJsonVars(t *testing.T) {
	e := &TestRunner{
		logger: log.New(io.Discard, "", 0),
	}

	tests := []struct {
		name     string
		jsonText string
		varMap   map[string]models.Variable
		expected string
	}{
		{
			name:     "replace string in JSON",
			jsonText: `{"name": "${name}", "age": 30}`,
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: `{"name": "John", "age": 30}`,
		},
		{
			name:     "replace number in JSON",
			jsonText: `{"name": "John", "age": ${age}}`,
			varMap: map[string]models.Variable{
				"age": {Value: float64(30), Type: "number"},
			},
			expected: `{"name": "John", "age": 30}`,
		},
		{
			name:     "replace boolean in JSON",
			jsonText: `{"active": ${is_active}, "verified": true}`,
			varMap: map[string]models.Variable{
				"is_active": {Value: true, Type: "boolean"},
			},
			expected: `{"active": true, "verified": true}`,
		},
		{
			name:     "replace null in JSON",
			jsonText: `{"data": ${null_val}, "status": "ok"}`,
			varMap: map[string]models.Variable{
				"null_val": {Value: nil, Type: "null"},
			},
			expected: `{"data": null, "status": "ok"}`,
		},
		{
			name:     "empty JSON object",
			jsonText: `{}`,
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: `{}`,
		},
		{
			name:     "empty JSON array",
			jsonText: `[]`,
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: `[]`,
		},
		{
			name:     "variable not found replaces with empty string",
			jsonText: `{"name": "${unknown}"}`,
			varMap: map[string]models.Variable{
				"known": {Value: "value", Type: "string"},
			},
			expected: `{"name": ""}`,
		},
		{
			name:     "nil number variable",
			jsonText: `{"count": ${count}}`,
			varMap: map[string]models.Variable{
				"count": {Value: nil, Type: "number"},
			},
			expected: `{"count": 0}`,
		},
		{
			name:     "nil boolean variable",
			jsonText: `{"active": ${active}}`,
			varMap: map[string]models.Variable{
				"active": {Value: nil, Type: "boolean"},
			},
			expected: `{"active": false}`,
		},
		{
			name:     "nil string variable",
			jsonText: `{"name": "${name}"}`,
			varMap: map[string]models.Variable{
				"name": {Value: nil, Type: "string"},
			},
			expected: `{"name": ""}`,
		},
		{
			name:     "nested JSON",
			jsonText: `{"user": {"name": "${name}", "id": ${id}}, "active": ${active}}`,
			varMap: map[string]models.Variable{
				"name":   {Value: "Bob", Type: "string"},
				"id":     {Value: float64(123), Type: "number"},
				"active": {Value: true, Type: "boolean"},
			},
			expected: `{"user": {"name": "Bob", "id": 123}, "active": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.replaceJsonVars(tt.jsonText, tt.varMap)
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

func TestReplaceJsonVarsEdgeCases(t *testing.T) {
	e := &TestRunner{
		logger: log.New(io.Discard, "", 0),
	}

	tests := []struct {
		name     string
		jsonText string
		varMap   map[string]models.Variable
		expected string
	}{
		{
			name:     "invalid JSON falls back to simple replacement",
			jsonText: `not valid json ${name}`,
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: `not valid json John`,
		},
		{
			name:     "empty string",
			jsonText: ``,
			varMap: map[string]models.Variable{
				"name": {Value: "John", Type: "string"},
			},
			expected: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.replaceJsonVars(tt.jsonText, tt.varMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceVarsInJsonString(t *testing.T) {
	e := &TestRunner{}

	tests := []struct {
		name     string
		jsonText string
		varMap   map[string]models.Variable
		expected string
	}{
		{
			name:     "replace string variable",
			jsonText: "Hello ${name}",
			varMap: map[string]models.Variable{
				"name": {Value: "World", Type: "string"},
			},
			expected: "Hello World",
		},
		{
			name:     "replace number variable",
			jsonText: "Count: ${count}",
			varMap: map[string]models.Variable{
				"count": {Value: float64(42), Type: "number"},
			},
			expected: "Count: 42",
		},
		{
			name:     "replace boolean variable",
			jsonText: "Active: ${active}",
			varMap: map[string]models.Variable{
				"active": {Value: true, Type: "boolean"},
			},
			expected: "Active: true",
		},
		{
			name:     "replace nil variable",
			jsonText: "Value: ${nil_var}",
			varMap: map[string]models.Variable{
				"nil_var": {Value: nil, Type: "string"},
			},
			expected: "Value: ",
		},
		{
			name:     "replace nil null type variable",
			jsonText: "Value: ${nil_var}",
			varMap: map[string]models.Variable{
				"nil_var": {Value: nil, Type: "null"},
			},
			expected: "Value: null",
		},
		{
			name:     "replace nil number type variable",
			jsonText: "Count: ${count}",
			varMap: map[string]models.Variable{
				"count": {Value: nil, Type: "number"},
			},
			expected: "Count: 0",
		},
		{
			name:     "replace nil boolean type variable",
			jsonText: "Active: ${active}",
			varMap: map[string]models.Variable{
				"active": {Value: nil, Type: "boolean"},
			},
			expected: "Active: false",
		},
		{
			name:     "variable not found",
			jsonText: "Hello ${unknown}",
			varMap: map[string]models.Variable{
				"name": {Value: "World", Type: "string"},
			},
			expected: "Hello ",
		},
		{
			name:     "multiple variables",
			jsonText: "${first} ${second} ${third}",
			varMap: map[string]models.Variable{
				"first":  {Value: "One", Type: "string"},
				"second": {Value: float64(2), Type: "number"},
				"third":  {Value: true, Type: "boolean"},
			},
			expected: "One 2 true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.replaceVarsInJsonString(tt.jsonText, tt.varMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}
