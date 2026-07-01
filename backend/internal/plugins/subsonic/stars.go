package subsonic

import (
	"context"
	"net/http"
	"strings"

	"github.com/soltros/Supernova/internal/models"
)

func (p *SubsonicPlugin) handleStar(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	
	r.ParseForm()
	ids := r.Form["id"]
	albumIds := r.Form["albumId"]
	artistIds := r.Form["artistId"]

	for _, id := range ids {
		p.repo.HeartEntity(context.Background(), user.ID, "track", id)
	}
	for _, id := range albumIds {
		p.repo.HeartEntity(context.Background(), user.ID, "album", id)
	}
	for _, id := range artistIds {
		p.repo.HeartEntity(context.Background(), user.ID, "artist", id)
	}

	p.writeResponse(w, r, nil)
}

func (p *SubsonicPlugin) handleUnstar(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)

	r.ParseForm()
	ids := r.Form["id"]
	albumIds := r.Form["albumId"]
	artistIds := r.Form["artistId"]

	for _, id := range ids {
		p.repo.UnheartEntity(context.Background(), user.ID, "track", id)
	}
	for _, id := range albumIds {
		p.repo.UnheartEntity(context.Background(), user.ID, "album", id)
	}
	for _, id := range artistIds {
		p.repo.UnheartEntity(context.Background(), user.ID, "artist", id)
	}

	p.writeResponse(w, r, nil)
}

func (p *SubsonicPlugin) handleGetStarred(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)

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

	p.writeResponse(w, r, map[string]interface{}{
		"starred2": starredInfo,
	})
}
