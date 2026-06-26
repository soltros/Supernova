package api

import (
	"net/http"
	"os"
)

// handleGetAlbumArt streams the extracted cover art directly to the browser.
// It leverages http.ServeFile which automatically handles Content-Type detection,
// byte-range requests, and ETag caching for maximum browser performance.
func (s *Server) handleGetAlbumArt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing album id", http.StatusBadRequest)
			return
		}

		// 1. Fetch the album metadata to get the CoverArtPath
		album, err := s.repo.GetAlbumByID(r.Context(), id)
		if err != nil || album.CoverArtPath == "" {
			// If the album doesn't exist or has no art, return a 404.
			// The React frontend will gracefully fallback to a default SVG.
			http.NotFound(w, r)
			return
		}

		// 2. Ensure the file actually exists on disk (hasn't been deleted externally)
		if _, err := os.Stat(album.CoverArtPath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		// 3. Stream the raw image bytes!
		w.Header().Del("Content-Type") // Fixes Flaw #1: Remove the global JSON header
		http.ServeFile(w, r, album.CoverArtPath)
	}
}
