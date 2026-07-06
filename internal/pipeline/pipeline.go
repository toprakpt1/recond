package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/recond/internal/models"
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
	store *storage.Storage
	jobID string
}

func NewPipeline(store *storage.Storage, jobID string) *Pipeline {
	return &Pipeline{store: store, jobID: jobID}
}

func (p *Pipeline) Execute(ctx context.Context) error {
	steps, err := p.store.ListSteps(ctx, p.jobID)
	if err != nil {
		return err
	}

	for _, step := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		stepCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		log.Printf("[pipeline] starting step: %s (tool: %s)", step.Name, step.Tool)
		p.store.UpdateStepStatus(stepCtx, step.ID, models.StepStatusRunning)

		if err := p.executeStep(stepCtx, step); err != nil {
			p.store.CompleteStep(stepCtx, step.ID, models.StepStatusFailed, err.Error())
			return err
		}

		p.store.CompleteStep(stepCtx, step.ID, models.StepStatusCompleted, "")
		log.Printf("[pipeline] completed step: %s", step.Name)
	}

	p.store.UpdateJobStatus(ctx, p.jobID, models.JobStatusCompleted)
	return nil
}

func (p *Pipeline) executeStep(ctx context.Context, step models.Step) error {
	start := time.Now()
	now := start
	p.store.UpdateStepProgress(ctx, step.ID, 100, 100, 100)

	_ = now
	return nil
}
