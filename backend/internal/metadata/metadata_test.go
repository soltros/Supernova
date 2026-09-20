package metadata

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDurationAndFormatFromRealAudio(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe required")
	}
	t.Setenv("SUPERNOVA_ENABLE_ESTIMATES", "false")
	for _, ext := range []string{"mp3", "flac", "m4a", "aac", "ogg", "opus", "wav", "aiff", "wma"} {
		t.Run(ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "untagged."+ext)
			cmd := exec.Command("ffmpeg", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-y", path)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generate %s: %v: %s", ext, err, out)
			}
			meta, err := Extract(path)
			if err != nil {
				t.Fatal(err)
			}
			if meta.DurationMs < 1900 || meta.DurationMs > 2200 {
				t.Fatalf("duration=%d, want approximately 2000 ms", meta.DurationMs)
			}
			if meta.Format != ext {
				t.Fatalf("format=%q, want %q", meta.Format, ext)
			}
			if meta.Title != "untagged" {
				t.Fatalf("filename fallback: %q", meta.Title)
			}
			if meta.Bitrate <= 0 {
				t.Fatalf("missing bitrate: %+v", meta)
			}
		})
	}
}

func TestInvalidAudioAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.mp3")
	if err := os.WriteFile(path, []byte("not music"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(path); err == nil {
		t.Fatal("invalid audio accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExtractContext(ctx, path); err == nil {
		t.Fatal("cancelled probe accepted")
	}
}

func TestCoverArtRejectsActiveContentAndHandlesNarrowImage(t *testing.T) {
	t.Setenv("ART_CACHE_PATH", t.TempDir())
	if got := processAndSaveImage([]byte("<html><script>alert(1)</script></html>")); got != "" {
		t.Fatalf("unsafe image saved: %s", got)
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1000))
	img.Set(0, 0, color.White)
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	path := processAndSaveImage(data.Bytes())
	if path == "" {
		t.Fatal("narrow image failed")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cfg, format, err := image.DecodeConfig(file)
	if err != nil || cfg.Width != 1 || cfg.Height != 500 || format != "jpeg" {
		t.Fatalf("invalid output: %+v %s %v", cfg, format, err)
	}
}
