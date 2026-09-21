package api

import (
	"archive/zip"
	"os"
	"fmt"
	"github.com/soltros/Supernova/internal/media"
	"io"
	"log"
	"mime"
	"net/http"
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

		file, err := media.Open(track.FilePath)
		if err != nil {
			http.Error(w, "media file unavailable", http.StatusNotFound)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			http.Error(w, "media file unavailable", http.StatusNotFound)
			return
		}
		safeTitle := strings.ReplaceAll(track.Title, "\"", "'")
		safeTitle = strings.ReplaceAll(safeTitle, "\n", " ")
		safeTitle = strings.ReplaceAll(safeTitle, "\r", "")
		filename := fmt.Sprintf("%s%s", safeTitle, filepath.Ext(info.Name()))

		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		w.Header().Del("Content-Type")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
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

		// Preflight every rooted open before committing ZIP headers.
		files := make([]*os.File, 0, len(tracks))
		for _, track := range tracks {
			file, err := media.Open(track.FilePath)
			if err != nil {
				for _, opened := range files { _ = opened.Close() }
				http.Error(w, "one or more track files are missing from disk", http.StatusInternalServerError)
				return
			}
			files = append(files, file)
		}
		defer func(){ for _, file := range files { _ = file.Close() } }()

		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeTitle + ".zip"}))
		w.Header().Set("Content-Type", "application/zip")

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		for i, track := range tracks {
			file := files[i]
			ext := filepath.Ext(track.FilePath)
			safeTrackTitle := strings.NewReplacer("/", "-", "\\", "-", "\r", "", "\n", " ").Replace(track.Title)
			fileName := fmt.Sprintf("%02d-%02d - %s-%s%s", track.DiscNumber, track.TrackNumber, safeTrackTitle, track.ID, ext)

			f, err := zipWriter.Create(fileName)
			if err != nil {
				log.Printf("Album ZIP entry creation failed: %v", err)
				return
			}

			if _, err := file.Seek(0, io.SeekStart); err != nil { log.Printf("Album file seek failed: %v", err); return }
			_, copyErr := io.Copy(f, file)
			if copyErr != nil {
				log.Printf("Album download interrupted: %v", copyErr)
				return
			}
		}
	}
}
