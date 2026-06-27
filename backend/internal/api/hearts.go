package api

import (
	"encoding/json"
	"net/http"

	"github.com/soltros/Supernova/internal/models"
)

// handleGetHearts fetches all hearts for the user
func (s *Server) handleGetHearts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		
		hearts, err := s.repo.GetAllHearts(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to get hearts", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(hearts)
	}
}

// handleGetHeartDetails fetches the fully populated tracks and albums that the user has hearted
func (s *Server) handleGetHeartDetails() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		tracks, albums, err := s.repo.GetHeartDetails(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to get heart details", http.StatusInternalServerError)
			return
		}

		response := struct {
			Tracks []models.Track `json:"tracks"`
			Albums []models.Album `json:"albums"`
		}{
			Tracks: tracks,
			Albums: albums,
		}

		json.NewEncoder(w).Encode(response)
	}
}

// handleAddHeart adds a new heart, letting the DB securely generate the UUID
func (s *Server) handleAddHeart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		var req struct {
			EntityType string `json:"entity_type"`
			EntityID   string `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := s.repo.HeartEntity(r.Context(), userID, req.EntityType, req.EntityID); err != nil {
			http.Error(w, "failed to add heart", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// handleRemoveHeart deletes a heart
func (s *Server) handleRemoveHeart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		entityType := r.URL.Query().Get("entity_type")
		entityID := r.URL.Query().Get("entity_id")
		if entityType == "" || entityID == "" {
			http.Error(w, "missing parameters", http.StatusBadRequest)
			return
		}
		if err := s.repo.UnheartEntity(r.Context(), userID, entityType, entityID); err != nil {
			http.Error(w, "failed to remove heart", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// handleExportHearts allows the user to backup their hearts as a JSON file
func (s *Server) handleExportHearts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		hearts, err := s.repo.ExportHearts(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to export hearts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=supernova_hearts_backup.json")
		json.NewEncoder(w).Encode(hearts)
	}
}

// handleImportHearts safely restores hearts by matching permanent file paths
func (s *Server) handleImportHearts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		var backups []models.HeartBackup
		if err := json.NewDecoder(r.Body).Decode(&backups); err != nil {
			http.Error(w, "invalid backup format", http.StatusBadRequest)
			return
		}
		for _, b := range backups {
			// We silently ignore errors per-item so a partial failure doesn't crash the import
			_ = s.repo.ImportHeartBackup(r.Context(), userID, b)
		}
		w.WriteHeader(http.StatusOK)
	}
}
