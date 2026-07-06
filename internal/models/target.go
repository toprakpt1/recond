package models

type TargetType string

const (
	TargetDomain    TargetType = "domain"
	TargetSubdomain TargetType = "subdomain"
	TargetURL       TargetType = "url"
)

type TargetStatus string

const (
	TargetPending   TargetStatus = "pending"
	TargetProcessed TargetStatus = "processed"
	TargetFailed    TargetStatus = "failed"
)

type Target struct {
	ID     string       `json:"id"`
	JobID  string       `json:"job_id"`
	Value  string       `json:"value"`
	Type   TargetType   `json:"type"`
	Status TargetStatus `json:"status"`
}
