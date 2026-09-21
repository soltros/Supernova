package albummerger

import (
	"encoding/json"
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

func (p *AlbumMergerPlugin) ID() string { return "albummerger" }

func (p *AlbumMergerPlugin) Name() string { return "Album Merge Preview" }

func (p *AlbumMergerPlugin) Description() string {
	return "Read-only administrator preview of possible album groupings. Destructive apply is disabled pending a reviewed recovery and undo design."
}

func (p *AlbumMergerPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *AlbumMergerPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/albummerger/run", p.handleRunMerger)
	mux.HandleFunc("/api/plugins/albummerger/preview", p.handlePreview)
}

func (p *AlbumMergerPlugin) handleRunMerger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(
		w,
		"destructive album merging is disabled until a reviewed recovery/undo plan is approved; inspect /api/plugins/albummerger/preview",
		http.StatusConflict,
	)
}

func (p *AlbumMergerPlugin) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := p.repo.DB().QueryContext(r.Context(), `
		SELECT a.id,a.title,COALESCE(a.release_year,0),COALESCE(a.musicbrainz_id,''),
		       COALESCE(aa.artist_id,''),COALESCE(ar.name,''),
		       (SELECT COUNT(*) FROM tracks t WHERE t.album_id=a.id),
		       (SELECT COUNT(*) FROM hearts h WHERE h.entity_type='album' AND h.entity_id=a.id)
		FROM albums a
		LEFT JOIN album_artists aa ON aa.album_id=a.id AND aa.role='primary'
		LEFT JOIN artists ar ON ar.id=aa.artist_id
		ORDER BY ar.name,a.title,a.id
	`)
	if err != nil {
		http.Error(w, "preview query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type item struct {
		ID, Title, MBID, ArtistID, Artist string
		Year, Tracks, Hearts              int
	}

	groups := map[string][]item{}
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.ID, &x.Title, &x.Year, &x.MBID, &x.ArtistID, &x.Artist, &x.Tracks, &x.Hearts); err != nil {
			http.Error(w, "preview scan failed", http.StatusInternalServerError)
			return
		}
		groups[normalizeName(x.Title)+"|"+x.ArtistID] = append(groups[normalizeName(x.Title)+"|"+x.ArtistID], x)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "preview iteration failed", http.StatusInternalServerError)
		return
	}

	out := make([][]item, 0)
	for _, group := range groups {
		if len(group) > 1 {
			out = append(out, group)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"mode":    "read-only",
		"warning": "edition/remaster/clean/explicit distinctions are not approved for automatic merge",
		"candidates": out,
	})
}

// normalizeName is intentionally preview-only. Matching normalized names is
// never sufficient authorization for a destructive merge.
func normalizeName(name string) string {
	lower := strings.ToLower(name)
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
