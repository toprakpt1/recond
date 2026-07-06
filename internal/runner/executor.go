package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Executor struct {
	runner      Runner
	input       string
	outputPath  string
	opts        RunOptions
	outputLines []string
	mu          sync.Mutex
	processed   int
	total       int
}

func NewExecutor(runner Runner, input string, outputPath string, opts RunOptions) *Executor {
	return &Executor{
		runner:     runner,
		input:      input,
		outputPath: outputPath,
		opts:       opts,
	}
}

func (e *Executor) Run(ctx context.Context) (*StepResult, error) {
	cmd := e.runner.BuildCommand(e.input, e.outputPath, e.opts)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", e.runner.Name(), err)
	}

	procDone := make(chan error, 1)
	go func() {
		procDone <- cmd.Wait()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go e.readLines(stdout, &wg)
	go e.drainStderr(stderr, &wg)

	go func() {
		wg.Wait()
	}()

	progressTicker := time.NewTicker(2 * time.Second)
	defer progressTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			cmd.Process.Signal(os.Interrupt)
			go func() {
				select {
				case <-procDone:
				case <-time.After(5 * time.Second):
					cmd.Process.Kill()
				}
			}()
			return nil, ctx.Err()

		case err := <-procDone:
			result := &StepResult{
				Items:      e.outputLines,
				OutputPath: e.outputPath,
				Checkpoint: e.buildCheckpoint(),
			}
			if err != nil {
				result.Error = err
			}

			if data, readErr := os.ReadFile(e.outputPath); readErr == nil {
				if parsed, parseErr := e.runner.ParseOutput(data); parseErr == nil {
					result.Items = parsed
				}
			}

			return result, nil

		case <-progressTicker.C:
			e.mu.Lock()
			if e.processed > e.total {
				e.total = e.processed
			}
			e.mu.Unlock()
		}
	}
}

func (e *Executor) readLines(reader io.ReadCloser, wg *sync.WaitGroup) {
	defer wg.Done()
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		e.mu.Lock()
		e.outputLines = append(e.outputLines, line)
		e.processed++
		e.mu.Unlock()
	}
}

func (e *Executor) drainStderr(reader io.ReadCloser, wg *sync.WaitGroup) {
	defer wg.Done()
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		_ = scanner.Text()
	}
}

func (e *Executor) ProcessedCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.processed
}

func (e *Executor) TotalCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.total < e.processed {
		return e.processed
	}
	return e.total
}

func (e *Executor) buildCheckpoint() CheckpointData {
	e.mu.Lock()
	defer e.mu.Unlock()
	return CheckpointData{
		ProcessedCount: e.processed,
		TotalCount:     e.total,
		ProcessedItems: e.outputLines,
		Position:       len(e.outputLines),
	}
}
