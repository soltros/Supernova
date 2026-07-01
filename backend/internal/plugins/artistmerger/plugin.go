package artistmerger

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type ArtistMergerPlugin struct {
	repo *database.Repository
}

func init() {
	plugins.Register("artistmerger", func() plugins.Plugin {
		return &ArtistMergerPlugin{}
	})
}

func (p *ArtistMergerPlugin) ID() string {
	return "artistmerger"
}

func (p *ArtistMergerPlugin) Name() string {
	return "Artist Merger"
}

func (p *ArtistMergerPlugin) Description() string {
	return "Automatically groups and merges misspelled or alternate artist names (e.g. 'Beatles' and 'The Beatles') into a single unified artist."
}

func (p *ArtistMergerPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *ArtistMergerPlugin) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/plugins/artistmerger/run", p.handleRunMerger)
}

func (p *ArtistMergerPlugin) handleRunMerger(w http.ResponseWriter, r *http.Request) {
	go p.runMergeJob()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "artist-merger job started in background"}`))
}

// normalizeName strips punctuation, lowercases, and removes common prefixes
func normalizeName(name string) string {
	lower := strings.ToLower(name)
	lower = strings.TrimPrefix(lower, "the ")
	lower = strings.TrimPrefix(lower, "a ")
	lower = strings.TrimPrefix(lower, "an ")
	
	var sb strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (p *ArtistMergerPlugin) runMergeJob() {
	log.Println("[ArtistMerger] Starting background merge job...")
	ctx := context.Background()
	db := p.repo.DB()

	// 1. Fetch all artists
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM artists")
	if err != nil {
		log.Printf("[ArtistMerger] Failed to fetch artists: %v\n", err)
		return
	}
	
	type artistData struct {
		id   string
		name string
	}
	
	groups := make(map[string][]artistData)
	
	for rows.Next() {
		var a artistData
		if err := rows.Scan(&a.id, &a.name); err == nil {
			norm := normalizeName(a.name)
			if norm != "" {
				groups[norm] = append(groups[norm], a)
			}
		}
	}
	rows.Close()

	mergeCount := 0

	// 2. Identify and merge duplicates
	for _, group := range groups {
		if len(group) > 1 {
			// Pick canonical artist (longest name string usually has better formatting, e.g. "The Beatles" over "Beatles")
			canonical := group[0]
			for _, a := range group {
				if len(a.name) > len(canonical.name) {
					canonical = a
				}
			}

			// Merge others into canonical
			for _, a := range group {
				if a.id == canonical.id {
					continue
				}
				
				log.Printf("[ArtistMerger] Merging '%s' into '%s'\n", a.name, canonical.name)

				// Update album_artists
				_, err := db.ExecContext(ctx, "UPDATE OR IGNORE album_artists SET artist_id = ? WHERE artist_id = ?", canonical.id, a.id)
				if err == nil {
					db.ExecContext(ctx, "DELETE FROM album_artists WHERE artist_id = ?", a.id)
				}

				// Update track_artists
				_, err = db.ExecContext(ctx, "UPDATE OR IGNORE track_artists SET artist_id = ? WHERE artist_id = ?", canonical.id, a.id)
				if err == nil {
					db.ExecContext(ctx, "DELETE FROM track_artists WHERE artist_id = ?", a.id)
				}

				// Delete duplicate artist
				db.ExecContext(ctx, "DELETE FROM artists WHERE id = ?", a.id)
				mergeCount++
			}
		}
	}

	log.Printf("[ArtistMerger] Background merge job completed. Merged %d duplicate artists.\n", mergeCount)
}
