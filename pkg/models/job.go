package models

type Job struct {
	JobID   string  `json:"jobId"`
	SceneID string  `json:"sceneId"`
	RunID   string  `json:"runId"`
	Scene   Scene   `json:"scene"`
	Frames  []Frame `json:"frames"`
}
