package runner

import (
	"time"
)

type Runner interface {
	Name() string
	IsInstalled() bool
	BuildCommand(opts RunOptions) ([]string, error)
	ParseOutput(data []byte) ([]string, error)
}

type RunOptions struct {
	Target      string
	InputFile   string
	OutputFile  string
	OutputDir   string
	Concurrency int
	RateLimit   int
	Timeout     time.Duration
	Wordlist    string
	IsResume    bool
}

type StepResult struct {
	Items      []string
	OutputPath string
	Err        error
}
