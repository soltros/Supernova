package podcasts

import (
	"bytes"
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type blockingTransport func(*http.Request) (*http.Response, error)

func (f blockingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestImportAdmissionCancellationAndShutdown(t *testing.T) {
	secret := strings.Repeat("s", 32)
	t.Setenv("JWT_SECRET", secret)
	t.Setenv("PODCAST_INDEX_API_KEY", "test")
	t.Setenv("PODCAST_INDEX_API_SECRET", "test")
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	user, err := repo.CreateUser(ctx, "alice", "hash")
	if err != nil {
		t.Fatal(err)
	}
	session, err := repo.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": user.ID, "jti": session, "kind": "session", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	p := &PodcastsPlugin{}
	if err := p.Init(plugins.PluginConfig{Context: ctx, Repo: repo}); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 1)
	p.client = &http.Client{Transport: blockingTransport(func(r *http.Request) (*http.Response, error) {
		started <- struct{}{}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	request := func(feeds int) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("file", "feeds.opml")
		if err != nil {
			t.Fatal(err)
		}
		file.Write([]byte("<opml><body>" + strings.Repeat(`<outline xmlUrl="https://example.test/feed"/>`, feeds) + "</body></opml>"))
		writer.Close()
		req := httptest.NewRequest("POST", "/api/plugins/podcasts/opml/import", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		p.handleImportOPML(w, req)
		return w
	}
	if got := request(1001); got.Code != 400 {
		t.Fatalf("oversized import: %d", got.Code)
	}
	if got := request(1); got.Code != 202 {
		t.Fatalf("import start: %d %s", got.Code, got.Body)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker never started")
	}
	if got := request(1); got.Code != 429 {
		t.Fatalf("concurrent worker admitted: %d", got.Code)
	}
	cancel()
	done := make(chan struct{})
	go func() { p.Shutdown(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("import worker did not stop")
	}
	if got := request(1); got.Code != 503 {
		t.Fatalf("import admitted after shutdown: %d", got.Code)
	}
}
