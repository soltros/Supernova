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

	// Browsing (GET)
	mux.HandleFunc("/rest/getIndexes", p.auth(p.handleGetIndexes))
	mux.HandleFunc("/rest/getArtists", p.auth(p.handleGetArtists))
	mux.HandleFunc("/rest/getArtist", p.auth(p.handleGetArtist))
	mux.HandleFunc("/rest/getMusicDirectory", p.auth(p.handleGetMusicDirectory))
	mux.HandleFunc("/rest/getAlbum", p.auth(p.handleGetAlbum))

	// Streaming
	mux.HandleFunc("/rest/stream", p.auth(p.handleStream))
}

func (p *SubsonicPlugin) handlePing(w http.ResponseWriter, r *http.Request) {
	p.writeResponse(w, r, nil)
}
