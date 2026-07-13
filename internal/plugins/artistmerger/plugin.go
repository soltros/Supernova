package artistmerger

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
	"golang.org/x/crypto/bcrypt"
)

type ArtistMergerPlugin struct {
	repo      *database.Repository
	isRunning atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
}

func init() {
	// Register an instance directly (Register expects a plugins.Plugin value)
	plugins.Register(&ArtistMergerPlugin{})
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
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return nil
}

func (p *ArtistMergerPlugin) auth(next http.HandlerFunc) http.HandlerFunc {
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

func (p *ArtistMergerPlugin) SetupRoutes(mux *http.ServeMux) {
	// ServeMux patterns are path-only; route by method inside the handler.
	mux.HandleFunc("/api/plugins/artistmerger/run", p.auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleRunMerger(w, r)
	}))
}

func (p *ArtistMergerPlugin) handleRunMerger(w http.ResponseWriter, r *http.Request) {
	if !p.isRunning.CompareAndSwap(false, true) {
		http.Error(w, "Job already running", http.StatusConflict)
		return
	}
	go func() {
		defer p.isRunning.Store(false)
		p.runMergeJob(p.ctx)
	}()
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

func (p *ArtistMergerPlugin) runMergeJob(ctx context.Context) {
	log.Println("[ArtistMerger] Starting background merge job...")
	db := p.repo.DB()

	type artistData struct {
		id   string
		name string
	}
	
	groups := make(map[string][]artistData)
	limit := 1000
	offset := 0

	// 1. Fetch all artists with pagination
	for {
		rows, err := db.QueryContext(ctx, "SELECT id, name FROM artists LIMIT ? OFFSET ?", limit, offset)
		if err != nil {
			log.Printf("[ArtistMerger] Failed to fetch artists: %v\n", err)
			return
		}
		
		count := 0
		for rows.Next() {
			var a artistData
			if err := rows.Scan(&a.id, &a.name); err == nil {
				norm := normalizeName(a.name)
				if norm != "" {
					groups[norm] = append(groups[norm], a)
				}
			}
			count++
		}
		rows.Close()

		if count < limit {
			break
		}
		offset += limit
	}

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

				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					log.Printf("[ArtistMerger] Failed to start tx: %v\n", err)
					continue
				}

				// Update album_artists
				_, err = tx.ExecContext(ctx, "UPDATE album_artists SET artist_id = ? WHERE artist_id = ? AND album_id NOT IN (SELECT album_id FROM album_artists WHERE artist_id = ?)", canonical.id, a.id, canonical.id)
				if err != nil {
					tx.Rollback()
					continue
				}
				_, err = tx.ExecContext(ctx, "DELETE FROM album_artists WHERE artist_id = ?", a.id)
				if err != nil {
					tx.Rollback()
					continue
				}

				// Update track_artists
				_, err = tx.ExecContext(ctx, "UPDATE track_artists SET artist_id = ? WHERE artist_id = ? AND track_id NOT IN (SELECT track_id FROM track_artists WHERE artist_id = ?)", canonical.id, a.id, canonical.id)
				if err != nil {
					tx.Rollback()
					continue
				}
				_, err = tx.ExecContext(ctx, "DELETE FROM track_artists WHERE artist_id = ?", a.id)
				if err != nil {
					tx.Rollback()
					continue
				}

				// Delete duplicate artist
				_, err = tx.ExecContext(ctx, "DELETE FROM artists WHERE id = ?", a.id)
				if err != nil {
					tx.Rollback()
					continue
				}
				
				tx.Commit()
				mergeCount++
			}
		}
	}

	log.Printf("[ArtistMerger] Background merge job completed. Merged %d duplicate artists.\n", mergeCount)
}
