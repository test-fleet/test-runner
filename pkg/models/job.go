package models

type Job struct {
	JobID     string  `json:"jobId"`
	Type      string  `json:"type"`
	RunID     string  `json:"runId"`
	Scene     Scene   `json:"scene"`
	Frames    []Frame `json:"frames"`
	CreatedAt string  `json:"createdAt"`
}
