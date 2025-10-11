package models

import "time"

type Frame struct {
	ID               string            `json:"_id"`
	SceneID          string            `json:"sceneId"`
	Name             string            `json:"name"`
	Order            int               `json:"order"`
	Enabled          bool              `json:"enabled"`
	Request          HTTPRequest       `json:"request"`
	ExtractVariables []VariableExtract `json:"extractVariables"`
	Assertions       []Assertion       `json:"assertions"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Timeout int               `json:"timeout"`
}

type VariableExtract struct {
	Name       string `json:"name"`
	Source     string `json:"source"`
	JSONPath   string `json:"jsonPath"`
	HeaderName string `json:"headerName"`
	Required   bool   `json:"required"`
}

type Assertion struct {
	Type       string      `json:"type"`
	Operator   string      `json:"operator"`
	Expected   interface{} `json:"expected"`
	Path       string      `json:"path"`
	HeaderName string      `json:"headerName"`
}
