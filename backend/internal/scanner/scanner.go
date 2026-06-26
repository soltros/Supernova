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
	repo      *database.Repository
	enricher  *Enricher
}

// New creates a new Scanner instance and initializes the file watcher.
func New(mediaPath string, repo *database.Repository, enricher *Enricher) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Scanner{
		mediaPath: mediaPath,
		watcher:   watcher,
		repo:      repo,
		enricher:  enricher,
	}, nil
}

// FullScan recursively walks the media directory using a high-performance Worker Pool.
func (s *Scanner) FullScan() error {
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
			}
		}()
	}

	// 3. Walk the directory rapidly and push files into the queue
	err := filepath.WalkDir(s.mediaPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Insert into SQLite instantly. We'll enhance this track in the background later.
	if err := s.repo.UpsertTrack(ctx, meta); err != nil {
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
