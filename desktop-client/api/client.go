package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// DefaultBaseURL is the default base URL for the Supernova API.
const DefaultBaseURL = "http://localhost:8080"

// Client is the Supernova API client.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// ==========================================
// Models
// ==========================================

type Album struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ArtistID string `json:"artist_id"`
	Year     int    `json:"year,omitempty"`
}

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	AlbumID  string `json:"album_id"`
	ArtistID string `json:"artist_id"`
	Duration int    `json:"duration"` // in seconds
	TrackNum int    `json:"track_num"`
}

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Heart struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

type Scrobble struct {
	TrackID string `json:"track_id"`
}

// ==========================================
// 1. Library (Metadata)
// ==========================================

func (c *Client) GetAlbums(limit, offset int) ([]Album, error) {
	reqURL := fmt.Sprintf("%s/api/albums?limit=%d&offset=%d", c.BaseURL, limit, offset)
	var albums []Album
	err := c.doGet(reqURL, &albums)
	return albums, err
}

func (c *Client) GetAlbum(id string) (*Album, error) {
	reqURL := fmt.Sprintf("%s/api/albums/%s", c.BaseURL, id)
	var album Album
	err := c.doGet(reqURL, &album)
	return &album, err
}

func (c *Client) GetTracks(albumID string, limit, offset int) ([]Track, error) {
	reqURL := fmt.Sprintf("%s/api/tracks?album_id=%s&limit=%d&offset=%d", c.BaseURL, albumID, limit, offset)
	var tracks []Track
	err := c.doGet(reqURL, &tracks)
	return tracks, err
}

func (c *Client) GetArtists(limit, offset int) ([]Artist, error) {
	reqURL := fmt.Sprintf("%s/api/artists?limit=%d&offset=%d", c.BaseURL, limit, offset)
	var artists []Artist
	err := c.doGet(reqURL, &artists)
	return artists, err
}

// ==========================================
// 2. Media Delivery
// ==========================================

// StreamAudioURL returns the URL to stream the audio file natively.
func (c *Client) StreamAudioURL(id, format, bitrate, timeOffset string) string {
	u, _ := url.Parse(fmt.Sprintf("%s/api/stream/%s", c.BaseURL, id))
	q := u.Query()
	if format != "" {
		q.Set("format", format)
	}
	if bitrate != "" {
		q.Set("bitrate", bitrate)
	}
	if timeOffset != "" {
		q.Set("time", timeOffset)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// AlbumArtURL returns the URL for the album art image.
func (c *Client) AlbumArtURL(id string) string {
	return fmt.Sprintf("%s/api/art/album/%s", c.BaseURL, id)
}

// ==========================================
// 3. Hearts (Favorites)
// ==========================================

func (c *Client) GetHearts() ([]Heart, error) {
	reqURL := fmt.Sprintf("%s/api/hearts", c.BaseURL)
	var hearts []Heart
	err := c.doGet(reqURL, &hearts)
	return hearts, err
}

func (c *Client) AddHeart(entityType, entityID string) error {
	reqURL := fmt.Sprintf("%s/api/hearts", c.BaseURL)
	heart := Heart{
		EntityType: entityType,
		EntityID:   entityID,
	}
	return c.doPost(reqURL, heart, nil)
}

func (c *Client) RemoveHeart(entityType, entityID string) error {
	reqURL := fmt.Sprintf("%s/api/hearts?entity_type=%s&entity_id=%s", c.BaseURL, entityType, entityID)
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

// ExportHearts returns the JSON backup of all hearts.
func (c *Client) ExportHearts() ([]byte, error) {
	reqURL := fmt.Sprintf("%s/api/hearts/export", c.BaseURL)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) ImportHearts(backupJSON []byte) error {
	reqURL := fmt.Sprintf("%s/api/hearts/import", c.BaseURL)
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(backupJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

// ==========================================
// 4. Scrobbling (Listen History)
// ==========================================

func (c *Client) LogScrobble(trackID string) error {
	reqURL := fmt.Sprintf("%s/api/scrobbles", c.BaseURL)
	scrobble := Scrobble{
		TrackID: trackID,
	}
	return c.doPost(reqURL, scrobble, nil)
}

func (c *Client) GetRecentScrobbles() ([]Scrobble, error) {
	reqURL := fmt.Sprintf("%s/api/scrobbles/recent", c.BaseURL)
	var scrobbles []Scrobble
	err := c.doGet(reqURL, &scrobbles)
	return scrobbles, err
}

// ==========================================
// Helper Methods
// ==========================================

func (c *Client) doGet(url string, v interface{}) error {
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) doPost(url string, body interface{}, response interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}
	return nil
}
