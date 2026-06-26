# Supernova API Documentation

The Supernova backend exposes a lightweight, strictly-typed JSON REST API.

## Base URL
`http://localhost:8080`

---

## 1. Library (Metadata)

### Get Albums
`GET /api/albums?limit=50&offset=0`
Returns a paginated list of albums in the library.

### Get Album by ID
`GET /api/albums/{id}`
Returns a specific album's metadata.

### Get Tracks
`GET /api/tracks?album_id={id}&limit=50&offset=0`
Returns all tracks for a specific album.

### Get Artists
`GET /api/artists?limit=50&offset=0`
Returns all artists in the database.

---

## 2. Media Delivery

### Stream Audio
`GET /api/stream/{id}?format={format}&bitrate={bitrate}&time={seconds}`
Streams the raw audio file natively using HTTP Range requests. 
- **Query Params (Optional for Transcoding):**
  - `format`: `mp3`, `aac`, `ogg`, `opus`
  - `bitrate`: e.g., `128`, `320`
  - `time`: Offset in seconds for fast seeking (e.g., `60`)

### Album Art
`GET /api/art/album/{id}`
Returns the raw binary image (JPEG/PNG) of the album art. Implements ETag caching.

---

## 3. Hearts (Favorites)

### Get Hearts
`GET /api/hearts`
Returns a list of all favorited tracks and albums (Database UUIDs).

### Add Heart
`POST /api/hearts`
**Body:**
```json
{
  "entity_type": "track", // or "album"
  "entity_id": "uuid-here"
}
```

### Remove Heart
`DELETE /api/hearts?entity_type={type}&entity_id={id}`
Removes a favorite.

### Export Backup
`GET /api/hearts/export`
Generates and downloads a JSON file containing permanent `file_path` mappings for all hearts, safe against database rebuilds.

### Import Backup
`POST /api/hearts/import`
**Body:** JSON output from `/api/hearts/export`.

---

## 4. Scrobbling (Listen History)

### Log Scrobble
`POST /api/scrobbles`
**Body:**
```json
{
  "track_id": "uuid-here"
}
```

### Get Recent Scrobbles
`GET /api/scrobbles/recent`
Returns the 20 most recently listened tracks.
