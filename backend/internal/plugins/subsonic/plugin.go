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
	// Root ping
	mux.HandleFunc("GET /rest/ping", p.auth(p.handlePing))
	mux.HandleFunc("POST /rest/ping", p.auth(p.handlePing))
	mux.HandleFunc("GET /rest/ping.view", p.auth(p.handlePing))
	mux.HandleFunc("POST /rest/ping.view", p.auth(p.handlePing))

	// License
	mux.HandleFunc("GET /rest/getLicense", p.auth(p.handleGetLicense))
	mux.HandleFunc("POST /rest/getLicense", p.auth(p.handleGetLicense))

	// Browsing
	mux.HandleFunc("GET /rest/getIndexes", p.auth(p.handleGetIndexes))
	mux.HandleFunc("GET /rest/getArtists", p.auth(p.handleGetArtists))
	mux.HandleFunc("GET /rest/getArtist", p.auth(p.handleGetArtist))
	mux.HandleFunc("GET /rest/getMusicDirectory", p.auth(p.handleGetMusicDirectory))
	mux.HandleFunc("GET /rest/getAlbum", p.auth(p.handleGetAlbum))

	// Streaming
	mux.HandleFunc("GET /rest/stream", p.auth(p.handleStream))
}

func (p *SubsonicPlugin) handlePing(w http.ResponseWriter, r *http.Request) {
	p.writeResponse(w, r, nil)
}
