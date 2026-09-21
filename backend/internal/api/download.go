package api

import (
	"archive/zip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/soltros/Supernova/internal/media"
)

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
		safeTitle := strings.NewReplacer("\"", "'", "\n", " ", "\r", "").Replace(track.Title)
		filename := fmt.Sprintf("%s%s", safeTitle, filepath.Ext(info.Name()))
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		w.Header().Del("Content-Type")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}

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

		tmp, err := os.CreateTemp("", "supernova-album-*.zip")
		if err != nil {
			http.Error(w, "failed to stage album export", http.StatusInternalServerError)
			return
		}
		tmpName := tmp.Name()
		defer os.Remove(tmpName)
		zw := zip.NewWriter(tmp)
		buildOK := false
		defer func() {
			if !buildOK {
				_ = zw.Close()
				_ = tmp.Close()
			}
		}()

		for _, track := range tracks {
			if err := r.Context().Err(); err != nil {
				http.Error(w, "request cancelled", http.StatusRequestTimeout)
				return
			}
			src, err := media.Open(track.FilePath)
			if err != nil {
				http.Error(w, "one or more track files are missing from disk", http.StatusInternalServerError)
				return
			}
			ext := filepath.Ext(track.FilePath)
			title := strings.NewReplacer("/", "-", "\\", "-", "\r", "", "\n", " ").Replace(track.Title)
			entryName := fmt.Sprintf("%02d-%02d - %s-%s%s", track.DiscNumber, track.TrackNumber, title, track.ID, ext)
			dst, err := zw.Create(entryName)
			if err != nil {
				_ = src.Close()
				http.Error(w, "failed to build album archive", http.StatusInternalServerError)
				return
			}
			_, copyErr := io.Copy(dst, src)
			closeErr := src.Close()
			if copyErr != nil || closeErr != nil {
				http.Error(w, "failed to read album media", http.StatusInternalServerError)
				return
			}
		}
		if err := zw.Close(); err != nil {
			http.Error(w, "failed to finalize album archive", http.StatusInternalServerError)
			return
		}
		if err := tmp.Sync(); err != nil {
			http.Error(w, "failed to finalize album archive", http.StatusInternalServerError)
			return
		}
		if err := tmp.Close(); err != nil {
			http.Error(w, "failed to finalize album archive", http.StatusInternalServerError)
			return
		}
		buildOK = true

		staged, err := os.Open(tmpName)
		if err != nil {
			http.Error(w, "failed to reopen album archive", http.StatusInternalServerError)
			return
		}
		defer staged.Close()
		info, err := staged.Stat()
		if err != nil {
			http.Error(w, "failed to inspect album archive", http.StatusInternalServerError)
			return
		}
		safeTitle := strings.NewReplacer("\"", "'", "\n", " ", "\r", "").Replace(album.Title)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeTitle + ".zip"}))
		w.Header().Set("Content-Type", "application/zip")
		http.ServeContent(w, r, safeTitle+".zip", info.ModTime(), staged)
	}
}
