package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/test-fleet/test-runner/pkg/models"
)

type Runner interface {
	Run(ctx context.Context, job *models.Job) *models.SceneResult
}

type TestRunner struct {
	logger     *log.Logger
	httpClient *http.Client
	runnerName string
}

func NewTestRunner(logger *log.Logger, runnerName string) *TestRunner {
	client := &http.Client{}
	return &TestRunner{
		logger:     logger,
		httpClient: client,
		runnerName: runnerName,
	}
}

func (e *TestRunner) Run(ctx context.Context, job *models.Job) *models.SceneResult {
	scene := job.Scene
	sceneVars := scene.Variables
	sceneStart := time.Now()

	sceneCtx, sceneCancel := context.WithTimeout(context.Background(), time.Duration(scene.Timeout)*time.Millisecond)
	defer sceneCancel()

	frames := job.Frames
	sort.Slice(frames, func(i int, j int) bool {
		return frames[i].Order < frames[j].Order
	})

	frameResults := make([]models.FrameResult, 0, len(frames))
	sceneStatus := "passed"

	for _, frame := range frames {
		frameCtx, frameCancel := context.WithTimeout(sceneCtx, time.Duration(frame.Request.Timeout)*time.Millisecond)
		frameResult := e.executeFrame(frame, sceneVars, frameCtx, sceneCtx, scene.Timeout)
		frameCancel()

		frameResults = append(frameResults, frameResult)

		if frameResult.Status != "passed" {
			sceneStatus = frameResult.Status
			sceneCancel()
			break
		}
	}

	completedAt := time.Now()
	return &models.SceneResult{
		RunID:       job.RunID,
		JobID:       job.JobID,
		SceneID:     scene.ID,
		RunnerID:    e.runnerName,
		StartedAt:   sceneStart,
		CompletedAt: completedAt,
		DurationMs:  completedAt.Sub(sceneStart).Milliseconds(),
		Status:      sceneStatus,
		Frames:      frameResults,
	}
}

func (e *TestRunner) executeFrame(frame models.Frame, vars map[string]models.Variable, frameCtx, sceneCtx context.Context, sceneTimeoutMs int) models.FrameResult {
	frameStart := time.Now()

	result := models.FrameResult{
		FrameID: frame.ID,
		Name:    frame.Name,
		Order:   frame.Order,
	}

	varsInUrl := e.findVariableRefs(frame.Request.URL)
	for _, v := range varsInUrl {
		if _, exists := vars[v]; !exists {
			return e.errorFrame(result, frameStart, fmt.Errorf("variable ${%s} referenced in URL but not defined", v))
		}
	}
	processedUrl := e.replaceUrlVars(frame.Request.URL, vars)

	for _, headerVal := range frame.Request.Headers {
		varsInHeader := e.findVariableRefs(headerVal)
		for _, v := range varsInHeader {
			if _, exists := vars[v]; !exists {
				return e.errorFrame(result, frameStart, fmt.Errorf("variable ${%s} referenced in headers but not defined", v))
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
				return e.errorFrame(result, frameStart, fmt.Errorf("variable ${%s} referenced in request body but not defined", v))
			}
		}
	}
	processedBody := e.replaceJsonVars(frame.Request.Body, vars)

	result.Request = models.FrameRequest{
		Method:  frame.Request.Method,
		URL:     processedUrl,
		Headers: headers,
	}

	req, err := http.NewRequest(frame.Request.Method, processedUrl, strings.NewReader(processedBody))
	if err != nil {
		return e.errorFrame(result, frameStart, fmt.Errorf("failed to create http request: %w", err))
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	reqStart := time.Now()
	res, err := e.sendHttpRequest(req, frameCtx)
	responseDurationMs := time.Since(reqStart).Milliseconds()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			if sceneCtx.Err() != nil {
				return e.errorFrame(result, frameStart, fmt.Errorf("scene timeout of %dms exceeded (frame ran for %dms)", sceneTimeoutMs, responseDurationMs))
			}
			return e.errorFrame(result, frameStart, fmt.Errorf("frame timeout of %dms exceeded (request took %dms)", frame.Request.Timeout, responseDurationMs))
		}
		return e.errorFrame(result, frameStart, fmt.Errorf("http request failed: %w", err))
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return e.errorFrame(result, frameStart, fmt.Errorf("failed to read response body: %w", err))
	}
	res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	result.Response = models.FrameResponse{
		StatusCode: res.StatusCode,
		Headers:    flattenHeaders(res.Header),
		BodySize:   len(bodyBytes),
		DurationMs: responseDurationMs,
	}

	if err := e.extractVariables(res, frame.Extractors, vars); err != nil {
		return e.errorFrame(result, frameStart, err)
	}

	assertionResults := e.validateAssertions(res, bodyBytes, frame.Assertions)
	result.Assertions = assertionResults

	frameStatus := "passed"
	for _, ar := range assertionResults {
		if !ar.Passed {
			frameStatus = "failed"
			break
		}
	}

	completedAt := time.Now()
	result.Status = frameStatus
	result.StartedAt = frameStart
	result.CompletedAt = completedAt
	result.DurationMs = completedAt.Sub(frameStart).Milliseconds()
	return result
}

func (e *TestRunner) errorFrame(result models.FrameResult, start time.Time, err error) models.FrameResult {
	completedAt := time.Now()
	result.Status = "error"
	result.Error = err.Error()
	result.StartedAt = start
	result.CompletedAt = completedAt
	result.DurationMs = completedAt.Sub(start).Milliseconds()
	return result
}

func flattenHeaders(h http.Header) map[string]string {
	flat := make(map[string]string, len(h))
	for key, values := range h {
		if len(values) > 0 {
			flat[key] = values[0]
		}
	}
	return flat
}
