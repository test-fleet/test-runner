package runner

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/test-fleet/test-runner/pkg/models"
)

type Runner interface {
	Run(ctx context.Context, job string) bool // should return result type
}

type TestRunner struct {
	logger     *log.Logger
	httpClient *http.Client
}

func NewTestRunner(logger *log.Logger) *TestRunner {
	client := &http.Client{}
	return &TestRunner{
		logger:     logger,
		httpClient: client,
	}
}

func (e *TestRunner) Run(ctx context.Context, job *models.Job) bool { // TODO: Add check on done chan to return timeout err
	scene := job.Scene
	sceneVars := scene.Variables

	sceneCtx, sceneCancel := context.WithTimeout(context.Background(), time.Duration(scene.Timeout)*time.Millisecond)
	defer sceneCancel()

	frames := job.Frames
	sort.Slice(frames, func(i int, j int) bool {
		return frames[i].Order < frames[j].Order
	})

	for _, frame := range frames {
		frameCtx, frameCancel := context.WithTimeout(sceneCtx, time.Duration(frame.Request.Timeout)*time.Millisecond)
		pass, err := e.executeFrame(frame, sceneVars, frameCtx)

		frameCancel()

		if !pass || err != nil {
			//!  send failed scene to res chan
			e.logger.Println("frame failed, closing scene")
			sceneCancel()
		}
	}

	return true
}

func (e *TestRunner) executeFrame(frame models.Frame, vars map[string]models.Variable, frameCtx context.Context) (bool, error) {
	varsInUrl := e.findVariableRefs(frame.Request.URL)
	for _, v := range varsInUrl {
		if _, exists := vars[v]; !exists {
			return false, fmt.Errorf("variable ${%s} referenced in URL but not defined", v)
		}
	}
	processedUrl := e.replaceUrlVars(frame.Request.URL, vars)

	for _, headerVal := range frame.Request.Headers {
		varsInHeader := e.findVariableRefs(headerVal)
		for _, v := range varsInHeader {
			if _, exists := vars[v]; !exists {
				return false, fmt.Errorf("variable ${%s} referenced in headers but not defined", v)
			}
		}
	}
	headers := make(map[string]string)
	for key, value := range frame.Request.Headers {
		headers[key] = value
	}
	e.ReplaceHeaderVars(headers, vars)

	if frame.Request.Body != "" {
		varsInBody := e.findVariableRefs(frame.Request.Body)
		for _, v := range varsInBody {
			if _, exists := vars[v]; !exists {
				return false, fmt.Errorf("variable ${%s} referenced in request body but not defined", v)
			}
		}
	}
	processedBody := e.replaceJsonVars(frame.Request.Body, vars)

	req, err := http.NewRequest(
		frame.Request.Method,
		processedUrl,
		strings.NewReader(processedBody),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create new http request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res, err := e.sendHttpRequest(req, frameCtx)
	if err != nil {
		return false, fmt.Errorf("http request err") // fail test
	}

	e.extractVariables(res, frame.Extractors, vars)

	return true, nil
}
