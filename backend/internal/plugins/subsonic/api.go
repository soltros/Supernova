package subsonic

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// auth middleware checks the subsonic credentials (u, p)
func (p *SubsonicPlugin) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		pwd := r.URL.Query().Get("p")
		t := r.URL.Query().Get("t")

		if t != "" && pwd == "" {
			p.writeError(w, r, 40, "Token auth (t/s) is unsupported by Supernova due to bcrypt security. Please enable Legacy Auth (Cleartext password) in your Subsonic client.")
			return
		}

		if u == "" || pwd == "" {
			p.writeError(w, r, 10, "Required parameter is missing.")
			return
		}

		// Handle enc: hex encoded passwords
		if strings.HasPrefix(pwd, "enc:") {
			decoded, err := hex.DecodeString(strings.TrimPrefix(pwd, "enc:"))
			if err != nil {
				p.writeError(w, r, 40, "Wrong username or password.")
				return
			}
			pwd = string(decoded)
		}

		user, hash, err := p.repo.GetUserByUsername(context.Background(), u)
		if err != nil || user == nil {
			p.writeError(w, r, 40, "Wrong username or password.")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)); err != nil {
			p.writeError(w, r, 40, "Wrong username or password.")
			return
		}

		// Store user in context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (p *SubsonicPlugin) writeResponse(w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	format := r.URL.Query().Get("f")
	if format == "" {
		format = "xml" // Subsonic defaults to XML
	}

	response := map[string]interface{}{
		"status":  "ok",
		"version": "1.16.1",
		"type":    "supernova",
	}
	
	for k, v := range data {
		response[k] = v
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"subsonic-response": response})
		return
	}

	// Just fallback to JSON if they didn't specify. Real XML support requires struct tags.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"subsonic-response": response})
}

func (p *SubsonicPlugin) writeError(w http.ResponseWriter, r *http.Request, code int, message string) {
	p.writeResponse(w, r, map[string]interface{}{
		"status": "failed",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

// handleGetLicense is required by some clients
func (p *SubsonicPlugin) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	p.writeResponse(w, r, map[string]interface{}{
		"license": map[string]interface{}{
			"valid":          true,
			"email":          "admin@supernova.local",
			"licenseExpires": "2099-01-01T00:00:00.000Z",
		},
	})
}

func (p *SubsonicPlugin) handleGetIndexes(w http.ResponseWriter, r *http.Request) {
	// Subsonic expects an alphabetic index of artists
	artists, err := p.repo.GetArtists(context.Background(), 1000, 0)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	// Group by first letter
	indexMap := make(map[string][]map[string]interface{})
	for _, a := range artists {
		if a.Name == "" {
			continue
		}
		letter := strings.ToUpper(string(a.Name[0]))
		if letter < "A" || letter > "Z" {
			letter = "#"
		}
		indexMap[letter] = append(indexMap[letter], map[string]interface{}{
			"id":   a.ID,
			"name": a.Name,
		})
	}

	var indexes []map[string]interface{}
	for letter, items := range indexMap {
		indexes = append(indexes, map[string]interface{}{
			"name":   letter,
			"artist": items,
		})
	}

	p.writeResponse(w, r, map[string]interface{}{
		"indexes": map[string]interface{}{
			"index": indexes,
		},
	})
}

func (p *SubsonicPlugin) handleGetArtists(w http.ResponseWriter, r *http.Request) {
	// Modern clients use getArtists (returns ID3 tags, grouped differently)
	artists, err := p.repo.GetArtists(context.Background(), 1000, 0)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	indexMap := make(map[string][]map[string]interface{})
	for _, a := range artists {
		if a.Name == "" {
			continue
		}
		letter := strings.ToUpper(string(a.Name[0]))
		if letter < "A" || letter > "Z" {
			letter = "#"
		}
		indexMap[letter] = append(indexMap[letter], map[string]interface{}{
			"id":   a.ID,
			"name": a.Name,
		})
	}

	var indexes []map[string]interface{}
	for letter, items := range indexMap {
		indexes = append(indexes, map[string]interface{}{
			"name":   letter,
			"artist": items,
		})
	}
	
	p.writeResponse(w, r, map[string]interface{}{
		"artists": map[string]interface{}{
			"ignoredArticles": "",
			"index": indexes,
		},
	})
}

func (p *SubsonicPlugin) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	artist, _ := p.repo.GetArtistByID(context.Background(), id)
	albums, _ := p.repo.GetAlbums(context.Background(), id, 100, 0)
	
	var albumList []map[string]interface{}
	for _, al := range albums {
		albumList = append(albumList, map[string]interface{}{
			"id":       al.ID,
			"name":     al.Title,
			"artist":   artist.Name,
			"artistId": artist.ID,
			"coverArt": al.ID,
		})
	}

	p.writeResponse(w, r, map[string]interface{}{
		"artist": map[string]interface{}{
			"id":     artist.ID,
			"name":   artist.Name,
			"album":  albumList,
		},
	})
}

func (p *SubsonicPlugin) handleGetMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	
	// First check if it's an artist
	artist, err := p.repo.GetArtistByID(context.Background(), id)
	if err == nil && artist != nil {
		// It's an artist, return their albums as directories
		albums, _ := p.repo.GetAlbumsByArtistID(context.Background(), id)
		var children []map[string]interface{}
		for _, album := range albums {
			children = append(children, map[string]interface{}{
				"id":       album.ID,
				"parent":   id,
				"isDir":    true,
				"title":    album.Title,
				"album":    album.Title,
				"artist":   artist.Name,
				"coverArt": album.ID,
			})
		}
		p.writeResponse(w, r, map[string]interface{}{
			"directory": map[string]interface{}{
				"id":    id,
				"name":  artist.Name,
				"child": children,
			},
		})
		return
	}

	// Try as an album
	album, err := p.repo.GetAlbumByID(context.Background(), id)
	if err == nil && album != nil {
		tracks, _ := p.repo.GetTracksByAlbumID(context.Background(), id)
		var children []map[string]interface{}
		for _, track := range tracks {
			children = append(children, map[string]interface{}{
				"id":          track.ID,
				"parent":      id,
				"isDir":       false,
				"title":       track.Title,
				"album":       album.Title,
				"artist":      track.ArtistName,
				"track":       track.TrackNumber,
				"duration":    track.DurationMs / 1000,
				"path":        track.FilePath,
				"coverArt":    album.ID,
				"contentType": "audio/" + track.Format,
				"suffix":      track.Format,
			})
		}
		p.writeResponse(w, r, map[string]interface{}{
			"directory": map[string]interface{}{
				"id":    id,
				"name":  album.Title,
				"child": children,
			},
		})
		return
	}

	p.writeError(w, r, 70, "The requested data was not found.")
}

func (p *SubsonicPlugin) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	album, err := p.repo.GetAlbumByID(context.Background(), id)
	if err != nil || album == nil {
		p.writeError(w, r, 70, "Album not found")
		return
	}
	
	tracks, _ := p.repo.GetTracks(context.Background(), id, "", 100, 0)
	
	var songList []map[string]interface{}
	for _, t := range tracks {
		songList = append(songList, map[string]interface{}{
			"id":       t.ID,
			"title":    t.Title,
			"album":    album.Title,
			"artist":   t.ArtistName,
			"track":    t.TrackNumber,
			"discNumber": t.DiscNumber,
			"coverArt": album.ID,
			"duration": t.DurationMs / 1000,
			"path":     t.FilePath,
			"contentType": "audio/flac", // Fallback, would need real mime type
		})
	}

	p.writeResponse(w, r, map[string]interface{}{
		"album": map[string]interface{}{
			"id":       album.ID,
			"name":     album.Title,
			"artist":   tracks[0].ArtistName,
			"coverArt": album.ID,
			"song":     songList,
		},
	})
}

func (p *SubsonicPlugin) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	track, err := p.repo.GetTrackByID(context.Background(), id)
	if err != nil || track == nil {
		http.Error(w, "Not found", 404)
		return
	}
	http.ServeFile(w, r, track.FilePath)
}
