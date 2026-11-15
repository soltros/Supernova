package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/jobs"
	"github.com/soltros/Supernova/internal/scanner"
)

// Opt-in integration soak: uses only a disposable synthetic library and database.
func TestProductionMemorySoak(t *testing.T) {
	seconds, _ := strconv.Atoi(os.Getenv("SUPERNOVA_SOAK_SECONDS"))
	if seconds <= 0 {
		t.Skip("set SUPERNOVA_SOAK_SECONDS to run profiling")
	}
	out := os.Getenv("SUPERNOVA_SOAK_OUTPUT")
	if out == "" {
		out = t.TempDir()
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	root := t.TempDir()
	music := filepath.Join(root, "music")
	if err := os.Mkdir(music, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEDIA_PATH", music)
	source := filepath.Join(music, "source.wav")
	if output, err := exec.Command("ffmpeg", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "sine=duration=0.1", "-y", source).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	total, _ := strconv.Atoi(os.Getenv("SUPERNOVA_SOAK_TRACKS"))
	if total <= 0 {
		total = 64
	}
	if total > 10000 {
		t.Fatal("maximum fixture size is 10000 tracks")
	}
	for i := 0; i < total-1; i++ {
		if err := os.WriteFile(filepath.Join(music, fmt.Sprintf("song-%d.wav", i)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	db, err := database.Init(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	supervisor := jobs.New(ctx)
	sc, err := scanner.New(ctx, music, repo, nil, supervisor)
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Close()
	sc.Watch()
	if err := sc.FullScan(); err != nil {
		t.Fatal(err)
	}
	user, err := repo.CreateUser(ctx, "profiler", "hash")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(repo, nil, nil, sc, nil)
	token, _, err := server.issueSessionToken(ctx, user.ID, "hash")
	if err != nil {
		t.Fatal(err)
	}
	var trackID string
	if err := db.QueryRow(`SELECT id FROM tracks LIMIT 1`).Scan(&trackID); err != nil {
		t.Fatal(err)
	}
	paths := []string{"/api/tracks?limit=64", "/api/albums", "/api/artists", "/api/dashboard", "/api/playlists", "/api/hearts", "/api/stream/" + trackID}
	batch := func() {
		for _, path := range paths {
			r := httptest.NewRequest("GET", path, nil)
			r.Header.Set("Authorization", "Bearer "+token)
			r.Header.Set("Range", "bytes=0-255")
			w := httptest.NewRecorder()
			server.ServeHTTP(w, r)
			if w.Code != 200 && w.Code != 206 {
				t.Fatalf("%s: %d %s", path, w.Code, w.Body)
			}
		}
	}
	for i := 0; i < 10; i++ {
		batch()
	}
	profile := func(name string) runtime.MemStats {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		f, err := os.Create(filepath.Join(out, name+".heap.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			t.Fatal(err)
		}
		f.Close()
		return m
	}
	before := profile("before")
	beforeG := runtime.NumGoroutine()
	cycles := 0
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	for time.Now().Before(deadline) {
		for i := 0; i < 25; i++ {
			batch()
		}
		if err := sc.FullScan(); err != nil {
			t.Fatal(err)
		}
		cycles++
	}
	after := profile("after")
	afterG := runtime.NumGoroutine()
	summary := map[string]any{"seconds": seconds, "cycles": cycles, "requests": cycles * 25 * len(paths), "tracks": total, "heap_before": before.HeapAlloc, "heap_after": after.HeapAlloc, "goroutines_before": beforeG, "goroutines_after": afterG}
	encoded, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "summary.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log(string(encoded))
	if after.HeapAlloc > before.HeapAlloc+32<<20 {
		t.Error("retained heap grew by more than 32 MiB")
	}
	if afterG > beforeG+10 {
		t.Error("goroutines did not stabilize")
	}
}
