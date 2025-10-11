package models

import "time"

type Scene struct {
	ID           string            `json:"_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	CronSchedule string            `json:"cronSchedule"`
	Enabled      bool              `json:"enabled"`
	Variables    map[string]string `json:"variables"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type SceneResult struct {
	ID                 string            `json:"_id,omitempty"`
	SceneID            string            `json:"sceneId"`
	RunID              string            `json:"runId"`
	ExecutedAt         time.Time         `json:"executedAt"`
	Duration           int64             `json:"duration"` // milliseconds
	Region             string            `json:"region"`
	RunnerID           string            `json:"runnerId"`
	Status             string            `json:"status"` // "success", "failure", "error"
	TotalFrames        int               `json:"totalFrames"`
	FailedFrames       int               `json:"failedFrames"`
	CollectedVariables map[string]string `json:"collectedVariables"`
	CreatedAt          time.Time         `json:"createdAt,omitempty"`
	UpdatedAt          time.Time         `json:"updatedAt,omitempty"`
}
