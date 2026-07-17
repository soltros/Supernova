# Supernova CLI: FUSE Mount Plans

## Objective
Implement a new subcommand in `supernova-cli` (`sn mount <mountpoint>`) that mounts a user's remote Supernova music library as a local, read-only Virtual File System (VFS). This allows users to browse and play their music seamlessly using any traditional local desktop music player (e.g., VLC, mpd, foobar2000) without needing a specialized Supernova client.

## Core Technologies
*   **Language:** Go (integrated natively into `supernova-cli`)
*   **FUSE Library:** `github.com/hanwen/go-fuse/v2` (provides a robust, modern Go interface for building FUSE filesystems).
*   **Backend Interaction:** The mount will communicate exclusively with Supernova's native JSON REST API (`http://localhost:8080/api/...`).

## Virtual Directory Structure
The FUSE mount will present a logical, human-readable directory hierarchy based on the API's metadata. 

```text
/mnt/supernova/
├── Artists/
│   └── <Artist Name>/
│       └── <Album Name>/
│           ├── 01 - <Track Title>.flac
│           └── 02 - <Track Title>.flac
├── Albums/
│   └── <Album Name>/
│       ├── 01 - <Track Title>.flac
│       └── ...
├── Favorites.m3u     <-- Dynamically generated
└── Playlists/        <-- Future implementation
```

## Implementation Details

### 1. Directory Browsing (Metadata Translation)
When a user or media player browses the mounted directory (e.g., running `ls /mnt/supernova/Artists`), the FUSE implementation will intercept the request and call the corresponding Supernova native API endpoints:
*   `GET /api/artists` for the root Artists folder.
*   `GET /api/albums?limit=...` for the Albums folder.
*   `GET /api/tracks?album_id={id}` to populate the contents of an album directory.

*Note: To ensure high performance, metadata may need to be lightly cached in memory by the CLI to prevent excessive API spam during media player library scans.*

### 2. Audio Streaming (On-Demand Reading)
Audio files are **not** downloaded in full when the filesystem mounts. 
When a local media player attempts to open and read a track (e.g., `01 - Track Title.flac`), FUSE will intercept the byte-read requests.
*   The CLI will map the file path to its internal Supernova `track_id` (UUID).
*   The CLI will proxy the read by making HTTP `Range` requests to `GET /api/stream/{id}`.
*   Seeking through a track locally will translate instantly to a new HTTP `Range` request, streaming only the necessary bytes.

### 3. Favorites and M3U Playlists
To provide natural integration for "Hearts" and custom playlists without breaking the local file illusion:
*   Favorites will be exposed as a dynamic `Favorites.m3u` file at the root of the mount.
*   When `Favorites.m3u` is read by the OS, the CLI will call `GET /api/hearts`, fetch the favored track UUIDs, and generate M3U plain-text on the fly.
*   The generated M3U file will contain relative paths pointing to the tracks within the mount (e.g., `Artists/Daft Punk/Discovery/01 - One More Time.flac`). 

## Phased Rollout
1.  **Phase 1:** Basic FUSE scaffolding. Mount a dummy directory structure and verify that `sn mount` successfully attaches a VFS to the OS.
2.  **Phase 2:** Connect the native `/api/artists` and `/api/albums` endpoints to populate read-only folders.
3.  **Phase 3:** Implement file `Read()` interception and map it to `GET /api/stream/{id}`. Test playback in VLC/mpv.
4.  **Phase 4:** Implement `Favorites.m3u` via `GET /api/hearts`.
5.  **Phase 5 (Future):** Introduce native `/api/playlists` endpoint on the backend and implement the `Playlists/` directory in the FUSE mount.
