package api

import (
	"encoding/json"
	"net/http"

	"github.com/soltros/Supernova/internal/models"
)

func (s *Server) handleGetPlaylists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		playlists, err := s.repo.GetPlaylists(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to fetch playlists", http.StatusInternalServerError)
			return
		}
		
		if playlists == nil {
			playlists = []models.Playlist{}
		}

		json.NewEncoder(w).Encode(playlists)
	}
}

func (s *Server) handleCreatePlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}

		p, err := s.repo.CreatePlaylist(r.Context(), userID, req.Name)
		if err != nil {
			http.Error(w, "failed to create playlist", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	}
}

func (s *Server) handleDeletePlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		playlistID := r.PathValue("id")

		if err := s.repo.DeletePlaylist(r.Context(), userID, playlistID); err != nil {
			http.Error(w, "failed to delete playlist", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handleGetPlaylistTracks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		playlistID := r.PathValue("id")

		tracks, err := s.repo.GetPlaylistTracks(r.Context(), userID, playlistID)
		if err != nil {
			http.Error(w, "failed to fetch playlist tracks", http.StatusInternalServerError)
			return
		}

		if tracks == nil {
			tracks = []models.Track{}
		}

		json.NewEncoder(w).Encode(tracks)
	}
}

func (s *Server) handleAddTrackToPlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		playlistID := r.PathValue("id")

		var req struct {
			TrackID string `json:"track_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}

		if err := s.repo.AddTrackToPlaylist(r.Context(), userID, playlistID, req.TrackID); err != nil {
			http.Error(w, "failed to add track to playlist", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) handleRemoveTrackFromPlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		playlistID := r.PathValue("id")
		trackID := r.PathValue("trackId")

		if err := s.repo.RemoveTrackFromPlaylist(r.Context(), userID, playlistID, trackID); err != nil {
			http.Error(w, "failed to remove track from playlist", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handleExportPlaylists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		backups, err := s.repo.ExportPlaylists(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to export playlists", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=supernova_playlists_backup.json")
		json.NewEncoder(w).Encode(backups)
	}
}

func (s *Server) handleImportPlaylists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		var backups []models.PlaylistBackup
		if err := json.NewDecoder(r.Body).Decode(&backups); err != nil {
			http.Error(w, "invalid backup format", http.StatusBadRequest)
			return
		}

		for _, b := range backups {
			_ = s.repo.ImportPlaylistBackup(r.Context(), userID, b)
		}

		w.WriteHeader(http.StatusOK)
	}
}
