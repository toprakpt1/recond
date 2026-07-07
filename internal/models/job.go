package models

import "time"

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusStopped   JobStatus = "stopped"
)

type Job struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Target     string    `json:"target"`
	Status     JobStatus `json:"status"`
	Profile    string    `json:"profile"`
	Debug      bool      `json:"debug"`
	ConfigJSON string    `json:"config_json,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
