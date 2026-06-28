package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Using ncruces/go-sqlite3 which is a 100% pure Go WebAssembly-based SQLite driver.
	// This guarantees we have zero CGO dependencies.
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

//go:embed schema.sql
var schemaSQL string

// DB wraps the standard sql.DB to provide custom repository methods later
type DB struct {
	*sql.DB
}

// Init initializes the database connection and runs our schema migrations
func Init(dbPath string) (*DB, error) {
	// Ensure the data directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connect to SQLite.
	// We enable WAL (Write-Ahead Logging) for significantly better concurrency,
	// allowing simultaneous reads (e.g. streaming) and writes (e.g. background scanning).
	// We also enforce foreign keys since they are disabled by default in SQLite.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", dbPath)
	
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// CONCURRENCY TUNING:
	// We cap max open connections so SQLite doesn't get overwhelmed with lock contention
	// by our background worker pool. WAL mode handles this gracefully with a timeout.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Automatically run our embedded schema migrations
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema migrations: %w", err)
	}

	// Add popularity column to existing DBs safely
	db.Exec("ALTER TABLE tracks ADD COLUMN popularity INTEGER DEFAULT 0;")

	return &DB{db}, nil
}
