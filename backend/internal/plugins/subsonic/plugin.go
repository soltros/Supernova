package subsonic

import (
	"encoding/json"
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
	// Subsonic endpoints
	mux.HandleFunc("GET /rest/ping", p.handlePing)
	mux.HandleFunc("POST /rest/ping", p.handlePing)
	mux.HandleFunc("GET /rest/ping.view", p.handlePing)
	mux.HandleFunc("POST /rest/ping.view", p.handlePing)
}

func (p *SubsonicPlugin) handlePing(w http.ResponseWriter, r *http.Request) {
	// Subsonic can respond in XML or JSON depending on the 'f' parameter.
	// For now, we only implement JSON to show the translation layer works.
	format := r.URL.Query().Get("f")
	
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"subsonic-response": map[string]interface{}{
				"status": "ok",
				"version": "1.16.1", // Tell the client we support a modern API version
			},
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Default XML response
	w.Header().Set("Content-Type", "text/xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.16.1">
</subsonic-response>`))
}
