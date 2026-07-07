package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type SubfinderRunner struct{}

func (r *SubfinderRunner) Name() string {
	return "subfinder"
}

func (r *SubfinderRunner) IsInstalled() bool {
	_, err := exec.LookPath("subfinder")
	return err == nil
}

func (r *SubfinderRunner) BuildCommand(opts RunOptions) ([]string, error) {
	if opts.Target == "" {
		return nil, fmt.Errorf("target domain is required")
	}

	args := []string{"subfinder"}

	args = append(args, "-d", opts.Target)

	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(opts.OutputDir, "subdomains.txt")
	}
	args = append(args, "-o", opts.OutputFile)

	if opts.RateLimit > 0 {
		args = append(args, "-rl", strconv.Itoa(opts.RateLimit))
	}

	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	args = append(args, "-silent")

	return args, nil
}

func (r *SubfinderRunner) ParseOutput(data []byte) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !seen[line] {
			seen[line] = true
			results = append(results, line)
		}
	}

	return results, nil
}
