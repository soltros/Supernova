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
)

// Scanner handles both the initial bulk scan and real-time directory watching.
type Scanner struct {
	mediaPath string
	watcher   *fsnotify.Watcher
	repo         *database.Repository
	enricher     *Enricher
	
	stateMu      sync.RWMutex
	status       string // "idle", "scanning"
	filesScanned int
}

// New creates a new Scanner instance and initializes the file watcher.
func New(mediaPath string, repo *database.Repository, enricher *Enricher) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Scanner{
		mediaPath:    mediaPath,
		watcher:      watcher,
		repo:         repo,
		enricher:     enricher,
		status:       "idle",
		filesScanned: 0,
	}, nil
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

	// 1. Create a buffered job channel and a WaitGroup
	jobs := make(chan string, 5000)
	var wg sync.WaitGroup

	// 2. Spawn 10 concurrent worker goroutines
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				s.processFile(path)
				s.stateMu.Lock()
				s.filesScanned++
				s.stateMu.Unlock()
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
			case event, ok := <-s.watcher.Events:
				if !ok { return }
				
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						s.watcher.Add(event.Name)
						// Recursively scan in case the user dragged a folder that already contains files
						filepath.WalkDir(event.Name, func(p string, d os.DirEntry, err error) error {
							if err == nil && d.IsDir() {
								s.watcher.Add(p)
							} else if err == nil && isAudioFile(p) {
								s.processFile(p)
							}
							return nil
						})
						if s.enricher != nil {
							s.enricher.Trigger()
						}
					} else if isAudioFile(event.Name) {
						s.processFile(event.Name)
						if s.enricher != nil {
							s.enricher.Trigger() // Wake up the enricher for the new file
						}
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

// processFile extracts metadata and saves it to the database instantly.
// WARNING: Do not call slow external APIs in this function.
func (s *Scanner) processFile(path string) {
	// Fast, purely native Go metadata extraction
	meta, err := metadata.Extract(path)
	if err != nil {
		return
	}
	meta.FilePath = path 

	// Insert into SQLite instantly. We'll enhance this track in the background later.
	// We use a background context because UpsertTrack acquires a global write mutex
	// and we don't want workers timing out while waiting in the lock queue.
	if err := s.repo.UpsertTrack(context.Background(), meta); err != nil {
		log.Printf("DB Insert Failed (%s): %v", filepath.Base(path), err)
	}
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
