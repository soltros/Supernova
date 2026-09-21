# Supernova Native API

Supernova exposes a JSON REST API used by the web client, Flutter desktop client, CLI, and integrations.

This document describes the native `/api/*` surface. The Subsonic/OpenSubsonic compatibility API lives under `/rest/*` and is intentionally documented separately from the native contract.

## Base URL

Use the root of the Supernova backend, for example:

```text
https://music.example.com
http://localhost:8080
```

Unless noted otherwise, request and response bodies are JSON.

## Authentication

### Register

`POST /api/auth/register`

```json
{
  "username": "alice",
  "password": "correct horse battery staple",
  "invite_code": "optional"
}
```

The first account becomes administrator. Later registrations require `REGISTRATION_INVITE_CODE` when the server has one configured.

Returns:

```json
{
  "token": "<session JWT>",
  "user": {
    "id": "<uuid>",
    "username": "alice",
    "is_admin": true
  }
}
```

### Login

`POST /api/auth/login`

```json
{"username":"alice","password":"..."}
```

Returns the same authentication envelope as registration.

### Current user

`GET /api/auth/me`

Requires `Authorization: Bearer <token>`.

### Logout

`POST /api/auth/logout`

Revokes the current persisted session.

### Change password

`POST /api/auth/change-password`

```json
{"current":"old password","new":"new password"}
```

Changing the password revokes other sessions.

### Media tickets

Native media clients should not put the full account token into stream/download URLs.

`POST /api/media-ticket`

```json
{"scope":"stream","resource":"<track-id>"}
```

Supported scopes:

- `stream`
- `download-track`
- `download-album`

Returns `{"ticket":"..."}`. Tickets are short lived, resource scoped, and tied to the active persisted session.

## Public library metadata

Library metadata reads are currently public. User-specific data and media delivery require authentication.

### Artists

- `GET /api/artists?limit=50&offset=0`
- `GET /api/artists?letter=A&limit=50&offset=0`
- `GET /api/artists/{id}`

### Albums

- `GET /api/albums?limit=50&offset=0`
- `GET /api/albums?artist_id={artist-id}&limit=50&offset=0`
- `GET /api/albums/{id}`

### Tracks

`GET /api/tracks?album_id={album-id}&artist_id={artist-id}&limit=50&offset=0`

Both filters are optional.

Pagination accepts `limit` up to 1000 and a non-negative `offset`.

### Search

`GET /api/search?q={query}`

Returns:

```json
{
  "artists": [],
  "albums": [],
  "tracks": []
}
```

### Album artwork

`GET /api/art/album/{album-id}`

Returns the image body and supports HTTP cache validators.

## Dashboard and discovery

Authenticated routes:

- `GET /api/dashboard`
- `GET /api/discovery`

The dashboard includes recently added albums, recently played tracks, and favorite tracks. Discovery provides the server-curated discovery payload used by first-party clients.

## Playback and downloads

### Stream a track

`GET /api/stream/{track-id}`

Authentication may be supplied through the bearer header or a valid `ticket` query parameter.

Without a `format`, the original file is served with normal HTTP Range support.

Optional transcoding parameters:

- `format=mp3|aac|ogg|opus`
- `bitrate=64..320`
- `time={seconds}`

Example:

```text
GET /api/stream/abc123?format=opus&bitrate=160&time=45&ticket=...
```

### Download a track

`GET /api/download/track/{track-id}`

Use a `download-track` media ticket for browser/player URLs.

### Download an album

`GET /api/download/album/{album-id}`

Returns a staged ZIP archive. Use a `download-album` media ticket for browser URLs.

## Favorites

All favorites routes require authentication.

### Raw favorites

`GET /api/hearts`

### Hydrated favorites

`GET /api/hearts/details`

Returns grouped hydrated data:

```json
{
  "tracks": [],
  "albums": [],
  "artists": [],
  "playlists": [],
  "radio": [],
  "podcasts": []
}
```

### Add a favorite

`POST /api/hearts`

```json
{
  "entity_type": "track",
  "entity_id": "<id>",
  "metadata": {}
}
```

Allowed entity types are `track`, `album`, `artist`, `playlist`, `radio`, and `podcast`. Metadata is primarily used for external radio/podcast entities.

### Remove a favorite

`DELETE /api/hearts?entity_type={type}&entity_id={id}`

### Backup

- `GET /api/hearts/export`
- `POST /api/hearts/import`

Current exports use a version-2 envelope and stable track references where possible. Imports are transactional.

## Playlists

All playlist routes require authentication.

- `GET /api/playlists`
- `POST /api/playlists` with `{"name":"Road Trip"}`
- `DELETE /api/playlists/{id}`
- `GET /api/playlists/{id}/tracks`
- `POST /api/playlists/{id}/tracks` with `{"track_id":"..."}`
- `DELETE /api/playlists/{id}/tracks/{trackId}`
- `GET /api/playlists/export`
- `POST /api/playlists/import`

Playlist entries are ordered and repeated tracks are valid.

## Scrobbles / listen history

- `POST /api/scrobbles` with `{"track_id":"..."}`
- `GET /api/scrobbles/recent`

Both require authentication.

## Library scanning and settings

### Start a full scan

`POST /api/scan`

Administrator only. A concurrent scan returns `409`.

### Scan status

`GET /api/scan/progress`

```json
{"status":"idle","files_scanned":1234}
```

### Reset artist enrichment

`POST /api/settings/reset-artists`

Administrator only.

## Plugin manifest

`GET /api/plugins`

Returns the registered plugin manifest and enabled state.

All `/api/plugins/*` routes require authentication. Maintenance write/preview routes additionally require an administrator account.

## Lyrics (LRCLIB)

`GET /api/plugins/lrclib/lyrics`

Query parameters:

- `artist_name`
- `track_name`
- `album_name`
- `duration` in seconds

The server proxies LRCLIB so first-party clients do not need separate CORS/rate-limit handling.

## Last.fm

When the Last.fm plugin is enabled:

- `GET /api/plugins/lastfm/auth-url?cb={callback-url}`
- `POST /api/plugins/lastfm/session` with `{"token":"..."}`
- `POST /api/plugins/lastfm/nowplaying`
- `POST /api/plugins/lastfm/scrobble`

Now-playing body:

```json
{"session_key":"...","artist":"Artist","track":"Song"}
```

Scrobble body additionally includes Unix `timestamp`.

## Podcasts

Podcast Index search requires the server to have Podcast Index credentials configured.

### Search and episodes

- `GET /api/plugins/podcasts/search?q={query}`
- `GET /api/plugins/podcasts/episodes?id={feed-id}`

### Account subscriptions

- `GET /api/plugins/podcasts/subscriptions`
- `POST /api/plugins/podcasts/subscriptions`
- `DELETE /api/plugins/podcasts/subscriptions?feed_id={id}`

Subscription body:

```json
{
  "feed_id": "123",
  "feed_url": "https://example.com/feed.xml",
  "title": "Podcast",
  "image_url": "https://..."
}
```

### Episode progress

- `POST /api/plugins/podcasts/progress`
- `POST /api/plugins/podcasts/progress/batch`

Save body:

```json
{"episode_id":"...","position_ms":42000,"completed":false}
```

Batch read body:

```json
{"episode_ids":["a","b","c"]}
```

### OPML

- `GET /api/plugins/podcasts/opml/export`
- `POST /api/plugins/podcasts/opml/import` as multipart form data with a `file` field

OPML import is accepted and completed asynchronously.

## Internet radio

### Search

`GET /api/plugins/radio/search?q={name}&country={country}&limit=50&offset=0`

At least `q` or `country` is required. Results are proxied from Radio-Browser.

### Saved stations

- `GET /api/plugins/radio/subscriptions`
- `POST /api/plugins/radio/subscriptions`
- `DELETE /api/plugins/radio/subscriptions?station_id={id}`

POST body:

```json
{
  "station_id": "...",
  "url": "https://stream.example.com",
  "name": "Station",
  "favicon": "https://..."
}
```

## AutoTagger

`POST /api/plugins/autotagger/run`

Administrator only. Starts the database-only path metadata repair job and returns `202 Accepted`. Supervised library mutation jobs are mutually exclusive.

## Maintenance previews

The destructive duplicate/merge implementations are intentionally disabled.

Administrator-only read-only previews:

- `GET /api/plugins/deduper/preview`
- `GET /api/plugins/albummerger/preview`
- `GET /api/plugins/artistmerger/preview`

Corresponding `POST .../run` routes return `409 Conflict` until the recovery/journal/undo design is approved.

## Health

`GET /api/health`

Public health response:

```json
{"status":"ok","version":"1.0"}
```

## Errors and limits

- JSON/API request bodies are capped at 10 MiB.
- Auth failures return `401`.
- Administrator-only routes return `403` for non-admin users.
- Registration/login throttling can return `429`.
- Busy supervised jobs can return `409` or `503`, depending on the subsystem.
- Clients should treat non-2xx responses as failures and surface the response text where useful.
