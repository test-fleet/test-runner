package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/oliveagle/jsonpath"
	"github.com/test-fleet/test-runner/pkg/models"
)

var (
	JSON   = "json"
	HEADER = "header"
)

func (e *TestRunner) extractVariables(res *http.Response, extractors []models.Extractors, varMap map[string]models.Variable) error {
	var jsonData interface{}
	var bodyParsed bool

	for _, ext := range extractors {
		extractType := strings.ToLower(ext.Type)
		switch extractType {
		case "json":
			if !bodyParsed {
				body, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("failed to read response body: %w", err)
				}

				if err := json.Unmarshal(body, &jsonData); err != nil {
					return fmt.Errorf("failed to parse JSON: %w", err)
				}
				bodyParsed = true
			}

			value, valueType, err := e.extractJsonFromParsed(ext.Source, jsonData)
			if err != nil {
				return fmt.Errorf("error extracting JSON variable %s: %w", ext.Name, err)
			}

			eVar := models.Variable{
				Value: value,
				Type:  valueType,
			}

			if ext.DataType != "" {
				eVar.Type = ext.DataType
			}

			varMap[ext.Name] = eVar

		case "header":
			value, err := e.extractHeader(ext.Source, res)
			if err != nil {
				return fmt.Errorf("error extracting header variable %s: %w", ext.Name, err)
			}

			eVar := models.Variable{
				Value: value,
				Type:  "string",
			}

			if ext.DataType != "" {
				eVar.Type = ext.DataType
			}

			varMap[ext.Name] = eVar

		default:
			return fmt.Errorf("unknown extractor type: %s", extractType)
		}
	}

	return nil
}

func (e *TestRunner) extractJsonFromParsed(path string, jsonData interface{}) (interface{}, string, error) {
	value, err := jsonpath.JsonPathLookup(jsonData, path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to extract JSON path %s: %w", path, err)
	}

	valueType := determineType(value)
	return value, valueType, nil
}

func (e *TestRunner) extractHeader(headerName string, res *http.Response) (string, error) {
	values, exists := res.Header[http.CanonicalHeaderKey(headerName)]
	if !exists {
		return "", fmt.Errorf("header %s not found", headerName)
	}
	if len(values) == 0 {
		return "", nil
	}
	return values[0], nil
}

func determineType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "string"
	}
}
