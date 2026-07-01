package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/external"
	"github.com/soltros/Supernova/internal/models"
	"github.com/soltros/Supernova/internal/plugins"
	"github.com/soltros/Supernova/internal/scanner"
)

type Server struct {
	repo          *database.Repository
	lastfm        *external.LastFmClient
	enricher      *scanner.Enricher
	scanner       *scanner.Scanner
	pluginManager *plugins.Manager
	mux           *http.ServeMux
	scanRunning   atomic.Bool // ERR-4: prevents concurrent scan goroutines
}

func NewServer(repo *database.Repository, lastfm *external.LastFmClient, enricher *scanner.Enricher, mediaScanner *scanner.Scanner, pluginManager *plugins.Manager) *Server {
	s := &Server{
		repo:          repo,
		lastfm:        lastfm,
		enricher:      enricher,
		scanner:       mediaScanner,
		pluginManager: pluginManager,
		mux:           http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5174" // Default for local docker
	}
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Authentication
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister())
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin())

	// Public read-only library routes
	s.mux.HandleFunc("GET /api/artists", s.handleGetArtists())
	s.mux.HandleFunc("GET /api/artists/{id}", s.handleGetArtistByID())
	s.mux.HandleFunc("GET /api/albums", s.handleGetAlbums())
	s.mux.HandleFunc("GET /api/albums/{id}", s.handleGetAlbumByID())
	s.mux.HandleFunc("GET /api/tracks", s.handleGetTracks())
	s.mux.HandleFunc("GET /api/search", s.handleSearch())
	
	// Protected User Data routes
	s.mux.HandleFunc("GET /api/hearts", s.requireAuth(s.handleGetHearts()))
	s.mux.HandleFunc("GET /api/hearts/details", s.requireAuth(s.handleGetHeartDetails()))
	s.mux.HandleFunc("POST /api/hearts", s.requireAuth(s.handleAddHeart()))
	s.mux.HandleFunc("DELETE /api/hearts", s.requireAuth(s.handleRemoveHeart()))
	s.mux.HandleFunc("GET /api/hearts/export", s.requireAuth(s.handleExportHearts()))
	s.mux.HandleFunc("POST /api/hearts/import", s.requireAuth(s.handleImportHearts()))
	
	// Streaming route — requireAuth prevents anonymous bandwidth abuse (SEC-3)
	s.mux.HandleFunc("GET /api/stream/{id}", s.requireAuth(s.handleStreamTrack()))

	// Dashboard route
	s.mux.HandleFunc("GET /api/dashboard", s.requireAuth(s.handleGetDashboard()))
	
	// Art Delivery
	s.mux.HandleFunc("GET /api/art/album/{id}", s.handleGetAlbumArt())
	
	// Scrobbling / Listen History (Protected)
	s.mux.HandleFunc("POST /api/scrobbles", s.requireAuth(s.handleScrobble()))
	s.mux.HandleFunc("GET /api/scrobbles/recent", s.requireAuth(s.handleGetRecentScrobbles()))
	
	// Scanning
	s.mux.HandleFunc("POST /api/scan", s.requireAuth(s.handleScanLibrary()))
	s.mux.HandleFunc("GET /api/scan/progress", s.requireAuth(s.handleScanStatus()))
	
	// Settings
	s.mux.HandleFunc("POST /api/settings/reset-artists", s.requireAuth(s.handleResetArtists()))
	// Playlists (Protected)
	s.mux.HandleFunc("GET /api/playlists", s.requireAuth(s.handleGetPlaylists()))
	s.mux.HandleFunc("POST /api/playlists", s.requireAuth(s.handleCreatePlaylist()))
	s.mux.HandleFunc("DELETE /api/playlists/{id}", s.requireAuth(s.handleDeletePlaylist()))
	s.mux.HandleFunc("GET /api/playlists/{id}/tracks", s.requireAuth(s.handleGetPlaylistTracks()))
	s.mux.HandleFunc("POST /api/playlists/{id}/tracks", s.requireAuth(s.handleAddTrackToPlaylist()))
	s.mux.HandleFunc("DELETE /api/playlists/{id}/tracks/{trackId}", s.requireAuth(s.handleRemoveTrackFromPlaylist()))
	s.mux.HandleFunc("GET /api/playlists/export", s.requireAuth(s.handleExportPlaylists()))
	s.mux.HandleFunc("POST /api/playlists/import", s.requireAuth(s.handleImportPlaylists()))
	
	// Plugins
	s.mux.HandleFunc("GET /api/plugins", s.requireAuth(s.handleGetPlugins()))
	
	if s.pluginManager != nil {
		s.pluginManager.SetupPluginRoutes(s.mux)
	}

	s.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0"})
	})
}

// parsePagination extracts limit and offset from the URL query with strict limits.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = 50 // Default items per page
	offset = 0 // Default starting point

	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 1000 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	return
}

func (s *Server) handleGetArtists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)
		letter := r.URL.Query().Get("letter")
		
		var artists []models.Artist
		var err error

		if letter != "" {
			artists, err = s.repo.GetArtistsByLetter(r.Context(), letter, limit, offset)
		} else {
			artists, err = s.repo.GetArtists(r.Context(), limit, offset)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(artists)
	}
}

func (s *Server) handleGetArtistByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		artist, err := s.repo.GetArtistByID(r.Context(), id)
		if err != nil {
			http.Error(w, "artist not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(artist)
	}
}

func (s *Server) handleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"artists": []interface{}{},
				"albums":  []interface{}{},
				"tracks":  []interface{}{},
			})
			return
		}
		
		limit := 20 // Default limit
		results, err := s.repo.Search(r.Context(), query, limit)
		if err != nil {
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(results)
	}
}

func (s *Server) handleGetAlbums() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artistID := r.URL.Query().Get("artist_id")
		limit, offset := parsePagination(r)
		
		albums, err := s.repo.GetAlbums(r.Context(), artistID, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(albums)
	}
}

func (s *Server) handleGetAlbumByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		album, err := s.repo.GetAlbumByID(r.Context(), id)
		if err != nil {
			http.Error(w, "album not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(album)
	}
}

func (s *Server) handleGetTracks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		albumID := r.URL.Query().Get("album_id")
		artistID := r.URL.Query().Get("artist_id")
		limit, offset := parsePagination(r)
		
		tracks, err := s.repo.GetTracks(r.Context(), albumID, artistID, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(tracks)
	}
}

// handleScrobble processes a track playback event for the authenticated user
func (s *Server) handleScrobble() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		var req struct {
			TrackID string `json:"track_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		// ERR-5: reject empty or missing track_id before hitting the DB
		if req.TrackID == "" {
			http.Error(w, "track_id is required", http.StatusBadRequest)
			return
		}
		if err := s.repo.ScrobbleTrack(r.Context(), userID, req.TrackID); err != nil {
			http.Error(w, "failed to scrobble track", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// handleGetRecentScrobbles returns the most recent listen history for the authenticated user
func (s *Server) handleGetRecentScrobbles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		tracks, err := s.repo.GetRecentScrobbles(r.Context(), userID, 20)
		if err != nil {
			http.Error(w, "failed to fetch scrobbles", http.StatusInternalServerError)
			return
		}
		if tracks == nil {
			tracks = []models.Track{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tracks)
	}
}

func (s *Server) handleGetDashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(userIDKey).(string)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		dashboard, err := s.repo.GetDashboard(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(dashboard)
	}
}

func (s *Server) handleResetArtists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.repo.ResetArtistEnrichment(r.Context())
		if err != nil {
			http.Error(w, `{"error":"failed to reset artists"}`, http.StatusInternalServerError)
			return
		}
		
		// Force the background enricher to wake up immediately
		if s.enricher != nil {
			s.enricher.Trigger()
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// handleScanLibrary triggers a full library rescan. Uses an atomic flag to prevent concurrent scans (ERR-4).
func (s *Server) handleScanLibrary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.scanner != nil {
			if !s.scanRunning.CompareAndSwap(false, true) {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"status":"already_running"}`))
				return
			}
			go func() {
				defer s.scanRunning.Store(false)
				if err := s.scanner.FullScan(); err != nil {
					log.Printf("[Scanner] FullScan error: %v", err)
				}
			}()
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// handleScanStatus returns the current scanning status
func (s *Server) handleScanStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "idle"
		scanned := 0
		if s.scanner != nil {
			status, scanned = s.scanner.GetStatus()
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": status,
			"files_scanned": scanned,
		})
	}
}

// handleGetPlugins returns the list of registered plugins and their status
func (s *Server) handleGetPlugins() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.pluginManager == nil {
			json.NewEncoder(w).Encode([]plugins.PluginInfo{})
			return
		}
		manifest := s.pluginManager.GetManifest()
		if manifest == nil {
			manifest = []plugins.PluginInfo{}
		}
		json.NewEncoder(w).Encode(manifest)
	}
}
