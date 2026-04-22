package runner

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/test-fleet/test-runner/pkg/models"
)

func newTestRunner(t *testing.T, httpClient *http.Client) *TestRunner {
	t.Helper()
	return &TestRunner{
		logger:     log.New(io.Discard, "", 0),
		httpClient: httpClient,
	}
}

func TestNewTestRunner(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	r := NewTestRunner(logger, "test-runner")

	assert.NotNil(t, r)
	assert.Equal(t, logger, r.logger)
	assert.NotNil(t, r.httpClient)
}

func TestRun_SingleFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	r := newTestRunner(t, server.Client())

	job := &models.Job{
		JobID: "job-1",
		Scene: models.Scene{
			Timeout:   5000,
			Variables: map[string]models.Variable{},
		},
		Frames: []models.Frame{
			{
				Order: 1,
				Request: models.HTTPRequest{
					Method:  "GET",
					URL:     server.URL + "/api/test",
					Headers: map[string]string{},
					Timeout: 5000,
				},
				Extractors: []models.Extractors{},
			},
		},
	}

	result := r.Run(context.Background(), job)
	assert.Equal(t, "passed", result.Status)
}

func TestRun_FrameOrdering(t *testing.T) {
	var requestPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	r := newTestRunner(t, server.Client())

	job := &models.Job{
		JobID: "job-ordering",
		Scene: models.Scene{
			Timeout:   5000,
			Variables: map[string]models.Variable{},
		},
		// Frames deliberately provided out of order
		Frames: []models.Frame{
			{
				Order: 3,
				Request: models.HTTPRequest{
					Method:  "GET",
					URL:     server.URL + "/third",
					Timeout: 5000,
					Headers: map[string]string{},
				},
				Extractors: []models.Extractors{},
			},
			{
				Order: 1,
				Request: models.HTTPRequest{
					Method:  "GET",
					URL:     server.URL + "/first",
					Timeout: 5000,
					Headers: map[string]string{},
				},
				Extractors: []models.Extractors{},
			},
			{
				Order: 2,
				Request: models.HTTPRequest{
					Method:  "GET",
					URL:     server.URL + "/second",
					Timeout: 5000,
					Headers: map[string]string{},
				},
				Extractors: []models.Extractors{},
			},
		},
	}

	r.Run(context.Background(), job)

	assert.Equal(t, []string{"/first", "/second", "/third"}, requestPaths)
}

func TestRun_VariableSubstitutionAcrossFrames(t *testing.T) {
	var capturedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "abc123"}`))
			return
		}
		capturedToken = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	r := newTestRunner(t, server.Client())

	job := &models.Job{
		JobID: "job-var-sub",
		Scene: models.Scene{
			Timeout:   5000,
			Variables: map[string]models.Variable{},
		},
		Frames: []models.Frame{
			{
				Order: 1,
				Request: models.HTTPRequest{
					Method:  "POST",
					URL:     server.URL + "/login",
					Headers: map[string]string{},
					Timeout: 5000,
				},
				Extractors: []models.Extractors{
					{Name: "TOKEN", Type: "json", Source: "$.token", DataType: "string"},
				},
			},
			{
				Order: 2,
				Request: models.HTTPRequest{
					Method:  "GET",
					URL:     server.URL + "/protected",
					Headers: map[string]string{"Authorization": "Bearer ${TOKEN}"},
					Timeout: 5000,
				},
				Extractors: []models.Extractors{},
			},
		},
	}

	result := r.Run(context.Background(), job)
	assert.Equal(t, "passed", result.Status)
	assert.Equal(t, "Bearer abc123", capturedToken)
}

func TestRun_NoFrames(t *testing.T) {
	r := newTestRunner(t, &http.Client{})

	job := &models.Job{
		JobID: "job-empty",
		Scene: models.Scene{
			Timeout:   5000,
			Variables: map[string]models.Variable{},
		},
		Frames: []models.Frame{},
	}

	result := r.Run(context.Background(), job)
	assert.Equal(t, "passed", result.Status)
}

func TestExecuteFrame_UndefinedURLVariable(t *testing.T) {
	r := newTestRunner(t, &http.Client{})

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "GET",
			URL:     "https://example.com/${undefined_var}/resource",
			Timeout: 5000,
			Headers: map[string]string{},
		},
		Extractors: []models.Extractors{},
	}

	result := r.executeFrame(frame, map[string]models.Variable{}, context.Background())
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "undefined_var")
}

func TestExecuteFrame_UndefinedHeaderVariable(t *testing.T) {
	r := newTestRunner(t, &http.Client{})

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "GET",
			URL:     "https://example.com/resource",
			Timeout: 5000,
			Headers: map[string]string{
				"Authorization": "Bearer ${undefined_token}",
			},
		},
		Extractors: []models.Extractors{},
	}

	result := r.executeFrame(frame, map[string]models.Variable{}, context.Background())
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "undefined_token")
}

func TestExecuteFrame_UndefinedBodyVariable(t *testing.T) {
	r := newTestRunner(t, &http.Client{})

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "POST",
			URL:     "https://example.com/resource",
			Body:    `{"id": "${undefined_id}"}`,
			Timeout: 5000,
			Headers: map[string]string{},
		},
		Extractors: []models.Extractors{},
	}

	result := r.executeFrame(frame, map[string]models.Variable{}, context.Background())
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "undefined_id")
}

func TestExecuteFrame_WithVariableSubstitution(t *testing.T) {
	var capturedPath string
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	r := newTestRunner(t, server.Client())

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "GET",
			URL:     server.URL + "/users/${user_id}",
			Timeout: 5000,
			Headers: map[string]string{
				"Authorization": "Bearer ${token}",
			},
		},
		Extractors: []models.Extractors{},
	}

	vars := map[string]models.Variable{
		"user_id": {Value: "123", Type: "string"},
		"token":   {Value: "secret-token", Type: "string"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := r.executeFrame(frame, vars, ctx)
	assert.Equal(t, "passed", result.Status)
	assert.Empty(t, result.Error)
	assert.Equal(t, "/users/123", capturedPath)
	assert.Equal(t, "Bearer secret-token", capturedAuth)
}

func TestExecuteFrame_RequestBodySent(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	r := newTestRunner(t, server.Client())

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "POST",
			URL:     server.URL + "/api/users",
			Body:    `{"name": "${name}", "age": ${age}}`,
			Timeout: 5000,
			Headers: map[string]string{},
		},
		Extractors: []models.Extractors{},
	}

	vars := map[string]models.Variable{
		"name": {Value: "Alice", Type: "string"},
		"age":  {Value: float64(30), Type: "number"},
	}

	result := r.executeFrame(frame, vars, context.Background())
	assert.Equal(t, "passed", result.Status)
	assert.Contains(t, string(capturedBody), "Alice")
	assert.Contains(t, string(capturedBody), "30")
}

func TestExecuteFrame_HTTPRequestFailure(t *testing.T) {
	r := newTestRunner(t, &http.Client{Timeout: 1 * time.Millisecond})

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "GET",
			URL:     "http://192.0.2.1/unreachable", // TEST-NET, guaranteed unreachable
			Timeout: 1,
			Headers: map[string]string{},
		},
		Extractors: []models.Extractors{},
	}

	result := r.executeFrame(frame, map[string]models.Variable{}, context.Background())
	assert.Equal(t, "error", result.Status)
	assert.NotEmpty(t, result.Error)
}

func TestExecuteFrame_ExtractsVariablesFromResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"userId": 42, "name": "Bob"}`))
	}))
	defer server.Close()

	r := newTestRunner(t, server.Client())

	frame := models.Frame{
		Request: models.HTTPRequest{
			Method:  "GET",
			URL:     server.URL + "/api/user",
			Timeout: 5000,
			Headers: map[string]string{},
		},
		Extractors: []models.Extractors{
			{Name: "USER_ID", Type: "json", Source: "$.userId", DataType: "number"},
			{Name: "USER_NAME", Type: "json", Source: "$.name", DataType: "string"},
		},
	}

	vars := map[string]models.Variable{}
	result := r.executeFrame(frame, vars, context.Background())

	assert.Equal(t, "passed", result.Status)
	assert.Empty(t, result.Error)
	assert.Equal(t, float64(42), vars["USER_ID"].Value)
	assert.Equal(t, "number", vars["USER_ID"].Type)
	assert.Equal(t, "Bob", vars["USER_NAME"].Value)
}
