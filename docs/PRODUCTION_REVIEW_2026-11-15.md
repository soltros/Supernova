# Production review — 2026-11-15

This review focused on authentication consistency, retained memory and asynchronous
work, playback lifecycle, and safe download publication. It is a source review
with regression tests, not a certification that every bug or memory leak is gone.

## Changes

- Password changes now replace bcrypt and reversible Subsonic credentials and
  revoke other sessions in one transaction. If the credential key is unavailable,
  the old encrypted password is cleared. Concurrent changes compare the previously
  verified hash; login and legacy migration cannot restore credentials for an old hash.
- Failed-authentication bookkeeping expires inactive entries and caps retained
  keys at 4096. At capacity new keys receive HTTP 429 until space is available;
  existing keys continue through the normal limiter. Login input lengths are bounded.
- OPML imports allow one background worker per server and at most 1000 feeds per
  upload, with a 15-minute job deadline. Podcast search and episode requests now
  cancel outbound work when the incoming request ends. Import timeouts can leave
  a partial import, as imports already apply subscriptions individually.
- Podcast resume listeners are aborted on track replacement and provider teardown.
  A late media ticket cannot replace the current source after another track starts.
- Toast timers are cleared on teardown. Confirmation promises resolve false when
  replaced or when the provider unmounts.
- CLI downloads publish using an exclusive hard link to avoid overwriting a file
  created during the download. Filesystems without hard-link support return a
  publication error; temporary downloads are cleaned up.
- Compatible frontend lockfile updates address the six npm audit findings,
  including DOMPurify, Vite, Vitest/mocker, source-map-js, and undici.

## Verification and limits

Backend tests and race detection, backend/CLI vet, CLI download regression test,
frontend tests/build/lint, and desktop security tests were run. Frontend lint
reports warnings. npm audit reported zero vulnerabilities after updates.

No production library, credentials, or service configuration was changed. No
production heap profiles or sustained playback/load soak tests were collected.
Those are still needed to establish memory behavior under real library sizes and
long-running workloads. Bounded caches and workers prevent the identified growth
paths; they do not bound total process memory under arbitrary request concurrency.
