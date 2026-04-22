package runner

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/test-fleet/test-runner/pkg/models"
)

func makeResponse(t *testing.T, statusCode int, headers map[string]string, body []byte) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	for k, v := range headers {
		rec.Header().Set(k, v)
	}
	rec.WriteHeader(statusCode)
	if len(body) > 0 {
		rec.Write(body)
	}
	return rec.Result()
}

func runAssertion(t *testing.T, a models.Assertion, statusCode int, headers map[string]string, body []byte) models.AssertionResult {
	t.Helper()
	r := newTestRunner(t, &http.Client{})
	res := makeResponse(t, statusCode, headers, body)
	return r.checkAssertion(res, body, a)
}

// --- status / status_code ---

func TestAssertion_Status_Eq_Pass(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "eq", Expected: float64(200), Source: "status"}, 200, nil, nil)
	assert.True(t, result.Passed)
	assert.Equal(t, 200, result.Actual)
}

func TestAssertion_Status_Eq_Fail(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "eq", Expected: float64(200), Source: "status"}, 404, nil, nil)
	assert.False(t, result.Passed)
	assert.Equal(t, 404, result.Actual)
}

func TestAssertion_StatusCode_Alias(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status_code", Operator: "eq", Expected: float64(201), Source: "status"}, 201, nil, nil)
	assert.True(t, result.Passed)
	assert.Equal(t, 201, result.Actual)
}

func TestAssertion_Status_Ne(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "ne", Expected: float64(500), Source: "status"}, 200, nil, nil)
	assert.True(t, result.Passed)
}

func TestAssertion_Status_Gt(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "gt", Expected: float64(199), Source: "status"}, 200, nil, nil)
	assert.True(t, result.Passed)
}

func TestAssertion_Status_Gte_Equal(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "gte", Expected: float64(200), Source: "status"}, 200, nil, nil)
	assert.True(t, result.Passed)
}

func TestAssertion_Status_Lt(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "lt", Expected: float64(400), Source: "status"}, 200, nil, nil)
	assert.True(t, result.Passed)
}

func TestAssertion_Status_Lte_Equal(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "status", Operator: "lte", Expected: float64(200), Source: "status"}, 200, nil, nil)
	assert.True(t, result.Passed)
}

// --- header ---

func TestAssertion_Header_Eq_Pass(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "header", Operator: "eq", Source: "Content-Type", Expected: "application/json"}, 200,
		map[string]string{"Content-Type": "application/json"}, nil)
	assert.True(t, result.Passed)
	assert.Equal(t, "application/json", result.Actual)
}

func TestAssertion_Header_Eq_Fail(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "header", Operator: "eq", Source: "Content-Type", Expected: "application/json"}, 200,
		map[string]string{"Content-Type": "text/plain"}, nil)
	assert.False(t, result.Passed)
	assert.Equal(t, "text/plain", result.Actual)
}

func TestAssertion_Header_Contains(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "header", Operator: "contains", Source: "Content-Type", Expected: "json"}, 200,
		map[string]string{"Content-Type": "application/json; charset=utf-8"}, nil)
	assert.True(t, result.Passed)
}

func TestAssertion_Header_Missing(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "header", Operator: "eq", Source: "X-Custom", Expected: "value"}, 200, nil, nil)
	assert.False(t, result.Passed)
	assert.Equal(t, "", result.Actual)
}

// --- body (raw) ---

func TestAssertion_Body_Raw_Contains(t *testing.T) {
	body := []byte(`{"message":"hello world"}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "contains", Source: "body", Expected: "hello"}, 200, nil, body)
	assert.True(t, result.Passed)
}

func TestAssertion_Body_Raw_NotContains(t *testing.T) {
	body := []byte(`{"message":"hello world"}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "not_contains", Source: "body", Expected: "error"}, 200, nil, body)
	assert.True(t, result.Passed)
}

// --- body (jsonpath) ---

func TestAssertion_Body_JSONPath_Eq_Pass(t *testing.T) {
	body := []byte(`{"status":"active","count":42}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "eq", Source: "$.status", Expected: "active"}, 200, nil, body)
	assert.True(t, result.Passed)
	assert.Equal(t, "active", result.Actual)
}

func TestAssertion_Body_JSONPath_Eq_Fail(t *testing.T) {
	body := []byte(`{"status":"inactive"}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "eq", Source: "$.status", Expected: "active"}, 200, nil, body)
	assert.False(t, result.Passed)
	assert.Equal(t, "inactive", result.Actual)
}

func TestAssertion_Body_JSONPath_Number_Gt(t *testing.T) {
	body := []byte(`{"count":100}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "gt", Source: "$.count", Expected: float64(50)}, 200, nil, body)
	assert.True(t, result.Passed)
}

func TestAssertion_Body_JSONPath_Nested(t *testing.T) {
	body := []byte(`{"user":{"id":99,"name":"Alice"}}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "eq", Source: "$.user.name", Expected: "Alice"}, 200, nil, body)
	assert.True(t, result.Passed)
}

func TestAssertion_Body_JSONPath_MissingKey(t *testing.T) {
	body := []byte(`{"other":"value"}`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "eq", Source: "$.missing", Expected: "x"}, 200, nil, body)
	assert.False(t, result.Passed)
	assert.Nil(t, result.Actual)
}

func TestAssertion_Body_JSONPath_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	result := runAssertion(t, models.Assertion{Type: "body", Operator: "eq", Source: "$.field", Expected: "x"}, 200, nil, body)
	assert.False(t, result.Passed)
	assert.Nil(t, result.Actual)
}

// --- unknown type ---

func TestAssertion_UnknownType_NilActual(t *testing.T) {
	result := runAssertion(t, models.Assertion{Type: "unknown", Operator: "eq", Expected: "x"}, 200, nil, nil)
	assert.False(t, result.Passed)
	assert.Nil(t, result.Actual)
}

// --- result fields ---

func TestAssertion_ResultFieldsPopulated(t *testing.T) {
	a := models.Assertion{Type: "status", Operator: "eq", Source: "status", Expected: float64(200)}
	result := runAssertion(t, a, 200, nil, nil)
	assert.Equal(t, "status", result.Type)
	assert.Equal(t, "eq", result.Operator)
	assert.Equal(t, "status", result.Source)
	assert.Equal(t, float64(200), result.Expected)
}
