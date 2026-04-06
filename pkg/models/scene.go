package models

type Scene struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Variables    map[string]Variable `json:"variables"`
	FrameIDs     []string            `json:"frameIds"`
	Timeout      int                 `json:"timeout"`
	OrgID        string              `json:"orgId"`
	CronSchedule string              `json:"cronSchedule"`
	Enabled      bool                `json:"enabled"`
}

type Variable struct {
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}
