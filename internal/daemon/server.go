package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/recond/internal/config"
	"github.com/recond/internal/models"
	"github.com/recond/internal/pipeline"
	"github.com/recond/internal/storage"
)

type Daemon struct {
	store      *storage.Storage
	cfg        *config.Config
	listener   net.Listener
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	activeJobs map[string]context.CancelFunc
}

func New() (*Daemon, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.EnsureDataDir(cfg.DataDir); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	store, err := storage.NewStorage(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &Daemon{
		store:      store,
		cfg:        cfg,
		activeJobs: make(map[string]context.CancelFunc),
	}, nil
}

func (d *Daemon) Start() error {
	socketPath := d.cfg.SocketPath

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	d.listener = listener

	log.Printf("daemon started (socket: %s)", socketPath)
	log.Printf("data directory: %s", d.cfg.DataDir)

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	d.wg.Add(1)
	go d.acceptLoop(ctx)

	return nil
}

func (d *Daemon) Stop() {
	d.cancel()

	if d.listener != nil {
		d.listener.Close()
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Println("timeout waiting for goroutines to finish")
	}

	if d.store != nil {
		d.store.Close()
	}

	socketPath := d.cfg.SocketPath
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	log.Println("daemon stopped")
}

func (d *Daemon) acceptLoop(ctx context.Context) {
	defer d.wg.Done()

	for {
		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}

		go d.handleConnection(ctx, conn)
	}
}

func (d *Daemon) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		json.NewEncoder(conn).Encode(Response{Error: "invalid request: " + err.Error()})
		return
	}

	var resp Response

	switch req.Action {
	case "start":
		resp = d.handleStart(ctx, req.Payload)
	case "pause":
		resp = d.handlePause(ctx, req.Payload)
	case "resume":
		resp = d.handleResume(ctx, req.Payload)
	case "stop":
		resp = d.handleStop(ctx, req.Payload)
	case "status":
		resp = d.handleStatus(ctx, req.Payload)
	case "list":
		resp = d.handleList(ctx, req.Payload)
	case "logs":
		resp = d.handleLogs(ctx, req.Payload)
	case "daemon-status":
		resp = Response{Success: true, Data: map[string]interface{}{
			"running":   true,
			"data_dir":  d.cfg.DataDir,
			"jobs_count": len(d.activeJobs),
		}}
	default:
		resp = Response{Error: "unknown action: " + req.Action}
	}

	json.NewEncoder(conn).Encode(resp)
}

func (d *Daemon) handleStart(ctx context.Context, payload json.RawMessage) Response {
	var req StartRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}
	if req.Target == "" {
		return Response{Error: "target is required"}
	}
	if req.Profile == "" {
		req.Profile = d.cfg.DefaultProfile
	}

	now := time.Now()
	jobID := fmt.Sprintf("job-%x", now.UnixNano())

	job := models.Job{
		ID:        jobID,
		Name:      req.Target,
		Target:    req.Target,
		Status:    models.JobStatusRunning,
		Profile:   req.Profile,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := d.store.CreateJob(ctx, job); err != nil {
		return Response{Error: "failed to create job: " + err.Error()}
	}

	steps := pipeline.CreateSteps(jobID)
	for _, step := range steps {
		if err := d.store.CreateStep(ctx, step); err != nil {
			return Response{Error: "failed to create step: " + err.Error()}
		}
	}

	d.store.InsertLog(ctx, models.Log{
		JobID:   jobID,
		Level:   models.LogInfo,
		Message: fmt.Sprintf("job created with profile: %s", req.Profile),
	})

	jobCtx, jobCancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.activeJobs[jobID] = jobCancel
	d.mu.Unlock()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			d.mu.Lock()
			delete(d.activeJobs, jobID)
			d.mu.Unlock()
		}()

		p := pipeline.NewPipeline(d.store, jobID)
		if err := p.Execute(jobCtx); err != nil {
			if err == context.Canceled {
				d.store.UpdateJobStatus(context.Background(), jobID, models.JobStatusPaused)
				d.store.InsertLog(context.Background(), models.Log{
					JobID:   jobID,
					Level:   models.LogInfo,
					Message: "job paused",
				})
			} else {
				d.store.UpdateJobStatus(context.Background(), jobID, models.JobStatusFailed)
				d.store.InsertLog(context.Background(), models.Log{
					JobID:   jobID,
					Level:   models.LogError,
					Message: "job failed: " + err.Error(),
				})
			}
			return
		}
		d.store.UpdateJobStatus(context.Background(), jobID, models.JobStatusCompleted)
		d.store.InsertLog(context.Background(), models.Log{
			JobID:   jobID,
			Level:   models.LogInfo,
			Message: "job completed",
		})
	}()

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"job_id":  jobID,
			"status":  job.Status,
			"target":  job.Target,
			"profile": job.Profile,
			"steps":   len(steps),
		},
	}
}

func (d *Daemon) handlePause(ctx context.Context, payload json.RawMessage) Response {
	var req ActionRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	if job.Status != models.JobStatusRunning {
		return Response{Error: fmt.Sprintf("job is not running (current status: %s)", job.Status)}
	}

	d.mu.Lock()
	if cancel, ok := d.activeJobs[req.JobID]; ok {
		cancel()
	}
	d.mu.Unlock()

	if err := d.store.UpdateJobStatus(ctx, req.JobID, models.JobStatusPaused); err != nil {
		return Response{Error: "failed to pause job: " + err.Error()}
	}

	steps, _ := d.store.ListSteps(ctx, req.JobID)
	for _, step := range steps {
		if step.Status == models.StepStatusRunning {
			d.store.UpdateStepStatus(ctx, step.ID, models.StepStatusPaused)
		}
	}

	d.store.InsertLog(ctx, models.Log{
		JobID:   req.JobID,
		Level:   models.LogInfo,
		Message: "job paused by user",
	})

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"job_id": req.JobID,
			"status": models.JobStatusPaused,
		},
	}
}

func (d *Daemon) handleResume(ctx context.Context, payload json.RawMessage) Response {
	var req ActionRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	if job.Status != models.JobStatusPaused {
		return Response{Error: fmt.Sprintf("job is not paused (current status: %s)", job.Status)}
	}

	if err := d.store.UpdateJobStatus(ctx, req.JobID, models.JobStatusRunning); err != nil {
		return Response{Error: "failed to resume job: " + err.Error()}
	}

	jobCtx, jobCancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.activeJobs[req.JobID] = jobCancel
	d.mu.Unlock()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			d.mu.Lock()
			delete(d.activeJobs, req.JobID)
			d.mu.Unlock()
		}()

		p := pipeline.NewPipeline(d.store, req.JobID)
		if err := p.Execute(jobCtx); err != nil {
			if err == context.Canceled {
				d.store.UpdateJobStatus(context.Background(), req.JobID, models.JobStatusPaused)
			} else {
				d.store.UpdateJobStatus(context.Background(), req.JobID, models.JobStatusFailed)
			}
			return
		}
		d.store.UpdateJobStatus(context.Background(), req.JobID, models.JobStatusCompleted)
	}()

	d.store.InsertLog(ctx, models.Log{
		JobID:   req.JobID,
		Level:   models.LogInfo,
		Message: "job resumed by user",
	})

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"job_id": req.JobID,
			"status": models.JobStatusRunning,
		},
	}
}

func (d *Daemon) handleStop(ctx context.Context, payload json.RawMessage) Response {
	var req ActionRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	d.mu.Lock()
	if cancel, ok := d.activeJobs[req.JobID]; ok {
		cancel()
	}
	d.mu.Unlock()

	if err := d.store.UpdateJobStatus(ctx, req.JobID, models.JobStatusStopped); err != nil {
		return Response{Error: "failed to stop job: " + err.Error()}
	}

	steps, _ := d.store.ListSteps(ctx, req.JobID)
	for _, step := range steps {
		if step.Status == models.StepStatusRunning {
			d.store.CompleteStep(ctx, step.ID, models.StepStatusFailed, "stopped by user")
		}
	}

	d.store.InsertLog(ctx, models.Log{
		JobID:   req.JobID,
		Level:   models.LogInfo,
		Message: "job stopped by user",
	})

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"job_id": req.JobID,
			"status": models.JobStatusStopped,
		},
	}
}

func (d *Daemon) handleStatus(ctx context.Context, payload json.RawMessage) Response {
	var req ActionRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	steps, err := d.store.ListSteps(ctx, req.JobID)
	if err != nil {
		return Response{Error: "failed to get steps: " + err.Error()}
	}

	outputs, _ := d.store.ListOutputs(ctx, req.JobID)

	currentStep := ""
	var overallProgress float64
	for _, s := range steps {
		if s.Status == models.StepStatusRunning {
			currentStep = s.Name
		}
		overallProgress += s.Progress
	}
	if len(steps) > 0 {
		overallProgress /= float64(len(steps))
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"job":     job,
			"steps":   steps,
			"outputs": outputs,
			"current_step": currentStep,
			"overall_progress": overallProgress,
		},
	}
}

func (d *Daemon) handleList(ctx context.Context, payload json.RawMessage) Response {
	filter := ""
	if payload != nil {
		var req struct {
			Status string `json:"status"`
		}
		json.Unmarshal(payload, &req)
		filter = req.Status
	}

	jobs, err := d.store.ListJobs(ctx, filter)
	if err != nil {
		return Response{Error: "failed to list jobs: " + err.Error()}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"jobs":  jobs,
			"count": len(jobs),
		},
	}
}

func (d *Daemon) handleLogs(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		JobID string `json:"job_id"`
		Level string `json:"level,omitempty"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}

	var logs []models.Log
	var err error

	if req.Level != "" {
		logs, err = d.store.ListLogsWithLevel(ctx, req.JobID, models.LogLevel(req.Level), req.Limit)
	} else {
		logs, err = d.store.ListLogs(ctx, req.JobID, req.Limit)
	}

	if err != nil {
		return Response{Error: "failed to get logs: " + err.Error()}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"logs":  logs,
			"count": len(logs),
		},
	}
}
