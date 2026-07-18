package subsonic

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

func (p *SubsonicPlugin) handleSearch3(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		p.writeError(w, r, 10, "Required parameter is missing: query")
		return
	}

	limitStr := r.URL.Query().Get("songCount")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := p.repo.Search(context.Background(), query, limit)
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

func (p *SubsonicPlugin) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	// For now, we return empty since repo doesn't have GetRandomTracks.
	// But returning empty array prevents errors.
	p.writeResponse(w, r, map[string]interface{}{
		"randomSongs": map[string]interface{}{
			"song": []map[string]interface{}{},
		},
	})
}

func (p *SubsonicPlugin) handleScrobble(w http.ResponseWriter, r *http.Request) {
	// Acknowledge scrobble/play count updates so clients don't complain
	p.writeResponse(w, r, map[string]interface{}{})
}
