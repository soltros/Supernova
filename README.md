<div align="center">
  <img src="https://raw.githubusercontent.com/soltros/Supernova/main/frontend/public/logo.svg" alt="Supernova Logo" width="180" />
</div>

# Supernova Music

Supernova is a self-hosted music server for large local libraries. It focuses on direct playback, responsive library browsing, user-owned data, and a web/PWA client. The PWA can cache the application shell for startup while offline; music, API data, and authenticated media are intentionally network-only.

Built with a highly-concurrent Go backend and a Progressive Web App (PWA) React frontend.

## Core Features

- **Concurrent Library Scanning:** Uses a tunable Go worker pool purely for CPU-bound ID3 metadata extraction from tens of thousands of files in seconds, batching results to a single dedicated database writer to completely eliminate SQLite write-lock contention.
- **Pure-Go Architecture:** Powered by `ncruces/go-sqlite3` (WASM-based SQLite) for zero CGO dependencies and true cross-platform compilation.
- **Audiophile Streaming:** Raw HTTP range-request streaming for lossless audio directly from your filesystem. 
- **Last.fm Enrichment:** A background daemon automatically fetches missing artist bios, high-resolution imagery, and global popularity rankings without blocking the user interface.
- **Scrub-Proof Scrobbling:** An internal playback engine calculates true listen thresholds, accurately logging your playback history independently of external services.
- **Hearts & Playlists System:** Full relational schema to favorite tracks, albums, and artists. Supports custom user playlists, ordering, and robust JSON export/import data portability.
- **Glassmorphism UI:** A stunning, highly dynamic React frontend built strictly around glassmorphism. It features blurred contextual backgrounds, smooth micro-animations, and viewport-aware right-click context menus rather than cheap native dialogs.
- **Progressive Web App (PWA):** Installs to supported desktop and mobile browsers. The service worker caches the UI shell/static assets only; it does not provide offline music downloads or cache authenticated media/API responses.
- **Extensible Plugin Architecture:** An `interface`-based compile-time Go registry for modular features such as Internet Radio, synchronized lyrics, podcasts, Last.fm, and Subsonic compatibility. Compiled plugins can be enabled or disabled with environment variables.
- **Comprehensive Wiki:** Full documentation covering the database architecture, design philosophy, API, and plugin internals is available on our [GitHub Wiki](https://github.com/soltros/Supernova/wiki).

## Getting Started

### Prerequisites
- Go 1.26.4+
- Node.js 24+
- FFmpeg (including `ffprobe`)

### Backend Development
The backend is a monolithic Go binary holding the SQLite database and static file servers.
```bash
cd backend
# Generate a fresh secret; configure MEDIA_PATH as needed.
export JWT_SECRET="$(openssl rand -hex 32)"
export CORS_ALLOWED_ORIGIN=http://localhost:5173
go run cmd/server/main.go
```
*The backend binds to `http://localhost:8080` and provisions its SQLite database at `./data/supernova.db` (relative to the backend working directory).*

### Frontend Development
The frontend is a Vite-powered React Single Page Application (SPA).
```bash
cd frontend
npm ci
npm run dev
```
*The frontend binds to `http://localhost:5173`.*

## Building Custom Clients

Supernova exposes a strictly typed RESTful JSON API. If you wish to build a native mobile app, a terminal UI, or an integration on top of Supernova, refer to the routing specifications below.

### Authentication
Supernova uses signed JWT session tokens backed by server-side session records. Most authenticated API routes use an `Authorization` header; logout revokes the current session and password changes revoke other sessions. Browser media playback/downloads use short-lived, resource-scoped media tickets instead of placing the full account bearer token in a URL.
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
- `GET /api/stream/{id}` - Raw streaming supports HTTP Range requests. Authenticated clients may use a Bearer header; browser media elements should first request a scoped `stream` ticket from `POST /api/media-ticket`.
- `GET /api/download/track/{id}` - Downloads the requested track after authenticated/rooted media validation.
- `GET /api/download/album/{id}` - Builds and validates the album ZIP in a temporary server-side file before committing the download response, preventing a late read error from masquerading as a successful partial archive.
- `POST /api/media-ticket` - Issues a short-lived ticket scoped to one stream, track download, or album download resource.
- `GET /api/art/album/{id}` - Serves extracted and highly-optimized embedded cover art.

#### Authentication
- `POST /api/auth/register` - Registers a new user. Accepts JSON `{ "username", "password", "invite_code" }`. The first account becomes administrator; later accounts require the owner-configured invite.
- `POST /api/auth/login` - Authenticates a user and creates a revocable session-backed JWT.
- `POST /api/auth/logout` - Revokes the current session.
- `POST /api/auth/change-password` - Changes the password after verifying the current password and revokes other sessions.

#### User Data (Requires Auth)
- `GET /api/dashboard` - Returns personalized layout data (recently added albums, recently played tracks, and favorite tracks).
- `GET /api/hearts` / `POST /api/hearts` / `DELETE /api/hearts` - Manage user favorites. Accepts `{ "entity_type", "entity_id" }`. Supported entities: `track`, `album`, `artist`, `playlist`, `radio`, `podcast`.
- `GET /api/hearts/details` - Returns hydrated tracks, albums, artists, playlists, radio stations, and podcasts for the authenticated user's favorites.
- `GET /api/hearts/export` / `POST /api/hearts/import` - Versioned JSON backup/restore. V2 prefers stable fingerprints/MusicBrainz identifiers and retains external favorite metadata; legacy array imports remain accepted.

#### Playlists (Requires Auth)
- `GET /api/playlists` - List user playlists.
- `POST /api/playlists` - Create a new playlist.
- `DELETE /api/playlists/{id}` - Delete a playlist.
- `GET /api/playlists/{id}/tracks` - Retrieve tracks for a specific playlist.
- `POST /api/playlists/{id}/tracks` - Add a track to a playlist.
- `DELETE /api/playlists/{id}/tracks/{trackId}` - Remove a track.
- `GET /api/playlists/export` / `POST /api/playlists/import` - Versioned JSON portability endpoints. V2 records content fingerprints plus legacy paths and restores an entire import set atomically; unresolved or ambiguous tracks fail explicitly rather than being silently skipped.

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

Then edit `.env`. At minimum you **must** set `JWT_SECRET`. Set `REGISTRATION_INVITE_CODE` to allow invited registrations after the first account; leaving it blank closes registration after initial setup. Rotate it by changing the value and recreating the backend container.

```bash
# Generate a cryptographically secure secret (run this in your terminal):
openssl rand -hex 32

# Paste the output as the value for JWT_SECRET in your .env file:
JWT_SECRET=paste_the_64_character_hex_output_here
```

> [!IMPORTANT]
> The server will **refuse to start** if `JWT_SECRET` is missing or shorter than 32 characters. This is intentional. A weak or missing secret allows anyone to forge login tokens for any account.

### 2. Set your music library and security configs

In `.env`, uncomment and set `MEDIA_PATH` to the absolute path of your music folder on the host:
```ini
MEDIA_PATH=/home/youruser/Music
```

For security, if you expose this server to the internet, it is strongly recommended to restrict API access by setting the CORS origin to match your frontend domain:
```ini
CORS_ALLOWED_ORIGIN=https://music.yourdomain.com
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
By default, all official plugins are bundled with the backend and are **enabled by default** (opt-out). You can explicitly disable them by setting their respective environment variables to `false`.

**With Docker Compose:** add to your `.env` file to disable specific plugins:
```ini
SUPERNOVA_PLUGIN_AUTOTAGGER=false
```

**Running directly** — export before starting the server:
```bash
# Required — generate with: openssl rand -hex 32
export JWT_SECRET=your_secret_here

# Optional — for Last.fm scrobbling
export LASTFM_API_KEY=your_api_key_here
export LASTFM_API_SECRET=your_api_secret_here

# Example of disabling a plugin
export SUPERNOVA_PLUGIN_LASTFM=false

go run cmd/server/main.go
```

### 1. Subsonic Translation Layer (`/rest/*`)
The Subsonic Translation plugin implements a compatibility subset of the Subsonic/OpenSubsonic REST API for third-party clients. Compatibility varies by client and endpoint, so it should not be treated as a claim of complete OpenSubsonic conformance.
**Implemented areas include:**
- Username/password authentication, including `enc:` hexadecimal passwords, and the standard token+salt flow (`t = md5(password + salt)`) after the user has logged in through Supernova once.
- XML and JSON responses for the implemented endpoints.
- Library browsing, directory traversal, paged search, album lists, playlists, favorites/starred data, scrobbling, raw streaming/downloads, cover art, and bounded on-the-fly transcoding for supported formats.
- Playlist creation/update operations preserve ordering and repeated songs and apply writes transactionally.

### 2. Last.fm Scrobbler (`/api/plugins/lastfm/*`)
For users deeply invested in tracking their listening habits, the Last.fm plugin provides seamless, background integration with the Last.fm ecosystem.
**Featureset:**
- **OAuth Integration:** Securely link your Last.fm account directly through the Supernova settings.
- **Dual-Scrobbling:** Works in tandem with Supernova's internal Scrub-Proof Scrobbling engine to log plays both locally and to Last.fm simultaneously.
- **"Now Playing" Support:** Instantly updates your Last.fm status to show the track you are currently listening to.
- **API Integration:** Sends now-playing and scrobble requests through the Last.fm integration. External-service/network failures can still occur and are surfaced or retried where the relevant workflow supports it.

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
- **High-Availability DNS:** Utilizes Radio-Browser's dynamic round-robin DNS to ensure the API never goes down.
- **Direct Integration:** Streams remote radio stations directly through the Supernova audio engine without cluttering your pristine local library.

### 5. Auto-Tagger (`/api/plugins/autotagger/*`)
A fully safe, non-destructive metadata enricher that fixes your library without modifying a single byte of your actual `.mp3` or `.flac` files on disk.
**Featureset:**
- **Smart Path Inference:** Automatically parses your folder structures (e.g., `/music/Artist Name/Album Name/01 - Track.mp3`) to infer missing metadata.
- **Background Processing:** Runs as an asynchronous background job, gracefully patching your Supernova database to fix "Unknown Artist" or generic "Track 1" entries.
- **Database-Only Execution:** Ensures your pristine local file tags are never overwritten or corrupted.

### 6. Album Merger (`/api/plugins/albummerger/*`)
The current implementation exposes an administrator-only **read-only preview** of possible album groups. Destructive apply is intentionally disabled until Supernova has a reviewed recovery-first workflow with backup/journal/undo semantics.

### 7. Artist Merger (`/api/plugins/artistmerger/*`)
The current implementation exposes an administrator-only **read-only preview** of normalized-name candidate groups. A preview is not merge authorization; destructive apply remains disabled pending the same recovery/undo design.

### 8. Deduper (`/api/plugins/deduper/*`)
The current implementation exposes an administrator-only **read-only preview** of duplicate-track candidates and affected user relationships. Destructive hiding/deletion is disabled pending the reviewed recovery-first workflow.

### 9. Podcasts (`/api/plugins/podcasts/*`)
A powerful podcast client integrated directly into Supernova, powered by the open PodcastIndex directory.
**Featureset:**
- **Massive Directory:** Search and browse millions of podcasts using the PodcastIndex API.
- **Direct Streaming:** Stream podcast episodes directly in the Supernova audio engine.
- **Heart Integration:** Favorite and save your top podcasts directly to your Hearts page.

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
4. Rebuild the backend so the blank import is compiled into the server. Compiled plugins are enabled by default unless `SUPERNOVA_PLUGIN_MYPLUGIN=false` is set. Supernova does not currently load arbitrary runtime `.so` plugin files.
