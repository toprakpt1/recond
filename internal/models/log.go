package models

import "time"

type LogLevel string

const (
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
	LogDebug LogLevel = "debug"
)

type Log struct {
	ID        int64     `json:"id"`
	JobID     string    `json:"job_id"`
	StepID    string    `json:"step_id,omitempty"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
