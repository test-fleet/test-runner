package models

import "time"

type SceneResult struct {
	RunID       string        `json:"runId"`
	JobID       string        `json:"jobId"`
	SceneID     string        `json:"sceneId"`
	RunnerID    string        `json:"runnerId"`
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt time.Time     `json:"completedAt"`
	DurationMs  int64         `json:"durationMs"`
	Status      string        `json:"status"` // "passed" | "failed" | "error"
	Frames      []FrameResult `json:"frames"`
}

type FrameResult struct {
	FrameID     string            `json:"frameId"`
	Name        string            `json:"name"`
	Order       int               `json:"order"`
	StartedAt   time.Time         `json:"startedAt"`
	CompletedAt time.Time         `json:"completedAt"`
	DurationMs  int64             `json:"durationMs"`
	Status      string            `json:"status"` // "passed" | "failed" | "error"
	Request     FrameRequest      `json:"request"`
	Response    FrameResponse     `json:"response"`
	Assertions  []AssertionResult `json:"assertions"`
	Error       string            `json:"error,omitempty"` // network error, timeout, etc
}

type FrameRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"` // post-variable-substitution
	Headers map[string]string `json:"headers"`
}

type FrameResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	BodySize   int               `json:"bodySize"` // bytes, not the full body
	DurationMs int64             `json:"durationMs"`
}

type AssertionResult struct {
	Type     string      `json:"type"`
	Operator string      `json:"operator"`
	Source   string      `json:"source"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Passed   bool        `json:"passed"`
}
