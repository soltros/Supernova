package subsonic

import (
	"context"
	"encoding/json"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormAuthenticationExtensionsAndScrobble(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repo.CreateUser(ctx, "listener", string(hash))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertTrack(ctx, &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: "/music/song.flac", Format: "flac", DurationMs: 120000}); err != nil {
		t.Fatal(err)
	}
	tracks, err := repo.GetTracks(ctx, "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := &SubsonicPlugin{repo: repo}
	mux := http.NewServeMux()
	p.SetupRoutes(mux)
	request := func(path string, values url.Values) map[string]any {
		values.Set("f", "json")
		r := httptest.NewRequest("POST", path, strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var data map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("%s: %s %v", path, w.Body, err)
		}
		return data["subsonic-response"].(map[string]any)
	}
	auth := func() url.Values { return url.Values{"u": {"listener"}, "p": {"password"}} }
	if got := request("/rest/ping.view", auth())["status"]; got != "ok" {
		t.Fatalf("POST auth: %v", got)
	}
	extensions := request("/rest/getOpenSubsonicExtensions", url.Values{})
	if extensions["status"] != "ok" {
		t.Fatal("extensions need auth")
	}
	if _, ok := extensions["openSubsonicExtensions"].([]any); !ok {
		t.Fatalf("invalid extensions shape: %+v", extensions)
	}
	values := auth()
	values.Set("id", tracks[0].ID)
	values.Set("time", "1700000000000")
	if result := request("/rest/scrobble", values); result["status"] != "ok" {
		t.Fatalf("scrobble: %+v", result)
	}
	var count int
	var timestamp string
	if err := db.QueryRow("SELECT COUNT(*), listened_at FROM scrobbles WHERE user_id=?", user.ID).Scan(&count, &timestamp); err != nil {
		t.Fatal(err)
	}
	if count != 1 || !strings.HasPrefix(timestamp, "2023-11-14") {
		t.Fatalf("scrobble not recorded: %d %s", count, timestamp)
	}
	values.Set("submission", "false")
	request("/rest/scrobble", values)
	if err := db.QueryRow("SELECT COUNT(*) FROM scrobbles").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("now playing inserted a play")
	}
	random := request("/rest/getRandomSongs", auth())["randomSongs"].(map[string]any)["song"].([]any)
	if len(random) != 1 {
		t.Fatal("random songs still empty")
	}
}
