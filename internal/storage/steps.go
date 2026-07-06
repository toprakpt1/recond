package storage

import (
	"context"

	"github.com/recond/internal/models"
)

func (s *Storage) CreateStep(ctx context.Context, step models.Step) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO steps (id, job_id, name, tool, ord, status, progress, total_items, processed_items,
		started_at, finished_at, checkpoint_json, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		step.ID, step.JobID, step.Name, step.Tool, step.Order, step.Status, step.Progress,
		step.TotalItems, step.ProcessedItems, step.StartedAt, step.FinishedAt,
		step.CheckpointJSON, step.ErrorMessage, step.CreatedAt, step.UpdatedAt)
	return err
}

func (s *Storage) GetStep(ctx context.Context, id string) (*models.Step, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, job_id, name, tool, ord, status, progress, total_items, processed_items,
		started_at, finished_at, COALESCE(checkpoint_json,''), COALESCE(error_message,''),
		created_at, updated_at FROM steps WHERE id = ?`, id)

	step := &models.Step{}
	err := row.Scan(&step.ID, &step.JobID, &step.Name, &step.Tool, &step.Order, &step.Status,
		&step.Progress, &step.TotalItems, &step.ProcessedItems, &step.StartedAt, &step.FinishedAt,
		&step.CheckpointJSON, &step.ErrorMessage, &step.CreatedAt, &step.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Storage) ListSteps(ctx context.Context, jobID string) ([]models.Step, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, name, tool, ord, status, progress, total_items, processed_items,
		started_at, finished_at, COALESCE(checkpoint_json,''), COALESCE(error_message,''),
		created_at, updated_at FROM steps WHERE job_id = ? ORDER BY ord ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []models.Step
	for rows.Next() {
		var s models.Step
		if err := rows.Scan(&s.ID, &s.JobID, &s.Name, &s.Tool, &s.Order, &s.Status,
			&s.Progress, &s.TotalItems, &s.ProcessedItems, &s.StartedAt, &s.FinishedAt,
			&s.CheckpointJSON, &s.ErrorMessage, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (s *Storage) UpdateStepStatus(ctx context.Context, id string, status models.StepStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE steps SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (s *Storage) UpdateStepProgress(ctx context.Context, id string, progress float64, processed, total int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE steps SET progress = ?, processed_items = ?, total_items = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		progress, processed, total, id)
	return err
}

func (s *Storage) UpdateStepCheckpoint(ctx context.Context, id string, checkpointJSON string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE steps SET checkpoint_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		checkpointJSON, id)
	return err
}

func (s *Storage) CompleteStep(ctx context.Context, id string, status models.StepStatus, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE steps SET status = ?, error_message = ?, finished_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, errMsg, id)
	return err
}
