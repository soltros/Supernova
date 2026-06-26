package external

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const lastFmBaseURL = "http://ws.audioscrobbler.com/2.0/"

// LastFmClient handles communication with the Last.fm API
type LastFmClient struct {
	apiKey    string
	apiSecret string
	client    *http.Client
}

// NewLastFmClient initializes the client. Requires both Key and Secret for scrobbling support.
func NewLastFmClient(apiKey, apiSecret string) *LastFmClient {
	return &LastFmClient{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// --- 1. RICH METADATA (Read-Only) ---

type ArtistInfoResponse struct {
	Artist struct {
		Name  string `json:"name"`
		Image []struct {
			URL  string `json:"#text"`
			Size string `json:"size"`
		} `json:"image"`
		Bio struct {
			Summary string `json:"summary"`
			Content string `json:"content"`
		} `json:"bio"`
	} `json:"artist"`
}

// GetArtistInfo fetches rich imagery and biography for an artist.
// This does not require authentication, just the API key.
func (c *LastFmClient) GetArtistInfo(artistName string) (*ArtistInfoResponse, error) {
	params := url.Values{}
	params.Add("method", "artist.getinfo")
	params.Add("artist", artistName)
	params.Add("api_key", c.apiKey)
	params.Add("format", "json")

	reqURL := fmt.Sprintf("%s?%s", lastFmBaseURL, params.Encode())
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data ArtistInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// --- 2. SCROBBLING (Authenticated Write Operations) ---

// generateSignature creates an MD5 signature. Last.fm requires this for all write operations (like scrobbling)
// to verify the request was actually made by our application using the secret key.
func (c *LastFmClient) generateSignature(params map[string]string) string {
	var keys []string
	for k := range params {
		if k != "format" && k != "callback" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys) // Must be alphabetical

	var sigStr string
	for _, k := range keys {
		sigStr += k + params[k]
	}
	sigStr += c.apiSecret

	hash := md5.Sum([]byte(sigStr))
	return hex.EncodeToString(hash[:])
}

// UpdateNowPlaying tells Last.fm that the user just started listening to a track.
// This displays "Scrobbling now" on their Last.fm profile.
func (c *LastFmClient) UpdateNowPlaying(sessionKey, artist, track string) error {
	params := map[string]string{
		"method":  "track.updateNowPlaying",
		"artist":  artist,
		"track":   track,
		"api_key": c.apiKey,
		"sk":      sessionKey,
	}

	return c.postAuthenticated(params)
}

// Scrobble records a track as definitively "played" on the user's Last.fm account.
// This should be called after a track has played for at least 50% of its duration, or 4 minutes.
func (c *LastFmClient) Scrobble(sessionKey, artist, track string, timestamp int64) error {
	params := map[string]string{
		"method":    "track.scrobble",
		"artist":    artist,
		"track":     track,
		"timestamp": fmt.Sprintf("%d", timestamp),
		"api_key":   c.apiKey,
		"sk":        sessionKey,
	}

	return c.postAuthenticated(params)
}

// postAuthenticated handles the boilerplate of signing and executing a write request
func (c *LastFmClient) postAuthenticated(params map[string]string) error {
	apiSig := c.generateSignature(params)
	
	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}
	form.Add("api_sig", apiSig)
	form.Add("format", "json")

	req, err := http.NewRequest(http.MethodPost, lastFmBaseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("last.fm api error: %s", string(body))
	}

	return nil
}
