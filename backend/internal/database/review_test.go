package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/soltros/Supernova/internal/models"
	"path/filepath"
	"sync"
	"testing"
)

func testRepository(t *testing.T) *Repository {
	t.Helper()
	db, err := Init(filepath.Join(t.TempDir(), "library?#.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRepository(db)
}
func TestRegistrationInviteAndConcurrentBootstrap(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	users := []*models.User{}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u, err := r.RegisterUser(ctx, fmt.Sprintf("user%d", i), "hash", "", "")
			if err != nil && !errors.Is(err, ErrInviteRequired) {
				t.Errorf("register: %v", err)
			}
			if u != nil {
				mu.Lock()
				users = append(users, u)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if len(users) != 1 || !users[0].IsAdmin {
		t.Fatalf("bootstrap: %+v", users)
	}
	if _, err := r.RegisterUser(ctx, "wrong", "hash", "bad", "invite"); !errors.Is(err, ErrInviteRequired) {
		t.Fatalf("wrong invite: %v", err)
	}
	u, err := r.RegisterUser(ctx, "invited", "hash", "invite", "invite")
	if err != nil || u.IsAdmin {
		t.Fatalf("invite: %+v %v", u, err)
	}
	persisted, err := r.GetUserByID(ctx, u.ID)
	if err != nil || persisted.IsAdmin {
		t.Fatalf("persisted admin: %+v %v", persisted, err)
	}
}
func TestRescanRepairsDurationAndPreservesOverrides(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	meta := &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: "/music/song.mp3", Format: "ID3v2.4", FileModifiedAt: 100}
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.Exec(`UPDATE tracks SET title='Manual title'`); err != nil {
		t.Fatal(err)
	}
	meta.DurationMs = 123456
	meta.Bitrate = 192
	meta.Format = "mp3"
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	tracks, err := r.GetTracks(ctx, "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].Title != "Manual title" || tracks[0].DurationMs != 123456 || tracks[0].Format != "mp3" {
		t.Fatalf("rescan: %+v", tracks)
	}
	// Replacing a file with an older timestamp must update its tags.
	meta.FileModifiedAt = 50
	meta.Title = "Replacement"
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	track, err := r.GetTrackByID(ctx, tracks[0].ID)
	if err != nil || track.Title != "Replacement" {
		t.Fatalf("older replacement: %+v %v", track, err)
	}
	// AutoTagger intentionally passes no timestamp and must still apply.
	meta.FileModifiedAt = 0
	meta.Title = "Tagged"
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	track, err = r.GetTrackByID(ctx, tracks[0].ID)
	if err != nil || track.Title != "Tagged" {
		t.Fatalf("tagger: %+v %v", track, err)
	}
}
func TestPlaylistOwnership(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	a, _ := r.CreateUser(ctx, "alice", "hash")
	b, _ := r.CreateUser(ctx, "bob", "hash")
	p, err := r.CreatePlaylist(ctx, a.ID, "Private")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetPlaylistTracks(ctx, b.ID, p.ID); err == nil {
		t.Fatal("private playlist readable")
	}
	if err := r.AddTrackToPlaylist(ctx, b.ID, p.ID, "missing"); err == nil {
		t.Fatal("private playlist writable")
	}
	if err := r.HeartEntity(ctx, b.ID, "playlist", p.ID); err == nil {
		t.Fatal("private playlist can be favorited")
	}
}

func TestV4UpgradePreservesAccountsAndAssignsOldestAdmin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = old.Exec(schemaSQL + `
 INSERT INTO users(id,username,password_hash,created_at) VALUES ('old','owner','hash','2026-01-01 00:00:00'),('new','member','hash','2026-01-02 00:00:00');
 PRAGMA user_version=4;`); err != nil {
		t.Fatal(err)
	}
	old.Close()
	db, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r := NewRepository(db)
	for _, id := range []string{"old", "new"} {
		u, err := r.GetUserByID(context.Background(), id)
		if err != nil || u.IsAdmin != (id == "old") {
			t.Fatalf("upgrade %s: %+v %v", id, u, err)
		}
	}
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil || result != "ok" {
		t.Fatalf("integrity: %s %v", result, err)
	}
	db.Close()
	again, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
}

func TestPlaylistImportRollsBackOnInsertFailure(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	user, err := r.CreateUser(ctx, "owner", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertTrack(ctx, &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: "/music/song.mp3"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.Exec(`CREATE TRIGGER reject_import BEFORE INSERT ON playlist_tracks BEGIN SELECT RAISE(ABORT,'simulated disk failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := r.ImportPlaylistBackup(ctx, user.ID, models.PlaylistBackup{Name: "Import", Tracks: []string{"/music/song.mp3"}}); err == nil {
		t.Fatal("expected insert failure")
	}
	lists, err := r.GetPlaylists(ctx, user.ID)
	if err != nil || len(lists) != 0 {
		t.Fatalf("partial playlist remains: %+v %v", lists, err)
	}
}
