package albummerger

import (
	"context"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type AlbumMergerPlugin struct {
	repo *database.Repository
}

func init() {
	plugins.Register(&AlbumMergerPlugin{})
}

func (p *AlbumMergerPlugin) ID() string {
	return "albummerger"
}

func (p *AlbumMergerPlugin) Name() string {
	return "Album Merger"
}

func (p *AlbumMergerPlugin) Description() string {
	return "Automatically groups and merges alternate album releases or multi-disc versions into a single unified album based on title similarity."
}

func (p *AlbumMergerPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *AlbumMergerPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/albummerger/run", p.handleRunMerger)
}

func (p *AlbumMergerPlugin) handleRunMerger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go p.runMergeJob()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "album-merger job started in background"}`))
}

// normalizeName strips punctuation, lowercases, and removes common variations
func normalizeName(name string) string {
	lower := strings.ToLower(name)
	
	// Strip common suffixes
	suffixes := []string{
		" (deluxe)", " (deluxe edition)", " [deluxe edition]", 
		" (remastered)", " [remastered]", " - remastered",
		" (bonus track version)", " (explicit)", " (clean)",
		" disc 1", " disc 2", " cd 1", " cd 2",
	}
	for _, suffix := range suffixes {
		lower = strings.ReplaceAll(lower, suffix, "")
	}

	var sb strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (p *AlbumMergerPlugin) runMergeJob() {
	log.Println("[AlbumMerger] Starting background merge job...")
	ctx := context.Background()
	db := p.repo.DB()

	// 1. Fetch all albums with their primary artist
	rows, err := db.QueryContext(ctx, `
		SELECT a.id, a.title, aa.artist_id
		FROM albums a
		LEFT JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
	`)
	if err != nil {
		log.Printf("[AlbumMerger] Failed to fetch albums: %v\n", err)
		return
	}
	
	type albumData struct {
		id       string
		title    string
		artistID string
	}
	
	groups := make(map[string][]albumData)
	
	for rows.Next() {
		var a albumData
		var artistID *string
		if err := rows.Scan(&a.id, &a.title, &artistID); err == nil {
			if artistID != nil {
				a.artistID = *artistID
			}
			norm := normalizeName(a.title)
			if norm != "" {
				// Group by normalized title + artist ID to prevent cross-artist merging
				key := norm + "|" + a.artistID
				groups[key] = append(groups[key], a)
			}
		}
	}
	rows.Close()

	mergeCount := 0

	// 2. Identify and merge duplicates
	for _, group := range groups {
		if len(group) > 1 {
			// Pick canonical album (shortest name is usually the clean/original one, unlike artists)
			canonical := group[0]
			for _, a := range group {
				if len(a.title) < len(canonical.title) {
					canonical = a
				}
			}

			// Merge others into canonical
			for _, a := range group {
				if a.id == canonical.id {
					continue
				}
				
				log.Printf("[AlbumMerger] Merging '%s' into '%s'\n", a.title, canonical.title)

				// Move tracks to canonical album
				db.ExecContext(ctx, "UPDATE tracks SET album_id = ? WHERE album_id = ?", canonical.id, a.id)

				// Move album-level hearts to canonical album
				_, err := db.ExecContext(ctx, "UPDATE OR IGNORE hearts SET entity_id = ? WHERE entity_type = 'album' AND entity_id = ?", canonical.id, a.id)
				if err == nil {
					db.ExecContext(ctx, "DELETE FROM hearts WHERE entity_type = 'album' AND entity_id = ?", a.id)
				}

				// Delete duplicate album (will cascade to album_artists)
				db.ExecContext(ctx, "DELETE FROM albums WHERE id = ?", a.id)
				mergeCount++
			}
		}
	}

	// 3. Deduplicate tracks on merged canonical albums to prevent duplicates from deluxe versions
	dedupeCount := 0
	for _, group := range groups {
		if len(group) > 1 {
			// Find canonical ID
			canonical := group[0]
			for _, a := range group {
				if len(a.title) < len(canonical.title) {
					canonical = a
				}
			}

			// Deduplicate tracks for this canonical album based on track name
			rowsT, err := db.QueryContext(ctx, "SELECT id, title, bitrate FROM tracks WHERE album_id = ? ORDER BY title, bitrate DESC", canonical.id)
			if err == nil {
				type trackData struct {
					id      string
					title   string
					bitrate int
				}
				var tData []trackData
				for rowsT.Next() {
					var t trackData
					if err := rowsT.Scan(&t.id, &t.title, &t.bitrate); err == nil {
						tData = append(tData, t)
					}
				}
				rowsT.Close()

				seen := make(map[string]bool)
				for _, t := range tData {
					normT := strings.ToLower(strings.TrimSpace(t.title))
					if seen[normT] {
						// Delete duplicate lower-quality track
						log.Printf("[AlbumMerger] Removing duplicate track '%s' from canonical album '%s'\n", t.title, canonical.title)
						db.ExecContext(ctx, "DELETE FROM tracks WHERE id = ?", t.id)
						dedupeCount++
					} else {
						seen[normT] = true
					}
				}
			}
		}
	}

	log.Printf("[AlbumMerger] Background merge job completed. Merged %d duplicate albums and removed %d duplicate tracks.\n", mergeCount, dedupeCount)
}
