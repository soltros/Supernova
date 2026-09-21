package artistmerger

import (
	"encoding/json"
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
	plugins.Register(&ArtistMergerPlugin{})
}

func (p *ArtistMergerPlugin) ID() string { return "artistmerger" }

func (p *ArtistMergerPlugin) Name() string { return "Artist Merge Preview" }

func (p *ArtistMergerPlugin) Description() string {
	return "Read-only administrator preview of possible artist groupings. Destructive apply is disabled pending a reviewed recovery and undo design."
}

func (p *ArtistMergerPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *ArtistMergerPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/artistmerger/run", p.handleRunMerger)
	mux.HandleFunc("/api/plugins/artistmerger/preview", p.handlePreview)
}

func (p *ArtistMergerPlugin) handleRunMerger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(
		w,
		"destructive artist merging is disabled until a reviewed recovery/undo plan is approved; inspect /api/plugins/artistmerger/preview",
		http.StatusConflict,
	)
}

func (p *ArtistMergerPlugin) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := p.repo.DB().QueryContext(r.Context(), `
		SELECT a.id,a.name,COALESCE(a.musicbrainz_id,''),
		       (SELECT COUNT(*) FROM album_artists aa WHERE aa.artist_id=a.id),
		       (SELECT COUNT(*) FROM track_artists ta WHERE ta.artist_id=a.id),
		       (SELECT COUNT(*) FROM hearts h WHERE h.entity_type='artist' AND h.entity_id=a.id)
		FROM artists a
		ORDER BY a.name,a.id
	`)
	if err != nil {
		http.Error(w, "preview query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type item struct {
		ID, Name, MBID       string
		Albums, Tracks, Hearts int
	}

	groups := map[string][]item{}
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.ID, &x.Name, &x.MBID, &x.Albums, &x.Tracks, &x.Hearts); err != nil {
			http.Error(w, "preview scan failed", http.StatusInternalServerError)
			return
		}
		key := normalizeName(x.Name)
		if key != "" {
			groups[key] = append(groups[key], x)
		}
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
		"mode":       "read-only",
		"warning":    "normalized-name matches are candidates only, not merge authorization",
		"candidates": out,
	})
}

// normalizeName is intentionally preview-only. Matching normalized names is
// never sufficient authorization for a destructive merge.
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
