package models

import "time"

type OutputKind string

const (
	OutputSubdomains  OutputKind = "subdomains"
	OutputAliveHosts  OutputKind = "alive"
	OutputURLs        OutputKind = "urls"
	OutputDirectories OutputKind = "directories"
	OutputResults     OutputKind = "results"
)

type Output struct {
	ID        string     `json:"id"`
	JobID     string     `json:"job_id"`
	StepID    string     `json:"step_id"`
	Path      string     `json:"path"`
	Kind      OutputKind `json:"kind"`
	SizeBytes int64      `json:"size_bytes,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
