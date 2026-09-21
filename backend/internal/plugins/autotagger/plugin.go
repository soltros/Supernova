package autotagger

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/jobs"
	"github.com/soltros/Supernova/internal/models"
	"github.com/soltros/Supernova/internal/plugins"
)

type AutoTaggerPlugin struct {
	repo *database.Repository
	jobs *jobs.Supervisor
}

func init() {
	plugins.Register(&AutoTaggerPlugin{})
}

func (p *AutoTaggerPlugin) ID() string { return "autotagger" }

func (p *AutoTaggerPlugin) Name() string { return "Auto-Tagger" }

func (p *AutoTaggerPlugin) Description() string {
	return "Automatically infers and fixes track metadata in the Supernova database by analyzing file paths."
}

func (p *AutoTaggerPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	p.jobs = config.Jobs
	return nil
}

func (p *AutoTaggerPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/plugins/autotagger/run", p.handleRunTagger)
}

func (p *AutoTaggerPlugin) handleRunTagger(w http.ResponseWriter, r *http.Request) {
	if p.jobs == nil || !p.jobs.GoMutation("autotagger", p.runTaggingJob) {
		http.Error(w, "another library mutation job is already running", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"auto-tagger job started in background"}`))
}

func (p *AutoTaggerPlugin) runTaggingJob(ctx context.Context) error {
	log.Println("[AutoTagger] Starting background tagging job...")
	rows, err := p.repo.DB().QueryContext(ctx, `
		SELECT t.file_path, t.id, COALESCE(t.duration_ms,0), COALESCE(t.format,''),
		       COALESCE(t.bitrate,0), COALESCE(a.cover_art_path,'')
		FROM tracks t
		LEFT JOIN albums a ON t.album_id = a.id
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists ar ON ta.artist_id = ar.id
		WHERE ar.name = 'Unknown Artist' OR a.title = 'Unknown Album' OR t.title LIKE 'Track %'
		GROUP BY t.id
	`)
	if err != nil {
		return fmt.Errorf("query tracks: %w", err)
	}
	defer rows.Close()

	var tracksToUpdate []models.TrackMetadata
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var path, id, format, coverArt string
		var durationMs, bitrate int
		if err := rows.Scan(&path, &id, &durationMs, &format, &bitrate, &coverArt); err != nil {
			return fmt.Errorf("scan track candidate: %w", err)
		}

		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) < 3 {
			continue
		}
		filename := parts[len(parts)-1]
		albumDir := parts[len(parts)-2]
		artistDir := parts[len(parts)-3]
		ext := filepath.Ext(filename)
		baseName := strings.TrimSuffix(filename, ext)

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
		if artistDir == "music" || albumDir == "music" || artistDir == "" {
			continue
		}
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
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate track candidates: %w", err)
	}

	count := 0
	for i := range tracksToUpdate {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.repo.UpsertTrack(ctx, &tracksToUpdate[i]); err != nil {
			return fmt.Errorf("update %s: %w", tracksToUpdate[i].FilePath, err)
		}
		count++
	}
	log.Printf("[AutoTagger] Background tagging job completed. Fixed %d tracks.", count)
	return nil
}
