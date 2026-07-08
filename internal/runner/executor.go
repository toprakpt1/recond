package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Executor struct {
	runner     Runner
	opts       RunOptions
	cmd        *exec.Cmd
	outputFile string
	stderr     bytes.Buffer
	processed  int64
	total      int64
	mu         sync.Mutex
	onDebug    func(string)
}

func NewExecutor(runner Runner, opts RunOptions) *Executor {
	return &Executor{
		runner: runner,
		opts:   opts,
	}
}

func (e *Executor) OnDebug(fn func(string)) {
	e.onDebug = fn
}

func (e *Executor) debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
	if e.onDebug != nil {
		e.onDebug(msg)
	}
}

func (e *Executor) Run(ctx context.Context) (*StepResult, error) {
	return e.RunWithRetry(ctx, 3, 2*time.Second)
}

func (e *Executor) RunWithRetry(ctx context.Context, maxRetries int, backoff time.Duration) (*StepResult, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			expBackoff := backoff * time.Duration(1<<uint(attempt-1))
			if expBackoff > 30*time.Second {
				expBackoff = 30 * time.Second
			}

			e.debugf("[executor] retry %d/%d after %v", attempt, maxRetries, expBackoff)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(expBackoff):
			}
		}

		result, err := e.runOnce(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err
		e.debugf("[executor] attempt %d failed: %v", attempt+1, err)
	}

	return nil, fmt.Errorf("all %d retries failed: %w", maxRetries+1, lastErr)
}

func (e *Executor) runOnce(ctx context.Context) (*StepResult, error) {
	args, err := e.runner.BuildCommand(e.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	e.outputFile = e.opts.OutputFile
	if err := os.MkdirAll(filepath.Dir(e.outputFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	outFile, err := os.Create(e.outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	e.debugf("[executor] running: %s", strings.Join(args, " "))

	e.cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	e.cmd.Stdout = outFile
	e.cmd.Stderr = &e.stderr
	setupSysProcAttr(e.cmd)

	if err := e.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- e.cmd.Wait()
	}()

	var execErr error
	select {
	case <-ctx.Done():
		e.debugf("[executor] context cancelled, killing process %d", e.cmd.Process.Pid)
		sendTermSignal(e.cmd.Process.Pid)
		select {
		case <-time.After(5 * time.Second):
			sendKillSignal(e.cmd.Process.Pid)
		case <-done:
		}
		execErr = ctx.Err()
	case execErr = <-done:
	}

	if execErr != nil {
		return nil, fmt.Errorf("command failed: %w\nstderr: %s", execErr, e.stderr.String())
	}

	data, err := os.ReadFile(e.outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	items, err := e.runner.ParseOutput(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	e.processed = int64(len(items))

	e.debugf("[executor] completed: %d items, output: %s", len(items), e.outputFile)

	return &StepResult{
		Items:      items,
		OutputPath: e.outputFile,
	}, nil
}

func (e *Executor) GetProgress() (processed int64, total int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.processed, e.total
}

func (e *Executor) OutputFile() string {
	return e.outputFile
}

func ReadInputLines(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func CountLines(filePath string) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	count := int64(0)
	lineSep := []byte{'\n'}

	for {
		n, err := f.Read(buf)
		count += int64(bytes.Count(buf[:n], lineSep))
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}
	}

	return count, nil
}

func WriteInputFile(filePath string, lines []string) error {
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}
