package storage

import (
	"context"
	"strings"

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

func (s *Storage) ListLogsFiltered(ctx context.Context, jobID string, level string, step string, search string, afterID int64, limit int) ([]models.Log, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, job_id, COALESCE(step_id,''), level, message, created_at FROM logs WHERE job_id = ?`
	args := []interface{}{jobID}

	if afterID > 0 {
		query += " AND id > ?"
		args = append(args, afterID)
	}

	if level != "" {
		query += " AND level = ?"
		args = append(args, level)
	}

	if step != "" {
		query += " AND step_id LIKE ?"
		args = append(args, "%"+step+"%")
	}

	if search != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY id ASC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
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

func (s *Storage) SearchLogs(ctx context.Context, jobID string, query string, limit int) ([]models.Log, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, COALESCE(step_id,''), level, message, created_at
		FROM logs WHERE job_id = ? AND message LIKE ? ORDER BY id ASC LIMIT ?`,
		jobID, "%"+query+"%", limit)
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

func (s *Storage) CountLogs(ctx context.Context, jobID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM logs WHERE job_id = ?`, jobID).Scan(&count)
	return count, err
}

func (s *Storage) DeleteOldLogs(ctx context.Context, jobID string, keepDays int) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM logs WHERE job_id = ? AND created_at < datetime('now', '-' || ? || ' days')`,
		jobID, keepDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func filterLogsBySearch(logs []models.Log, search string) []models.Log {
	if search == "" {
		return logs
	}

	search = strings.ToLower(search)
	var filtered []models.Log
	for _, l := range logs {
		if strings.Contains(strings.ToLower(l.Message), search) {
			filtered = append(filtered, l)
		}
	}
	return filtered
}
