package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type GauRunner struct{}

func (r *GauRunner) Name() string {
	return "gau"
}

func (r *GauRunner) IsInstalled() bool {
	_, err := exec.LookPath("gau")
	return err == nil
}

func (r *GauRunner) BuildCommand(opts RunOptions) ([]string, error) {
	if opts.InputFile == "" {
		return nil, fmt.Errorf("input file is required")
	}

	args := []string{"gau"}

	args = append(args, opts.InputFile)

	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(opts.OutputDir, "urls.txt")
	}
	args = append(args, "--o", opts.OutputFile)

	if opts.Concurrency > 0 {
		args = append(args, "--threads", strconv.Itoa(opts.Concurrency))
	}

	return args, nil
}

func (r *GauRunner) ParseOutput(data []byte) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "http") && !seen[line] {
			seen[line] = true
			results = append(results, line)
		}
	}

	return results, nil
}
