package radiobrowser

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/plugins"
)

type RadioPlugin struct {
	repo   *database.Repository
	client *http.Client
}

// Ensure RadioPlugin implements plugins.Plugin
var _ plugins.Plugin = (*RadioPlugin)(nil)

func init() {
	plugins.Register(&RadioPlugin{})
}

func (p *RadioPlugin) ID() string {
	return "radiobrowser"
}

func (p *RadioPlugin) Name() string {
	return "Internet Radio Browser"
}

func (p *RadioPlugin) Description() string {
	return "Browse and stream over 40,000 global internet radio stations."
}

func (p *RadioPlugin) Init(config plugins.PluginConfig) error {
	p.repo = config.Repo
	p.client = &http.Client{
		Timeout: 10 * time.Second,
	}
	return nil
}

func (p *RadioPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/radio/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleSearch(w, r)
	})
}

func (p *RadioPlugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Make request to a random Radio-Browser instance using the round-robin DNS
	apiURL := "https://all.api.radio-browser.info/json/stations/search"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}
	offset := r.URL.Query().Get("offset")
	if offset == "" {
		offset = "0"
	}

	q := req.URL.Query()
	q.Add("name", query)
	q.Add("limit", limit)
	q.Add("offset", offset)
	q.Add("hidebroken", "true")
	q.Add("order", "clickcount")
	q.Add("reverse", "true")
	req.URL.RawQuery = q.Encode()

	// RadioBrowser requires a descriptive user agent
	req.Header.Set("User-Agent", "Supernova/1.0.0 (https://github.com/soltros/Supernova)")

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, "upstream api failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var stations []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		http.Error(w, "failed to parse upstream response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stations)
}
