package subsonic

import (
	"context"
	"net/http"
	"strings"

	"github.com/soltros/Supernova/internal/models"
)

func (p *SubsonicPlugin) resolveEntityType(ctx context.Context, id string) string {
	if t, err := p.repo.GetTrackByID(ctx, id); err == nil && t != nil {
		return "track"
	}
	if a, err := p.repo.GetAlbumByID(ctx, id); err == nil && a != nil {
		return "album"
	}
	if a, err := p.repo.GetArtistByID(ctx, id); err == nil && a.ID != "" {
		return "artist"
	}
	return "track"
}

func (p *SubsonicPlugin) handleStar(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || user == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}
	
	r.ParseForm()
	ids := r.Form["id"]
	albumIds := r.Form["albumId"]
	artistIds := r.Form["artistId"]

	for _, id := range ids {
		entityType := p.resolveEntityType(r.Context(), id)
		p.repo.HeartEntity(r.Context(), user.ID, entityType, id)
	}
	for _, id := range albumIds {
		p.repo.HeartEntity(r.Context(), user.ID, "album", id)
	}
	for _, id := range artistIds {
		p.repo.HeartEntity(r.Context(), user.ID, "artist", id)
	}

	p.writeResponse(w, r, nil)
}

func (p *SubsonicPlugin) handleUnstar(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || user == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	r.ParseForm()
	ids := r.Form["id"]
	albumIds := r.Form["albumId"]
	artistIds := r.Form["artistId"]

	for _, id := range ids {
		entityType := p.resolveEntityType(r.Context(), id)
		p.repo.UnheartEntity(r.Context(), user.ID, entityType, id)
	}
	for _, id := range albumIds {
		p.repo.UnheartEntity(r.Context(), user.ID, "album", id)
	}
	for _, id := range artistIds {
		p.repo.UnheartEntity(r.Context(), user.ID, "artist", id)
	}

	p.writeResponse(w, r, nil)
}

func (p *SubsonicPlugin) handleGetStarred(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || user == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	tracks, albums, artists, _, err := p.repo.GetHeartDetails(context.Background(), user.ID)
	if err != nil {
		p.writeError(w, r, 0, err.Error())
		return
	}

	var songNodes []map[string]interface{}
	for _, t := range tracks {
		contentType := "audio/" + strings.ToLower(t.Format)
		if t.Format == "" {
			contentType = "audio/mpeg"
		}
		songNodes = append(songNodes, map[string]interface{}{
			"id":          t.ID,
			"title":       t.Title,
			"albumId":     t.AlbumID,
			"artist":      t.ArtistName,
			"track":       t.TrackNumber,
			"discNumber":  t.DiscNumber,
			"coverArt":    t.AlbumID,
			"duration":    t.DurationMs / 1000,
			"path":        t.FilePath,
			"contentType": contentType,
			"suffix":      strings.ToLower(t.Format),
			"bitRate":     t.Bitrate,
		})
	}

	var albumNodes []map[string]interface{}
	for _, a := range albums {
		albumNodes = append(albumNodes, map[string]interface{}{
			"id":       a.ID,
			"name":     a.Title,
			"artist":   a.ArtistName,
			"coverArt": a.ID,
		})
	}

	var artistNodes []map[string]interface{}
	for _, a := range artists {
		artistNodes = append(artistNodes, map[string]interface{}{
			"id":   a.ID,
			"name": a.Name,
		})
	}

	starredInfo := map[string]interface{}{}
	if len(songNodes) > 0 {
		starredInfo["song"] = songNodes
	}
	if len(albumNodes) > 0 {
		starredInfo["album"] = albumNodes
	}
	if len(artistNodes) > 0 {
		starredInfo["artist"] = artistNodes
	}

	key := "starred"
	if strings.HasSuffix(r.URL.Path, "getStarred2") || strings.HasSuffix(r.URL.Path, "getStarred2.view") {
		key = "starred2"
	}

	p.writeResponse(w, r, map[string]interface{}{
		key: starredInfo,
	})
}
