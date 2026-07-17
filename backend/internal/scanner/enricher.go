package scanner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/external"
	"github.com/soltros/Supernova/internal/models"
)

// Enricher handles the slow, rate-limited process of querying external APIs
// in the background without blocking the main application or library scanning.
type Enricher struct {
	repo     *database.Repository
	mbClient *external.MusicBrainzClient
	lastfm   *external.LastFmClient
	trigger  chan struct{}
}

// NewEnricher initializes the background daemon.
func NewEnricher(repo *database.Repository, mbClient *external.MusicBrainzClient, lastfm *external.LastFmClient) *Enricher {
	return &Enricher{
		repo:     repo,
		mbClient: mbClient,
		lastfm:   lastfm,
		trigger:  make(chan struct{}, 1),
	}
}

// Trigger wakes up the enricher to check for new unenriched tracks.
func (e *Enricher) Trigger() {
	select {
	case e.trigger <- struct{}{}:
	default:
	}
}

// Start begins the background worker loop. This should run for the lifecycle of the app.
func (e *Enricher) Start(ctx context.Context) {
	log.Println("Background Enrichment Worker started")
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Enrichment Worker shutting down...")
				return
			case <-e.trigger:
				e.processQueue(ctx)
				e.processArtistQueue(ctx)
				e.processAlbumQueue(ctx)
			}
		}
	}()
}

// processQueue iterates over the database finding albums missing an MBID
func (e *Enricher) processQueue(ctx context.Context) {
	log.Println("Enricher waking up to check database for missing metadata...")
	
	// Process batches of 50 unenriched albums at a time
	for {
		albums, err := e.repo.GetUnenrichedAlbums(ctx, 50)
		if err != nil {
			log.Printf("Enricher DB error: %v", err)
			return
		}
		
		if len(albums) == 0 {
			// If no albums need MBID enrichment, check for missing artist data via LastFM
			e.processArtistQueue(ctx)
			return
		}

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 3)

		for _, a := range albums {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			semaphore <- struct{}{}

			go func(album database.UnenrichedAlbum) {
				defer wg.Done()
				defer func() { <-semaphore }()
				
				meta := &models.TrackMetadata{
					Title:  album.TrackTitle,
					Album:  album.AlbumTitle,
					Artist: album.ArtistName,
				}
				
				// mbClient automatically sleeps for 1.1s to respect MusicBrainz rate limits
				err := e.mbClient.EnhanceMetadata(meta)
				
				if err == nil {
					// If we found an MBID, update the database
					if meta.AlbumMBID != "" {
						_ = e.repo.UpdateMBIDs(ctx, album.AlbumID, meta.AlbumMBID, album.ArtistID, meta.ArtistMBID)
						log.Printf("Successfully background-enriched album: %s", album.AlbumTitle)
					} else {
						// To prevent infinite loops on albums that don't exist in MusicBrainz, 
						// we write a special flag 'NOT_FOUND' so GetUnenrichedAlbums ignores it next time.
						_ = e.repo.UpdateMBIDs(ctx, album.AlbumID, "NOT_FOUND", album.ArtistID, "")
						log.Printf("No MusicBrainz data found for album: %s", album.AlbumTitle)
					}
				} else {
					// Prevent infinite loop on API/network errors by marking it with a failure flag
					_ = e.repo.UpdateMBIDs(ctx, album.AlbumID, "ERROR", album.ArtistID, "")
					log.Printf("MusicBrainz API error for album %s: %v", album.AlbumTitle, err)
				}
			}(a)
		}
		wg.Wait()
	}
}

// processArtistQueue iterates over the database finding artists missing LastFM imagery/bios
func (e *Enricher) processArtistQueue(ctx context.Context) {
	for {
		artists, err := e.repo.GetUnenrichedArtists(ctx, 10) // Small batches for LastFM
		if err != nil {
			log.Printf("Enricher DB error: %v", err)
			return
		}

		if len(artists) == 0 {
			log.Println("Enricher finished processing all pending artists. Sleeping.")
			return
		}

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 5) // Limit concurrent Last.fm API calls to 5

		for _, a := range artists {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			semaphore <- struct{}{} // Acquire semaphore

			go func(artist models.Artist) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release semaphore

				// If LastFM isn't configured, mark as NOT_FOUND to avoid infinite loops
				if e.lastfm == nil {
					_ = e.repo.UpdateArtistInfo(ctx, artist.ID, "NOT_FOUND", "")
					return
				}

				info, err := e.lastfm.GetArtistInfo(artist.Name)
				if err != nil || info == nil || len(info.Artist.Image) == 0 {
					log.Printf("No LastFM data found for artist: %s", artist.Name)
					_ = e.repo.UpdateArtistInfo(ctx, artist.ID, "NOT_FOUND", "")
					return
				}

				// Find the largest image (usually last in array)
				imgURL := ""
				for _, img := range info.Artist.Image {
					if img.URL != "" && !strings.Contains(img.URL, "2a96cbd8b46e442fc41c2b86b821562f") {
						imgURL = img.URL
					}
				}

				// If we didn't find a valid image, try scraping the Last.fm website directly
				if imgURL == "" {
					scrapedImage := e.lastfm.ScrapeArtistImage(artist.Name)
					if scrapedImage != "" {
						imgURL = scrapedImage
					}
				}

				if imgURL == "" {
					imgURL = "NOT_FOUND"
				}

				bio := info.Artist.Bio.Summary
				err = e.repo.UpdateArtistInfo(ctx, artist.ID, imgURL, bio)
				if err != nil {
					log.Printf("Failed to update artist info in DB for %s: %v", artist.Name, err)
				} else {
					log.Printf("Successfully enriched artist via LastFM: %s", artist.Name)
				}
				
				// Additionally fetch top tracks to update local track popularity
				topTracks, err := e.lastfm.GetArtistTopTracks(artist.Name)
				if err == nil && topTracks != nil {
					for _, track := range topTracks.Toptracks.Track {
						// We'll use listeners as a proxy for popularity
						var popularity int
						if track.Listeners != "" {
							fmt.Sscanf(track.Listeners, "%d", &popularity)
						} else {
							fmt.Sscanf(track.Playcount, "%d", &popularity)
						}
						
						if popularity > 0 {
							_ = e.repo.UpdateArtistTracksPopularity(ctx, artist.ID, track.Name, popularity)
						}
					}
					log.Printf("Updated top tracks popularity for artist: %s", artist.Name)
				}
			}(a)
		}
		
		wg.Wait() // Wait for batch to finish before fetching next batch
	}
}

// processAlbumQueue iterates over the database finding albums missing LastFM bios/track durations
func (e *Enricher) processAlbumQueue(ctx context.Context) {
	for {
		albums, err := e.repo.GetAlbumsMissingBio(ctx, 10)
		if err != nil {
			log.Printf("Enricher DB error: %v", err)
			return
		}

		if len(albums) == 0 {
			log.Println("Enricher finished processing all pending album bios. Sleeping.")
			return
		}

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 5)

		for _, a := range albums {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			semaphore <- struct{}{}

			go func(album database.UnenrichedAlbum) {
				defer wg.Done()
				defer func() { <-semaphore }()

				if e.lastfm == nil {
					_ = e.repo.UpdateAlbumBio(ctx, album.AlbumID, "NOT_FOUND")
					return
				}

				info, err := e.lastfm.GetAlbumInfo(album.ArtistName, album.AlbumTitle, "", 0)
				if err != nil || info == nil || info.Error > 0 {
					log.Printf("No LastFM data found for album: %s by %s", album.AlbumTitle, album.ArtistName)
					_ = e.repo.UpdateAlbumBio(ctx, album.AlbumID, "NOT_FOUND")
					return
				}

				bio := info.Album.Wiki.Summary
				if bio == "" {
					bio = "NOT_FOUND" // mark to avoid retrying
				}

				err = e.repo.UpdateAlbumBio(ctx, album.AlbumID, bio)
				if err != nil {
					log.Printf("Failed to update album bio in DB for %s: %v", album.AlbumTitle, err)
				} else if bio != "NOT_FOUND" {
					log.Printf("Successfully enriched album bio via LastFM: %s", album.AlbumTitle)
				}

				// Update track durations from Last.fm response
				for _, track := range info.Album.Tracks.Track {
					if track.Duration != "" {
						var durSecs int
						fmt.Sscanf(track.Duration, "%d", &durSecs)
						if durSecs > 0 {
							_ = e.repo.UpdateTrackDuration(ctx, album.AlbumID, track.Name, durSecs*1000)
						}
					}
				}
			}(a)
		}
		wg.Wait()
	}
}
