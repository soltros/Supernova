# Supernova Music

Supernova is a self-hosted, lightning-fast audiophile music server designed for massive local music libraries. It acts as an open-source, high-fidelity alternative to streaming platforms, focusing on raw performance, direct streaming, and uncompromising offline playback.

Built with a highly-concurrent Go backend and a Progressive Web App (PWA) React frontend.

## Core Features

- **Concurrent Library Scanning:** Leverages a tunable Go worker pool to extract ID3 metadata, album art, and audio parameters from tens of thousands of files (FLAC, ALAC, MP3, OPUS, M4A) in seconds.
- **Pure-Go Architecture:** Powered by `ncruces/go-sqlite3` (WASM-based SQLite) for zero CGO dependencies and true cross-platform compilation.
- **Audiophile Streaming:** Raw HTTP range-request streaming for lossless audio directly from your filesystem. 
- **Last.fm Enrichment:** A background daemon automatically fetches missing artist bios, high-resolution imagery, and global popularity rankings without blocking the user interface.
- **Scrub-Proof Scrobbling:** An internal playback engine calculates true listen thresholds, accurately logging your playback history independently of external services.
- **Hearts & Playlists System:** Full relational schema to favorite tracks, albums, and artists. Supports custom user playlists, ordering, and robust JSON export/import data portability.
- **Progressive Web App (PWA):** Installs directly to your Desktop, iOS, or Android homescreen as a standalone native-feeling application.
- **Extensible Plugin Architecture:** An `interface`-based registry system allowing modular features like Internet Radio, synchronized lyrics, and third-party scrobblers to be added and enabled seamlessly.

## Getting Started

### Prerequisites
- Go 1.22+
- Node.js 20+

### Backend Development
The backend is a monolithic Go binary holding the SQLite database and static file servers.
```bash
cd backend
go run cmd/server/main.go
```
*The backend binds to `http://localhost:8080` and provisions its SQLite database at `~/.supernova/supernova.db`.*

### Frontend Development
The frontend is a Vite-powered React Single Page Application (SPA).
```bash
cd frontend
npm install
npm run dev
```
*The frontend binds to `http://localhost:5173`.*

## Building Custom Clients

Supernova exposes a strictly typed RESTful JSON API. If you wish to build a native mobile app, a terminal UI, or an integration on top of Supernova, refer to the routing specifications below.

### Authentication
Supernova uses JWT (JSON Web Tokens) for authentication. Most routes (except library discovery) require an `Authorization` header.
```http
Authorization: Bearer <your_jwt_token>
```

### Core API Routes

#### Public Library (Read-Only)
- `GET /api/artists` - Returns a paginated list of all artists (`?limit=50&offset=0&letter=A`).
- `GET /api/artists/{id}` - Returns specific artist metadata, including Last.fm enriched bios and imagery.
- `GET /api/albums` - Returns a paginated list of albums.
- `GET /api/albums/{id}` - Returns specific album data.
- `GET /api/tracks` - Returns tracks, optionally filtered by `?album_id=` or `?artist_id=`. Note: When querying by `artist_id`, Supernova automatically sorts the tracks by global popularity.

#### Streaming & Media
- `GET /api/stream/{id}` - The core audio streaming endpoint. Supports HTTP Range Requests for seeking and buffering. Can be injected directly into `<audio src="...">` tags.
- `GET /api/art/album/{id}` - Serves extracted and highly-optimized embedded cover art.

#### Authentication
- `POST /api/auth/register` - Registers a new user. Accepts JSON `{ "username", "password" }`.
- `POST /api/auth/login` - Authenticates a user and returns the JWT token.

#### User Data (Requires Auth)
- `GET /api/dashboard` - Returns personalized layout data (recently added albums, recently played tracks, and favorite tracks).
- `GET /api/hearts` / `POST /api/hearts` / `DELETE /api/hearts` - Manage user favorites. Accepts `{ "entity_type", "entity_id" }`.
- `GET /api/hearts/details` - Returns hydrated track/album models for all of a user's hearted items.

#### Playlists (Requires Auth)
- `GET /api/playlists` - List user playlists.
- `POST /api/playlists` - Create a new playlist.
- `DELETE /api/playlists/{id}` - Delete a playlist.
- `GET /api/playlists/{id}/tracks` - Retrieve tracks for a specific playlist.
- `POST /api/playlists/{id}/tracks` - Add a track to a playlist.
- `DELETE /api/playlists/{id}/tracks/{trackId}` - Remove a track.
- `GET /api/playlists/export` / `POST /api/playlists/import` - JSON portability endpoints.

#### Internal Scrobbling (Requires Auth)
- `POST /api/scrobbles` - Log a completed listen. Accepts `{ "track_id" }`.
- `GET /api/scrobbles/recent` - Retrieve chronological listening history.

## Plugin Ecosystem
Supernova is built to be highly modular. Enabled plugins expose their own API endpoints mounted under `/api/plugins/`.

### 1. Subsonic Translation Layer (`/rest/*`)
Exposes a fully OpenSubsonic-compliant API. This allows you to connect any standard Subsonic app (like Symfonium, DSub, Play:Sub, or Ultrasonic) directly to Supernova.
- Automatically handles token, cleartext, and API Key authentication.
- Maps internal UUIDs and relationships to Subsonic's strict XML/JSON payloads.

### 2. Last.fm Scrobbler (`/api/plugins/lastfm/*`)
Allows users to link their Last.fm accounts via OAuth and dual-scrobble their plays to Last.fm in real-time.
- `POST /api/plugins/lastfm/session`
- `POST /api/plugins/lastfm/nowplaying`
- `POST /api/plugins/lastfm/scrobble`

### 3. LRCLib Synchronized Lyrics (`/api/plugins/lrclib/*`)
Fetches time-synced lyrics on-demand for the currently playing track by proxying queries through the backend to avoid CORS restrictions and leverage caching.
- `GET /api/plugins/lrclib/lyrics`

### 4. Radio-Browser (`/api/plugins/radiobrowser/*`)
Integrates tens of thousands of global internet radio stations directly into the Supernova player, supporting searching, tagging, and direct streaming without modifying the local library.
- `GET /api/plugins/radiobrowser/search`

## Writing Your Own Plugin
Supernova's plugin system is designed to be highly accessible for developers. To create your own plugin:

1. Create a new directory under `backend/internal/plugins/yourplugin`.
2. Implement the `plugins.Plugin` interface:
   ```go
   package yourplugin

   import (
       "net/http"
       "github.com/soltros/Supernova/internal/plugins"
   )

   type MyPlugin struct {}

   func init() {
       plugins.Register(&MyPlugin{})
   }

   func (p *MyPlugin) ID() string { return "myplugin" }
   func (p *MyPlugin) Name() string { return "My Custom Plugin" }
   func (p *MyPlugin) Description() string { return "Does something cool!" }
   func (p *MyPlugin) Init(config plugins.PluginConfig) error { return nil }
   func (p *MyPlugin) SetupRoutes(mux *http.ServeMux) {
       mux.HandleFunc("GET /api/plugins/myplugin/hello", func(w http.ResponseWriter, r *http.Request) {
           w.Write([]byte("Hello from my plugin!"))
       })
   }
   ```
3. Import your package anonymously in `backend/cmd/server/main.go`:
   ```go
   import _ "github.com/soltros/Supernova/internal/plugins/yourplugin"
   ```
4. Enable it by setting the environment variable `SUPERNOVA_PLUGIN_MYPLUGIN=true`.
