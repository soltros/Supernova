package api

import (
	"archive/zip"
	"fmt"
	"github.com/soltros/Supernova/internal/media"
	"io"
	"log"
	"mime"
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

		resolved, err := media.Resolve(track.FilePath)
		if err != nil {
			http.Error(w, "media file unavailable", http.StatusNotFound)
			return
		}
		track.FilePath = resolved
		safeTitle := strings.ReplaceAll(track.Title, "\"", "'")
		safeTitle = strings.ReplaceAll(safeTitle, "\n", " ")
		safeTitle = strings.ReplaceAll(safeTitle, "\r", "")
		filename := fmt.Sprintf("%s%s", safeTitle, filepath.Ext(track.FilePath))

		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
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

		tracks, err := s.repo.GetTracks(r.Context(), albumID, "", -1, 0)
		if err != nil {
			http.Error(w, "failed to get tracks", http.StatusInternalServerError)
			return
		}

		safeTitle := strings.ReplaceAll(album.Title, "\"", "'")
		safeTitle = strings.ReplaceAll(safeTitle, "\n", " ")
		safeTitle = strings.ReplaceAll(safeTitle, "\r", "")

		// Pre-check files exist
		for i := range tracks {
			resolved, err := media.Resolve(tracks[i].FilePath)
			if err != nil {
				http.Error(w, "one or more track files are missing from disk", http.StatusInternalServerError)
				return
			}
			tracks[i].FilePath = resolved
		}

		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeTitle + ".zip"}))
		w.Header().Set("Content-Type", "application/zip")

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		for _, track := range tracks {
			file, err := os.Open(track.FilePath)
			if err != nil {
				continue
			}

			ext := filepath.Ext(track.FilePath)
			safeTrackTitle := strings.NewReplacer("/", "-", "\\", "-", "\r", "", "\n", " ").Replace(track.Title)
			fileName := fmt.Sprintf("%02d-%02d - %s-%s%s", track.DiscNumber, track.TrackNumber, safeTrackTitle, track.ID, ext)

			f, err := zipWriter.Create(fileName)
			if err != nil {
				file.Close()
				continue
			}

			_, copyErr := io.Copy(f, file)
			file.Close()
			if copyErr != nil {
				log.Printf("Album download interrupted: %v", copyErr)
				return
			}
		}
	}
}
