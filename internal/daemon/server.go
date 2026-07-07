package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/toprakpt1/recond/internal/config"
	"github.com/toprakpt1/recond/internal/models"
	"github.com/toprakpt1/recond/internal/pipeline"
	"github.com/toprakpt1/recond/internal/runner"
	"github.com/toprakpt1/recond/internal/storage"
	templatepkg "github.com/toprakpt1/recond/internal/template"
	"gopkg.in/yaml.v3"
)

type Daemon struct {
	store      *storage.Storage
	cfg        *config.Config
	registry   *runner.Registry
	listener   net.Listener
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	activeJobs map[string]context.CancelFunc
	debug      bool
}

func New(debug bool) (*Daemon, error) {
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
		registry:   runner.NewRegistry(),
		activeJobs: make(map[string]context.CancelFunc),
		debug:      debug,
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
	case "profiles":
		resp = d.handleProfiles(ctx, req.Payload)
	case "health":
		resp = d.handleHealth(ctx)
	case "export":
		resp = d.handleExport(ctx, req.Payload)
	case "outputs":
		resp = d.handleOutputs(ctx, req.Payload)
	case "templates":
		resp = d.handleTemplates(ctx)
	case "template-show":
		resp = d.handleTemplateShow(ctx, req.Payload)
	case "template-create":
		resp = d.handleTemplateCreate(ctx, req.Payload)
	case "template-delete":
		resp = d.handleTemplateDelete(ctx, req.Payload)
	case "delete":
		resp = d.handleDelete(ctx, req.Payload)
	case "delete-all":
		resp = d.handleDeleteAll(ctx)
	case "retry":
		resp = d.handleRetry(ctx, req.Payload)
	case "duplicate":
		resp = d.handleDuplicate(ctx, req.Payload)
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

	profile, ok := config.GetProfile(req.Profile)
	if !ok {
		return Response{Error: fmt.Sprintf("profile '%s' not found", req.Profile)}
	}

	now := time.Now()
	jobID := fmt.Sprintf("job-%x", now.UnixNano())

	job := models.Job{
		ID:        jobID,
		Name:      req.Target,
		Target:    req.Target,
		Status:    models.JobStatusRunning,
		Profile:   req.Profile,
		Debug:     req.Debug,
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
		Message: fmt.Sprintf("job created with profile: %s (concurrency=%d, rate_limit=%d, timeout=%v)", req.Profile, profile.Concurrency, profile.RateLimit, profile.Timeout),
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

		p := pipeline.NewPipeline(d.store, jobID, profile, d.debug || req.Debug)
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

	profile, _ := config.GetProfile(job.Profile)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			d.mu.Lock()
			delete(d.activeJobs, req.JobID)
			d.mu.Unlock()
		}()

		p := pipeline.NewPipeline(d.store, req.JobID, profile, d.debug || job.Debug)
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
		JobID  string  `json:"job_id"`
		Level  string  `json:"level,omitempty"`
		Step   string  `json:"step,omitempty"`
		Search string  `json:"search,omitempty"`
		Limit  int     `json:"limit"`
		After  float64 `json:"after,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}

	var logs []models.Log
	var err error

	logs, err = d.store.ListLogsFiltered(ctx, req.JobID, req.Level, req.Step, req.Search, int64(req.After), req.Limit)

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

func (d *Daemon) handleHealth(ctx context.Context) Response {
	activeJobs := make([]map[string]interface{}, 0)
	d.mu.Lock()
	for jobID := range d.activeJobs {
		job, err := d.store.GetJob(ctx, jobID)
		if err == nil {
			activeJobs = append(activeJobs, map[string]interface{}{
				"id":     job.ID,
				"name":   job.Name,
				"status": job.Status,
			})
		}
	}
	d.mu.Unlock()

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"running":     true,
			"data_dir":    d.cfg.DataDir,
			"socket_path": d.cfg.SocketPath,
			"profile":     d.cfg.DefaultProfile,
			"active_jobs": activeJobs,
			"tool_status": d.registry.CheckTools(),
		},
	}
}

func (d *Daemon) handleProfiles(ctx context.Context, payload json.RawMessage) Response {
	profiles := config.ListProfiles()

	var profileList []map[string]interface{}
	for _, p := range profiles {
		profileList = append(profileList, map[string]interface{}{
			"name":        p.Name,
			"concurrency": p.Concurrency,
			"rate_limit":  p.RateLimit,
			"cpu_max":     p.CPUMax,
			"ram_max":     p.RAMMax,
			"timeout":     p.Timeout.String(),
		})
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"profiles": profileList,
			"count":    len(profileList),
		},
	}
}

func (d *Daemon) handleExport(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		JobID  string `json:"job_id"`
		Type   string `json:"type"`
		Format string `json:"format"`
		Output string `json:"output,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".recond", "jobs", job.ID)

	outputType := req.Type
	if outputType == "" {
		outputType = "all"
	}

	items, err := d.readOutputFile(dataDir, outputType)
	if err != nil {
		return Response{Error: err.Error()}
	}

	format := req.Format
	if format == "" {
		format = "text"
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"items": items,
			"count": len(items),
			"type":  outputType,
			"format": format,
		},
	}
}

func (d *Daemon) readOutputFile(dataDir, outputType string) ([]string, error) {
	var fileName string
	switch outputType {
	case "subdomains":
		fileName = "subfinder.txt"
	case "alive":
		fileName = "httpx.txt"
	case "urls":
		fileName = "katana.txt"
	case "directories":
		fileName = "directories.json"
	default:
		return nil, fmt.Errorf("unknown output type: %s", outputType)
	}

	filePath := filepath.Join(dataDir, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("no %s output found", outputType)
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

func (d *Daemon) handleOutputs(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".recond", "jobs", job.ID)

	var outputs []string
	types := map[string]string{
		"subdomains":  "subfinder.txt",
		"alive":       "httpx.txt",
		"crawled":     "katana.txt",
		"urls":        "gau.txt",
		"directories": "directories.json",
	}

	for name, file := range types {
		path := filepath.Join(dataDir, file)
		if _, err := os.Stat(path); err == nil {
			data, _ := os.ReadFile(path)
			lines := strings.Split(string(data), "\n")
			count := 0
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					count++
				}
			}
			outputs = append(outputs, fmt.Sprintf("%s: %d items (%s)", name, count, path))
		}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"outputs": outputs,
			"count":   len(outputs),
		},
	}
}

func (d *Daemon) handleTemplates(ctx context.Context) Response {
	templates, err := templatepkg.ListTemplates()
	if err != nil {
		templates = nil
	}

	var templateList []map[string]interface{}
	for _, t := range templates {
		templateList = append(templateList, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"steps":       len(t.Steps),
		})
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"templates": templateList,
			"count":     len(templateList),
		},
	}
}

func (d *Daemon) handleTemplateShow(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	t, err := templatepkg.LoadTemplate(req.Name)
	if err != nil {
		return Response{Error: err.Error()}
	}

	var steps []map[string]interface{}
	for _, s := range t.Steps {
		steps = append(steps, map[string]interface{}{
			"name":      s.Name,
			"tool":      s.Tool,
			"order":     s.Order,
			"parallel":  s.Parallel,
		})
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"steps":       steps,
		},
	}
}

func (d *Daemon) handleTemplateCreate(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	var t templatepkg.Template
	if err := yaml.Unmarshal([]byte(req.Data), &t); err != nil {
		return Response{Error: "invalid template YAML: " + err.Error()}
	}

	if err := t.Validate(); err != nil {
		return Response{Error: err.Error()}
	}

	if err := templatepkg.CreateTemplate(req.Name, &t); err != nil {
		return Response{Error: err.Error()}
	}

	return Response{Success: true}
}

func (d *Daemon) handleTemplateDelete(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	if err := templatepkg.DeleteTemplate(req.Name); err != nil {
		return Response{Error: err.Error()}
	}

	return Response{Success: true}
}

func (d *Daemon) handleDelete(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	if job.Status == models.JobStatusRunning {
		if cancel, ok := d.activeJobs[req.JobID]; ok {
			cancel()
		}
	}

	if err := d.store.DeleteJob(ctx, req.JobID); err != nil {
		return Response{Error: "failed to delete job: " + err.Error()}
	}

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".recond", "jobs", req.JobID)
	os.RemoveAll(dataDir)

	return Response{Success: true}
}

func (d *Daemon) handleDeleteAll(ctx context.Context) Response {
	jobs, err := d.store.ListJobs(ctx, "completed")
	if err != nil {
		return Response{Error: "failed to list jobs: " + err.Error()}
	}

	deleted := 0
	for _, job := range jobs {
		if err := d.store.DeleteJob(ctx, job.ID); err == nil {
			home, _ := os.UserHomeDir()
			dataDir := filepath.Join(home, ".recond", "jobs", job.ID)
			os.RemoveAll(dataDir)
			deleted++
		}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"deleted": deleted,
		},
	}
}

func (d *Daemon) handleRetry(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		JobID    string `json:"job_id"`
		Profile  string `json:"profile,omitempty"`
		FromStep string `json:"from_step,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	profileName := req.Profile
	if profileName == "" {
		profileName = job.Profile
	}

	profile, ok := config.GetProfile(profileName)
	if !ok {
		return Response{Error: "profile not found: " + profileName}
	}

	now := time.Now()
	newJobID := fmt.Sprintf("job-%x", now.UnixNano())

	newJob := models.Job{
		ID:        newJobID,
		Name:      job.Name,
		Target:    job.Target,
		Status:    models.JobStatusRunning,
		Profile:   profileName,
		Debug:     job.Debug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := d.store.CreateJob(ctx, newJob); err != nil {
		return Response{Error: "failed to create job: " + err.Error()}
	}

	steps := pipeline.CreateSteps(newJobID)
	for _, step := range steps {
		if req.FromStep != "" && step.Name != req.FromStep {
			step.Status = models.StepStatusCompleted
		}
		if err := d.store.CreateStep(ctx, step); err != nil {
			return Response{Error: "failed to create step: " + err.Error()}
		}
	}

	d.store.InsertLog(ctx, models.Log{
		JobID:   newJobID,
		Level:   models.LogInfo,
		Message: fmt.Sprintf("job retried from %s (original: %s)", req.FromStep, req.JobID),
	})

	jobCtx, jobCancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.activeJobs[newJobID] = jobCancel
	d.mu.Unlock()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			d.mu.Lock()
			delete(d.activeJobs, newJobID)
			d.mu.Unlock()
		}()

		p := pipeline.NewPipeline(d.store, newJobID, profile, d.debug || job.Debug)
		if err := p.Execute(jobCtx); err != nil {
			if err == context.Canceled {
				d.store.UpdateJobStatus(context.Background(), newJobID, models.JobStatusPaused)
			} else {
				d.store.UpdateJobStatus(context.Background(), newJobID, models.JobStatusFailed)
			}
			return
		}
		d.store.UpdateJobStatus(context.Background(), newJobID, models.JobStatusCompleted)
	}()

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"new_job_id": newJobID,
			"target":     job.Target,
		},
	}
}

func (d *Daemon) handleDuplicate(ctx context.Context, payload json.RawMessage) Response {
	var req struct {
		JobID   string `json:"job_id"`
		Profile string `json:"profile,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Error: "invalid payload: " + err.Error()}
	}

	job, err := d.store.GetJob(ctx, req.JobID)
	if err != nil {
		return Response{Error: "job not found: " + err.Error()}
	}

	profileName := req.Profile
	if profileName == "" {
		profileName = job.Profile
	}

	profile, ok := config.GetProfile(profileName)
	if !ok {
		return Response{Error: "profile not found: " + profileName}
	}

	now := time.Now()
	newJobID := fmt.Sprintf("job-%x", now.UnixNano())

	newJob := models.Job{
		ID:        newJobID,
		Name:      job.Name,
		Target:    job.Target,
		Status:    models.JobStatusRunning,
		Profile:   profileName,
		Debug:     job.Debug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := d.store.CreateJob(ctx, newJob); err != nil {
		return Response{Error: "failed to create job: " + err.Error()}
	}

	steps := pipeline.CreateSteps(newJobID)
	for _, step := range steps {
		if err := d.store.CreateStep(ctx, step); err != nil {
			return Response{Error: "failed to create step: " + err.Error()}
		}
	}

	d.store.InsertLog(ctx, models.Log{
		JobID:   newJobID,
		Level:   models.LogInfo,
		Message: fmt.Sprintf("job duplicated from %s", req.JobID),
	})

	jobCtx, jobCancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.activeJobs[newJobID] = jobCancel
	d.mu.Unlock()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			d.mu.Lock()
			delete(d.activeJobs, newJobID)
			d.mu.Unlock()
		}()

		p := pipeline.NewPipeline(d.store, newJobID, profile, d.debug || job.Debug)
		if err := p.Execute(jobCtx); err != nil {
			if err == context.Canceled {
				d.store.UpdateJobStatus(context.Background(), newJobID, models.JobStatusPaused)
			} else {
				d.store.UpdateJobStatus(context.Background(), newJobID, models.JobStatusFailed)
			}
			return
		}
		d.store.UpdateJobStatus(context.Background(), newJobID, models.JobStatusCompleted)
	}()

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"new_job_id": newJobID,
			"target":     job.Target,
		},
	}
}
