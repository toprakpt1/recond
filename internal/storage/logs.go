package storage

import (
	"context"

	"github.com/recond/internal/models"
)

func (s *Storage) InsertLog(ctx context.Context, logEntry models.Log) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO logs (job_id, step_id, level, message, created_at) VALUES (?, ?, ?, ?, ?)`,
		logEntry.JobID, logEntry.StepID, logEntry.Level, logEntry.Message, logEntry.CreatedAt)
	return err
}

func (s *Storage) ListLogs(ctx context.Context, jobID string, limit int) ([]models.Log, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, COALESCE(step_id,''), level, message, created_at
		FROM logs WHERE job_id = ? ORDER BY id ASC LIMIT ?`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.Log
	for rows.Next() {
		var l models.Log
		if err := rows.Scan(&l.ID, &l.JobID, &l.StepID, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Storage) ListLogsWithLevel(ctx context.Context, jobID string, level models.LogLevel, limit int) ([]models.Log, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, COALESCE(step_id,''), level, message, created_at
		FROM logs WHERE job_id = ? AND level = ? ORDER BY id ASC LIMIT ?`, jobID, level, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.Log
	for rows.Next() {
		var l models.Log
		if err := rows.Scan(&l.ID, &l.JobID, &l.StepID, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
