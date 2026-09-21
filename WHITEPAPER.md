# Supernova: Project Whitepaper

## 1. Vision and scope

Supernova is a self-hosted music server for local libraries. Its current implementation combines a Go backend, SQLite, FFmpeg/FFprobe, a React/Vite web client, a native Flutter Linux desktop client, and a compile-time Go plugin registry.

This document distinguishes **implemented behavior**, **review-gated behavior**, and **planned work**. It is not a promise that every Subsonic/OpenSubsonic client, platform package, or future feature is already supported.

## 2. Current implementation

### Library and metadata

- Recursively scans a configured local media tree and watches it for filesystem changes.
- Extracts local audio metadata and probes duration with FFprobe.
- Stores artists, albums, tracks, playlists, favorites, scrobbles, external subscriptions, and related state in SQLite.
- Uses nanosecond modification time, file size, and a bounded content fingerprint to detect changes and preserve track identity across ordinary file moves where the fingerprint match is unambiguous.
- Uses folder-aware fallback grouping for media with incomplete tags.
- Runs background MusicBrainz and Last.fm enrichment with bounded external-request concurrency and retryable transient failures.
- AutoTagger can infer missing/generic database metadata from paths. It updates Supernova's database; it does not rewrite the media files on disk.

### Playback and media delivery

- Serves original media with HTTP range support.
- Supports bounded FFmpeg transcoding for the formats implemented by the native and Subsonic compatibility routes.
- Opens library media through rooted filesystem handles so containment validation and opening are performed together.
- Uses a global transcode budget and a shared external-HTTP admission budget to prevent unbounded fan-out.
- Album downloads are staged and finalized before the HTTP success response is committed.

### Accounts and user data

- The first registered account becomes administrator.
- Subsequent registration requires the configured invite code; leaving it blank closes later registration.
- Authentication uses signed JWTs backed by persisted, revocable sessions.
- Reversible Subsonic compatibility credentials use a separate `SUBSONIC_CREDENTIAL_KEY`; legacy JWT-secret-encrypted values are accepted only as an upgrade path.
- Logout revokes the current session. Password changes revoke other sessions.
- Browser media elements use short-lived, resource-scoped media tickets instead of full bearer tokens in media URLs.
- Playlists support stable ordered entries, including repeated tracks.
- Playlist and favorite exports use versioned backup envelopes with stable references where available. They are portability backups, not substitutes for a verified SQLite disaster-recovery backup.

### Web and PWA client

- Provides responsive library browsing, search, playback, playlists, favorites, podcasts, radio, lyrics, and settings.
- Uses the Media Session API where supported.
- Supports complete paged library retrieval instead of silently stopping at fixed client-side limits.
- Guards stale asynchronous UI requests and maintains per-track context for mixed queues.
- The service worker caches the application shell/static assets only. Authenticated API traffic and music are network-only; Supernova does not currently advertise offline music downloads.
- Includes keyboard/focus improvements, accessible labels for core controls, reduced-motion handling, and mobile layout fixes. This is not an accessibility-conformance certification.

### Desktop and command-line clients

- The Flutter desktop wrapper validates configured HTTP(S) origins, restricts navigation/IPC, denies unapproved new windows and permissions, and isolates the Last.fm OAuth flow.
- The Go CLI uses atomic download publication and does not overwrite an existing destination implicitly.
- The Python GUI normalizes grouped favorite responses, distinguishes entity types, guards stale requests, discovers the bundled CLI relative to its own location, and uses bounded child-process cleanup.

### Plugins

Plugins are registered at **compile time** through the Go plugin interface. Supernova does not dynamically load arbitrary runtime `.so` files.

Built-in integrations include:

- Subsonic/OpenSubsonic compatibility layer
- Last.fm
- LRCLIB
- Podcast Index
- Radio-Browser
- AutoTagger
- maintenance preview tools for duplicate tracks, albums, and artists

The Subsonic/OpenSubsonic surface is an implemented compatibility subset, not a claim of complete protocol conformance. It includes the routes and semantics present in the current server, including paged search/listing, playlists, favorites, scrobbling, raw media delivery, and supported transcoding.

## 3. Review-gated maintenance behavior

Duplicate-track, album-merge, and artist-merge writes are intentionally disabled.

The code review identified data-loss risks in the previous destructive heuristics. The current server therefore exposes administrator-only, read-only candidate previews while destructive apply returns a conflict response.

Before write behavior is enabled, the maintenance design requires:

1. A read-only preview based on stable candidate identities and affected relationship counts.
2. A verified SQLite recovery point that correctly includes WAL state, checksum/version metadata, integrity checking, and a rehearsed restore.
3. Explicit reviewed source/canonical mappings revalidated immediately before apply.
4. Transactional preservation of playlist entries, favorites, scrobbles, timestamps, and unaffected IDs.
5. A reversible journal/visibility model and tested undo path rather than implicit irreversible deletion.
6. Acceptance fixtures covering ambiguous editions, repeated playlist entries, same-title distinct tracks, exact copies, Unicode/case variants, interrupted writes, full-disk behavior, concurrent jobs, apply/undo/reapply, and independent restore.

No destructive maintenance operation is considered production-ready until that recovery-first design is reviewed and its acceptance tests pass.

## 4. Architecture

### Backend

- Go
- `ncruces/go-sqlite3`
- FFmpeg / FFprobe
- `fsnotify`
- shared job supervision for coordinated mutation work
- shared outbound HTTP budget for external integrations

### Frontend

- React
- TypeScript
- Vite
- browser Media Session API
- service-worker shell caching

### Desktop

- Electron wrapper around the configured Supernova web origin
- isolated handling for allowed external OAuth navigation

### Plugin model

Plugins implement the Go `plugins.Plugin` interface, register with `plugins.Register`, and are compiled into the backend through imports. Environment variables can disable built-in compiled plugins.

## 5. Reliability and release model

The repository contains regression CI covering backend race-enabled tests/vet, CLI tests/vet, frontend tests/build/lint, website build, and desktop security-helper tests.

Container publication is test-gated and uses the supported component build contexts. Published image metadata includes immutable commit-derived tags alongside the documented moving/release tags, with provenance/SBOM generation in the publish workflow.

Release/package integrity should remain tied to reproducible artifacts and recorded checksums. Platform packaging and actual install/upgrade/restore rehearsals are release-validation activities and are not implied merely by a passing source test suite.

## 6. Compatibility boundaries

Supernova's native JSON API is the primary application API.

The Subsonic/OpenSubsonic layer is provided for compatibility with third-party clients, but client behavior varies. A passing protocol fixture does not establish universal client compatibility. Real chosen-client sync, playlist, browsing, transcoding, and playback sessions remain part of release validation.

Likewise, passing unit/integration tests do not certify every browser, desktop installer, reverse proxy, filesystem, or external service.

## 7. Planned or exploratory work

The following are **not current capabilities unless separately implemented later**:

- GraphQL API
- runtime loading of arbitrary shared-object plugins
- first-class folder-view browsing
- offline music-library downloads in the PWA
- waveform generation and waveform seek UI
- automatic dominant-color theming
- automatic destructive deduplication/album/artist consolidation
- single-file ingest API
- fully automatic compilation/soundtrack grouping
- a claim of flawless seeking/transcoding across every codec and third-party client

These ideas may be useful roadmap items, but documentation should not present them as shipped behavior.

## 8. Operational guidance

Before upgrading a real library:

- keep a verified database recovery copy;
- configure a strong JWT secret and a separate Subsonic credential key; keep the previous JWT secret available during migration until legacy compatibility credentials have been refreshed by normal web logins;
- configure the desired registration invite policy;
- install FFmpeg including FFprobe;
- persist database and art-cache storage;
- test migrations against a copied database first;
- rescan representative media and verify durations/identity behavior;
- verify administrator-only maintenance access, playback/seek, favorites/playlist backup round trips, podcast resume, and logout/session revocation;
- keep destructive maintenance writes disabled until the recovery-first design above is explicitly reviewed and validated.

Supernova's JSON export features improve portability, but they are not a complete replacement for a consistent SQLite backup and tested restore procedure.
