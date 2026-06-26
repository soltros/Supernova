# Supernova Music

![Supernova](https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=1200&h=400&fit=crop)

Supernova is a lightning-fast, highly-concurrent audiophile music server. It acts as a self-hosted alternative to Spotify or Apple Music, specifically designed for massive, high-fidelity local music libraries (FLAC, ALAC, MP3). 

Built with a pure-Go backend and a React (Vite) Progressive Web App frontend.

## Key Features

- **Blazing Fast Scanning:** Utilizes a highly-concurrent 10-worker pool to scan and extract ID3 metadata from 10,000+ tracks in seconds.
- **Pure-Go Architecture:** Powered by `ncruces/go-sqlite3` (WASM-based SQLite) for zero CGO dependencies and true cross-platform compilation.
- **Audiophile Streaming:** Raw HTTP range-request streaming for lossless audio, with dynamic real-time `ffmpeg` transcoding (MP3/Opus) for low-bandwidth mobile streaming.
- **Progressive Web App (PWA):** Installs directly to your Desktop, iOS, or Android homescreen. Includes a Service Worker for instant offline UI booting.
- **Internal Scrobbling:** An intelligent, scrub-proof playback engine that accurately logs your listen history.
- **Hearts System:** Secure, cascading SQLite relational architecture to favorite tracks and albums, with robust export/import capabilities.

## Architecture

- **Backend:** Go 1.22+, `net/http` standard library (no bloatware frameworks), SQLite WAL mode for extreme concurrency.
- **Frontend:** React 18, Vite, TypeScript, standard CSS (Glassmorphism design).
- **Metadata:** `dhowden/tag` for parsing audio tags, `golang.org/x/image/draw` for highly optimized embedded cover art extraction and resizing.

## Getting Started (Local Development)

### Prerequisites
- Go 1.22+
- Node.js 20+
- FFmpeg (must be installed on your system path for transcoding)

### Backend
```bash
cd backend
go run cmd/server/main.go
```
*The backend runs on `http://localhost:8080` and stores your library in `~/.supernova`.*

### Frontend
```bash
cd frontend
npm install
npm run dev
```
*The frontend runs on `http://localhost:5173`.*

## Docker Deployment

Supernova is completely containerized.

```bash
docker-compose up -d
```
