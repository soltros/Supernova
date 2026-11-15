package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestLiveBackupRestoreAndRefuseOverwrite(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "owner", "hash")
	if err != nil {
		t.Fatal(err)
	}
	playlist, err := r.CreatePlaylist(ctx, u.ID, "important playlist")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	if err := r.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&source); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "snapshot.db")
	if err := Backup(ctx, source, dest); err != nil {
		t.Fatal(err)
	}
	if err := Backup(ctx, source, dest); err == nil {
		t.Fatal("overwrote backup")
	}
	restored, err := Init(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	recovered := NewRepository(restored)
	user, hash, err := recovered.GetUserByUsername(ctx, "owner")
	if err != nil || user == nil || hash != "hash" {
		t.Fatalf("lost account: %v", err)
	}
	playlists, err := recovered.GetPlaylists(ctx, user.ID)
	if err != nil || len(playlists) != 1 || playlists[0].ID != playlist.ID {
		t.Fatalf("lost playlist: %v %v", playlists, err)
	}
}
func TestBackupFailureLeavesNoDestination(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "snapshot.db")
	if err := Backup(context.Background(), filepath.Join(dir, "missing.db"), dest); err == nil {
		t.Fatal("backed up missing database")
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatalf("partial files retained: %v %v", files, err)
	}
}
func TestFutureSchemaRejectedWithoutSchemaWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version=999; CREATE TABLE future_marker(id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if unexpected, err := Init(path); err == nil {
		unexpected.Close()
		t.Fatal("future schema accepted")
	}
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("schema modified: %d %v", count, err)
	}
}
