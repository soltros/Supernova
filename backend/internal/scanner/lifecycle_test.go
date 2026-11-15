package scanner

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/jobs"
	"github.com/soltros/Supernova/internal/models"
)

func TestCancelledScanPreservesLibrary(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	root := t.TempDir()
	if err := repo.UpsertTrack(context.Background(), &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: filepath.Join(root, "missing.wav"), Format: "wav"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sc, err := New(ctx, root, repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Close()
	if err := sc.FullScan(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled scan: %v", err)
	}
	tracks, err := repo.GetTracks(context.Background(), "", "", 100, 0)
	if err != nil || len(tracks) != 1 {
		t.Fatalf("cancelled scan deleted library: %v %v", tracks, err)
	}
}
func TestOverflowSchedulesRecoveryScan(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root := t.TempDir()
	repo := database.NewRepository(db)
	if output, err := exec.Command("ffmpeg", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "sine=duration=0.1", "-y", filepath.Join(root, "new.wav")).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	sc, err := New(ctx, root, repo, nil, jobs.New(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Close()
	// Fill the admission queue without workers to deterministically exercise overflow.
	stub := &Scanner{ctx: ctx, realtimeEvents: make(chan realtimeEvent, 1)}
	stub.enqueueEvent("one", false)
	stub.enqueueEvent("two", false)
	if !stub.rescanNeeded.Load() {
		t.Fatal("dropped event did not schedule recovery")
	}
	sc.rescanNeeded.Store(true)
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		tracks, err := repo.GetTracks(ctx, "", "", 100, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(tracks) == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("overflow recovery never reconciled library")
}
func TestEnricherJoinsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	e := NewEnricher(nil, nil, nil)
	e.Start(ctx)
	cancel()
	done := make(chan struct{})
	go func() { e.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("enricher did not stop")
	}
}
