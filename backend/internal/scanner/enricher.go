package scanner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	running atomic.Bool
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
		ticker:=time.NewTicker(5*time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Enrichment Worker shutting down...")
				return
			case <-e.trigger:
			case <-ticker.C:
			}
			if !e.running.CompareAndSwap(false,true){continue}
			e.processQueue(ctx)
			e.processArtistQueue(ctx)
			e.processAlbumQueue(ctx)
			e.running.Store(false)
		}
	}()
}

// processQueue iterates over the database finding albums missing an MBID
func (e *Enricher) processQueue(ctx context.Context) {
	if e.mbClient==nil{return}
	albums,err:=e.repo.GetUnenrichedAlbums(ctx,50)
	if err!=nil{log.Printf("Enricher DB error: %v",err);return}
	var wg sync.WaitGroup
	sem:=make(chan struct{},3)
	for _,a:=range albums{
		if ctx.Err()!=nil{return}
		wg.Add(1);sem<-struct{}{}
		go func(album database.UnenrichedAlbum){
			defer wg.Done();defer func(){<-sem}()
			meta:=&models.TrackMetadata{Title:album.TrackTitle,Album:album.AlbumTitle,Artist:album.ArtistName}
			if err:=e.mbClient.EnhanceMetadata(meta);err!=nil{
				log.Printf("MusicBrainz transient error for %s; leaving pending for retry: %v",album.AlbumTitle,err)
				return
			}
			albumMBID:=meta.AlbumMBID
			if albumMBID==""{albumMBID="NOT_FOUND"}
			if err:=e.repo.UpdateMBIDs(ctx,album.AlbumID,albumMBID,album.ArtistID,meta.ArtistMBID);err!=nil{
				log.Printf("Failed to persist MusicBrainz result for %s: %v",album.AlbumTitle,err)
				return
			}
			if albumMBID!="NOT_FOUND"{log.Printf("Successfully background-enriched album: %s",album.AlbumTitle)}
		}(a)
	}
	wg.Wait()
}

// processArtistQueue iterates over the database finding artists missing LastFM imagery/bios
func (e *Enricher) processArtistQueue(ctx context.Context) {
	if e.lastfm==nil{return}
	{
		artists, err := e.repo.GetUnenrichedArtists(ctx, 10) // bounded fair batch
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

				info, err := e.lastfm.GetArtistInfo(artist.Name)
				if err != nil {
					log.Printf("Last.fm transient error for artist %s; leaving pending for retry: %v",artist.Name,err)
					return
				}
				if info == nil || len(info.Artist.Image) == 0 {
					if err:=e.repo.UpdateArtistInfo(ctx,artist.ID,"NOT_FOUND","");err!=nil{log.Printf("Failed to persist artist NOT_FOUND for %s: %v",artist.Name,err)}
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
	if e.lastfm==nil{return}
	{
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

				info, err := e.lastfm.GetAlbumInfo(album.ArtistName, album.AlbumTitle, "", 0)
				if err != nil {
					log.Printf("Last.fm transient error for album %s; leaving pending for retry: %v",album.AlbumTitle,err)
					return
				}
				if info == nil || info.Error > 0 {
					if err:=e.repo.UpdateAlbumBio(ctx,album.AlbumID,"NOT_FOUND");err!=nil{log.Printf("Failed to persist album NOT_FOUND for %s: %v",album.AlbumTitle,err)}
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
