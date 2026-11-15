# Backup, recovery, and production checks

## Consistent live backups

The server image includes `supernova-backup`, which opens the source database
read-only, includes committed WAL data, verifies the snapshot with SQLite's
integrity check, and publishes a new file without replacing an existing backup.
The destination directory must exist and support hard links. The file contains
account hashes, encrypted Subsonic credentials, and active sessions: keep it private.

For the root Dockerfile's backend image and the standard Compose paths:

```sh
umask 077
docker compose exec backend /app/supernova-backup /root/.supernova/db/supernova.db /root/.supernova/db/snapshot.db
docker cp supernova-backend:/root/.supernova/db/snapshot.db ./supernova-snapshot.db
```

Use a new destination filename each time. Copy the completed snapshot to separate,
protected storage. A snapshot in the same Docker volume does not protect against
loss of that volume or disk. The standalone `backend/Dockerfile` places the utility
at `/root/supernova-backup` instead.

Before upgrades, retain the image version, database snapshot, `JWT_SECRET`, and
`SUBSONIC_CREDENTIAL_KEY`. Artwork is rebuildable; playlists and account state are
in the database. Music needs its own backup.

## Restore and upgrade checks

Restore into a **new empty volume or directory**, with the snapshot named
`supernova.db`, and start an isolated server against it. Do not combine a snapshot
with WAL/SHM files from a different database. Keep the original volume untouched
until login, playlists, favorites, playback, and permissions have been checked.
Use the same signing and credential keys if existing sessions and Subsonic token
credentials must remain usable. Verify file ownership matches the server user.

Startup rejects a database with a schema version newer than the server supports.
Use a compatible server or restore the pre-upgrade snapshot rather than attempting
to run an old server against a newer database.

The disposable smoke test exercises live backup, restore of accounts/sessions/
playlists, proxy routing, permission boundaries, cancellation during scanning,
and restart. It creates its own synthetic library and Docker network and removes
its containers afterward:

```sh
docker build --target backend -t supernova-review-backend .
docker build --target frontend -t supernova-review-frontend .
python3 scripts/production-smoke.py
```

The test backend runs as the invoking user with a read-only root, no capabilities,
one CPU, 256 MiB of RAM, 128 PIDs, and a 64 MiB temporary filesystem. These limits
are test conditions, not sizing recommendations for every library. Monitor RSS,
CPU, filesystem capacity, and open descriptors with a representative library.
Compose gives shutdown 30 seconds; HTTP draining uses up to 10 seconds before
active connections are closed and background work is cancelled and joined.

## Memory profiling

Use Go 1.26.11 or newer and FFmpeg. This test uses a new database and generated
music; it does not read a configured production library:

```sh
cd backend
SUPERNOVA_SOAK_SECONDS=90 SUPERNOVA_SOAK_TRACKS=1024 SUPERNOVA_SOAK_OUTPUT=/tmp/supernova-profile \
  go test ./internal/api -run TestProductionMemorySoak -count=1 -v
go tool pprof -top -base /tmp/supernova-profile/before.heap.pprof /tmp/supernova-profile/after.heap.pprof
```

The harness compares post-GC retained heap and goroutine counts after warmup.
It fails on more than 32 MiB retained growth or ten additional goroutines. Increase
duration for soak testing; the fixture count is configurable up to 10,000 tracks.

For browser navigation, build the frontend and install Playwright separately:

```sh
npm ci --prefix frontend
npm run build --prefix frontend
npm install --prefix /tmp/supernova-browser playwright@1.65.1
/tmp/supernova-browser/node_modules/.bin/playwright install chromium
PLAYWRIGHT_MODULE=/tmp/supernova-browser/node_modules/playwright node scripts/browser-memory-soak.cjs
```

This serves the built app on loopback, uses synthetic API responses, and compares
Chromium heap, DOM, and listener counts after repeated client-side navigation.
It does not validate real external service responses or hours of audio decoding.
Unit tests cover cancelled playback requests, resume listener cleanup, confirmation
settlement, timer cleanup, and lyrics-cache eviction.

The manually triggered `Production readiness soak` GitHub workflow runs these
checks and retains profiles and JSON summaries. Normal regression CI runs race
checks, migration/recovery regressions, and Go vulnerability scanning.

Metadata responses from external services are capped at 8 MiB, with eight shared
outbound slots. Browser lyrics caching retains at most 128 entries and skips
individual cached payloads larger than 128 KiB. API request deadlines are 30
seconds. These are deliberate budgets; oversized provider responses or stalled
requests fail instead of retaining unbounded work.
