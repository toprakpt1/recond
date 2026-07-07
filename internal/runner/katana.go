package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type KatanaRunner struct{}

func (r *KatanaRunner) Name() string {
	return "katana"
}

func (r *KatanaRunner) IsInstalled() bool {
	_, err := exec.LookPath("katana")
	return err == nil
}

func (r *KatanaRunner) BuildCommand(opts RunOptions) ([]string, error) {
	if opts.InputFile == "" {
		return nil, fmt.Errorf("input file is required")
	}

	args := []string{"katana"}

	args = append(args, "-list", opts.InputFile)

	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(opts.OutputDir, "crawled.txt")
	}
	args = append(args, "-o", opts.OutputFile)

	args = append(args, "-silent")

	if opts.Concurrency > 0 {
		args = append(args, "-c", strconv.Itoa(opts.Concurrency))
	}

	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	args = append(args, "-d", "3", "-jc", "-ef", "css,png,jpg,gif,svg,woff,woff2,ico")

	return args, nil
}

func (r *KatanaRunner) ParseOutput(data []byte) ([]string, error) {
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
