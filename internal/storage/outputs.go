package storage

import (
	"context"

	"github.com/recond/internal/models"
)

func (s *Storage) CreateOutput(ctx context.Context, output models.Output) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO outputs (id, job_id, step_id, path, kind, size_bytes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		output.ID, output.JobID, output.StepID, output.Path, output.Kind, output.SizeBytes, output.CreatedAt)
	return err
}

func (s *Storage) ListOutputs(ctx context.Context, jobID string) ([]models.Output, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, COALESCE(step_id,''), path, kind, size_bytes, created_at
		FROM outputs WHERE job_id = ? ORDER BY created_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outputs []models.Output
	for rows.Next() {
		var o models.Output
		if err := rows.Scan(&o.ID, &o.JobID, &o.StepID, &o.Path, &o.Kind, &o.SizeBytes, &o.CreatedAt); err != nil {
			return nil, err
		}
		outputs = append(outputs, o)
	}
	return outputs, rows.Err()
}
