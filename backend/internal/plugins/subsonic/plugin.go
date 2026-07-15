package subsonic

import (
	"net/http"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type SubsonicPlugin struct {
	repo *database.Repository
}

// Ensure SubsonicPlugin implements plugins.Plugin
var _ plugins.Plugin = (*SubsonicPlugin)(nil)

func init() {
	plugins.Register(&SubsonicPlugin{})
}

func (p *SubsonicPlugin) ID() string {
	return "subsonic"
}

func (p *SubsonicPlugin) Name() string {
	return "Subsonic API Layer"
}

func (p *SubsonicPlugin) Description() string {
	return "Exposes a /rest/* API compatible with Subsonic/Navidrome clients like Symfonium."
}

func (p *SubsonicPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	return nil
}

func (p *SubsonicPlugin) SetupRoutes(mux *http.ServeMux) {
	// Root ping (allow GET and POST)
	h := p.auth(p.handlePing)
	mux.HandleFunc("/rest/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	})
	mux.HandleFunc("/rest/ping.view", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	})

	// License (allow GET and POST)
	hGetLicense := p.auth(p.handleGetLicense)
	mux.HandleFunc("/rest/getLicense", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		hGetLicense(w, r)
	})

	// Register routes with and without .view suffix
	routes := map[string]http.HandlerFunc{
		"/rest/getIndexes":        p.auth(p.handleGetIndexes),
		"/rest/getArtists":        p.auth(p.handleGetArtists),
		"/rest/getArtist":         p.auth(p.handleGetArtist),
		"/rest/getMusicDirectory": p.auth(p.handleGetMusicDirectory),
		"/rest/getAlbum":          p.auth(p.handleGetAlbum),
		"/rest/getAlbumList":      p.auth(p.handleGetAlbumList),
		"/rest/getAlbumList2":     p.auth(p.handleGetAlbumList),
		"/rest/getPlaylists":      p.auth(p.handleGetPlaylists),
		"/rest/getPlaylist":       p.auth(p.handleGetPlaylist),
		"/rest/stream":            p.auth(p.handleStream),
		"/rest/star":              p.auth(p.handleStar),
		"/rest/unstar":            p.auth(p.handleUnstar),
		"/rest/getStarred":        p.auth(p.handleGetStarred),
		"/rest/getStarred2":       p.auth(p.handleGetStarred),
	}

	for path, handler := range routes {
		mux.HandleFunc(path, handler)
		mux.HandleFunc(path+".view", handler)
	}
}

func (p *SubsonicPlugin) handlePing(w http.ResponseWriter, r *http.Request) {
	p.writeResponse(w, r, nil)
}
