package models

import "time"

type Frame struct {
	ID         string       `json:"frameId"`
	SceneID    string       `json:"sceneId"`
	Name       string       `json:"name"`
	Order      int          `json:"order"`
	Enabled    bool         `json:"enabled"`
	Request    HTTPRequest  `json:"request"`
	Extractors []Extractors `json:"extractors"`
	Assertions []Assertion  `json:"assertions"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}

type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Timeout int               `json:"timeout"`
}

type Extractors struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Source   string `json:"source"`
	DataType string `json:"dataType"`
}

type Assertion struct {
	Type     string      `json:"type"`
	Operator string      `json:"operator"`
	Expected interface{} `json:"expected"`
	Source   string      `json:"source"`
}
