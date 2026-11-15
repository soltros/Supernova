package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// Backup creates a consistent SQLite snapshot, including committed WAL data,
// without running migrations or changing the source. Publication never replaces
// an existing destination. The destination must support hard links.
func Backup(ctx context.Context, source, destination string) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	u := url.URL{Scheme: "file", Path: source}
	q := u.Query()
	q.Set("mode", "ro")
	q.Set("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".supernova-backup-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Close(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, name); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	snapshotURL := url.URL{Scheme: "file", Path: name, RawQuery: "mode=ro"}
	snapshot, err := sql.Open("sqlite3", snapshotURL.String())
	if err != nil {
		return err
	}
	var result string
	err = snapshot.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result)
	snapshot.Close()
	if err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("snapshot integrity: %s", result)
	}
	file, err := os.OpenFile(name, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	err = file.Sync()
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Link(name, destination); err != nil {
		return fmt.Errorf("publish snapshot: %w", err)
	}
	return nil
}
