package models

import "time"

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusPaused    StepStatus = "paused"
)

type Step struct {
	ID             string     `json:"id"`
	JobID          string     `json:"job_id"`
	Name           string     `json:"name"`
	Tool           string     `json:"tool"`
	Order          int        `json:"order"`
	Status         StepStatus `json:"status"`
	Progress       float64    `json:"progress"`
	TotalItems     int        `json:"total_items"`
	ProcessedItems int        `json:"processed_items"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	CheckpointJSON string     `json:"checkpoint_json,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type StepDef struct {
	Name  string `json:"name"`
	Tool  string `json:"tool"`
	Order int    `json:"order"`
}
