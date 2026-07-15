package lastfm

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/external"
	"github.com/soltros/Supernova/internal/plugins"
)

type LastFmPlugin struct {
	repo   *database.Repository
	client *external.LastFmClient
}

// Ensure LastFmPlugin implements plugins.Plugin
var _ plugins.Plugin = (*LastFmPlugin)(nil)

func init() {
	plugins.Register(&LastFmPlugin{})
}

func (p *LastFmPlugin) ID() string {
	return "lastfm"
}

func (p *LastFmPlugin) Name() string {
	return "Last.fm Scrobbler"
}

func (p *LastFmPlugin) Description() string {
	return "Scrobble your plays directly to your Last.fm account."
}

func (p *LastFmPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	apiKey := os.Getenv("LASTFM_API_KEY")
	apiSecret := os.Getenv("LASTFM_API_SECRET")
	p.client = external.NewLastFmClient(apiKey, apiSecret)
	return nil
}

func (p *LastFmPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/lastfm/scrobble", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleScrobble(w, r)
	})

	mux.HandleFunc("/api/plugins/lastfm/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleGetSession(w, r)
	})

	mux.HandleFunc("/api/plugins/lastfm/nowplaying", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleNowPlaying(w, r)
	})

	mux.HandleFunc("/api/plugins/lastfm/auth-url", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleGetAuthUrl(w, r)
	})
}

func (p *LastFmPlugin) handleGetAuthUrl(w http.ResponseWriter, r *http.Request) {
	cb := r.URL.Query().Get("cb")
	apiKey := os.Getenv("LASTFM_API_KEY")
	if apiKey == "" {
		http.Error(w, "Last.fm API key not configured on server", http.StatusServiceUnavailable)
		return
	}
	url := "http://www.last.fm/api/auth/?api_key=" + apiKey + "&cb=" + cb
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": url,
	})
}

func (p *LastFmPlugin) handleNowPlaying(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionKey string `json:"session_key"`
		Artist     string `json:"artist"`
		Track      string `json:"track"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if payload.SessionKey == "" {
		http.Error(w, "missing session_key", http.StatusBadRequest)
		return
	}

	err := p.client.UpdateNowPlaying(payload.SessionKey, payload.Artist, payload.Track)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (p *LastFmPlugin) handleScrobble(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionKey string `json:"session_key"`
		Artist     string `json:"artist"`
		Track      string `json:"track"`
		Timestamp  int64  `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if payload.SessionKey == "" {
		http.Error(w, "missing session_key", http.StatusBadRequest)
		return
	}

	err := p.client.Scrobble(payload.SessionKey, payload.Artist, payload.Track, payload.Timestamp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (p *LastFmPlugin) handleGetSession(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if payload.Token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	session, err := p.client.GetSession(payload.Token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_key": session,
	})
}
