package api

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleDownloadTrack handles GET /api/download/track/{id}
func (s *Server) handleDownloadTrack() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trackID := r.PathValue("id")
		if trackID == "" {
			http.Error(w, "track ID required", http.StatusBadRequest)
			return
		}

		track, err := s.repo.GetTrackByID(r.Context(), trackID)
		if err != nil {
			http.Error(w, "track not found", http.StatusNotFound)
			return
		}

		safeTitle := strings.ReplaceAll(track.Title, "\"", "'")
		filename := fmt.Sprintf("%s%s", safeTitle, filepath.Ext(track.FilePath))

		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Del("Content-Type")
		http.ServeFile(w, r, track.FilePath)
	}
}

// handleDownloadAlbum handles GET /api/download/album/{id}
func (s *Server) handleDownloadAlbum() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		albumID := r.PathValue("id")
		if albumID == "" {
			http.Error(w, "album ID required", http.StatusBadRequest)
			return
		}

		album, err := s.repo.GetAlbumByID(r.Context(), albumID)
		if err != nil {
			http.Error(w, "album not found", http.StatusNotFound)
			return
		}

		tracks, err := s.repo.GetTracks(r.Context(), albumID, "", 1000, 0)
		if err != nil {
			http.Error(w, "failed to get tracks", http.StatusInternalServerError)
			return
		}

		safeTitle := strings.ReplaceAll(album.Title, "\"", "'")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, safeTitle))
		w.Header().Set("Content-Type", "application/zip")

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		for _, track := range tracks {
			file, err := os.Open(track.FilePath)
			if err != nil {
				continue
			}

			ext := filepath.Ext(track.FilePath)
			safeTrackTitle := strings.ReplaceAll(track.Title, "/", "-")
			fileName := fmt.Sprintf("%02d - %s%s", track.TrackNumber, safeTrackTitle, ext)

			f, err := zipWriter.Create(fileName)
			if err != nil {
				file.Close()
				continue
			}

			_, _ = io.Copy(f, file)
			file.Close()
		}
	}
}
