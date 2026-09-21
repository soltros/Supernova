#!/usr/bin/env bash
set -euo pipefail

# Repair the v7 file-identity migration on an existing Supernova database.
#
# Usage:
#   ./scripts/repair-file-fingerprint-migration.sh
#   ./scripts/repair-file-fingerprint-migration.sh /absolute/path/to/supernova.db
#
# The script:
#   - stops the compose stack
#   - auto-detects the backend database mount when no path is supplied
#   - backs up supernova.db plus WAL/SHM sidecars
#   - refuses to skip migrations older than v6
#   - adds only missing v7 columns/indexes
#   - advances user_version from 6 to 7 only when appropriate
#   - runs PRAGMA integrity_check
#   - restarts the compose stack unless --no-start is supplied

DB_PATH=""
START_STACK=1

for arg in "$@"; do
  case "$arg" in
    --no-start)
      START_STACK=0
      ;;
    -h|--help)
      sed -n '1,32p' "$0"
      exit 0
      ;;
    *)
      if [[ -n "$DB_PATH" ]]; then
        echo "error: unexpected argument: $arg" >&2
        exit 2
      fi
      DB_PATH="$arg"
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is required" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "error: python3 is required (Ubuntu Server normally includes it)" >&2
  exit 1
fi

DOCKER=(docker)
if ! docker info >/dev/null 2>&1; then
  if command -v sudo >/dev/null 2>&1 && sudo docker info >/dev/null 2>&1; then
    DOCKER=(sudo docker)
  else
    echo "error: cannot access Docker; run as a user with Docker access or with sudo" >&2
    exit 1
  fi
fi

COMPOSE=("${DOCKER[@]}" compose)

echo "==> Stopping Supernova containers"
"${COMPOSE[@]}" down

if [[ -z "$DB_PATH" ]]; then
  echo "==> Locating the backend database mount"

  cid=$("${COMPOSE[@]}" ps -aq backend 2>/dev/null || true)

  # docker compose down removes containers, so fall back to creating the backend
  # container (without starting it) in order to inspect its mounts.
  if [[ -z "$cid" ]]; then
    "${COMPOSE[@]}" create backend >/dev/null
    cid=$("${COMPOSE[@]}" ps -aq backend)
  fi

  if [[ -z "$cid" ]]; then
    echo "error: could not find/create the Compose backend container" >&2
    exit 1
  fi

  mount_source=$("${DOCKER[@]}" inspect "$cid" --format     '{{range .Mounts}}{{if eq .Destination "/root/.supernova/db"}}{{.Source}}{{end}}{{end}}')

  if [[ -z "$mount_source" ]]; then
    echo "error: could not locate a mount at /root/.supernova/db" >&2
    echo "Pass the database path explicitly:" >&2
    echo "  $0 /absolute/path/to/supernova.db" >&2
    exit 1
  fi

  DB_PATH="$mount_source/supernova.db"
fi

DB_PATH=$(readlink -f "$DB_PATH")
DB_DIR=$(dirname "$DB_PATH")

FS=()
if [[ ! -r "$DB_PATH" || ! -w "$DB_PATH" || ! -w "$DB_DIR" ]]; then
  if command -v sudo >/dev/null 2>&1 && sudo test -f "$DB_PATH"; then
    FS=(sudo)
  fi
fi

if ! "${FS[@]}" test -f "$DB_PATH"; then
  echo "error: database not found or inaccessible: $DB_PATH" >&2
  exit 1
fi

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP_DIR="$DB_DIR/repair-backup-$timestamp"

echo "==> Database: $DB_PATH"
echo "==> Backup:   $BACKUP_DIR"

"${FS[@]}" mkdir -p "$BACKUP_DIR"
"${FS[@]}" cp -a "$DB_PATH" "$BACKUP_DIR/"

for sidecar in "$DB_PATH-wal" "$DB_PATH-shm"; do
  if "${FS[@]}" test -f "$sidecar"; then
    "${FS[@]}" cp -a "$sidecar" "$BACKUP_DIR/"
  fi
done

echo "==> Repairing v7 file-identity schema"

"${FS[@]}" python3 - "$DB_PATH" <<'PY'
import sqlite3
import sys

path = sys.argv[1]
conn = sqlite3.connect(path)
conn.execute("PRAGMA foreign_keys=ON")
conn.execute("PRAGMA busy_timeout=5000")

try:
    version = conn.execute("PRAGMA user_version").fetchone()[0]
    print(f"    current user_version: {version}")

    if version < 6:
        raise SystemExit(
            "error: database is older than migration v6; refusing to skip earlier migrations. "
            "Deploy the fixed backend image and let Supernova migrate it normally."
        )

    columns = {
        row[1]
        for row in conn.execute("PRAGMA table_info(tracks)")
    }

    required = {
        "file_modified_ns": "INTEGER DEFAULT 0",
        "file_size": "INTEGER DEFAULT 0",
        "file_fingerprint": "TEXT DEFAULT ''",
    }

    with conn:
        for name, definition in required.items():
            if name not in columns:
                print(f"    adding tracks.{name}")
                conn.execute(f"ALTER TABLE tracks ADD COLUMN {name} {definition}")
            else:
                print(f"    tracks.{name} already exists")

        print("    ensuring idx_tracks_fingerprint exists")
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_tracks_fingerprint "
            "ON tracks(file_fingerprint)"
        )

        # Only advance the migration marker when this database was exactly at
        # the version immediately before the repaired migration. Never lower a
        # newer database or jump over unknown earlier migrations.
        if version == 6:
            conn.execute("PRAGMA user_version = 7")
            print("    advanced user_version: 6 -> 7")
        else:
            print(f"    preserving user_version: {version}")

    result = conn.execute("PRAGMA integrity_check").fetchone()[0]
    if result != "ok":
        raise SystemExit(f"error: PRAGMA integrity_check failed: {result}")

    print("    integrity_check: ok")
finally:
    conn.close()
PY

echo "==> Repair complete"
echo "    Backup kept at: $BACKUP_DIR"

if [[ "$START_STACK" -eq 1 ]]; then
  echo "==> Starting Supernova"
  "${COMPOSE[@]}" up -d
  echo
  echo "Follow the backend migration/startup log with:"
  echo "  ${COMPOSE[*]} logs -f backend"
else
  echo "==> Stack left stopped (--no-start)"
fi
