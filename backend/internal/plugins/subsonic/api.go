package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/soltros/Supernova/internal/api"
	"github.com/soltros/Supernova/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string
const userContextKey contextKey = "subsonic_user"

// auth middleware checks the subsonic credentials (u, p or u, t, s)
func (p *SubsonicPlugin) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		pwd := r.URL.Query().Get("p")
		t := r.URL.Query().Get("t")
		s := r.URL.Query().Get("s")

		if u == "" {
			p.writeError(w, r, 10, "Required parameter is missing: u")
			return
		}

		if t != "" && s != "" {
			// Token-based auth: client sends t = md5(password + salt), s = salt
			user, _, err := p.repo.GetUserByUsername(context.Background(), u)
			if err != nil || user == nil {
				p.writeError(w, r, 40, "Wrong username or password.")
				return
			}
			
			encPass, err := p.repo.GetSubsonicPassword(context.Background(), u)
			if err != nil || encPass == "" {
				p.writeError(w, r, 40, "Please login via the web UI once to enable Subsonic token authentication.")
				return
			}

			// We retrieve the symmetric JWT_SECRET to decrypt the password
			secret := os.Getenv("JWT_SECRET")
			if len(secret) < 32 {
				p.writeError(w, r, 40, "Server configuration error.")
				return
			}

			// Decrypt using the crypto utility
			plain, err := api.DecryptPassword(encPass, []byte(secret))
			if err != nil {
				p.writeError(w, r, 40, "Wrong username or password.")
				return
			}

			expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(plain + s)))
			if expectedToken != t {
				p.writeError(w, r, 40, "Wrong username or password.")
				return
			}

			// Valid!
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if pwd == "" {
			p.writeError(w, r, 10, "Required parameter is missing: p or t+s")
			return
		}

		// Handle enc: hex encoded passwords
		if strings.HasPrefix(pwd, "enc:") {
			decoded, err := hex.DecodeString(strings.TrimPrefix(pwd, "enc:"))
			if err != nil {
				p.writeError(w, r, 10, "Malformed hex-encoded password.")
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
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (p *SubsonicPlugin) writeResponse(w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	format := r.URL.Query().Get("f")
	if format == "" {
		format = "xml" // Subsonic defaults to XML
	}

	response := map[string]interface{}{
		"version":       "1.16.1",
		"type":          "supernova",
		"serverVersion": "1.0.0",
		"openSubsonic":  true,
	}
	
	status := "ok"
	if s, ok := data["status"].(string); ok {
		status = s
		delete(data, "status")
	}
	response["status"] = status
	
	for k, v := range data {
		response[k] = v
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"subsonic-response": response})
		return
	}

	// Generate XML
	w.Header().Set("Content-Type", "application/xml")
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	response["xmlns"] = "http://subsonic.org/restapi"
	p.writeXML(&sb, "subsonic-response", response)
	w.Write([]byte(sb.String()))
}

func isPrimitive(val interface{}) bool {
	switch val.(type) {
	case string, int, int64, float64, float32, bool:
		return true
	default:
		return false
	}
}

func (p *SubsonicPlugin) writeXML(sb *strings.Builder, nodeName string, data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		sb.WriteString("<" + nodeName)
		for k, val := range v {
			if isPrimitive(val) {
				sb.WriteString(fmt.Sprintf(` %s="`, k))
				xml.EscapeText(sb, []byte(fmt.Sprint(val)))
				sb.WriteString(`"`)
			}
		}
		
		hasChildren := false
		for _, val := range v {
			if !isPrimitive(val) {
				hasChildren = true
				break
			}
		}
		
		if !hasChildren {
			sb.WriteString(" />\n")
		} else {
			sb.WriteString(">\n")
			for k, val := range v {
				if !isPrimitive(val) {
					p.writeXML(sb, k, val)
				}
			}
			sb.WriteString("</" + nodeName + ">\n")
		}
	case []map[string]interface{}:
		for _, item := range v {
			p.writeXML(sb, nodeName, item)
		}
	case []interface{}:
		for _, item := range v {
			p.writeXML(sb, nodeName, item)
		}
	}
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
			"email":          "soltros@proton.me",
			"licenseExpires": "2099-01-01T00:00:00.000Z",
		},
	})
}

func (p *SubsonicPlugin) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userParam := r.URL.Query().Get("username")
	if userParam == "" {
		// Default to authenticated user if not provided
		userParam = r.URL.Query().Get("u")
	}

	p.writeResponse(w, r, map[string]interface{}{
		"user": map[string]interface{}{
			"username":          userParam,
			"email":             userParam + "@example.com", // Stub email
			"scrobblingEnabled": true,
			"adminRole":         true,
			"settingsRole":      true,
			"downloadRole":      true,
			"uploadRole":        false,
			"playlistRole":      true,
			"coverArtRole":      true,
			"commentRole":       false,
			"podcastRole":       true,
			"streamRole":        true,
			"jukeboxRole":       false,
			"shareRole":         false,
		},
	})
}

func (p *SubsonicPlugin) handleGetOpenSubsonicExtensions(w http.ResponseWriter, r *http.Request) {
	p.writeResponse(w, r, map[string]interface{}{
		"openSubsonicExtensions": map[string]interface{}{
			"extension": []map[string]interface{}{
				{
					"name": "supernova",
					"versions": []int{1},
				},
			},
		},
	})
}

func (p *SubsonicPlugin) handleGetMusicFolders(w http.ResponseWriter, r *http.Request) {
	p.writeResponse(w, r, map[string]interface{}{
		"musicFolders": map[string]interface{}{
			"musicFolder": []map[string]interface{}{
				{
					"id":   "1",
					"name": "Supernova Library",
				},
			},
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
			"lastModified":    0,
			"ignoredArticles": "The El La Los Las Le Les",
			"index":           indexes,
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
	artist, err := p.repo.GetArtistByID(context.Background(), id)
	if err != nil {
		p.writeError(w, r, 70, "Artist not found")
		return
	}
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
			"id":         artist.ID,
			"name":       artist.Name,
			"coverArt":   artist.ID,
			"albumCount": len(albumList),
			"album":      albumList,
		},
	})
}

func (p *SubsonicPlugin) handleGetMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	
	// First check if it's an artist
	artist, err := p.repo.GetArtistByID(context.Background(), id)
	if err == nil {
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
	if err == nil {
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
	if err != nil {
		p.writeError(w, r, 70, "Album not found")
		return
	}
	
	tracks, _ := p.repo.GetTracks(context.Background(), id, "", 100, 0)
	
	var songList []map[string]interface{}
	for _, t := range tracks {
		contentType := "audio/" + strings.ToLower(t.Format)
		if t.Format == "" {
			contentType = "audio/mpeg"
		}
		songList = append(songList, map[string]interface{}{
			"id":          t.ID,
			"title":       t.Title,
			"album":       album.Title,
			"artist":      t.ArtistName,
			"track":       t.TrackNumber,
			"discNumber":  t.DiscNumber,
			"coverArt":    album.ID,
			"duration":    t.DurationMs / 1000,
			"path":        t.FilePath,
			"contentType": contentType,
			"suffix":      strings.ToLower(t.Format),
			"bitRate":     t.Bitrate,
		})
	}

	artistName := ""
	if len(tracks) > 0 {
		artistName = tracks[0].ArtistName
	}

	p.writeResponse(w, r, map[string]interface{}{
		"album": map[string]interface{}{
			"id":       album.ID,
			"name":     album.Title,
			"artist":   artistName,
			"coverArt": album.ID,
			"song":     songList,
		},
	})
}

func (p *SubsonicPlugin) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	track, err := p.repo.GetTrackByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Not found", 404)
		return
	}
	if !strings.HasPrefix(track.FilePath, os.Getenv("MEDIA_PATH")) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, track.FilePath)
}

func (p *SubsonicPlugin) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	track, err := p.repo.GetTrackByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Not found", 404)
		return
	}
	if !strings.HasPrefix(track.FilePath, os.Getenv("MEDIA_PATH")) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	// Set headers for download
	filename := track.Title + ".flac" // Or get extension from file path
	if idx := strings.LastIndex(track.FilePath, "."); idx != -1 {
		filename = track.Title + track.FilePath[idx:]
	}
	
	// Escape filename quotes
	filename = strings.ReplaceAll(filename, "\"", "")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	
	http.ServeFile(w, r, track.FilePath)
}
func (p *SubsonicPlugin) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	playlists, err := p.repo.GetPlaylists(context.Background(), u.ID)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	var playlistList []map[string]interface{}
	for _, pl := range playlists {
		playlistList = append(playlistList, map[string]interface{}{
			"id":        pl.ID,
			"name":      pl.Name,
			"owner":     u.Username,
			"public":    false,
			"songCount": 0,
			"duration":  0,
			"created":   pl.CreatedAt,
			"changed":   pl.CreatedAt,
		})
	}

	if playlistList == nil {
		playlistList = make([]map[string]interface{}, 0)
	}

	p.writeResponse(w, r, map[string]interface{}{
		"playlists": map[string]interface{}{
			"playlist": playlistList,
		},
	})
}

func (p *SubsonicPlugin) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		p.writeError(w, r, 10, "Required parameter is missing: id")
		return
	}

	playlists, err := p.repo.GetPlaylists(context.Background(), u.ID)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}
	
	var playlist *models.Playlist
	for _, pl := range playlists {
		if pl.ID == id {
			playlist = &pl
			break
		}
	}
	
	if playlist == nil {
		p.writeError(w, r, 70, "Playlist not found")
		return
	}

	tracks, err := p.repo.GetPlaylistTracks(context.Background(), u.ID, id)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	var entryList []map[string]interface{}
	for _, t := range tracks {
		contentType := "audio/" + strings.ToLower(t.Format)
		if t.Format == "" {
			contentType = "audio/mpeg"
		}
		entryList = append(entryList, map[string]interface{}{
			"id":          t.ID,
			"title":       t.Title,
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
	
	if entryList == nil {
		entryList = make([]map[string]interface{}, 0)
	}

	p.writeResponse(w, r, map[string]interface{}{
		"playlist": map[string]interface{}{
			"id":        playlist.ID,
			"name":      playlist.Name,
			"owner":     u.Username,
			"public":    false,
			"songCount": len(entryList),
			"duration":  0,
			"created":   playlist.CreatedAt,
			"changed":   playlist.CreatedAt,
			"entry":     entryList,
		},
	})
}

func (p *SubsonicPlugin) handleGetAlbumList(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	
	var albums []models.Album
	var err error

	if listType == "starred" {
		u, ok := r.Context().Value(userContextKey).(*models.User)
		if ok && u != nil {
			_, albums, _, _, err = p.repo.GetHeartDetails(context.Background(), u.ID)
		} else {
			p.writeError(w, r, 0, "Not authenticated")
			return
		}
	} else {
		// Placeholder for other types (newest, random, frequent, recent, etc.)
		albums, err = p.repo.GetAlbums(context.Background(), "", 100, 0)
	}

	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	var albumList []map[string]interface{}
	for _, a := range albums {
		albumList = append(albumList, map[string]interface{}{
			"id":       a.ID,
			"name":     a.Title,
			"title":    a.Title, // some clients use title instead of name
			"artist":   a.ArtistName, // Not strictly fetched in GetAlbums, but GetAlbums query joins artists! Wait, does models.Album have ArtistName?
			"artistId": "",
			"coverArt": a.ID,
			"songCount": 10,
		})
	}

	if albumList == nil {
		albumList = make([]map[string]interface{}, 0)
	}
	
	// getAlbumList uses albumList, getAlbumList2 uses albumList2.
	// Since we handle both with this one function, we can check path
	isList2 := strings.Contains(r.URL.Path, "getAlbumList2")
	key := "albumList"
	if isList2 {
		key = "albumList2"
	}

	p.writeResponse(w, r, map[string]interface{}{
		key: map[string]interface{}{
			"album": albumList,
		},
	})
}

func (p *SubsonicPlugin) handleGetCoverArt(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	// Try as album ID first
	if album, err := p.repo.GetAlbumByID(context.Background(), id); err == nil && album != nil && album.CoverArtPath != "" {
		http.ServeFile(w, r, album.CoverArtPath)
		return
	}

	// Fallback: try as track ID to fetch track's album cover art
	if track, err := p.repo.GetTrackByID(context.Background(), id); err == nil && track != nil && track.AlbumID != "" {
		if album, err := p.repo.GetAlbumByID(context.Background(), track.AlbumID); err == nil && album != nil && album.CoverArtPath != "" {
			http.ServeFile(w, r, album.CoverArtPath)
			return
		}
	}

	// Fallback: artist cover art could be supported if added to schema, currently ignoring
	http.Error(w, "Cover art not found", http.StatusNotFound)
}
