package deduper

import (
	"encoding/json"
	"context"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type DeduperPlugin struct {
	repo    *database.Repository
	running atomic.Bool
}

func init() {
	// Register the plugin instance. plugins.Register expects a plugins.Plugin value.
	plugins.Register(&DeduperPlugin{})
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

func (p *DeduperPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/deduper/run", p.handleRunDeduper)
	mux.HandleFunc("/api/plugins/deduper/preview", p.handlePreview)
}

func (p *DeduperPlugin) handleRunDeduper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w,"method not allowed",http.StatusMethodNotAllowed); return }
	http.Error(w,"destructive deduplication is disabled until a reviewed recovery/undo plan is approved; inspect /api/plugins/deduper/preview",http.StatusConflict)
}

func (p *DeduperPlugin) handlePreview(w http.ResponseWriter,r *http.Request){
	if r.Method!=http.MethodGet{http.Error(w,"method not allowed",http.StatusMethodNotAllowed);return}
	rows,err:=p.repo.DB().QueryContext(r.Context(),`
		SELECT t.id,t.title,t.album_id,a.title,COALESCE(ar.id,''),COALESCE(ar.name,''),
		       COALESCE(t.disc_number,0),COALESCE(t.duration_ms,0),COALESCE(t.format,''),
		       COALESCE(t.bitrate,0),t.file_path,COALESCE(t.file_fingerprint,''),
		       (SELECT COUNT(*) FROM playlist_tracks pt WHERE pt.track_id=t.id),
		       (SELECT COUNT(*) FROM hearts h WHERE h.entity_type='track' AND h.entity_id=t.id),
		       (SELECT COUNT(*) FROM scrobbles s WHERE s.track_id=t.id)
		FROM tracks t JOIN albums a ON a.id=t.album_id
		LEFT JOIN track_artists ta ON ta.track_id=t.id
		LEFT JOIN artists ar ON ar.id=ta.artist_id
		ORDER BY ar.name,a.title,t.disc_number,t.track_number,t.title,t.id
	`)
	if err!=nil{http.Error(w,"preview query failed",500);return}
	defer rows.Close()
	type item struct{ID,Title,AlbumID,Album,ArtistID,Artist,Format,FilePath,Fingerprint string;Disc,Duration,Bitrate,PlaylistEntries,Hearts,Scrobbles int}
	groups:=map[string][]item{}
	for rows.Next(){var x item;if err:=rows.Scan(&x.ID,&x.Title,&x.AlbumID,&x.Album,&x.ArtistID,&x.Artist,&x.Disc,&x.Duration,&x.Format,&x.Bitrate,&x.FilePath,&x.Fingerprint,&x.PlaylistEntries,&x.Hearts,&x.Scrobbles);err!=nil{http.Error(w,"preview scan failed",500);return};key:=strings.ToLower(strings.TrimSpace(x.ArtistID+"|"+x.AlbumID+"|"+x.Title));groups[key]=append(groups[key],x)}
	out:=make([][]item,0)
	for _,g:=range groups{if len(g)>1{out=append(out,g)}}
	w.Header().Set("Content-Type","application/json");_ = json.NewEncoder(w).Encode(map[string]any{"mode":"read-only","candidates":out})
}

func (p *DeduperPlugin) runDeduperJob() {
	log.Println("[Deduper] Starting background duplicate removal job...")
	ctx := context.Background()
	db := p.repo.DB()

	// Find tracks that have the same title and album, but keep the one with the highest bitrate
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.title, a.title, t.bitrate, t.file_path
		FROM tracks t
		JOIN albums a ON t.album_id = a.id
		ORDER BY t.title, a.title, t.bitrate DESC
	`)
	if err != nil {
		log.Printf("[Deduper] Failed to fetch tracks: %v\n", err)
		return
	}

	type trackData struct {
		id       string
		title    string
		album    string
		bitrate  int
		filePath string
	}

	var allTracks []trackData
	for rows.Next() {
		var t trackData
		if err := rows.Scan(&t.id, &t.title, &t.album, &t.bitrate, &t.filePath); err == nil {
			allTracks = append(allTracks, t)
		}
	}
	rows.Close()

	seen := make(map[string]bool)
	deleteCount := 0

	for _, t := range allTracks {
		key := strings.ToLower(t.title) + "|" + strings.ToLower(t.album)
		if seen[key] {
			// It's a duplicate and since we ordered by bitrate DESC, this is the lower quality one.
			// We delete it from the database so it's "hidden" from the UI.
			log.Printf("[Deduper] Hiding duplicate track: %s (Bitrate: %d)\n", t.title, t.bitrate)
			db.ExecContext(ctx, "INSERT OR IGNORE INTO ignored_files (file_path, reason) VALUES (?, 'deduper')", t.filePath)
			db.ExecContext(ctx, "DELETE FROM tracks WHERE id = ?", t.id)
			deleteCount++
		} else {
			seen[key] = true
		}
	}

	log.Printf("[Deduper] Background deduper job completed. Hidden %d duplicate tracks.\n", deleteCount)
}
