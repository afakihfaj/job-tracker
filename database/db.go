package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = "job_tracker.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite best practice to avoid concurrency locks
	db.SetMaxOpenConns(1)

	// Enable WAL mode and Foreign Keys
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		log.Printf("Warning: failed to set WAL mode: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS applications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		company TEXT NOT NULL,
		role TEXT NOT NULL,
		platform TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'Applied',
		salary INTEGER NOT NULL DEFAULT 0,
		notes TEXT NOT NULL DEFAULT '',
		applied_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS status_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		application_id INTEGER NOT NULL,
		old_status TEXT NOT NULL,
		new_status TEXT NOT NULL,
		changed_at DATETIME NOT NULL,
		FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_status_logs_application_id ON status_logs(application_id);
	CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(status);
	`

	_, err := db.Exec(schema)
	return err
}
