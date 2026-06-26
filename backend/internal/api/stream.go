package api

import (
	"log"
	"net/http"
	"os/exec"
	"strconv"
)

// handleStreamTrack handles GET /api/stream/{id}
func (s *Server) handleStreamTrack() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trackID := r.PathValue("id")
		if trackID == "" {
			http.Error(w, "track ID required", http.StatusBadRequest)
			return
		}

		track, err := s.repo.GetTrackByID(r.Context(), trackID)
		if err != nil {
			http.Error(w, "track not found", http.StatusNotFound)
			return
		}

		format := r.URL.Query().Get("format")

		// If no transcode requested, serve the raw file directly.
		// http.ServeFile automatically handles HTTP Range requests for seeking.
		if format == "" {
			w.Header().Del("Content-Type") // Fixes Flaw #1: Remove the global JSON header
			http.ServeFile(w, r, track.FilePath)
			return
		}

		// --- 1. VALIDATE INPUT PARAMETERS ---
		
		// Strict Whitelist for formats
		switch format {
		case "mp3", "aac", "ogg", "opus":
			// valid
		default:
			http.Error(w, "unsupported transcode format", http.StatusBadRequest)
			return
		}

		// Parse and clamp bitrate (64k to 320k)
		bitrateStr := r.URL.Query().Get("bitrate")
		bitrate := 128 // Default
		if bitrateStr != "" {
			if b, err := strconv.Atoi(bitrateStr); err == nil {
				if b < 64 {
					bitrate = 64
				} else if b > 320 {
					bitrate = 320
				} else {
					bitrate = b
				}
			}
		}

		// --- 2. SUPPORT SEEKING (TIME OFFSET) ---
		
		// Parse optional time offset (in seconds) for fast-forwarding
		timeStr := r.URL.Query().Get("time")
		seekTime := 0
		if timeStr != "" {
			if t, err := strconv.Atoi(timeStr); err == nil && t > 0 {
				seekTime = t
			}
		}

		// --- 3. SET HTTP STREAM HEADERS ---
		
		if format == "mp3" {
			w.Header().Set("Content-Type", "audio/mpeg")
		} else if format == "aac" {
			w.Header().Set("Content-Type", "audio/aac")
		} else if format == "ogg" {
			w.Header().Set("Content-Type", "audio/ogg")
		} else if format == "opus" {
			w.Header().Set("Content-Type", "audio/ogg; codecs=opus")
		}

		// Force the browser to treat this as a live, chunked stream without caching
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Accept-Ranges", "none") // Crucial: Prevent the browser from breaking the pipe with Range requests

		// --- BUILD FFMPEG COMMAND ---
		
		args := []string{}
		
		// If seeking is requested, pass -ss BEFORE the input file for extremely fast seeking
		if seekTime > 0 {
			args = append(args, "-ss", strconv.Itoa(seekTime))
		}
		
		args = append(args,
			"-i", track.FilePath,
			"-map", "0:a:0", // Strip massive embedded cover art to save bandwidth
			"-f", format,
			"-ab", strconv.Itoa(bitrate)+"k",
			"-loglevel", "error",
			"pipe:1",
		)

		cmd := exec.CommandContext(r.Context(), "ffmpeg", args...)

		// Pipe the stdout of ffmpeg directly to the HTTP response writer
		cmd.Stdout = w
		
		// Flush headers so the browser begins playback immediately without buffering
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		if err := cmd.Run(); err != nil {
			// Expected behavior: if the user skips a track, the HTTP context is cancelled
			// which automatically sends SIGKILL to ffmpeg.
			log.Printf("Transcode stream ended for track %s: %v", trackID, err)
		}
	}
}
