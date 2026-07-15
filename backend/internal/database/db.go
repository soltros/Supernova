package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	// We serialize all database access with 1 open connection. This prevents 
	// SQLite 'database is locked' deadlocks when multiple goroutines read-then-write.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Automatically run our embedded schema migrations
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema migrations: %w", err)
	}

	// Track migration states using user_version
	var version int
	err = db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to read user_version: %w", err)
	}

	if version < 1 {
		log.Println("Migrating database to version 1...")
		_, err = db.Exec("ALTER TABLE tracks ADD COLUMN popularity INTEGER DEFAULT 0;")
		if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("migration to v1 failed: %w", err)
		}
		db.Exec("PRAGMA user_version = 1")
		version = 1
	}

	if version < 2 {
		log.Println("Migrating database to version 2 (Podcasts)...")
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS podcast_subscriptions (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				feed_id TEXT NOT NULL,
				feed_url TEXT NOT NULL,
				title TEXT NOT NULL,
				image_url TEXT,
				subscribed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(user_id, feed_id),
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
			CREATE TABLE IF NOT EXISTS podcast_progress (
				user_id TEXT NOT NULL,
				episode_id TEXT NOT NULL,
				position_ms INTEGER NOT NULL DEFAULT 0,
				completed BOOLEAN NOT NULL DEFAULT 0,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY(user_id, episode_id),
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
		if err != nil {
			return nil, fmt.Errorf("migration to v2 failed: %w", err)
		}
		db.Exec("PRAGMA user_version = 2")
	}

	if version < 3 {
		log.Println("Migrating database to version 3 (Radio)...")
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS radio_subscriptions (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				station_id TEXT NOT NULL,
				url TEXT NOT NULL,
				name TEXT NOT NULL,
				favicon TEXT,
				subscribed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(user_id, station_id),
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
		if err != nil {
			return nil, fmt.Errorf("migration to v3 failed: %w", err)
		}
		db.Exec("PRAGMA user_version = 3")
	}

	if version < 4 {
		log.Println("Migrating database to version 4 (Plugin Persistence)...")
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS ignored_files (
				file_path TEXT PRIMARY KEY,
				reason TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			ALTER TABLE tracks ADD COLUMN file_modified_at INTEGER DEFAULT 0;
			ALTER TABLE albums ADD COLUMN bio TEXT;
		`)
		if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("migration to v4 failed: %w", err)
		}
		db.Exec("PRAGMA user_version = 4")
	}

	return &DB{db}, nil
}
