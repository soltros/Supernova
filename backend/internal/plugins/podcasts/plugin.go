package podcasts

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/soltros/Supernova/internal/plugins"
)

type PodcastsPlugin struct {
	config plugins.PluginConfig
}

func init() {
	plugins.Register(&PodcastsPlugin{})
}

func (p *PodcastsPlugin) ID() string {
	return "podcasts"
}

func (p *PodcastsPlugin) Name() string {
	return "Podcast Index"
}

func (p *PodcastsPlugin) Description() string {
	return "Stream millions of podcasts via PodcastIndex.org API (Podcasting 2.0)"
}

func (p *PodcastsPlugin) Init(config plugins.PluginConfig) error {
	p.config = config
	return nil
}

func (p *PodcastsPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/plugins/podcasts/search", p.handleSearch)
	mux.HandleFunc("GET /api/plugins/podcasts/episodes", p.handleEpisodes)
}

func (p *PodcastsPlugin) doPodcastIndexRequest(endpoint string, queryValues string) (*http.Response, error) {
	apiKey := os.Getenv("PODCAST_INDEX_API_KEY")
	apiSecret := os.Getenv("PODCAST_INDEX_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("Podcast Index API keys are not configured in the .env file")
	}

	req, err := http.NewRequest("GET", "https://api.podcastindex.org/api/1.0"+endpoint+"?"+queryValues, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Supernova/1.0")

	// Authenticated request
	now := fmt.Sprintf("%d", time.Now().Unix())
	hash := sha1.New()
	hash.Write([]byte(apiKey + apiSecret + now))
	authHeader := fmt.Sprintf("%x", hash.Sum(nil))

	req.Header.Set("X-Auth-Date", now)
	req.Header.Set("X-Auth-Key", apiKey)
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	return client.Do(req)
}

func (p *PodcastsPlugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query", http.StatusBadRequest)
		return
	}

	resp, err := p.doPodcastIndexRequest("/search/byterm", "q="+query)
	if err != nil {
		if err.Error() == "Podcast Index API keys are not configured in the .env file" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to contact Podcast Index", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Podcast Index returned an error", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result["feeds"])
}

func (p *PodcastsPlugin) handleEpisodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing podcast ID", http.StatusBadRequest)
		return
	}

	resp, err := p.doPodcastIndexRequest("/episodes/bypodcastid", "id="+id)
	if err != nil {
		if err.Error() == "Podcast Index API keys are not configured in the .env file" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to contact Podcast Index", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Podcast Index returned an error", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result["items"])
}
