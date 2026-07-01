package scanner

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/metadata"
	"github.com/soltros/Supernova/internal/models"
)

// Scanner handles both the initial bulk scan and real-time directory watching.
type Scanner struct {
	mediaPath    string
	watcher      *fsnotify.Watcher
	repo         *database.Repository
	enricher     *Enricher
	ctx          context.Context
	realtimeJobs chan string

	stateMu      sync.RWMutex
	status       string // "idle", "scanning"
	filesScanned int
}

// New creates a new Scanner instance and initializes the file watcher.
func New(ctx context.Context, mediaPath string, repo *database.Repository, enricher *Enricher) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	scanner := &Scanner{
		mediaPath:    mediaPath,
		watcher:      watcher,
		repo:         repo,
		enricher:     enricher,
		ctx:          ctx,
		realtimeJobs: make(chan string, 1000), // Buffer to absorb rapid drops
		status:       "idle",
		filesScanned: 0,
	}

	// Start a background worker for real-time events:
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case path := <-scanner.realtimeJobs:
				meta := scanner.extractMetadata(path)
				if meta != nil {
					if err := scanner.repo.UpsertTrack(scanner.ctx, meta); err != nil {
						log.Printf("DB Insert Failed (%s): %v", filepath.Base(path), err)
					}
				}
				if scanner.enricher != nil {
					scanner.enricher.Trigger()
				}
			}
		}
	}()

	return scanner, nil
}

// FullScan recursively walks the media directory using a high-performance Worker Pool.
func (s *Scanner) FullScan() error {
	s.stateMu.Lock()
	if s.status == "scanning" {
		s.stateMu.Unlock()
		return nil // Already scanning
	}
	s.status = "scanning"
	s.filesScanned = 0
	s.stateMu.Unlock()

	defer func() {
		s.stateMu.Lock()
		s.status = "idle"
		s.stateMu.Unlock()
	}()

	log.Printf("Starting highly concurrent library scan at: %s", s.mediaPath)
	startTime := time.Now()

	// 1. Create buffered job channels and WaitGroups
	jobs := make(chan string, 5000)
	dbJobs := make(chan *models.TrackMetadata, 5000)
	var wg sync.WaitGroup
	var dbWg sync.WaitGroup

	// Dedicated database writer goroutine to batch inserts and avoid SQLite WAL lock contention
	dbWg.Add(1)
	go func() {
		defer dbWg.Done()
		for meta := range dbJobs {
			if err := s.repo.UpsertTrack(s.ctx, meta); err != nil {
				log.Printf("DB Insert Failed (%s): %v", filepath.Base(meta.FilePath), err)
			} else {
				s.stateMu.Lock()
				s.filesScanned++
				s.stateMu.Unlock()
			}
		}
	}()

	// 2. Spawn 10 concurrent worker goroutines purely for CPU-bound ID3 extraction
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if meta := s.extractMetadata(path); meta != nil {
					dbJobs <- meta
				}
			}
		}()
	}

	// 3. Walk the directory rapidly and push files into the queue
	err := filepath.WalkDir(s.mediaPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.Printf("Scanner permission error skipping path %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			s.watcher.Add(path) // Watch for real-time changes
			return nil
		}
		if isAudioFile(path) {
			jobs <- path // Instantly dispatch to a worker
		}
		return nil
	})

	// 4. Close the channel to signal workers no more jobs are coming
	close(jobs)

	// 5. Block until all workers have finished
	wg.Wait()
	
	// Close DB jobs channel and wait for DB writer
	close(dbJobs)
	dbWg.Wait()

	log.Printf("Full scan completed in %v.", time.Since(startTime))

	// 6. Wake up the background enricher to slowly fetch MusicBrainz data
	if s.enricher != nil {
		s.enricher.Trigger()
	}

	return err
}

// GetStatus returns the current scanning state
func (s *Scanner) GetStatus() (string, int) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.status, s.filesScanned
}

// Watch starts listening for real-time file system events (adds, deletes, renames)
func (s *Scanner) Watch() {
	log.Println("Starting real-time file watcher...")
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case event, ok := <-s.watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						s.watcher.Add(event.Name)
						// Recursively scan asynchronously in case the user dragged a folder
						filepath.WalkDir(event.Name, func(p string, d os.DirEntry, err error) error {
							if err == nil && !d.IsDir() && isAudioFile(p) {
								s.realtimeJobs <- p // Non-blocking dispatch
							} else if err == nil && d.IsDir() {
								s.watcher.Add(p)
							}
							return nil
						})
					} else if isAudioFile(event.Name) {
						s.realtimeJobs <- event.Name // Non-blocking dispatch
					}
				}

			case <-s.watcher.Errors:
				// ignore
			}
		}
	}()
}

// Close gracefully shuts down the file watcher
func (s *Scanner) Close() error {
	return s.watcher.Close()
}

// extractMetadata extracts metadata without hitting the database
func (s *Scanner) extractMetadata(path string) *models.TrackMetadata {
	// Fast, purely native Go metadata extraction
	meta, err := metadata.Extract(path)
	if err != nil {
		return nil
	}
	meta.FilePath = path
	return meta
}

// isAudioFile checks if the given file has a supported audio extension
func isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3", ".flac", ".ogg", ".m4a", ".aac", ".opus", ".wav", ".alac", ".wma", ".aiff", ".m4b":
		return true
	}
	return false
}
