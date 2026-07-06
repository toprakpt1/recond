package daemon

import (
	"encoding/json"
)

type Request struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type StartRequest struct {
	Target  string `json:"target"`
	Profile string `json:"profile,omitempty"`
}

type ActionRequest struct {
	JobID string `json:"job_id"`
}
