package runner

import (
	"os/exec"
	"time"
)

type Runner interface {
	Name() string
	IsInstalled() bool
	BuildCommand(input string, outputPath string, opts RunOptions) *exec.Cmd
	ParseOutput(data []byte) ([]string, error)
	GetCheckpoint() CheckpointData
	ResumeFromCheckpoint(cp CheckpointData) error
}

type RunOptions struct {
	Concurrency int
	RateLimit   int
	Timeout     time.Duration
	OutputDir   string
	Profile     string
	Wordlist    string
}

type CheckpointData struct {
	ProcessedCount int      `json:"processed_count"`
	TotalCount     int      `json:"total_count"`
	ProcessedItems []string `json:"processed_items,omitempty"`
	Position       int      `json:"position"`
	Data           string   `json:"data,omitempty"`
}

type StepResult struct {
	Items      []string     `json:"items"`
	OutputPath string       `json:"output_path"`
	Error      error        `json:"-"`
	Checkpoint CheckpointData `json:"checkpoint"`
}

func IsToolInstalled(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
