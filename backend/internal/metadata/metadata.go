package metadata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
	"github.com/soltros/Supernova/internal/models"
	"golang.org/x/image/draw"
)

var (
	artCacheDir  string
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

	// Check decoded dimensions before allocating pixels; compressed size alone is insufficient.
	if len(rawData) > 15*1024*1024 {
		return ""
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(rawData))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 25_000_000 {
		return ""
	}
	img, _, err := image.Decode(bytes.NewReader(rawData))
	if err != nil {
		return ""
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

		newWidth = max(1, newWidth)
		newHeight = max(1, newHeight)
		dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
	}

	// Save as a highly optimized JPEG
	out, err := os.CreateTemp(cacheDir, ".cover-*.jpg")
	if err != nil {
		return ""
	}
	defer os.Remove(out.Name())
	defer out.Close()

	// 85 quality drastically reduces bytes while remaining visually flawless for 500x500 UI elements
	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 85}); err != nil {
		out.Close()
		os.Remove(out.Name())
		return ""
	}
	if err := out.Close(); err != nil {
		return ""
	}
	if err := os.Rename(out.Name(), finalPath); err != nil {
		return ""
	}
	return finalPath
}

// Extract reads tags and probes the audio stream, including untagged files.
func Extract(filePath string) (*models.TrackMetadata, error) {
	return ExtractContext(context.Background(), filePath)
}

func ExtractContext(ctx context.Context, filePath string) (*models.TrackMetadata, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular audio file")
	}
	duration, bitrate, err := probeAudio(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("probe audio: %w", err)
	}
	meta := &models.TrackMetadata{
		Title:      strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		DurationMs: duration, Bitrate: bitrate,
		Format: strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), "."),
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Missing ID3/Vorbis tags must not exclude otherwise playable audio.
	if m, err := tag.ReadFrom(f); err == nil {
		if m.Title() != "" {
			meta.Title = m.Title()
		}
		meta.Album, meta.Artist, meta.AlbumArtist = m.Album(), m.Artist(), m.AlbumArtist()
		meta.TrackNumber, _ = m.Track()
		meta.DiscNumber, _ = m.Disc()
		meta.Year = m.Year()
		if pic := m.Picture(); pic != nil {
			meta.CoverArtPath = processAndSaveImage(pic.Data)
		}
	}
	if meta.CoverArtPath == "" {
		for _, name := range []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png", "front.jpg"} {
			path := filepath.Join(filepath.Dir(filePath), name)
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() <= 15*1024*1024 {
				if data, err := os.ReadFile(path); err == nil {
					meta.CoverArtPath = processAndSaveImage(data)
				}
				if meta.CoverArtPath != "" {
					break
				}
			}
		}
	}
	return meta, nil
}

// ffprobe is shipped with ffmpeg in both Docker images. It reads container/frame
// timing rather than guessing from file size or using unrelated online releases.
func probeAudio(ctx context.Context, filePath string) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	absolute, err := filepath.Abs(filePath)
	if err != nil {
		return 0, 0, err
	}
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-protocol_whitelist", "file,pipe", "-select_streams", "a:0", "-show_entries", "format=duration,bit_rate:stream=duration,bit_rate", "-of", "json", absolute)
	data, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	type timing struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	}
	var result struct {
		Format  timing   `json:"format"`
		Streams []timing `json:"streams"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, 0, err
	}
	if len(result.Streams) == 0 {
		return 0, 0, fmt.Errorf("no audio stream")
	}
	seconds, _ := strconv.ParseFloat(result.Streams[0].Duration, 64)
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		seconds, _ = strconv.ParseFloat(result.Format.Duration, 64)
	}
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > float64(math.MaxInt)/1000 {
		return 0, 0, fmt.Errorf("invalid audio duration")
	}
	bits, _ := strconv.Atoi(result.Streams[0].BitRate)
	if bits <= 0 {
		bits, _ = strconv.Atoi(result.Format.BitRate)
	}
	return int(math.Round(seconds * 1000)), max(0, bits/1000), nil
}
