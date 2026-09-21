package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net/url"
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
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: absPath}
	dsn := u.String() + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	success := false
	defer func() {
		if !success {
			db.Close()
		}
	}()

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// WAL allows concurrent readers; repository transactions serialize related writes.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Automatically run our embedded schema migrations
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema migrations: %w", err)
	}

	// Ensure new columns exist for older databases
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN subsonic_password TEXT DEFAULT '';")

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
		if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 1: %w", err)
		}
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
		if _, err := db.Exec("PRAGMA user_version = 2"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 2: %w", err)
		}
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
		if _, err := db.Exec("PRAGMA user_version = 3"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 3: %w", err)
		}
	}

	if version < 4 {
		log.Println("Migrating database to version 4 (Plugin Persistence)...")
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS ignored_files (
				file_path TEXT PRIMARY KEY,
				reason TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return nil, fmt.Errorf("migration to v4 failed: %w", err)
		}

		_, err = db.Exec(`ALTER TABLE tracks ADD COLUMN file_modified_at INTEGER DEFAULT 0;`)
		if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("migration to v4 failed: %w", err)
		}

		_, err = db.Exec(`ALTER TABLE albums ADD COLUMN bio TEXT;`)
		if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("migration to v4 failed: %w", err)
		}

		if _, err := db.Exec("PRAGMA user_version = 4"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 4: %w", err)
		}
	}

	if version < 5 {
		tx, err := db.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;`); err != nil {
			return nil, err
		}
		// Preserve ownership on upgrade: the oldest account becomes administrator.
		if _, err := tx.Exec(`UPDATE users SET is_admin = 1 WHERE id = (SELECT id FROM users ORDER BY created_at, rowid LIMIT 1); PRAGMA user_version = 5;`); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}

	if version < 6 {
		log.Println("Migrating database to version 6 (ordered playlist entries)...")
		var hasEntryID int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('playlist_tracks') WHERE name = 'entry_id'`).Scan(&hasEntryID); err != nil {
			return nil, fmt.Errorf("migration to v6 inspection failed: %w", err)
		}
		if hasEntryID == 0 {
			tx, err := db.Begin()
			if err != nil {
				return nil, fmt.Errorf("migration to v6 begin failed: %w", err)
			}
			if _, err := tx.Exec(`
				CREATE TABLE playlist_tracks_v6 (
					entry_id TEXT PRIMARY KEY,
					playlist_id TEXT NOT NULL,
					track_id TEXT NOT NULL,
					position INTEGER NOT NULL,
					added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
					FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE,
					UNIQUE(playlist_id, position)
				);
				INSERT INTO playlist_tracks_v6(entry_id, playlist_id, track_id, position, added_at)
				SELECT lower(hex(randomblob(16))), playlist_id, track_id, position, added_at
				FROM playlist_tracks
				ORDER BY playlist_id, position;
				DROP TABLE playlist_tracks;
				ALTER TABLE playlist_tracks_v6 RENAME TO playlist_tracks;
				CREATE INDEX idx_playlist_tracks_playlist_id_pos ON playlist_tracks(playlist_id, position);
			`); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("migration to v6 failed: %w", err)
			}
			if err := tx.Commit(); err != nil {
				return nil, fmt.Errorf("migration to v6 commit failed: %w", err)
			}
		}
		if _, err := db.Exec("PRAGMA user_version = 6"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 6: %w", err)
		}
		version = 6
	}

	if version < 7 {
		log.Println("Migrating database to version 7 (stable file identity)...")
		for _, stmt := range []string{
			"ALTER TABLE tracks ADD COLUMN file_modified_ns INTEGER DEFAULT 0;",
			"ALTER TABLE tracks ADD COLUMN file_size INTEGER DEFAULT 0;",
			"ALTER TABLE tracks ADD COLUMN file_fingerprint TEXT DEFAULT '';",
		} {
			if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
				return nil, fmt.Errorf("migration to v7 failed: %w", err)
			}
		}
		if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tracks_fingerprint ON tracks(file_fingerprint)"); err != nil {
			return nil, fmt.Errorf("migration to v7 index failed: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 7"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 7: %w", err)
		}
		version = 7
	}

	if version < 8 {
		log.Println("Migrating database to version 8 (revocable sessions)...")
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL,
				revoked_at DATETIME,
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_sessions_user_active ON sessions(user_id, expires_at, revoked_at);
		`); err != nil {
			return nil, fmt.Errorf("migration to v8 failed: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 8"); err != nil {
			return nil, fmt.Errorf("failed to write user_version 8: %w", err)
		}
		version = 8
	}
	success = true
	return &DB{db}, nil
}
