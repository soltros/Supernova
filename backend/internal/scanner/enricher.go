package scanner

import (
	"context"
	"log"

	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/external"
	"github.com/soltros/Supernova/internal/models"
)

// Enricher handles the slow, rate-limited process of querying external APIs
// in the background without blocking the main application or library scanning.
type Enricher struct {
	repo     *database.Repository
	mbClient *external.MusicBrainzClient
	trigger  chan struct{}
}

// NewEnricher initializes the background daemon.
func NewEnricher(repo *database.Repository, mbClient *external.MusicBrainzClient) *Enricher {
	return &Enricher{
		repo:     repo,
		mbClient: mbClient,
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
			log.Println("Enricher finished processing all pending albums. Sleeping.")
			return
		}

		for _, a := range albums {
			// Check if app is shutting down
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			meta := &models.TrackMetadata{
				Title:  a.TrackTitle,
				Album:  a.AlbumTitle,
				Artist: a.ArtistName,
			}
			
			// mbClient automatically sleeps for 1.1s to respect MusicBrainz rate limits
			err := e.mbClient.EnhanceMetadata(meta)
			
			if err == nil {
				// If we found an MBID, update the database
				if meta.AlbumMBID != "" {
					_ = e.repo.UpdateMBIDs(ctx, a.AlbumID, meta.AlbumMBID, a.ArtistID, meta.ArtistMBID)
					log.Printf("Successfully background-enriched album: %s", a.AlbumTitle)
				} else {
					// To prevent infinite loops on albums that don't exist in MusicBrainz, 
					// we write a special flag 'NOT_FOUND' so GetUnenrichedAlbums ignores it next time.
					_ = e.repo.UpdateMBIDs(ctx, a.AlbumID, "NOT_FOUND", a.ArtistID, "")
					log.Printf("No MusicBrainz data found for album: %s", a.AlbumTitle)
				}
			}
		}
	}
}
