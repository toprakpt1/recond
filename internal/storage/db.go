package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(dataDir string) (*Storage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "recond.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			target TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			profile TEXT DEFAULT 'balanced',
			config_json TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS steps (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			name TEXT NOT NULL,
			tool TEXT NOT NULL,
			ord INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			progress REAL DEFAULT 0,
			total_items INTEGER DEFAULT 0,
			processed_items INTEGER DEFAULT 0,
			started_at DATETIME,
			finished_at DATETIME,
			checkpoint_json TEXT,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		)`,
		`CREATE TABLE IF NOT EXISTS targets (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			value TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'domain',
			status TEXT NOT NULL DEFAULT 'pending',
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		)`,
		`CREATE TABLE IF NOT EXISTS outputs (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			step_id TEXT,
			path TEXT NOT NULL,
			kind TEXT NOT NULL,
			size_bytes INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES jobs(id),
			FOREIGN KEY (step_id) REFERENCES steps(id)
		)`,
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			step_id TEXT,
			level TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_steps_job_id ON steps(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_job_id ON logs(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_outputs_job_id ON outputs(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_targets_job_id ON targets(job_id)`,
	}

	for _, q := range schema {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}

	// Add debug column if it doesn't exist (for existing databases)
	if _, err := s.db.Exec(`ALTER TABLE jobs ADD COLUMN debug INTEGER DEFAULT 0`); err != nil {
		// Ignore error if column already exists
	}

	return nil
}
