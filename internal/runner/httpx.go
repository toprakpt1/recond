package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type HttpxRunner struct{}

func (r *HttpxRunner) Name() string {
	return "httpx"
}

func (r *HttpxRunner) IsInstalled() bool {
	_, err := exec.LookPath("httpx")
	return err == nil
}

func (r *HttpxRunner) BuildCommand(opts RunOptions) ([]string, error) {
	if opts.InputFile == "" {
		return nil, fmt.Errorf("input file is required")
	}

	args := []string{"httpx"}

	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(opts.OutputDir, "alive.txt")
	}

	args = append(args, "-l", opts.InputFile)
	args = append(args, "-o", opts.OutputFile)
	args = append(args, "-silent")

	if opts.Concurrency > 0 {
		args = append(args, "-threads", strconv.Itoa(opts.Concurrency))
	}

	if opts.RateLimit > 0 {
		args = append(args, "-rl", strconv.Itoa(opts.RateLimit))
	}

	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	if opts.IsResume {
		args = append(args, "-resume")
	}

	args = append(args, "-follow-redirects", "-status-code", "-content-length")

	return args, nil
}

func (r *HttpxRunner) ParseOutput(data []byte) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "[") || strings.Contains(line, "http") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				url := parts[0]
				if !seen[url] {
					seen[url] = true
					results = append(results, url)
				}
			}
		}
	}

	return results, nil
}
