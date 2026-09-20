package subsonic

import (
	"github.com/soltros/Supernova/internal/media"
	"github.com/soltros/Supernova/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (p *SubsonicPlugin) handleSearch3(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	if query == "" {
		p.writeError(w, r, 10, "Required parameter is missing: query")
		return
	}

	limitStr := r.FormValue("songCount")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	results, err := p.repo.Search(r.Context(), query, limit)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	// Map DB generic maps to Subsonic XML/JSON
	var artists []map[string]interface{}
	if dbArtists, ok := results["artists"].([]map[string]interface{}); ok {
		for _, a := range dbArtists {
			artists = append(artists, map[string]interface{}{
				"id":         a["id"],
				"name":       a["name"],
				"albumCount": 1,
			})
		}
	}

	var albums []map[string]interface{}
	if dbAlbums, ok := results["albums"].([]map[string]interface{}); ok {
		for _, a := range dbAlbums {
			albums = append(albums, map[string]interface{}{
				"id":        a["id"],
				"title":     a["title"],
				"name":      a["title"],
				"artist":    a["artist_name"],
				"coverArt":  a["id"],
				"songCount": 1,
			})
		}
	}

	var songs []map[string]interface{}
	if dbTracks, ok := results["tracks"].([]map[string]interface{}); ok {
		for _, t := range dbTracks {
			songs = append(songs, map[string]interface{}{
				"id":       t["id"],
				"title":    t["title"],
				"album":    t["album_title"],
				"artist":   t["artist_name"],
				"coverArt": t["album_id"],
				"duration": t["duration_ms"].(int) / 1000,
				"parent":   t["album_id"],
				"albumId":  t["album_id"],
				"isDir":    false,
			})
		}
	}

	searchKey := "searchResult3"
	if strings.HasPrefix(r.URL.Path, "/rest/search2") {
		searchKey = "searchResult2"
	}

	resultMap := map[string]interface{}{}
	if len(artists) > 0 {
		resultMap["artist"] = artists
	}
	if len(albums) > 0 {
		resultMap["album"] = albums
	}
	if len(songs) > 0 {
		resultMap["song"] = songs
	}

	p.writeResponse(w, r, map[string]interface{}{
		searchKey: resultMap,
	})
}

func songNode(t models.Track) map[string]interface{} {
	return map[string]interface{}{"id": t.ID, "parent": t.AlbumID, "albumId": t.AlbumID, "isDir": false, "title": t.Title, "artist": t.ArtistName, "coverArt": t.AlbumID, "duration": t.DurationMs / 1000, "suffix": t.Format, "contentType": media.ContentType(t.Format), "bitRate": t.Bitrate, "track": t.TrackNumber, "discNumber": t.DiscNumber}
}

func (p *SubsonicPlugin) handleGetSong(w http.ResponseWriter, r *http.Request) {
	track, err := p.repo.GetTrackByID(r.Context(), r.FormValue("id"))
	if err != nil {
		p.writeError(w, r, 70, "Song not found")
		return
	}
	p.writeResponse(w, r, map[string]interface{}{"song": songNode(*track)})
}

func (p *SubsonicPlugin) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	size := 10
	if v, err := strconv.Atoi(r.FormValue("size")); err == nil && v >= 0 && v <= 500 {
		size = v
	}
	tracks, err := p.repo.GetRandomTracks(r.Context(), size)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}
	songs := make([]map[string]interface{}, 0, len(tracks))
	for _, t := range tracks {
		songs = append(songs, songNode(t))
	}
	p.writeResponse(w, r, map[string]interface{}{"randomSongs": map[string]interface{}{"song": songs}})
}

func (p *SubsonicPlugin) handleScrobble(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*models.User)
	ids, times := r.Form["id"], r.Form["time"]
	if len(ids) == 0 || (len(times) != 0 && len(times) != len(ids)) {
		p.writeError(w, r, 10, "Invalid id/time parameters")
		return
	}
	submission := r.FormValue("submission")
	if submission != "" && submission != "true" && submission != "false" {
		p.writeError(w, r, 10, "Invalid submission")
		return
	}
	timestamps := make([]time.Time, len(ids))
	for i, id := range ids {
		if _, err := p.repo.GetTrackByID(r.Context(), id); err != nil {
			p.writeError(w, r, 70, "Song not found")
			return
		}
		timestamps[i] = time.Now()
		if len(times) != 0 {
			ms, err := strconv.ParseInt(times[i], 10, 64)
			if err != nil || ms <= 0 {
				p.writeError(w, r, 10, "Invalid time")
				return
			}
			timestamps[i] = time.UnixMilli(ms)
		}
	}
	if submission != "false" {
		if err := p.repo.ScrobbleBatch(r.Context(), user.ID, ids, timestamps); err != nil {
			p.writeError(w, r, 0, "Failed to scrobble")
			return
		}
	}
	p.writeResponse(w, r, nil)
}
