package api

import (
	"context"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"io"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscodingFormatsAndMissingEncoder(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	dir := t.TempDir()
	t.Setenv("MEDIA_PATH", dir)
	path := filepath.Join(dir, "source.wav")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=duration=0.2", "-y", path).CombinedOutput(); err != nil {
		t.Fatalf("audio fixture: %v %s", err, out)
	}
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	if err := repo.UpsertTrack(context.Background(), &models.TrackMetadata{Title: "Test", Album: "Album", Artist: "Artist", FilePath: path, Format: "wav", DurationMs: 200}); err != nil {
		t.Fatal(err)
	}
	tracks, err := repo.GetTracks(context.Background(), "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(repo, nil, nil, nil, nil)
	for _, format := range []string{"mp3", "aac", "ogg", "opus"} {
		t.Run(format, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/stream/test?format="+format, nil)
			r.SetPathValue("id", tracks[0].ID)
			w := httptest.NewRecorder()
			server.handleStreamTrack()(w, r)
			body, _ := io.ReadAll(w.Result().Body)
			if w.Code != 200 || len(body) < 100 || !strings.HasPrefix(w.Header().Get("Content-Type"), "audio/") {
				t.Fatalf("format %s: %d, %d bytes, %s", format, w.Code, len(body), w.Header().Get("Content-Type"))
			}
			if format == "aac" && (body[0] != 0xff || body[1]&0xf0 != 0xf0) {
				t.Fatal("AAC missing ADTS header")
			}
		})
	}
	t.Setenv("PATH", t.TempDir())
	r := httptest.NewRequest("GET", "/api/stream/test?format=mp3", nil)
	r.SetPathValue("id", tracks[0].ID)
	w := httptest.NewRecorder()
	server.handleStreamTrack()(w, r)
	if w.Code != 503 {
		t.Fatalf("missing ffmpeg returned %d", w.Code)
	}
}
