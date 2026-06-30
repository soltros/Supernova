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

type ArtistTopTracksResponse struct {
	Toptracks struct {
		Track []struct {
			Name       string `json:"name"`
			Playcount  string `json:"playcount"`
			Listeners  string `json:"listeners"`
		} `json:"track"`
	} `json:"toptracks"`
}

// GetArtistTopTracks fetches the most popular tracks for a given artist globally.
func (c *LastFmClient) GetArtistTopTracks(artistName string) (*ArtistTopTracksResponse, error) {
	params := url.Values{}
	params.Add("method", "artist.gettoptracks")
	params.Add("artist", artistName)
	params.Add("api_key", c.apiKey)
	params.Add("format", "json")
	params.Add("limit", "100")

	reqURL := fmt.Sprintf("%s?%s", lastFmBaseURL, params.Encode())
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data ArtistTopTracksResponse
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
	// Must be strictly alphabetical according to UTF-8/ASCII byte values,
	// which is required by Last.fm (e.g., artist[10] must come before artist[1]).
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

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
		"method":       "track.scrobble",
		"artist[0]":    artist,
		"track[0]":     track,
		"timestamp[0]": fmt.Sprintf("%d", timestamp),
		"api_key":      c.apiKey,
		"sk":           sessionKey,
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

// AlbumInfoResponse represents the Last.fm album.getInfo response
type AlbumInfoResponse struct {
	Album struct {
		Name   string `json:"name"`
		Artist string `json:"artist"`
		Mbid   string `json:"mbid"`
		Image  []struct {
			URL  string `json:"#text"`
			Size string `json:"size"`
		} `json:"image"`
		Tracks struct {
			Track []struct {
				Name     string `json:"name"`
				Duration string `json:"duration"` // Returned as string seconds often
			} `json:"track"`
		} `json:"tracks"`
	} `json:"album"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// TrackInfoResponse represents the Last.fm track.getInfo response
type TrackInfoResponse struct {
	Track struct {
		Name     string `json:"name"`
		Mbid     string `json:"mbid"`
		Duration string `json:"duration"` // milliseconds
		Album    struct {
			Title string `json:"title"`
			Image []struct {
				URL  string `json:"#text"`
				Size string `json:"size"`
			} `json:"image"`
		} `json:"album"`
	} `json:"track"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// TopAlbumsResponse represents the Last.fm tag.getTopAlbums response
type TopAlbumsResponse struct {
	Albums struct {
		Album []struct {
			Name  string `json:"name"`
			Mbid  string `json:"mbid"`
			Image []struct {
				URL  string `json:"#text"`
				Size string `json:"size"`
			} `json:"image"`
			Artist struct {
				Name string `json:"name"`
				Mbid string `json:"mbid"`
			} `json:"artist"`
		} `json:"album"`
	} `json:"albums"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// TopTracksResponse represents the Last.fm tag.getTopTracks response
type TopTracksResponse struct {
	Tracks struct {
		Track []struct {
			Name     string `json:"name"`
			Duration string `json:"duration"`
			Mbid     string `json:"mbid"`
			Artist   struct {
				Name string `json:"name"`
				Mbid string `json:"mbid"`
			} `json:"artist"`
		} `json:"track"`
	} `json:"tracks"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// GetAlbumInfo fetches album metadata and tracklist
func (c *LastFmClient) GetAlbumInfo(artist, album, mbid string, autocorrect int) (*AlbumInfoResponse, error) {
	params := url.Values{}
	params.Add("method", "album.getinfo")
	if mbid != "" {
		params.Add("mbid", mbid)
	} else {
		params.Add("artist", artist)
		params.Add("album", album)
	}
	if autocorrect > 0 {
		params.Add("autocorrect", "1")
	}
	params.Add("api_key", c.apiKey)
	params.Add("format", "json")

	reqURL := fmt.Sprintf("%s?%s", lastFmBaseURL, params.Encode())
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data AlbumInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error > 0 {
		return nil, fmt.Errorf("last.fm api error %d: %s", data.Error, data.Message)
	}
	return &data, nil
}

// GetTrackInfo fetches track metadata
func (c *LastFmClient) GetTrackInfo(artist, track, mbid string, autocorrect int) (*TrackInfoResponse, error) {
	params := url.Values{}
	params.Add("method", "track.getinfo")
	if mbid != "" {
		params.Add("mbid", mbid)
	} else {
		params.Add("artist", artist)
		params.Add("track", track)
	}
	if autocorrect > 0 {
		params.Add("autocorrect", "1")
	}
	params.Add("api_key", c.apiKey)
	params.Add("format", "json")

	reqURL := fmt.Sprintf("%s?%s", lastFmBaseURL, params.Encode())
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data TrackInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error > 0 {
		return nil, fmt.Errorf("last.fm api error %d: %s", data.Error, data.Message)
	}
	return &data, nil
}

// GetTopAlbumsByTag fetches top albums for a tag
func (c *LastFmClient) GetTopAlbumsByTag(tag string, limit, page int) (*TopAlbumsResponse, error) {
	params := url.Values{}
	params.Add("method", "tag.gettopalbums")
	params.Add("tag", tag)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if page > 0 {
		params.Add("page", fmt.Sprintf("%d", page))
	}
	params.Add("api_key", c.apiKey)
	params.Add("format", "json")

	reqURL := fmt.Sprintf("%s?%s", lastFmBaseURL, params.Encode())
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data TopAlbumsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error > 0 {
		return nil, fmt.Errorf("last.fm api error %d: %s", data.Error, data.Message)
	}
	return &data, nil
}

// GetTopTracksByTag fetches top tracks for a tag
func (c *LastFmClient) GetTopTracksByTag(tag string, limit, page int) (*TopTracksResponse, error) {
	params := url.Values{}
	params.Add("method", "tag.gettoptracks")
	params.Add("tag", tag)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if page > 0 {
		params.Add("page", fmt.Sprintf("%d", page))
	}
	params.Add("api_key", c.apiKey)
	params.Add("format", "json")

	reqURL := fmt.Sprintf("%s?%s", lastFmBaseURL, params.Encode())
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data TopTracksResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error > 0 {
		return nil, fmt.Errorf("last.fm api error %d: %s", data.Error, data.Message)
	}
	return &data, nil
}

// ScrapeArtistImage fetches the real artist image from Last.fm's HTML, bypassing their API limitation
func (c *LastFmClient) ScrapeArtistImage(artistName string) string {
	reqURL := "https://www.last.fm/music/" + url.PathEscape(artistName)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return ""
	}
	// Mimic a browser to avoid simple bot blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	
	resp, err := c.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	body := string(bodyBytes)

	// Simple string matching to find the og:image meta tag
	startStr := `property="og:image"`
	idx := strings.Index(body, startStr)
	if idx == -1 {
		return ""
	}
	body = body[idx:]
	contentIdx := strings.Index(body, `content="`)
	if contentIdx == -1 {
		return ""
	}
	body = body[contentIdx+9:]
	endIdx := strings.Index(body, `"`)
	if endIdx == -1 {
		return ""
	}
	imgURL := body[:endIdx]
	
	// If it returns the default star even in og:image, reject it
	if strings.Contains(imgURL, "2a96cbd8b46e442fc41c2b86b821562f") {
		return ""
	}
	return imgURL
}
