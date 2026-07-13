package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
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

func (c *Client) GetAlbums(ctx context.Context, limit, offset int) ([]Album, error) {
	reqURL := fmt.Sprintf("%s/api/albums?limit=%d&offset=%d", c.BaseURL, limit, offset)
	var albums []Album
	err := c.doGet(ctx, reqURL, &albums)
	return albums, err
}

func (c *Client) GetAlbum(ctx context.Context, id string) (*Album, error) {
	reqURL := fmt.Sprintf("%s/api/albums/%s", c.BaseURL, url.PathEscape(id))
	var album Album
	err := c.doGet(ctx, reqURL, &album)
	return &album, err
}

func (c *Client) GetTracks(ctx context.Context, albumID string, limit, offset int) ([]Track, error) {
	reqURL := fmt.Sprintf("%s/api/tracks?album_id=%s&limit=%d&offset=%d", c.BaseURL, url.QueryEscape(albumID), limit, offset)
	var tracks []Track
	err := c.doGet(ctx, reqURL, &tracks)
	return tracks, err
}

func (c *Client) GetArtists(ctx context.Context, limit, offset int) ([]Artist, error) {
	reqURL := fmt.Sprintf("%s/api/artists?limit=%d&offset=%d", c.BaseURL, limit, offset)
	var artists []Artist
	err := c.doGet(ctx, reqURL, &artists)
	return artists, err
}

// ==========================================
// 2. Media Delivery
// ==========================================

// StreamAudioURL returns the URL to stream the audio file natively.
func (c *Client) StreamAudioURL(id, format, bitrate, timeOffset string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/stream/%s", c.BaseURL, url.PathEscape(id)))
	if err != nil {
		return "", err
	}
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
	return u.String(), nil
}

// AlbumArtURL returns the URL for the album art image.
func (c *Client) AlbumArtURL(id string) string {
	return fmt.Sprintf("%s/api/art/album/%s", c.BaseURL, url.PathEscape(id))
}

// ==========================================
// 3. Hearts (Favorites)
// ==========================================

func (c *Client) GetHearts(ctx context.Context) ([]Heart, error) {
	reqURL := fmt.Sprintf("%s/api/hearts", c.BaseURL)
	var hearts []Heart
	err := c.doGet(ctx, reqURL, &hearts)
	return hearts, err
}

func (c *Client) AddHeart(ctx context.Context, entityType, entityID string) error {
	reqURL := fmt.Sprintf("%s/api/hearts", c.BaseURL)
	heart := Heart{
		EntityType: entityType,
		EntityID:   entityID,
	}
	return c.doPost(ctx, reqURL, heart, nil)
}

func (c *Client) RemoveHeart(ctx context.Context, entityType, entityID string) error {
	q := url.Values{}
	q.Set("entity_type", entityType)
	q.Set("entity_id", entityID)
	reqURL := fmt.Sprintf("%s/api/hearts?%s", c.BaseURL, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(body))
	}
	return nil
}

// ExportHearts returns the JSON backup of all hearts.
func (c *Client) ExportHearts(ctx context.Context) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/api/hearts/export", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(body))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) ImportHearts(ctx context.Context, backupJSON []byte) error {
	reqURL := fmt.Sprintf("%s/api/hearts/import", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(backupJSON))
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(body))
	}
	return nil
}

// ==========================================
// 4. Scrobbling (Listen History)
// ==========================================

func (c *Client) LogScrobble(ctx context.Context, trackID string) error {
	reqURL := fmt.Sprintf("%s/api/scrobbles", c.BaseURL)
	scrobble := Scrobble{
		TrackID: trackID,
	}
	return c.doPost(ctx, reqURL, scrobble, nil)
}

func (c *Client) GetRecentScrobbles(ctx context.Context) ([]Scrobble, error) {
	reqURL := fmt.Sprintf("%s/api/scrobbles/recent", c.BaseURL)
	var scrobbles []Scrobble
	err := c.doGet(ctx, reqURL, &scrobbles)
	return scrobbles, err
}

// ==========================================
// Helper Methods
// ==========================================

func (c *Client) doGet(ctx context.Context, url string, v interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) doPost(ctx context.Context, url string, body interface{}, response interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(b))
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
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(respBody))
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}
	return nil
}
