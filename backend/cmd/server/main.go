package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/soltros/Supernova/internal/api"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/external"
	"github.com/soltros/Supernova/internal/scanner"
)

func main() {
	// Attempt to load .env file if it exists (useful for local development)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables.")
	}

	log.Println("Booting Supernova Media Server...")

	// 1. Read Environment Variables (these are provided by our docker-compose.yml)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/supernova.db"
	}
	mediaPath := os.Getenv("MEDIA_PATH")
	if mediaPath == "" {
		mediaPath = "./music"
	}

	// 2. Initialize the Database
	log.Printf("Connecting to pure-Go SQLite at: %s", dbPath)
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// db embeds *sql.DB, so we can defer its Close method safely
	defer db.Close()

	// 3. Initialize the Repository
	repo := database.NewRepository(db)

	// 4. Initialize External APIs
	// (Last.fm keys will be pulled from ENV if the user wants rich bios/scrobbling)
	lastfmKey := os.Getenv("LASTFM_API_KEY")
	lastfmSecret := os.Getenv("LASTFM_API_SECRET")
	lastfmClient := external.NewLastFmClient(lastfmKey, lastfmSecret)

	mbClient := external.NewMusicBrainzClient("SupernovaMediaServer", "1.0.0", "admin@supernova.local")

	// 5. Initialize the Background Enricher
	enricher := scanner.NewEnricher(repo, mbClient, lastfmClient)
	ctx, cancelEnricher := context.WithCancel(context.Background())
	defer cancelEnricher()
	enricher.Start(ctx)

	// 6. Initialize the File Scanner
	log.Printf("Initializing media scanner for path: %s", mediaPath)
	mediaScanner, err := scanner.New(mediaPath, repo, enricher)
	if err != nil {
		log.Fatalf("Failed to initialize media scanner: %v", err)
	}
	defer mediaScanner.Close()

	// 7. Start the real-time fsnotify file watcher
	mediaScanner.Watch()

	// 8. Kick off an asynchronous full bulk scan
	// This runs in the background and populates the DB while the web server comes online
	go func() {
		if err := mediaScanner.FullScan(); err != nil {
			log.Printf("Error during full library scan: %v", err)
		}
	}()

	// 8. Initialize the HTTP API Server
	apiServer := api.NewServer(repo, lastfmClient, enricher, mediaScanner)

	// Configure the HTTP Server with sensible production timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      apiServer,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 9. Start the HTTP server in a background goroutine
	go func() {
		log.Printf("Supernova API successfully listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 10. Graceful Shutdown listener
	// This keeps the main thread alive until Docker or the user sends a termination signal (CTRL+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Supernova server gracefully...")
	
	// Create a deadline to wait for currently active requests
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
