package api

import (
	"github.com/soltros/Supernova/internal/media"
	"net/http"
)

func (s *Server) handleGetAlbumArt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		album, err := s.repo.GetAlbumByID(r.Context(), r.PathValue("id"))
		if err != nil || album.CoverArtPath == "" || !media.ServeArt(w, r, album.CoverArtPath) {
			http.NotFound(w, r)
		}
	}
}
