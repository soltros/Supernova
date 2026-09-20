package api

import (
	"github.com/soltros/Supernova/internal/database"
	"image"
	"image/jpeg"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAlbumArtRejectsActiveContentAndOutsidePaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ART_CACHE_PATH", root)
	t.Setenv("MEDIA_PATH", root)
	db, err := database.Init(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := NewServer(database.NewRepository(db), nil, nil, nil, nil)
	jpg := filepath.Join(root, "valid.jpg")
	f, err := os.Create(jpg)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	f.Close()
	html := filepath.Join(root, "legacy.bin")
	os.WriteFile(html, []byte("<html><script>alert(1)</script></html>"), 0600)
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	bytes, _ := os.ReadFile(jpg)
	os.WriteFile(outside, bytes, 0600)
	for _, tc := range []struct {
		path string
		want int
	}{{jpg, 200}, {html, 404}, {outside, 404}} {
		_, err := db.Exec(`INSERT OR REPLACE INTO albums(id,title,release_year,musicbrainz_id,cover_art_path,bio) VALUES ('test','Art',0,'',?,'')`, tc.path)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/api/art/album/test", nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s: got %d, body %s", tc.path, rec.Code, rec.Body)
		}
		if rec.Code == 200 && rec.Header().Get("Content-Type") != "image/jpeg" {
			t.Fatal("wrong content type")
		}
	}
}
