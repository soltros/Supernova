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

## Follow-up readiness pass

The follow-up added real Chromium navigation measurements, a synthetic Go
streaming/rescan soak, and disposable Docker recovery/resource tests. Repeatable
commands and limitations are in [OPERATIONS.md](OPERATIONS.md).

Additional fixes include bounded lyrics retention and upstream response sizes,
cancellable/joined enrichment and podcast work, scanner recovery after event
queue overflow, consistent WAL-aware backup tooling, stale-login rejection after
password changes, and rejection of unsupported future database schemas. Compose
now forwards the dedicated Subsonic key. Docker contexts exclude local dependency
folders and credential directories. Website builds preserve hand-written docs.

Frontend and website lint were reviewed: initialization/loading effects and
co-located context hooks have narrow, explained exemptions; stale callback,
unused-variable, and cleanup issues were fixed. Desktop minimatch overrides use
11.0.8 and have a packaging-glob compatibility regression test; native macOS and
Windows installers still require their platform build checks.

### Measured results

- Final Go 1.26.11 soak: 1,024 generated tracks, 90 seconds, 17 rescans, and
  2,975 API/stream requests. Post-GC heap went from 1,494,808 to 1,483,656 bytes;
  goroutines remained at nine.
- Chromium: 200 client-side navigations. Post-GC heap went from 3,668,760 to
  4,229,952 bytes; DOM nodes remained at 192 and listeners at 199. No page errors.
- Disposable Docker recovery tests passed with the limits described in OPERATIONS.
- Backend race tests/vet, CLI tests/vet, 16 frontend tests, frontend/website builds
  and lint, desktop security/glob tests, and final image builds passed.
- Go vulnerability scanning originally found 21 reachable standard-library
  advisories with Go 1.26.4. The minimum compiler and Docker builders now use
  1.26.11; x/crypto and x/sys were updated. Final govulncheck and the three npm
  audits report no vulnerabilities.

API fetches now have 30-second deadlines and scan-status polling does not overlap
requests. This bounds retention when a server stops responding; lyrics overlays
and session verification also cancel work on teardown.

The machine-readable [results](production-readiness-results.json) record these
measurements. These are regression signals from small synthetic libraries and
short runs, not proof of zero leaks. Real audio decoding, external providers,
very large libraries, hours-long workloads, and native platform installers are
not covered by the browser/navigation fixture.
