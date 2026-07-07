package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/recond/internal/config"
	"github.com/recond/internal/models"
	"github.com/recond/internal/runner"
	"github.com/recond/internal/storage"
)

var DefaultSteps = []models.StepDef{
	{Name: "subdomain-discovery", Tool: "subfinder", Order: 1},
	{Name: "alive-check", Tool: "httpx", Order: 2},
	{Name: "crawling", Tool: "katana", Order: 3},
	{Name: "url-collection", Tool: "gau", Order: 4},
	{Name: "directory-fuzzing", Tool: "ffuf", Order: 5},
}

func CreateSteps(jobID string) []models.Step {
	now := time.Now()
	steps := make([]models.Step, len(DefaultSteps))
	for i, def := range DefaultSteps {
		steps[i] = models.Step{
			ID:        fmt.Sprintf("step-%s-%s", jobID[len(jobID)-8:], def.Name[:8]),
			JobID:     jobID,
			Name:      def.Name,
			Tool:      def.Tool,
			Order:     def.Order,
			Status:    models.StepStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return steps
}

type Pipeline struct {
	store     *storage.Storage
	jobID     string
	registry  *runner.Registry
	outputDir string
	profile   config.Profile
}

func NewPipeline(store *storage.Storage, jobID string, profile config.Profile) *Pipeline {
	home, _ := os.UserHomeDir()
	outputDir := filepath.Join(home, ".recond", "jobs", jobID)

	return &Pipeline{
		store:     store,
		jobID:     jobID,
		registry:  runner.NewRegistry(),
		outputDir: outputDir,
		profile:   profile,
	}
}

func (p *Pipeline) Execute(ctx context.Context) error {
	os.MkdirAll(p.outputDir, 0755)

	job, err := p.store.GetJob(ctx, p.jobID)
	if err != nil {
		return err
	}

	steps, err := p.store.ListSteps(ctx, p.jobID)
	if err != nil {
		return err
	}

	log.Printf("[pipeline] starting job %s with %d steps (profile: %s)", p.jobID, len(steps), p.profile.Name)
	log.Printf("[pipeline] concurrency=%d rate_limit=%d timeout=%v cpu_max=%d ram_max=%s",
		p.profile.Concurrency, p.profile.RateLimit, p.profile.Timeout, p.profile.CPUMax, p.profile.RAMMax)

	for _, step := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if step.Status == models.StepStatusCompleted || step.Status == models.StepStatusSkipped {
			log.Printf("[pipeline] skipping completed step: %s", step.Name)
			continue
		}

		log.Printf("[pipeline] starting step: %s (tool: %s)", step.Name, step.Tool)

		if step.Status != models.StepStatusRunning {
			p.store.UpdateStepStatus(ctx, step.ID, models.StepStatusRunning)
			p.store.UpdateStepCheckpoint(ctx, step.ID, "")
		}

		if err := p.executeStep(ctx, job, step); err != nil {
			if err == context.Canceled {
				p.store.UpdateStepStatus(ctx, step.ID, models.StepStatusPaused)
				return err
			}

			p.store.CompleteStep(ctx, step.ID, models.StepStatusFailed, err.Error())
			p.store.InsertLog(ctx, models.Log{
				JobID:   p.jobID,
				StepID:  step.ID,
				Level:   models.LogError,
				Message: fmt.Sprintf("step %s failed: %v", step.Name, err),
			})
			return err
		}

		p.store.CompleteStep(ctx, step.ID, models.StepStatusCompleted, "")
		p.store.InsertLog(ctx, models.Log{
			JobID:   p.jobID,
			StepID:  step.ID,
			Level:   models.LogInfo,
			Message: fmt.Sprintf("step %s completed", step.Name),
		})
		log.Printf("[pipeline] completed step: %s", step.Name)
	}

	log.Printf("[pipeline] job %s completed successfully", p.jobID)
	return nil
}

func (p *Pipeline) executeStep(ctx context.Context, job *models.Job, step models.Step) error {
	r, err := p.registry.Get(step.Tool)
	if err != nil {
		return err
	}

	if !r.IsInstalled() {
		return fmt.Errorf("tool %s is not installed. install it and try again", step.Tool)
	}

	inputFile, err := p.getStepInput(step)
	if err != nil {
		return err
	}

	outputFile := filepath.Join(p.outputDir, step.Tool+".txt")
	if step.Tool == "ffuf" {
		outputFile = filepath.Join(p.outputDir, "directories.json")
	}

	opts := runner.RunOptions{
		Target:      job.Target,
		InputFile:   inputFile,
		OutputFile:  outputFile,
		OutputDir:   p.outputDir,
		Concurrency: p.profile.Concurrency,
		RateLimit:   p.profile.RateLimit,
		Timeout:     p.profile.Timeout,
	}

	executor := runner.NewExecutor(r, opts)

	p.store.InsertLog(ctx, models.Log{
		JobID:   p.jobID,
		StepID:  step.ID,
		Level:   models.LogInfo,
		Message: fmt.Sprintf("executing %s (concurrency=%d, rate_limit=%d, timeout=%v)",
			step.Tool, opts.Concurrency, opts.RateLimit, opts.Timeout),
	})

	progressCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-progressCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				processed, total := executor.GetProgress()
				var progress float64
				if total > 0 {
					progress = float64(processed) / float64(total) * 100
				}
				p.store.UpdateStepProgress(ctx, step.ID, progress, int(processed), int(total))
			}
		}
	}()

	result, err := executor.Run(ctx)
	close(progressCh)

	if err != nil {
		return err
	}

	p.store.UpdateStepProgress(ctx, step.ID, 100, len(result.Items), len(result.Items))

	outputKind := models.OutputResults
	switch step.Tool {
	case "subfinder":
		outputKind = models.OutputSubdomains
	case "httpx":
		outputKind = models.OutputAliveHosts
	case "katana", "gau":
		outputKind = models.OutputURLs
	case "ffuf":
		outputKind = models.OutputDirectories
	}

	p.store.CreateOutput(ctx, models.Output{
		ID:        fmt.Sprintf("out-%s-%s", step.ID[:8], step.Tool),
		JobID:     p.jobID,
		StepID:    step.ID,
		Path:      result.OutputPath,
		Kind:      outputKind,
		SizeBytes: int64(len(result.Items)),
		CreatedAt: time.Now(),
	})

	cpData := map[string]interface{}{
		"step":         step.Name,
		"tool":         step.Tool,
		"items_found":  len(result.Items),
		"completed_at": time.Now().Format(time.RFC3339),
	}
	cpJSON, _ := json.Marshal(cpData)
	p.store.UpdateStepCheckpoint(ctx, step.ID, string(cpJSON))

	p.store.InsertLog(ctx, models.Log{
		JobID:   p.jobID,
		StepID:  step.ID,
		Level:   models.LogInfo,
		Message: fmt.Sprintf("found %d items in %s", len(result.Items), step.Tool),
	})

	return nil
}

func (p *Pipeline) getStepInput(step models.Step) (string, error) {
	switch step.Tool {
	case "subfinder":
		return "", nil
	case "httpx":
		prev := filepath.Join(p.outputDir, "subfinder.txt")
		if _, err := os.Stat(prev); os.IsNotExist(err) {
			return "", fmt.Errorf("subfinder output not found: %s", prev)
		}
		return prev, nil
	case "katana":
		prev := filepath.Join(p.outputDir, "httpx.txt")
		if _, err := os.Stat(prev); os.IsNotExist(err) {
			return "", fmt.Errorf("httpx output not found: %s", prev)
		}
		return prev, nil
	case "gau":
		prev := filepath.Join(p.outputDir, "httpx.txt")
		if _, err := os.Stat(prev); os.IsNotExist(err) {
			return "", fmt.Errorf("httpx output not found: %s", prev)
		}
		return prev, nil
	case "ffuf":
		prev := filepath.Join(p.outputDir, "httpx.txt")
		if _, err := os.Stat(prev); os.IsNotExist(err) {
			return "", fmt.Errorf("httpx output not found: %s", prev)
		}
		lines, err := runner.ReadInputLines(prev)
		if err != nil {
			return "", err
		}
		if len(lines) == 0 {
			return "", fmt.Errorf("no alive hosts found for ffuf")
		}
		return prev, nil
	default:
		return "", fmt.Errorf("unknown tool: %s", step.Tool)
	}
}
