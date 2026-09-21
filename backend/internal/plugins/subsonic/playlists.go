package subsonic

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/soltros/Supernova/internal/models"
)

func (p *SubsonicPlugin) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil { p.writeError(w,r,0,"Not authenticated"); return }
	if err := r.ParseForm(); err != nil { p.writeError(w,r,10,"Invalid request parameters"); return }

	name := strings.TrimSpace(r.FormValue("name"))
	playlistID := strings.TrimSpace(r.FormValue("playlistId"))
	songIDs := append([]string(nil), r.Form["songId"]...)
	if name == "" && playlistID == "" {
		p.writeError(w,r,10,"Required parameter is missing: name or playlistId")
		return
	}

	var playlist *models.Playlist
	var err error
	if playlistID != "" {
		playlist, err = p.repo.ReplacePlaylist(r.Context(),u.ID,playlistID,name,songIDs)
	} else {
		playlist, err = p.repo.CreatePlaylistWithTracks(r.Context(),u.ID,name,songIDs)
	}
	if err != nil {
		p.writeError(w,r,70,err.Error())
		return
	}
	count, duration, err := p.repo.PlaylistStats(r.Context(),u.ID,playlist.ID)
	if err != nil { p.writeError(w,r,0,"Failed to read playlist state"); return }

	now := time.Now().UTC().Format(time.RFC3339)
	p.writeResponse(w,r,map[string]interface{}{
		"playlist":map[string]interface{}{
			"id":playlist.ID, "name":playlist.Name, "owner":u.Username,
			"public":false, "songCount":count, "duration":duration,
			"created":now, "changed":now,
		},
	})
}

func (p *SubsonicPlugin) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil { p.writeError(w,r,0,"Not authenticated"); return }
	if err := r.ParseForm(); err != nil { p.writeError(w,r,10,"Invalid request parameters"); return }

	playlistID := strings.TrimSpace(r.FormValue("playlistId"))
	if playlistID == "" { p.writeError(w,r,10,"Required parameter is missing: playlistId"); return }

	var name *string
	if values, exists := r.Form["name"]; exists {
		v := ""
		if len(values) > 0 { v = values[len(values)-1] }
		name = &v
	}
	add := append([]string(nil),r.Form["songIdToAdd"]...)
	remove := make([]int,0,len(r.Form["songIndexToRemove"]))
	for _, raw := range r.Form["songIndexToRemove"] {
		idx, err := strconv.Atoi(raw)
		if err != nil || idx < 0 {
			p.writeError(w,r,10,"Invalid songIndexToRemove")
			return
		}
		remove = append(remove,idx)
	}
	if err := p.repo.UpdatePlaylist(r.Context(),u.ID,playlistID,name,add,remove); err != nil {
		p.writeError(w,r,70,err.Error())
		return
	}
	p.writeResponse(w,r,map[string]interface{}{})
}

func (p *SubsonicPlugin) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil { p.writeError(w,r,0,"Not authenticated"); return }
	id := r.FormValue("id")
	if id == "" { p.writeError(w,r,10,"Required parameter is missing: id"); return }
	if err := p.repo.DeletePlaylist(r.Context(),u.ID,id); err != nil {
		p.writeError(w,r,70,err.Error())
		return
	}
	p.writeResponse(w,r,map[string]interface{}{})
}
