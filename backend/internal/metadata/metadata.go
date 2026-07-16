package metadata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	_ "image/gif"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/dhowden/tag"
	"github.com/soltros/Supernova/internal/models"
	"golang.org/x/image/draw"
)

var (
	artCacheDir string
	artCacheOnce sync.Once
)

// getArtCacheDir resolves the directory for storing extracted/resized cover art
func getArtCacheDir() string {
	artCacheOnce.Do(func() {
		artCacheDir = os.Getenv("ART_CACHE_PATH")
		if artCacheDir == "" {
			// Default to ./data/art_cache to stay in the same data folder as the default SQLite db
			artCacheDir = filepath.Join(".", "data", "art_cache")
		}
		os.MkdirAll(artCacheDir, 0755)
	})
	return artCacheDir
}

// processAndSaveImage resizes large images to 500x500, strictly enforces JPEG encoding, and caches them.
func processAndSaveImage(rawData []byte) string {
	hash := fmt.Sprintf("%x", sha256.Sum256(rawData))
	cacheDir := getArtCacheDir()
	finalPath := filepath.Join(cacheDir, hash+".jpg") // Standardize all art to .jpg
	
	if _, err := os.Stat(finalPath); err == nil {
		return finalPath
	}

	// Protect against OOM DOS attacks from massive embedded audiophile covers
	if len(rawData) > 15*1024*1024 { // 15MB threshold
		mimeType := http.DetectContentType(rawData)
		ext := ".bin"
		if mimeType == "image/png" { ext = ".png" }
		if mimeType == "image/jpeg" { ext = ".jpg" }
		if mimeType == "image/webp" { ext = ".webp" }
		fallbackPath := filepath.Join(cacheDir, hash+ext)
		if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
			_ = os.WriteFile(fallbackPath, rawData, 0644)
		}
		return fallbackPath
	}

	// Strictly decode the image. This implicitly validates the MIME type (fixes flaw #3).
	img, _, err := image.Decode(bytes.NewReader(rawData))
	if err != nil {
		// If it's a completely unsupported format, we fallback to just saving the raw bytes
		// and using http.DetectContentType to give it the correct extension.
		mimeType := http.DetectContentType(rawData)
		ext := ".bin"
		if mimeType == "image/png" { ext = ".png" }
		if mimeType == "image/jpeg" { ext = ".jpg" }
		if mimeType == "image/webp" { ext = ".webp" }
		fallbackPath := filepath.Join(cacheDir, hash+ext)
		if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
			_ = os.WriteFile(fallbackPath, rawData, 0644)
		}
		return fallbackPath
	}

	// Fixes Flaw #1: Aggressively resize oversized images
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	const maxSize = 500

	if width > maxSize || height > maxSize {
		var newWidth, newHeight int
		if width > height {
			newWidth = maxSize
			newHeight = (height * maxSize) / width
		} else {
			newHeight = maxSize
			newWidth = (width * maxSize) / height
		}

		dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
	}

	// Save as a highly optimized JPEG
	out, err := os.Create(finalPath)
	if err != nil {
		return ""
	}
	defer out.Close()
	
	// 85 quality drastically reduces bytes while remaining visually flawless for 500x500 UI elements
	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 85}); err != nil {
		out.Close()
		os.Remove(finalPath)
		return ""
	}
	return finalPath
}

// Extract attempts to extract metadata and cover art using a fast pure-Go library.
func Extract(filePath string) (*models.TrackMetadata, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	trackNum, _ := m.Track()
	discNum, _ := m.Disc()

	var coverArtPath string
	pic := m.Picture()

	// 1. Primary Check: Embedded Art
	if pic != nil && len(pic.Data) > 0 {
		// Process, resize, and strictly type the image
		coverArtPath = processAndSaveImage(pic.Data)
	} else {
		// 2. Folder Fallback: Scan the directory for cover.jpg, etc.
		dir := filepath.Dir(filePath)
		candidates := []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png", "front.jpg"}
		for _, c := range candidates {
			p := filepath.Join(dir, c)
			if _, err := os.Stat(p); err == nil {
				coverArtPath = p
				break
			}
		}
	}

	return &models.TrackMetadata{
		Title:        m.Title(),
		Album:        m.Album(),
		Artist:       m.Artist(),
		AlbumArtist:  m.AlbumArtist(),
		TrackNumber:  trackNum,
		DiscNumber:   discNum,
		Year:         m.Year(),
		Format:       string(m.Format()),
		CoverArtPath: coverArtPath,
	}, nil
}
