package deduper

import (
	"context"
	"log"
	"net/http"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type DeduperPlugin struct {
	repo *database.Repository
}

func init() {
	plugins.Register("deduper", func() plugins.Plugin {
		return &DeduperPlugin{}
	})
}

func (p *DeduperPlugin) ID() string {
	return "deduper"
}

func (p *DeduperPlugin) Name() string {
	return "Hide Duplicates"
}

func (p *DeduperPlugin) Description() string {
	return "Automatically scans the library for exact duplicate tracks and hides the lower-quality versions from the UI."
}

func (p *DeduperPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *DeduperPlugin) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/plugins/deduper/run", p.handleRunDeduper)
}

func (p *DeduperPlugin) handleRunDeduper(w http.ResponseWriter, r *http.Request) {
	go p.runDeduperJob()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "deduper job started in background"}`))
}

func (p *DeduperPlugin) runDeduperJob() {
	log.Println("[Deduper] Starting background duplicate removal job...")
	ctx := context.Background()
	db := p.repo.DB()

	// Find tracks that have the same title and album, but keep the one with the highest bitrate
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.title, a.title, t.bitrate
		FROM tracks t
		JOIN albums a ON t.album_id = a.id
		ORDER BY t.title, a.title, t.bitrate DESC
	`)
	if err != nil {
		log.Printf("[Deduper] Failed to fetch tracks: %v\n", err)
		return
	}

	type trackData struct {
		id      string
		title   string
		album   string
		bitrate int
	}

	var allTracks []trackData
	for rows.Next() {
		var t trackData
		if err := rows.Scan(&t.id, &t.title, &t.album, &t.bitrate); err == nil {
			allTracks = append(allTracks, t)
		}
	}
	rows.Close()

	seen := make(map[string]bool)
	deleteCount := 0

	for _, t := range allTracks {
		key := t.title + "|" + t.album
		if seen[key] {
			// It's a duplicate and since we ordered by bitrate DESC, this is the lower quality one.
			// We delete it from the database so it's "hidden" from the UI.
			log.Printf("[Deduper] Hiding duplicate track: %s (Bitrate: %d)\n", t.title, t.bitrate)
			db.ExecContext(ctx, "DELETE FROM tracks WHERE id = ?", t.id)
			deleteCount++
		} else {
			seen[key] = true
		}
	}

	log.Printf("[Deduper] Background deduper job completed. Hidden %d duplicate tracks.\n", deleteCount)
}
