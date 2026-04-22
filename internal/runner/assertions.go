package runner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/oliveagle/jsonpath"
	"github.com/test-fleet/test-runner/pkg/models"
)

func (e *TestRunner) validateAssertions(res *http.Response, body []byte, assertions []models.Assertion) []models.AssertionResult {
	results := make([]models.AssertionResult, 0, len(assertions))
	for _, a := range assertions {
		results = append(results, e.checkAssertion(res, body, a))
	}
	return results
}

func (e *TestRunner) checkAssertion(res *http.Response, body []byte, a models.Assertion) models.AssertionResult {
	result := models.AssertionResult{
		Type:     a.Type,
		Operator: a.Operator,
		Source:   a.Source,
		Expected: a.Expected,
	}

	var actual interface{}
	switch strings.ToLower(a.Type) {
	case "status", "status_code":
		actual = res.StatusCode
	case "header":
		actual = res.Header.Get(a.Source)
	case "body":
		if strings.HasPrefix(a.Source, "$.") {
			var jsonData interface{}
			if err := json.Unmarshal(body, &jsonData); err == nil {
				if val, err := jsonpath.JsonPathLookup(jsonData, a.Source); err == nil {
					actual = val
				}
			}
		} else {
			actual = string(body)
		}
	}

	result.Actual = actual
	result.Passed = evaluate(a.Operator, actual, a.Expected)
	return result
}

func evaluate(operator string, actual, expected interface{}) bool {
	switch strings.ToLower(operator) {
	case "eq", "equals":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "ne", "not_equals":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
	case "not_contains":
		return !strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
	case "gt":
		a, aOk := toFloat64(actual)
		b, bOk := toFloat64(expected)
		return aOk && bOk && a > b
	case "gte":
		a, aOk := toFloat64(actual)
		b, bOk := toFloat64(expected)
		return aOk && bOk && a >= b
	case "lt":
		a, aOk := toFloat64(actual)
		b, bOk := toFloat64(expected)
		return aOk && bOk && a < b
	case "lte":
		a, aOk := toFloat64(actual)
		b, bOk := toFloat64(expected)
		return aOk && bOk && a <= b
	}
	return false
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err == nil
	}
	return 0, false
}
