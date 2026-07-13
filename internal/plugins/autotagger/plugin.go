package autotagger

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"github.com/soltros/Supernova/internal/plugins"
	"golang.org/x/crypto/bcrypt"
)

type AutoTaggerPlugin struct {
	repo      *database.Repository
	isRunning atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
}

func init() {
	plugins.Register(&AutoTaggerPlugin{})
}

func (p *AutoTaggerPlugin) ID() string {
	return "autotagger"
}

func (p *AutoTaggerPlugin) Name() string {
	return "Auto-Tagger"
}

func (p *AutoTaggerPlugin) Description() string {
	return "Automatically infers and fixes track metadata in the Supernova database by analyzing file paths."
}

func (p *AutoTaggerPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return nil
}

func (p *AutoTaggerPlugin) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, pwd, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, hash, err := p.repo.GetUserByUsername(r.Context(), u)
		if err != nil || user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (p *AutoTaggerPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/autotagger/run", p.auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleRunTagger(w, r)
	}))
}

func (p *AutoTaggerPlugin) handleRunTagger(w http.ResponseWriter, r *http.Request) {
	if !p.isRunning.CompareAndSwap(false, true) {
		http.Error(w, "Job already running", http.StatusConflict)
		return
	}
	go func() {
		defer p.isRunning.Store(false)
		p.runTaggingJob(p.ctx)
	}()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "auto-tagger job started in background"}`))
}

func (p *AutoTaggerPlugin) runTaggingJob(ctx context.Context) {
	log.Println("[AutoTagger] Starting background tagging job...")
	db := p.repo.DB()
	trackPrefixRegex := regexp.MustCompile(`^\d+[\s\-\.]+(.*)`)

	// Query tracks with "Unknown Artist", "Unknown Album", or titles like "Track %"
	rows, err := db.QueryContext(ctx, `
		SELECT t.file_path, t.id, t.duration_ms, t.format, t.bitrate, a.cover_art_path 
		FROM tracks t
		JOIN albums a ON t.album_id = a.id
		JOIN artists ar ON a.artist_id = ar.id
		WHERE ar.name = 'Unknown Artist' OR a.title = 'Unknown Album' OR t.title LIKE 'Track %'
	`)
	if err != nil {
		log.Printf("[AutoTagger] Failed to query tracks: %v\n", err)
		return
	}
	defer rows.Close()

	var tracksToUpdate []models.TrackMetadata
	for rows.Next() {
		var path, id, format, coverArt string
		var durationMs, bitrate int
		if err := rows.Scan(&path, &id, &durationMs, &format, &bitrate, &coverArt); err != nil {
			continue
		}

		// Try to parse file path: e.g. /music/Artist/Album/01 - Title.mp3
		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) >= 3 {
			filename := parts[len(parts)-1]
			albumDir := parts[len(parts)-2]
			artistDir := parts[len(parts)-3]

			// Strip extension
			ext := filepath.Ext(filename)
			baseName := strings.TrimSuffix(filename, ext)
			
			// Remove leading track numbers (e.g., "01 - Song" -> "Song", "1. Song" -> "Song")
			title := baseName
			if matches := trackPrefixRegex.FindStringSubmatch(title); len(matches) > 1 {
				title = matches[1]
			}
			if title == "" {
				title = baseName
			}
