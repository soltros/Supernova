package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
)

type Server struct {
	repo *database.Repository
	mux  *http.ServeMux
}

func NewServer(repo *database.Repository) *Server {
	s := &Server{
		repo: repo,
		mux:  http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
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
	s.mux.HandleFunc("GET /api/albums", s.handleGetAlbums())
	s.mux.HandleFunc("GET /api/albums/{id}", s.handleGetAlbumByID())
	s.mux.HandleFunc("GET /api/tracks", s.handleGetTracks())
	
	// Hearts / Favorites (Protected)
	s.mux.HandleFunc("GET /api/hearts", requireAuth(s.handleGetHearts()))
	s.mux.HandleFunc("POST /api/hearts", requireAuth(s.handleAddHeart()))
	s.mux.HandleFunc("DELETE /api/hearts", requireAuth(s.handleRemoveHeart()))
	s.mux.HandleFunc("GET /api/hearts/export", requireAuth(s.handleExportHearts()))
	s.mux.HandleFunc("POST /api/hearts/import", requireAuth(s.handleImportHearts()))
	
	// Streaming route leveraging Go 1.22 path variables
	s.mux.HandleFunc("GET /api/stream/{id}", s.handleStreamTrack())
	
	// Art Delivery
	s.mux.HandleFunc("GET /api/art/album/{id}", s.handleGetAlbumArt())
	
	// Scrobbling / Listen History (Protected)
	s.mux.HandleFunc("POST /api/scrobbles", requireAuth(s.handleScrobble()))
	s.mux.HandleFunc("GET /api/scrobbles/recent", requireAuth(s.handleGetRecentScrobbles()))
	
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
		artists, err := s.repo.GetArtists(r.Context(), limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(artists)
	}
}

func (s *Server) handleGetAlbums() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)
		albums, err := s.repo.GetAlbums(r.Context(), limit, offset)
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
		limit, offset := parsePagination(r)
		
		tracks, err := s.repo.GetTracks(r.Context(), albumID, limit, offset)
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
		userID := r.Context().Value("user_id").(string)

		var req struct {
			TrackID string `json:"track_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
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
		userID := r.Context().Value("user_id").(string)

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
