package podcasts

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"context"
	"encoding/xml"
	"bytes"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"github.com/soltros/Supernova/internal/plugins"
	"github.com/golang-jwt/jwt/v5"
)

type PodcastsPlugin struct {
	config plugins.PluginConfig
	repo   *database.Repository
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
	p.repo = config.Repo
	return nil
}

func (p *PodcastsPlugin) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/plugins/podcasts/search", p.handleSearch)
	mux.HandleFunc("GET /api/plugins/podcasts/episodes", p.handleEpisodes)
	
	mux.HandleFunc("GET /api/plugins/podcasts/subscriptions", p.handleGetSubscriptions)
	mux.HandleFunc("POST /api/plugins/podcasts/subscriptions", p.handleSubscribe)
	mux.HandleFunc("DELETE /api/plugins/podcasts/subscriptions", p.handleUnsubscribe)
	
	mux.HandleFunc("POST /api/plugins/podcasts/progress", p.handleSaveProgress)
	mux.HandleFunc("POST /api/plugins/podcasts/progress/batch", p.handleGetProgress)
	
	mux.HandleFunc("GET /api/plugins/podcasts/opml/export", p.handleExportOPML)
	mux.HandleFunc("POST /api/plugins/podcasts/opml/import", p.handleImportOPML)
}

func (p *PodcastsPlugin) authenticate(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		return "", fmt.Errorf("missing token")
	}
	tokenString := authHeader[7:]
	
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("missing user_id")
	}
	return userID, nil
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

	resp, err := p.doPodcastIndexRequest("/episodes/byfeedid", "id="+id)
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

func (p *PodcastsPlugin) handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	subs, err := p.repo.GetPodcastSubscriptions(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []models.PodcastSubscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (p *PodcastsPlugin) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var sub models.PodcastSubscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	sub.ID = fmt.Sprintf("psub_%d", time.Now().UnixNano())
	sub.UserID = userID
	if err := p.repo.AddPodcastSubscription(r.Context(), sub); err != nil {
		http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (p *PodcastsPlugin) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	feedID := r.URL.Query().Get("feed_id")
	if feedID == "" {
		http.Error(w, "Missing feed_id", http.StatusBadRequest)
		return
	}
	if err := p.repo.RemovePodcastSubscription(r.Context(), userID, feedID); err != nil {
		http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (p *PodcastsPlugin) handleSaveProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var prog models.PodcastProgress
	if err := json.NewDecoder(r.Body).Decode(&prog); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	prog.UserID = userID
	if err := p.repo.SavePodcastProgress(r.Context(), prog); err != nil {
		http.Error(w, "Failed to save progress", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (p *PodcastsPlugin) handleGetProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var req struct {
		EpisodeIDs []string `json:"episode_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	
	prog, err := p.repo.GetPodcastProgress(r.Context(), userID, req.EpisodeIDs)
	if err != nil {
		http.Error(w, "Failed to get progress", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prog)
}

func (p *PodcastsPlugin) handleExportOPML(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	subs, err := p.repo.GetPodcastSubscriptions(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=\"supernova_podcasts.opml\"")
	
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<opml version=\"1.0\">\n<head><title>Supernova Podcast Subscriptions</title></head>\n<body>\n<outline text=\"Feeds\">\n"))
	for _, sub := range subs {
		var bufTitle bytes.Buffer
		xml.EscapeText(&bufTitle, []byte(sub.Title))
		
		var bufURL bytes.Buffer
		xml.EscapeText(&bufURL, []byte(sub.FeedURL))
		
		w.Write([]byte(fmt.Sprintf("\t<outline type=\"rss\" text=\"%s\" xmlUrl=\"%s\" />\n", bufTitle.String(), bufURL.String())))
	}
	w.Write([]byte("</outline>\n</body>\n</opml>"))
}

type OPMLOutline struct {
	Type     string        `xml:"type,attr"`
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	Outlines []OPMLOutline `xml:"outline"`
}

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Body    struct {
		Outlines []OPMLOutline `xml:"outline"`
	} `xml:"body"`
}

func extractFeeds(outlines []OPMLOutline) []string {
	var urls []string
	for _, o := range outlines {
		if o.XMLURL != "" {
			urls = append(urls, o.XMLURL)
		}
		urls = append(urls, extractFeeds(o.Outlines)...)
	}
	return urls
}

func (p *PodcastsPlugin) handleImportOPML(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	
	// Parse OPML file
	r.ParseMultipartForm(10 << 20) // 10 MB
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var opml OPML
	if err := xml.NewDecoder(file).Decode(&opml); err != nil {
		http.Error(w, "Invalid OPML file", http.StatusBadRequest)
		return
	}
	
	urls := extractFeeds(opml.Body.Outlines)
	
	// For each URL, fetch podcast info from PodcastIndex
	go func(urls []string, userID string) {
		for _, u := range urls {
			resp, err := p.doPodcastIndexRequest("/podcasts/byfeedurl", "url="+u)
			if err != nil || resp.StatusCode != 200 {
				continue
			}
			
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				if feed, ok := result["feed"].(map[string]interface{}); ok {
					idFloat, _ := feed["id"].(float64)
					title, _ := feed["title"].(string)
					img, _ := feed["image"].(string)
					
					sub := models.PodcastSubscription{
						ID:       fmt.Sprintf("psub_%d", time.Now().UnixNano()),
						UserID:   userID,
						FeedID:   fmt.Sprintf("%.0f", idFloat),
						FeedURL:  u,
						Title:    title,
						ImageURL: img,
					}
					p.repo.AddPodcastSubscription(context.Background(), sub)
				}
			}
			resp.Body.Close()
			time.Sleep(100 * time.Millisecond) // Don't hammer the API
		}
	}(urls, userID)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Import started in background"))
}
