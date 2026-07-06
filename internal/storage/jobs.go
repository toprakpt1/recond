package storage

import (
	"context"
	"fmt"

	"github.com/recond/internal/models"
)

func (s *Storage) CreateJob(ctx context.Context, job models.Job) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (id, name, target, status, profile, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Name, job.Target, job.Status, job.Profile, job.ConfigJSON,
		job.CreatedAt, job.UpdatedAt)
	return err
}

func (s *Storage) GetJob(ctx context.Context, id string) (*models.Job, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, target, status, profile, COALESCE(config_json,''), created_at, updated_at
		FROM jobs WHERE id = ?`, id)

	job := &models.Job{}
	err := row.Scan(&job.ID, &job.Name, &job.Target, &job.Status, &job.Profile, &job.ConfigJSON,
		&job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Storage) ListJobs(ctx context.Context, status string) ([]models.Job, error) {
	query := `SELECT id, name, target, status, profile, COALESCE(config_json,''), created_at, updated_at FROM jobs`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var j models.Job
		if err := rows.Scan(&j.ID, &j.Name, &j.Target, &j.Status, &j.Profile, &j.ConfigJSON,
			&j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Storage) UpdateJobStatus(ctx context.Context, id string, status models.JobStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (s *Storage) UpdateJob(ctx context.Context, job models.Job) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = ?, profile = ?, config_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		job.Status, job.Profile, job.ConfigJSON, job.ID)
	return err
}

func (s *Storage) DeleteJob(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"logs", "outputs", "targets", "steps"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE job_id = ?", table), id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM jobs WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}
