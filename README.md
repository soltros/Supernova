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

## Docker Deployment

The recommended way to run Supernova in production is with Docker Compose.

### 1. Configure your environment

Copy the example file and fill in your values:
```bash
cp .env.example .env
```

Then edit `.env`. At minimum you **must** set `JWT_SECRET`:

```bash
# Generate a cryptographically secure secret (run this in your terminal):
openssl rand -hex 32

# Paste the output as the value for JWT_SECRET in your .env file:
JWT_SECRET=paste_the_64_character_hex_output_here
```

> [!IMPORTANT]
> The server will **refuse to start** if `JWT_SECRET` is missing or shorter than 32 characters. This is intentional — a weak or missing secret allows anyone to forge login tokens for any account.

### 2. Set your music library path

In `.env`, uncomment and set `MEDIA_PATH` to the absolute path of your music folder on the host:
```ini
MEDIA_PATH=/home/youruser/Music
```

### 3. Start the stack
```bash
docker compose up -d
```

The web UI will be available at **http://your-server:5174**.

> [!NOTE]
> The frontend container will not start until the backend passes its health check (`/api/health`). This prevents the nginx DNS crash that occurs when the backend hasn't launched yet.

---

## Plugin Ecosystem
Supernova is built to be highly modular. Enabled plugins expose their own API endpoints mounted under `/api/plugins/`.

### Enabling Plugins
By default, all official plugins are bundled with the backend but must be explicitly enabled via environment variables.

**With Docker Compose** — add to your `.env` file:
```ini
SUPERNOVA_PLUGIN_SUBSONIC=true
SUPERNOVA_PLUGIN_LASTFM=true
SUPERNOVA_PLUGIN_LRCLIB=true
SUPERNOVA_PLUGIN_RADIOBROWSER=true
SUPERNOVA_PLUGIN_AUTOTAGGER=true
SUPERNOVA_PLUGIN_ARTISTMERGER=true
SUPERNOVA_PLUGIN_DEDUPER=true
```

**Running directly** — export before starting the server:
```bash
# Required — generate with: openssl rand -hex 32
export JWT_SECRET=your_secret_here

# Optional — for Last.fm scrobbling
export LASTFM_API_KEY=your_api_key_here
export LASTFM_API_SECRET=your_api_secret_here

export SUPERNOVA_PLUGIN_LASTFM=true
# ... other plugins as needed

go run cmd/server/main.go
```

### 1. Subsonic Translation Layer (`/rest/*`)
The Subsonic Translation plugin bridges the gap between Supernova's modern architecture and the massive, established ecosystem of Subsonic clients. By translating API calls in real-time, it enables full compatibility with dozens of third-party apps without needing a dedicated Supernova mobile app.
**Featureset:**
- **Universal Compatibility:** Connect standard apps like Symfonium, DSub, Play:Sub, Ultrasonic, and AVSub directly to your Supernova server.
- **On-the-fly Translation:** Intercepts OpenSubsonic XML/JSON payloads, maps them to Supernova's UUID relational database, and returns perfectly formatted OpenSubsonic responses.
- **Complete Auth Support:** Automatically handles Token, Cleartext, and API Key authentication.
- **Deep Integration:** Supports library browsing, directory traversal, full-text search, and direct media streaming.

### 2. Last.fm Scrobbler (`/api/plugins/lastfm/*`)
For users deeply invested in tracking their listening habits, the Last.fm plugin provides seamless, background integration with the Last.fm ecosystem.
**Featureset:**
- **OAuth Integration:** Securely link your Last.fm account directly through the Supernova settings.
- **Dual-Scrobbling:** Works in tandem with Supernova's internal Scrub-Proof Scrobbling engine to log plays both locally and to Last.fm simultaneously.
- **"Now Playing" Support:** Instantly updates your Last.fm status to show the track you are currently listening to.
- **Real-time API Sync:** Strictly adheres to Last.fm's Scrobbling 2.0 API guidelines for zero dropped scrobbles.

### 3. LRCLib Synchronized Lyrics (`/api/plugins/lrclib/*`)
Enhance your listening experience with real-time, karaoke-style synchronized lyrics powered by the open-source LRCLib database.
**Featureset:**
- **Time-Synced Lyrics:** Automatically fetches LRC formatted lyrics that sync line-by-line with audio playback.
- **CORS Bypass Proxy:** Proxies queries through the Go backend to bypass strict browser CORS restrictions, ensuring lyrics load flawlessly in the PWA.
- **Smart Fallbacks:** Falls back to plain-text lyrics if time-synced versions are unavailable for a specific track.
- **Performance Caching:** Highly optimized to avoid redundant external network requests.

### 4. Radio-Browser (`/api/plugins/radiobrowser/*`)
Transform Supernova into an internet radio powerhouse. This plugin integrates directly with the community-driven Radio-Browser database.
**Featureset:**
- **Massive Directory:** Search and browse tens of thousands of global internet radio stations by genre, language, or country.
- **High-Availability DNS:** Leverages Radio-Browser's dynamic round-robin DNS to ensure the API never goes down.
- **Direct Integration:** Streams remote radio stations directly through the Supernova audio engine without cluttering your pristine local library.

### 5. Auto-Tagger (`/api/plugins/autotagger/*`)
A fully safe, non-destructive metadata enricher that fixes your library without modifying a single byte of your actual `.mp3` or `.flac` files on disk.
**Featureset:**
- **Smart Path Inference:** Automatically parses your folder structures (e.g., `/music/Artist Name/Album Name/01 - Track.mp3`) to infer missing metadata.
- **Background Processing:** Runs as an asynchronous background job, gracefully patching your Supernova database to fix "Unknown Artist" or generic "Track 1" entries.
- **Database-Only Execution:** Ensures your pristine local file tags are never overwritten or corrupted.

### 6. Artist Merger (`/api/plugins/artistmerger/*`)
A powerful library cleaner that groups similar artists to eliminate frustrating duplicates from bad metadata.
**Featureset:**
- **String Normalization:** Intelligently strips out punctuation, spaces, and prefixes (like "The " or "A ") to find matches.
- **Canonical Merging:** Automatically detects pairs like "Beatles" and "The Beatles" or "AC DC" and "AC/DC", picking the best formatted name as the canonical artist.
- **Relational Re-routing:** Safely migrates all albums, tracks, and favorites pointing to the duplicates over to the canonical artist before deleting the orphaned records.

### 7. Deduper ("Hide Duplicates") (`/api/plugins/deduper/*`)
An automatic library cleaner that identifies duplicate tracks (same title, same album) and gracefully deletes the lower quality (lower bitrate) version from your database, keeping your library pristine.

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
