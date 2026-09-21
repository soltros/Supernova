package deduper

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type DeduperPlugin struct {
	repo *database.Repository
}

func init() {
	plugins.Register(&DeduperPlugin{})
}

func (p *DeduperPlugin) ID() string { return "deduper" }

func (p *DeduperPlugin) Name() string { return "Duplicate Preview" }

func (p *DeduperPlugin) Description() string {
	return "Read-only administrator preview of duplicate-track candidates and affected user relationships. Destructive apply is disabled pending a reviewed recovery and undo design."
}

func (p *DeduperPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *DeduperPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/deduper/run", p.handleRunDeduper)
	mux.HandleFunc("/api/plugins/deduper/preview", p.handlePreview)
}

func (p *DeduperPlugin) handleRunDeduper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(
		w,
		"destructive deduplication is disabled until a reviewed recovery/undo plan is approved; inspect /api/plugins/deduper/preview",
		http.StatusConflict,
	)
}

func (p *DeduperPlugin) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := p.repo.DB().QueryContext(r.Context(), `
		SELECT t.id,t.title,t.album_id,a.title,COALESCE(ar.id,''),COALESCE(ar.name,''),
		       COALESCE(t.disc_number,0),COALESCE(t.duration_ms,0),COALESCE(t.format,''),
		       COALESCE(t.bitrate,0),t.file_path,COALESCE(t.file_fingerprint,''),
		       (SELECT COUNT(*) FROM playlist_tracks pt WHERE pt.track_id=t.id),
		       (SELECT COUNT(*) FROM hearts h WHERE h.entity_type='track' AND h.entity_id=t.id),
		       (SELECT COUNT(*) FROM scrobbles s WHERE s.track_id=t.id)
		FROM tracks t
		JOIN albums a ON a.id=t.album_id
		LEFT JOIN track_artists ta ON ta.track_id=t.id
		LEFT JOIN artists ar ON ar.id=ta.artist_id
		ORDER BY ar.name,a.title,t.disc_number,t.track_number,t.title,t.id
	`)
	if err != nil {
		http.Error(w, "preview query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type item struct {
		ID, Title, AlbumID, Album, ArtistID, Artist, Format, FilePath, Fingerprint string
		Disc, Duration, Bitrate, PlaylistEntries, Hearts, Scrobbles                  int
	}

	groups := map[string][]item{}
	for rows.Next() {
		var x item
		if err := rows.Scan(
			&x.ID, &x.Title, &x.AlbumID, &x.Album, &x.ArtistID, &x.Artist,
			&x.Disc, &x.Duration, &x.Format, &x.Bitrate, &x.FilePath, &x.Fingerprint,
			&x.PlaylistEntries, &x.Hearts, &x.Scrobbles,
		); err != nil {
			http.Error(w, "preview scan failed", http.StatusInternalServerError)
			return
		}
		key := strings.ToLower(strings.TrimSpace(x.ArtistID + "|" + x.AlbumID + "|" + x.Title))
		groups[key] = append(groups[key], x)
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
		"warning":    "candidate grouping is not authorization to merge or delete",
		"candidates": out,
	})
}
