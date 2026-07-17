package external

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"sync"

	"github.com/soltros/Supernova/internal/models"
)

const mbBaseURL = "https://musicbrainz.org/ws/2"

// MusicBrainzClient handles communication with the official MusicBrainz API
type MusicBrainzClient struct {
	client    *http.Client
	userAgent string
	mu        sync.Mutex
}

// NewMusicBrainzClient creates a new client. MusicBrainz strictly requires a descriptive User-Agent.
func NewMusicBrainzClient(appName, version, contact string) *MusicBrainzClient {
	return &MusicBrainzClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent: fmt.Sprintf("%s/%s ( %s )", appName, version, contact),
	}
}

// mbSearchResponse represents the JSON response from the MusicBrainz recording search API
type mbSearchResponse struct {
	Recordings []mbRecording `json:"recordings"`
}

type mbRecording struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Releases []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"releases"`
	ArtistCredit []struct {
		Name   string `json:"name"`
		Artist struct {
			ID string `json:"id"`
		} `json:"artist"`
	} `json:"artist-credit"`
}

// EnhanceMetadata takes raw local metadata (from ID3/FLAC tags) and queries MusicBrainz
// to find official IDs and correct any spelling/casing mistakes.
func (c *MusicBrainzClient) EnhanceMetadata(raw *models.TrackMetadata) error {
	// If it doesn't have an artist or title, we can't search effectively
	if raw.Title == "" || raw.Artist == "" {
		return nil
	}

	// Build a Lucene query string for the MusicBrainz API
	query := fmt.Sprintf("recording:\"%s\" AND artist:\"%s\"", raw.Title, raw.Artist)
	if raw.Album != "" {
		query += fmt.Sprintf(" AND release:\"%s\"", raw.Album)
	}

	reqURL := fmt.Sprintf("%s/recording/?query=%s&fmt=json&limit=1", mbBaseURL, url.QueryEscape(query))
	
	// Globally lock to enforce 1 req/sec rate limit across all worker routines
	c.mu.Lock()
	defer c.mu.Unlock()
	// WARNING: MusicBrainz rate limits heavily (1 request per second per IP).
	// We MUST sleep while holding the lock to avoid getting banned.
	defer time.Sleep(1100 * time.Millisecond)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("musicbrainz returned status: %d", resp.StatusCode)
	}

	var data mbSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	// If we found a match, apply the highly accurate MusicBrainz data
	if len(data.Recordings) > 0 {
		rec := data.Recordings[0]
		
		// Set the official IDs (crucial for smart de-duplication later)
		raw.TrackMBID = rec.ID
		raw.Title = rec.Title // Corrects capitalization/spelling
		
		// Update Artist information
		if len(rec.ArtistCredit) > 0 {
			raw.Artist = rec.ArtistCredit[0].Name
			raw.ArtistMBID = rec.ArtistCredit[0].Artist.ID
		}

		// Update Album information
		if len(rec.Releases) > 0 {
			raw.Album = rec.Releases[0].Title
			raw.AlbumMBID = rec.Releases[0].ID
		}
	}

	return nil
}
