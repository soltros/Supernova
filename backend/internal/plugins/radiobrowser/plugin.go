package radiobrowser

import (
	"encoding/json"
	"net/http"
	"time"
	"fmt"
	"os"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"github.com/soltros/Supernova/internal/plugins"
	"github.com/golang-jwt/jwt/v5"
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

	mux.HandleFunc("GET /api/plugins/radio/subscriptions", p.handleGetSubscriptions)
	mux.HandleFunc("POST /api/plugins/radio/subscriptions", p.handleSubscribe)
	mux.HandleFunc("DELETE /api/plugins/radio/subscriptions", p.handleUnsubscribe)
}

func (p *RadioPlugin) authenticate(r *http.Request) (string, error) {
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

func (p *RadioPlugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	country := r.URL.Query().Get("country")
	
	if query == "" && country == "" {
		http.Error(w, "query parameter 'q' or 'country' is required", http.StatusBadRequest)
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
	if query != "" {
		q.Add("name", query)
	}
	if country != "" {
		q.Add("country", country)
	}
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

	var results []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		http.Error(w, "failed to parse upstream response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (p *RadioPlugin) handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	subs, err := p.repo.GetRadioSubscriptions(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []models.RadioSubscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (p *RadioPlugin) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var sub models.RadioSubscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	sub.ID = fmt.Sprintf("rsub_%d", time.Now().UnixNano())
	sub.UserID = userID
	if err := p.repo.AddRadioSubscription(r.Context(), sub); err != nil {
		http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (p *RadioPlugin) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	userID, err := p.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	stationID := r.URL.Query().Get("station_id")
	if stationID == "" {
		http.Error(w, "Missing station_id", http.StatusBadRequest)
		return
	}
	if err := p.repo.RemoveRadioSubscription(r.Context(), userID, stationID); err != nil {
		http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
