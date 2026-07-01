package autotagger

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"github.com/soltros/Supernova/internal/plugins"
)

type AutoTaggerPlugin struct {
	repo *database.Repository
}

func init() {
	plugins.Register("autotagger", func() plugins.Plugin {
		return &AutoTaggerPlugin{}
	})
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
	return nil
}

func (p *AutoTaggerPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/plugins/autotagger/run", p.handleRunTagger)
}

func (p *AutoTaggerPlugin) handleRunTagger(w http.ResponseWriter, r *http.Request) {
	go p.runTaggingJob()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "auto-tagger job started in background"}`))
}

func (p *AutoTaggerPlugin) runTaggingJob() {
	log.Println("[AutoTagger] Starting background tagging job...")
	ctx := context.Background()
	db := p.repo.DB()

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
			for i, char := range title {
				if char < '0' || char > '9' {
					title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(title[i:]), "-"))
					break
				}
			}
			if title == "" {
				title = baseName
			}

			if artistDir != "music" && albumDir != "music" && artistDir != "" {
				tracksToUpdate = append(tracksToUpdate, models.TrackMetadata{
					Title:        title,
					Album:        albumDir,
					Artist:       artistDir,
					AlbumArtist:  artistDir,
					DurationMs:   durationMs,
					Format:       format,
					Bitrate:      bitrate,
					FilePath:     path,
					CoverArtPath: coverArt,
				})
			}
		}
	}
	rows.Close()

	count := 0
	for _, meta := range tracksToUpdate {
		if err := p.repo.UpsertTrack(ctx, &meta); err == nil {
			count++
		}
	}

	log.Printf("[AutoTagger] Background tagging job completed. Fixed %d tracks.\n", count)
}
