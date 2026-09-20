package subsonic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/soltros/Supernova/internal/models"
)

func (p *SubsonicPlugin) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	name := r.FormValue("name")
	if name == "" {
		p.writeError(w, r, 10, "Required parameter is missing: name")
		return
	}

	playlist, err := p.repo.CreatePlaylist(r.Context(), u.ID, name)
	if err != nil {
		p.writeError(w, r, 0, "Failed to create playlist")
		return
	}

	// Add any provided songs
	r.ParseForm()
	if songIds, ok := r.Form["songId"]; ok {
		for _, songId := range songIds {
			p.repo.AddTrackToPlaylist(r.Context(), u.ID, playlist.ID, songId)
		}
	}

	// Return the empty playlist object
	p.writeResponse(w, r, map[string]interface{}{
		"playlist": map[string]interface{}{
			"id":        playlist.ID,
			"name":      playlist.Name,
			"owner":     u.Username,
			"public":    false,
			"songCount": len(r.Form["songId"]),
			"duration":  0,
			"created":   time.Now().Format(time.RFC3339),
			"changed":   time.Now().Format(time.RFC3339),
		},
	})
}

func (p *SubsonicPlugin) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	playlistId := r.FormValue("playlistId")
	if playlistId == "" {
		p.writeError(w, r, 10, "Required parameter is missing: playlistId")
		return
	}

	name := r.FormValue("name")
	if name != "" {
		// Update name not directly supported by repo yet?
		// We would do p.repo.UpdatePlaylistName
	}

	r.ParseForm()
	// Add songs
	if songIdsToAdd, ok := r.Form["songIdToAdd"]; ok {
		for _, songId := range songIdsToAdd {
			p.repo.AddTrackToPlaylist(r.Context(), u.ID, playlistId, songId)
		}
	}

	// Remove songs
	if songIndexesToRemove, ok := r.Form["songIndexToRemove"]; ok {
		// repo.RemoveTrackFromPlaylist takes trackId, but Subsonic gives songIndexToRemove.
		// For a barebones implementation, we might need to fetch the playlist tracks,
		// find the track ID at that index, and delete it.
		tracks, err := p.repo.GetPlaylistTracks(r.Context(), u.ID, playlistId)
		if err == nil {
			for _, idxStr := range songIndexesToRemove {
				// Convert to int
				var idx int
				if _, err := fmt.Sscanf(idxStr, "%d", &idx); err == nil {
					if idx >= 0 && idx < len(tracks) {
						p.repo.RemoveTrackFromPlaylist(r.Context(), u.ID, playlistId, tracks[idx].ID)
					}
				}
			}
		}
	}

	p.writeResponse(w, r, map[string]interface{}{})
}

func (p *SubsonicPlugin) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	id := r.FormValue("id")
	if id == "" {
		p.writeError(w, r, 10, "Required parameter is missing: id")
		return
	}

	err := p.repo.DeletePlaylist(r.Context(), u.ID, id)
	if err != nil {
		p.writeError(w, r, 0, "Failed to delete playlist")
		return
	}

	p.writeResponse(w, r, map[string]interface{}{})
}
