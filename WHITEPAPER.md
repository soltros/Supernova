# Supernova: Project Whitepaper

## 1. Vision & Mission
**Supernova** is a modern, self-hosted music streaming platform designed to replace legacy systems like Navidrome. Our mission is to provide users with direct, uncompromising control over their personal music libraries without sacrificing the premium, fluid experience expected from commercial streaming services (like Spotify or Apple Music).

Supernova achieves this by wrapping a high-performance Go-based backend and raw FFmpeg processing power in an elegant, glassmorphic Progressive Web App (PWA) with native system integrations (lockscreen, MPRIS).

## 2. Core Features
- **Direct Library Connection:** Connects directly to local folders via Docker volume mounts. No complex ingest pipelines.
- **Smart Library Management:**
  - **Metadata Enrichment:** Fetch rich artist bios, high-res artist pictures, and missing album art using user-provided Last.fm API keys.
  - **Tag Correction:** Automatic scanning and fixing of audio tags using MusicBrainz integration.
  - **Smart De-duplication:** Automatically hide or merge duplicate tracks within albums to maintain a clean UI.
- **Modern PWA Client:** An installable web app that feels native on desktop and mobile.
- **Native Playback & Integrations:** 
  - Media Session API support for lockscreen controls and Bluetooth metadata.
  - Linux Desktop MPRIS support for native media key handling.
- **Advanced Media Processing:** 
  - On-the-fly transcoding via FFmpeg to save bandwidth (e.g., FLAC to Opus).
  - Waveform extraction for SoundCloud-style precise seek bars.
- **Premium Aesthetics:** Dynamic color palettes extracted from album art, glassmorphism, fluid micro-animations, and responsive layout.

## 3. Architecture & Tech Stack
- **Backend (The Core Engine):** Go (Golang)
  - Blazing fast concurrency for directory walking and metadata scanning.
  - Lightweight and static binary, perfect for tiny Docker images.
- **Database:** SQLite
  - Embedded directly into the Go binary. Zero-configuration, incredibly fast for read-heavy operations, and stored locally in the container volume.
- **Media Processing:** FFmpeg
  - Invoked via Go `os/exec` for transcoding, metadata extraction, and waveform generation.
- **Frontend (The Client):** React + Vite (TypeScript)
  - Fast, component-driven UI. Vanilla CSS (CSS Modules) for maximum design control without relying on bloated UI libraries.

## 4. Phased Implementation Plan

### Phase 1: Foundation (Completed)
- [x] Establish tech stack and architecture.
- [x] Initialize Go backend, Vite React frontend, and Docker multi-stage configuration.
- [x] Draft Project Whitepaper.

### Phase 2: Core Backend & Data Ingestion (Goal: Serve Metadata)
- [ ] **DB Schema:** Design SQLite schema for Artists, Albums, Tracks, and Playlists (including fields for rich metadata).
- [ ] **Scanner Service:** Build the Go routine to recursively walk the `/music` volume.
- [ ] **Metadata Extraction:** Use Go libraries or FFprobe to parse ID3/FLAC tags and save them to SQLite.
- [ ] **Smart Tagging & Deduplication:** Use MusicBrainz API to validate/fix tags and implement logic to hide album duplicates during the scan.
- [ ] **Rich Metadata Enrichment:** Implement Last.fm API integration to pull down artist imagery and bios using user-provided API keys.
- [ ] **REST API:** Create endpoints for serving the library (`/api/artists`, `/api/albums`, `/api/tracks`).

### Phase 3: Core Frontend (Goal: UI Skeleton & Data Fetching)
- [ ] **Design System:** Implement base CSS (CSS variables for colors, typography, glassmorphism utilities).
- [ ] **Layout:** Build the responsive shell (Sidebar, Main Content Area, Persistent Bottom Player).
- [ ] **Data Hookup:** Fetch and display Artists, Albums, and Tracks from the Go API.

### Phase 4: Audio Playback & Transcoding (Goal: Hear the Music)
- [ ] **Stream Endpoint:** Build a Go endpoint (`/api/stream/{id}`) that pipes the raw audio file to the HTTP response.
- [ ] **Frontend Player:** Implement the HTML5 Audio API to play streams, control volume, and track progress.
- [ ] **On-the-fly Transcoding:** Enhance the Go stream endpoint to pipe data through FFmpeg (e.g., converting to 128kbps Opus) based on a query parameter.

### Phase 5: PWA & Native Integrations (Goal: Make it Native)
- [ ] **PWA Manifest & Service Worker:** Make the frontend installable and capable of caching UI assets.
- [ ] **Media Session API:** Hook up the frontend player to the browser's Media Session API so the OS lockscreen can control playback.
- [ ] **Waveform Generation:** Use FFmpeg on the backend to generate JSON waveform data; build a custom React canvas component to render the seek bar.

### Phase 6: Polish & Advanced Features
- [ ] **Dynamic Theming:** Extract dominant colors from Album Art to dynamically theme the background and player.
- [ ] **Playlists & Favorites:** Implement user-specific playlists and liking/favoriting functionality.
- [ ] **Performance Tuning:** Add pagination/virtualization to the frontend for massive libraries.

## 5. UI/UX Design Philosophy
The interface must WOW the user. We will strictly avoid generic bootstrapper looks.
- **Typography:** Inter or Outfit (Google Fonts).
- **Backgrounds:** Deep, dark modes (e.g., `hsl(220, 20%, 10%)`) heavily augmented with soft, glowing blurred circles based on album art colors.
- **Interactions:** Subtle hover lifts, smooth play/pause transitions, and responsive feedback on every click.

## 6. Beyond Navidrome (Solving Subsonic Limitations)
Supernova is built to explicitly solve the legacy constraints of the Subsonic API and pain points in Navidrome:
- **Proper Multi-Artist / Multi-Genre Support:** A truly relational database schema that handles complex tracks (e.g., "Artist A feat. Artist B") so tracks appear under all relevant artists, rather than creating a bloated "Artist A & Artist B" entry.
- **Real-Time Library Syncing:** Using `fsnotify`, the Go backend will instantly detect file changes (drops, deletes, renames) and update the library in real-time, removing the need for scheduled cron scans.
- **Folder View Browsing:** While tags are the primary organization method, Supernova will include a first-class "Folder View" for users who meticulously organize their directories and prefer to browse by folder tree.
- **Modern REST/GraphQL API:** No legacy XML bloat. The API will be lean, allowing for lightning-fast loads even on constrained devices (like smartwatches), and will support instantaneous single-file ingest endpoints.
- **Smart Compilations:** Intelligent grouping of soundtracks and compilations so they do not pollute the primary artist roster.
- **Flawless Transcoding & Seeking:** Robust handling of modern codecs (Opus, AAC) with the ability to accurately seek within active transcoded streams.
- **Discovery Features:** Modern UI features like "Appears On" sections, "Similar Artists" (via Last.fm), and intelligent mix builders.
