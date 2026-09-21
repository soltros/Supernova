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

		tracks, albums, artists, playlists, err := s.repo.GetHeartDetails(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to get heart details", http.StatusInternalServerError)
			return
		}

		radio, podcasts, err := s.repo.GetExternalHeartMetadata(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to get external heart details", http.StatusInternalServerError)
			return
		}
		response := struct {
			Tracks    []models.Track    `json:"tracks"`
			Albums    []models.Album    `json:"albums"`
			Artists   []models.Artist   `json:"artists"`
			Playlists []models.Playlist `json:"playlists"`
			Radio     []json.RawMessage `json:"radio"`
			Podcasts  []json.RawMessage `json:"podcasts"`
		}{
			Tracks: tracks, Albums: albums, Artists: artists, Playlists: playlists,
			Radio: radio, Podcasts: podcasts,
		}

		json.NewEncoder(w).Encode(response)
	}
}

// handleAddHeart adds a new heart, letting the DB securely generate the UUID
func (s *Server) handleAddHeart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)

		var req struct {
			EntityType string          `json:"entity_type"`
			EntityID   string          `json:"entity_id"`
			Metadata   json.RawMessage `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		// SEC-2: allowlist entity_type to prevent dirty data / future query confusion
		switch req.EntityType {
		case "track", "album", "artist", "playlist", "radio", "podcast":
			// valid
		default:
			http.Error(w, "invalid entity_type: must be track, album, artist, playlist, radio, or podcast", http.StatusBadRequest)
			return
		}
		// ERR-6: reject empty entity_id
		if req.EntityID == "" {
			http.Error(w, "entity_id is required", http.StatusBadRequest)
			return
		}
		if err := s.repo.HeartEntityWithMetadata(r.Context(), userID, req.EntityType, req.EntityID, req.Metadata); err != nil {
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
		switch entityType {
		case "track", "album", "artist", "playlist", "radio", "podcast":
			// valid
		default:
			http.Error(w, "invalid entity_type", http.StatusBadRequest)
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

		hearts, err := s.repo.ExportHeartsV2(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to export hearts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=supernova_hearts_backup.json")
		json.NewEncoder(w).Encode(models.HeartBackupEnvelope{Version: 2, Hearts: hearts})
	}
}

// handleImportHearts safely restores versioned or legacy heart backups.
func (s *Server) handleImportHearts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey).(string)
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			http.Error(w, "invalid backup format", http.StatusBadRequest)
			return
		}
		var backups []models.HeartBackup
		if len(raw) > 0 && raw[0] == '{' {
			var envelope models.HeartBackupEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Version != 2 {
				http.Error(w, "unsupported favorite backup version", http.StatusBadRequest)
				return
			}
			backups = envelope.Hearts
		} else if err := json.Unmarshal(raw, &backups); err != nil {
			http.Error(w, "invalid legacy favorite backup", http.StatusBadRequest)
			return
		}
		if err := s.repo.ImportHeartBackupsV2(r.Context(), userID, backups); err != nil {
			http.Error(w, "favorite import failed: "+err.Error(), http.StatusConflict)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"version": 2, "imported": len(backups), "skipped": 0})
	}
}
