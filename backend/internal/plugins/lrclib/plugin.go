package lrclib

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/soltros/Supernova/internal/plugins"
)

type LRCLibPlugin struct {
	client *http.Client
}

// Ensure LRCLibPlugin implements plugins.Plugin
var _ plugins.Plugin = (*LRCLibPlugin)(nil)

func init() {
	plugins.Register(&LRCLibPlugin{})
}

func (p *LRCLibPlugin) ID() string {
	return "lrclib"
}

func (p *LRCLibPlugin) Name() string {
	return "LRCLib Time-Synced Lyrics"
}

func (p *LRCLibPlugin) Description() string {
	return "Fetches time-synced lyrics from LRCLib for the currently playing track."
}

func (p *LRCLibPlugin) Init(config plugins.PluginConfig) error {
	p.client = &http.Client{
		Timeout: 15 * time.Second,
	}
	return nil
}

func (p *LRCLibPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugins/lrclib/lyrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.handleGetLyrics(w, r)
	})
}

func (p *LRCLibPlugin) handleGetLyrics(w http.ResponseWriter, r *http.Request) {
	// LRCLib expects artist_name, track_name, album_name, duration
	artist := r.URL.Query().Get("artist_name")
	track := r.URL.Query().Get("track_name")
	album := r.URL.Query().Get("album_name")
	durationStr := r.URL.Query().Get("duration") // in seconds

	if artist == "" || track == "" {
		http.Error(w, "missing artist_name or track_name", http.StatusBadRequest)
		return
	}

	query := url.Values{}
	query.Add("artist_name", artist)
	query.Add("track_name", track)
	if album != "" {
		query.Add("album_name", album)
	}
	if durationStr != "" && durationStr != "0" {
		query.Add("duration", durationStr)
	}

	reqURL := fmt.Sprintf("https://lrclib.net/api/get?%s", query.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Supernova Media Server v2026.07.17")

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[LRCLib] Request failed: %v\n", err)
		http.Error(w, "failed to connect to lrclib", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		http.Error(w, "lyrics not found", http.StatusNotFound)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "lrclib api error", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		http.Error(w, "failed to read response", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(body)
	if err != nil {
		// Can't do much if writing fails, just log it or ignore
		return
	}
}
