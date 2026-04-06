package runner

import (
	"context"
	"net/http"
)

func (e *TestRunner) sendHttpRequest(req *http.Request, frameCtx context.Context) (*http.Response, error) {
	resp, err := e.httpClient.Do(req.WithContext(frameCtx))
	if err != nil {
		return nil, err
	}

	// err = e.printResponse(resp)
	// if err != nil {
	// 	e.logger.Println("err: failed to print response")
	// }
	return resp, nil
}

// func (e *TestRunner) printResponse(resp *http.Response) error {
// 	if resp == nil {
// 		return fmt.Errorf("response is nil")
// 	}

// 	e.logger.Println("************RES************")
// 	e.logger.Printf("Status: %s (%d)\n", resp.Status, resp.StatusCode)

// 	hiddenHeaders := map[string]bool{
// 		"age":                              true,
// 		"alt-svc":                          true,
// 		"cache-control":                    true,
// 		"cf-cache-status":                  true,
// 		"cf-ray":                           true,
// 		"connection":                       true,
// 		"date":                             true,
// 		"etag":                             true,
// 		"expires":                          true,
// 		"keep-alive":                       true,
// 		"nel":                              true,
// 		"pragma":                           true,
// 		"report-to":                        true,
// 		"reporting-endpoints":              true,
// 		"server":                           true,
// 		"server-timing":                    true,
// 		"via":                              true,
// 		"vary":                             true,
// 		"x-content-type-options":           true,
// 		"x-frame-options":                  true,
// 		"x-powered-by":                     true,
// 		"x-ratelimit-limit":                true,
// 		"x-ratelimit-remaining":            true,
// 		"x-ratelimit-reset":                true,
// 		"access-control-allow-origin":      true,
// 		"access-control-allow-methods":     true,
// 		"access-control-allow-headers":     true,
// 		"access-control-allow-credentials": true,
// 	}

// 	e.logger.Println("Headers:")
// 	headerCount := 0

// 	for key, values := range resp.Header {
// 		if !hiddenHeaders[strings.ToLower(key)] {
// 			e.logger.Printf("  %s: %s\n", key, values[0])
// 			headerCount++
// 		}
// 	}

// 	if headerCount == 0 {
// 		e.logger.Println("  <Only standard headers present - hidden>")
// 	}

// 	if resp.Body != nil {
// 		bodyBytes, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			return fmt.Errorf("failed to read response body: %w", err)
// 		}
// 		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

// 		e.logger.Println("Body:")
// 		e.logger.Println(string(bodyBytes))
// 	} else {
// 		e.logger.Println("Body: <nil>")
// 	}

// 	e.logger.Println("**********END RES*********")
// 	return nil
// }

// func (e *TestRunner) printRequest(req *http.Request) {
// 	e.logger.Println("------------HTTP REQ------------")
// 	e.logger.Printf("%s %s\n", req.Method, req.URL.String())

// 	// Define headers to hide
// 	hiddenHeaders := map[string]bool{
// 		"accept":                    true,
// 		"accept-encoding":           true,
// 		"accept-language":           true,
// 		"cache-control":             true,
// 		"connection":                true,
// 		"cookie":                    true,
// 		"dnt":                       true,
// 		"host":                      true,
// 		"origin":                    true,
// 		"pragma":                    true,
// 		"referer":                   true,
// 		"sec-fetch-dest":            true,
// 		"sec-fetch-mode":            true,
// 		"sec-fetch-site":            true,
// 		"sec-fetch-user":            true,
// 		"user-agent":                true,
// 		"upgrade-insecure-requests": true,
// 	}

// 	e.logger.Println("HEADERS:")
// 	headerCount := 0

// 	for key, values := range req.Header {
// 		if !hiddenHeaders[strings.ToLower(key)] {
// 			e.logger.Printf("%s: %s\n", key, values[0])
// 			headerCount++
// 		}
// 	}

// 	if headerCount == 0 {
// 		e.logger.Println("<Only standard headers present - hidden>")
// 	}

// 	if req.Body != nil {
// 		bodyBytes, _ := io.ReadAll(req.Body)
// 		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

// 		e.logger.Println("BODY:")
// 		e.logger.Println(string(bodyBytes))
// 	} else {
// 		e.logger.Println("BODY:")
// 	}

// 	e.logger.Println("-----------REQ END------------")
// }
