package runner

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/test-fleet/test-runner/pkg/models"
)

func (e *TestRunner) findVariableRefs(input string) []string {
	re := regexp.MustCompile(`\${([A-Za-z0-9_]+)}`)
	matches := re.FindAllStringSubmatch(input, -1)
	vars := []string{}

	for _, match := range matches {
		if len(match) > 1 {
			vars = append(vars, match[1])
		}
	}
	return vars
}

func (e *TestRunner) replaceVars(text string, varMap map[string]models.Variable) string {
	if text == "" {
		return text
	}
	res := text
	re := regexp.MustCompile(`\${([A-Za-z0-9_]+)}`)
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			placeholder := match[0]
			varName := match[1]
			if variable, ok := varMap[varName]; ok {
				var varVal string
				if variable.Value == nil {
					varVal = ""
				} else {
					switch v := variable.Value.(type) {
					case string:
						varVal = v
					case float64:
						varVal = strconv.FormatFloat(v, 'f', -1, 64)
					case int:
						varVal = strconv.Itoa(v)
					case bool:
						varVal = strconv.FormatBool(v)
					default:
						varVal = fmt.Sprintf("%v", v)
					}
				}
				res = strings.Replace(res, placeholder, varVal, -1)
			}
		}
	}
	return res
}

func (e *TestRunner) replaceUrlVars(urlStr string, varMap map[string]models.Variable) string {
	if urlStr == "" {
		return urlStr
	}

	processedUrl := e.replaceVars(urlStr, varMap)

	u, err := url.Parse(processedUrl)
	if err != nil {
		return processedUrl
	}

	path := u.Path
	segments := strings.Split(path, "/")
	var cleanSegments []string
	for _, seg := range segments {
		if seg != "" {
			cleanSegments = append(cleanSegments, seg)
		}
	}

	u.Path = "/" + strings.Join(cleanSegments, "/")

	return u.String()
}

func (e *TestRunner) ReplaceHeaderVars(headers map[string]string, varMap map[string]models.Variable) {
	for key, val := range headers {
		headerValue := e.replaceVars(val, varMap)
		headers[key] = headerValue
	}
}

func (e *TestRunner) replaceJsonVars(jsonText string, varMap map[string]models.Variable) string {
	if jsonText == "" {
		return jsonText
	}

	var jsonData interface{}
	isValidJson := json.Unmarshal([]byte(jsonText), &jsonData) == nil

	trimmed := strings.TrimSpace(jsonText)
	if !isValidJson || trimmed == "{}" || trimmed == "[]" {
		return e.replaceVarsInJsonString(jsonText, varMap)
	}

	result := jsonText

	re := regexp.MustCompile(`\${([A-Za-z0-9_]+)}`)
	matches := re.FindAllStringSubmatch(jsonText, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			placeholder := match[0]
			varName := match[1]

			if variable, ok := varMap[varName]; ok {
				var replacement string

				if variable.Value == nil {
					switch variable.Type {
					case "null":
						replacement = "null"
					case "number":
						replacement = "0"
					case "boolean":
						replacement = "false"
					default:
						replacement = "\"\""
					}
					result = strings.Replace(result, placeholder, replacement, -1)
					continue
				}

				switch variable.Type {
				case "number":
					switch v := variable.Value.(type) {
					case float64:
						replacement = strconv.FormatFloat(v, 'f', -1, 64)
					case int:
						replacement = strconv.Itoa(v)
					case string:
						if _, err := strconv.ParseFloat(v, 64); err == nil {
							replacement = v
						} else {
							replacement = "0"
						}
					default:
						replacement = "0"
					}

				case "boolean":
					switch v := variable.Value.(type) {
					case bool:
						replacement = strconv.FormatBool(v)
					case string:
						if v == "true" {
							replacement = "true"
						} else if v == "false" {
							replacement = "false"
						} else {
							replacement = "false"
						}
					default:
						replacement = "false"
					}

				case "null":
					replacement = "null"

				case "string", "":
					var str string
					switch v := variable.Value.(type) {
					case string:
						str = v
					default:
						str = fmt.Sprintf("%v", v)
					}

					bytes, _ := json.Marshal(str)
					replacement = string(bytes)

				default:
					str := fmt.Sprintf("%v", variable.Value)
					bytes, _ := json.Marshal(str)
					replacement = string(bytes)
				}

				result = strings.Replace(result, placeholder, replacement, -1)
			} else {
				result = strings.Replace(result, placeholder, "\"\"", -1)
			}
		}
	}

	if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
		e.logger.Printf("Warning: JSON replacement resulted in invalid JSON, falling back to simple replacement")
		return e.replaceVarsInJsonString(jsonText, varMap)
	}

	return result
}

func (e *TestRunner) replaceVarsInJsonString(jsonText string, varMap map[string]models.Variable) string {
	res := jsonText
	re := regexp.MustCompile(`\${([A-Za-z0-9_]+)}`)
	matches := re.FindAllStringSubmatch(jsonText, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			placeholder := match[0]
			varName := match[1]

			if variable, ok := varMap[varName]; ok {
				var replacement string

				if variable.Value == nil {
					switch variable.Type {
					case "null":
						replacement = "null"
					case "number":
						replacement = "0"
					case "boolean":
						replacement = "false"
					default:
						replacement = ""
					}
				} else {
					switch v := variable.Value.(type) {
					case string:
						replacement = v
					case float64:
						replacement = strconv.FormatFloat(v, 'f', -1, 64)
					case int:
						replacement = strconv.Itoa(v)
					case bool:
						replacement = strconv.FormatBool(v)
					default:
						replacement = fmt.Sprintf("%v", v)
					}
				}

				res = strings.Replace(res, placeholder, replacement, -1)
			} else {
				res = strings.Replace(res, placeholder, "", -1)
			}
		}
	}

	return res
}
