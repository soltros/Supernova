package subsonic

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/soltros/Supernova/internal/media"
	"net/http"
	"os"
	"sort"
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
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			p.writeError(w, r, 10, "Invalid request parameters")
			return
		}
		u := r.FormValue("u")
		pwd := r.FormValue("p")
		t := r.FormValue("t")
		s := r.FormValue("s")

		if u == "" {
			p.writeError(w, r, 10, "Required parameter is missing: u")
			return
		}

		if t != "" && s != "" {
			// Token-based auth: client sends t = md5(password + salt), s = salt
			user, _, err := p.repo.GetUserByUsername(r.Context(), u)
			if err != nil || user == nil {
				p.writeError(w, r, 40, "Wrong username or password.")
				return
			}

			encPass, err := p.repo.GetSubsonicPassword(r.Context(), u)
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

			expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(plain+s)))
			if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(strings.ToLower(t))) != 1 {
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

		user, hash, err := p.repo.GetUserByUsername(r.Context(), u)
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
	format := r.FormValue("f")
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
	case []int:
		for _, item := range v {
			fmt.Fprintf(sb, "<%s>%d</%s>\n", nodeName, item, nodeName)
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
			"valid": true,

			"licenseExpires": "2099-01-01T00:00:00.000Z",
		},
	})
}

func (p *SubsonicPlugin) handleGetUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*models.User)
	userParam := r.FormValue("username")
	if userParam == "" {
		userParam = user.Username
	}
	if userParam != user.Username {
		p.writeError(w, r, 50, "Only your own account is available")
		return
	}

	p.writeResponse(w, r, map[string]interface{}{
		"user": map[string]interface{}{
			"username": userParam,

			"scrobblingEnabled": true,
			"adminRole":         user.IsAdmin,
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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		p.writeError(w, r, 10, "Invalid parameters")
		return
	}
	p.writeResponse(w, r, map[string]interface{}{
		"openSubsonicExtensions": []map[string]interface{}{{"name": "formPost", "versions": []int{1}}},
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
	artists, err := p.repo.GetArtists(r.Context(), 1000, 0)
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
			"id":         a.ID,
			"name":       a.Name,
			"albumCount": 1, // Satisfy strict clients that require albumCount > 0
		})
	}

	var indexes []map[string]interface{}
	for letter, items := range indexMap {
		indexes = append(indexes, map[string]interface{}{
			"name":   letter,
			"artist": items,
		})
	}

	sort.Slice(indexes, func(i, j int) bool { return indexes[i]["name"].(string) < indexes[j]["name"].(string) })
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
	artists, err := p.repo.GetArtists(r.Context(), 1000, 0)
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
			"id":         a.ID,
			"name":       a.Name,
			"albumCount": 1, // Satisfy strict clients that require albumCount > 0
		})
	}

	var indexes []map[string]interface{}
	for letter, items := range indexMap {
		indexes = append(indexes, map[string]interface{}{
			"name":   letter,
			"artist": items,
		})
	}

	sort.Slice(indexes, func(i, j int) bool { return indexes[i]["name"].(string) < indexes[j]["name"].(string) })
	p.writeResponse(w, r, map[string]interface{}{
		"artists": map[string]interface{}{
			"ignoredArticles": "",
			"index":           indexes,
		},
	})
}

func (p *SubsonicPlugin) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	artist, err := p.repo.GetArtistByID(r.Context(), id)
	if err != nil {
		p.writeError(w, r, 70, "Artist not found")
		return
	}
	albums, _ := p.repo.GetAlbums(r.Context(), id, -1, 0)

	var albumList []map[string]interface{}
	for _, al := range albums {
		albumList = append(albumList, map[string]interface{}{
			"id":        al.ID,
			"name":      al.Title,
			"artist":    artist.Name,
			"artistId":  artist.ID,
			"coverArt":  al.ID,
			"songCount": 1, // Satisfy strict clients
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
	id := r.FormValue("id")

	if id == "1" {
		// They are requesting the root music folder, return artists as directories
		artists, _ := p.repo.GetArtists(r.Context(), 1000, 0)
		var children []map[string]interface{}
		for _, a := range artists {
			children = append(children, map[string]interface{}{
				"id":     a.ID,
				"parent": "1",
				"isDir":  true,
				"title":  a.Name,
				"artist": a.Name,
			})
		}
		p.writeResponse(w, r, map[string]interface{}{
			"directory": map[string]interface{}{
				"id":    "1",
				"name":  "Supernova Library",
				"child": children,
			},
		})
		return
	}

	// First check if it's an artist
	artist, err := p.repo.GetArtistByID(r.Context(), id)
	if err == nil {
		// It's an artist, return their albums as directories
		albums, _ := p.repo.GetAlbumsByArtistID(r.Context(), id)
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
	album, err := p.repo.GetAlbumByID(r.Context(), id)
	if err == nil {
		tracks, _ := p.repo.GetTracksByAlbumID(r.Context(), id)
		var children []map[string]interface{}
		for _, track := range tracks {
			children = append(children, map[string]interface{}{
				"id":          track.ID,
				"parent":      id,
				"isDir":       false,
				"title":       track.Title,
				"album":       album.Title,
				"albumId":     album.ID,
				"artist":      track.ArtistName,
				"track":       track.TrackNumber,
				"duration":    track.DurationMs / 1000,
				"path":        track.FilePath,
				"coverArt":    album.ID,
				"contentType": media.ContentType(track.Format),
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
	id := r.FormValue("id")
	album, err := p.repo.GetAlbumByID(r.Context(), id)
	if err != nil {
		p.writeError(w, r, 70, "Album not found")
		return
	}

	tracks, _ := p.repo.GetTracks(r.Context(), id, "", -1, 0)

	var songList []map[string]interface{}
	for _, t := range tracks {
		contentType := media.ContentType(t.Format)
		if t.Format == "" {
			contentType = "audio/mpeg"
		}
		songList = append(songList, map[string]interface{}{
			"id":          t.ID,
			"parent":      album.ID,
			"isDir":       false,
			"title":       t.Title,
			"album":       album.Title,
			"albumId":     album.ID,
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
			"id":        album.ID,
			"name":      album.Title,
			"artist":    artistName,
			"coverArt":  album.ID,
			"songCount": len(songList),
			"song":      songList,
		},
	})
}

func (p *SubsonicPlugin) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	track, err := p.repo.GetTrackByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", 404)
		return
	}
	resolved, err := media.Resolve(track.FilePath)
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, resolved)
}

func (p *SubsonicPlugin) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	track, err := p.repo.GetTrackByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", 404)
		return
	}
	resolved, err := media.Resolve(track.FilePath)
	if err != nil {
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

	http.ServeFile(w, r, resolved)
}
func (p *SubsonicPlugin) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil {
		p.writeError(w, r, 0, "Not authenticated")
		return
	}

	playlists, err := p.repo.GetPlaylists(r.Context(), u.ID)
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

	id := r.FormValue("id")
	if id == "" {
		p.writeError(w, r, 10, "Required parameter is missing: id")
		return
	}

	playlists, err := p.repo.GetPlaylists(r.Context(), u.ID)
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

	tracks, err := p.repo.GetPlaylistTracks(r.Context(), u.ID, id)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	var entryList []map[string]interface{}
	for _, t := range tracks {
		contentType := media.ContentType(t.Format)
		if t.Format == "" {
			contentType = "audio/mpeg"
		}
		entryList = append(entryList, map[string]interface{}{
			"id":          t.ID,
			"title":       t.Title,
			"album":       "", // Not easily available in GetPlaylistTracks
			"albumId":     t.AlbumID,
			"parent":      t.AlbumID,
			"isDir":       false,
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
	listType := r.FormValue("type")

	var albums []models.Album
	var err error

	if listType == "starred" {
		u, ok := r.Context().Value(userContextKey).(*models.User)
		if ok && u != nil {
			_, albums, _, _, err = p.repo.GetHeartDetails(r.Context(), u.ID)
		} else {
			p.writeError(w, r, 0, "Not authenticated")
			return
		}
	} else {
		// Placeholder for other types (newest, random, frequent, recent, etc.)
		albums, err = p.repo.GetAlbums(r.Context(), "", 100, 0)
	}

	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}

	var albumList []map[string]interface{}
	for _, a := range albums {
		albumList = append(albumList, map[string]interface{}{
			"id":        a.ID,
			"name":      a.Title,
			"title":     a.Title, // some clients use title instead of name
			"artist":    a.ArtistName,
			"artistId":  a.ArtistID,
			"coverArt":  a.ID,
			"songCount": 1, // Minimum 1 to show as a valid album
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
	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	// Try as album ID first
	if album, err := p.repo.GetAlbumByID(r.Context(), id); err == nil && album != nil && album.CoverArtPath != "" {
		if !media.ServeArt(w, r, album.CoverArtPath) {
			http.NotFound(w, r)
		}
		return
	}

	// Fallback: try as track ID to fetch track's album cover art
	if track, err := p.repo.GetTrackByID(r.Context(), id); err == nil && track != nil && track.AlbumID != "" {
		if album, err := p.repo.GetAlbumByID(r.Context(), track.AlbumID); err == nil && album != nil && album.CoverArtPath != "" {
			if !media.ServeArt(w, r, album.CoverArtPath) {
				http.NotFound(w, r)
			}
			return
		}
	}

	// Fallback: artist cover art could be supported if added to schema, currently ignoring
	http.Error(w, "Cover art not found", http.StatusNotFound)
}

func (p *SubsonicPlugin) handleGetLyrics(w http.ResponseWriter, r *http.Request) {
	// Stub to prevent 404s when clients check for OpenSubsonic lyrics
	p.writeResponse(w, r, map[string]interface{}{
		"lyrics": map[string]interface{}{},
	})
}
