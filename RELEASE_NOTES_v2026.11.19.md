# Supernova v2026.11.19 — Draft release notes

> **Release status:** Draft notes only. This document is not a published GitHub release.
>
> **Changes compared with:** `v2026.09.21`  
> **Reviewed baseline:** `f492400551d3d766e2dd6336dfb1578f861cfacc` (19 commits ahead)  
> **Suggested release tag:** `v2026.11.19`  
> **Suggested release title:** Supernova v2026.11.19 — Reliability & UI Refinements

Supernova's November update focuses on making the server more dependable and the web player easier to use. It tightens authentication and background-task cleanup, adds recovery and diagnostic tooling, refreshes the desktop/mobile client documentation, and fixes several visible interface problems. This is primarily a stability and usability release rather than a major change to the music library.

## Highlights

### Web player improvements

- **Scrollable playlists:** Long playlist collections now scroll inside the sidebar without pushing the account controls offscreen.
- **Visible account controls:** The active username and log-out/switch-user actions are available in the sidebar and top bar. Switching users returns promptly to the login screen.
- **Radio artwork fallback:** Stations with no image, or a broken image URL, use a branded Supernova Radio placeholder in discovery, subscriptions, and playback artwork.
- **Cleaner metadata wording:** Removed the "automagic" wording from the library metadata description.

### Server and playback reliability

- Hardened password-change and login concurrency, session invalidation, legacy credential handling, and authentication rate-limiting.
- Bounded retained asynchronous work and made background operations more responsive to cancellation, including podcast and metadata-enrichment requests.
- Improved scanner recovery around queue overflow and task teardown; added resource and request limits to reduce runaway retention.
- Fixed races involving delayed playback media tickets and podcast resume listeners, and improved UI timer and async cleanup.
- Improved CLI download publication so an in-progress transfer cannot overwrite a newly created destination file.
- Added safer, WAL-aware database backup and recovery tooling, with explicit validation of unsupported future schema versions.

### Operations, testing, and security

- Added an operations guide, production review, synthetic soak tests, browser memory/navigation checks, and disposable recovery validation.
- Updated Go and JavaScript dependency constraints as part of the production-readiness review.
- Added dedicated production-readiness checks and strengthened test-gated Docker publication.
- Removed an obsolete CI test step pointing to the deleted `desktop-app` directory. The independent client project is now documented separately.

### Desktop and mobile client information

- Updated the website and README to showcase the standalone Supernova desktop and mobile clients, including Linux DEB/RPM/AppImage previews and iOS/Android preview installation information.
- The separate client project is maintained at [soltros/supernova-desktop](https://github.com/soltros/supernova-desktop). These client packages are **not** bundled in this server release; download availability and signing requirements must be checked in that repository.
- Removed the old in-repository `desktop-app` Electron tree. This is a repository/layout change; existing separately installed clients are not migrated automatically.

## Upgrade notes

- **Docker:** After the tagged server/frontend images are published, pull the new images and recreate the containers using the same persistent volumes and configuration. Back up the database before upgrading.
- **Web/PWA:** Reload the application after the frontend update; an available service-worker update may offer an explicit reload.
- **Clients:** Desktop, Android, and iOS preview packages are distributed by the separate client project, not the Supernova server image.
- No automatic migration of a previously installed in-repository desktop app is provided by this repository.

## Validation and limitations

The `main` branch at `f492400` passed the [Docker publish workflow](https://github.com/soltros/Supernova/actions/runs/44593996812) (including its required regression jobs) and [GitHub Pages deployment](https://github.com/soltros/Supernova/actions/runs/44593996508). The frontend tests (16 tests), TypeScript/Vite build, frontend lint, website build, and backend test jobs passed in the preceding regression run; the final publishing workflow completed successfully after the CI correction.

The [production review](docs/PRODUCTION_REVIEW_2026-11-15.md) reports short synthetic load and browser navigation measurements. Those measurements do not establish zero memory leaks for real large libraries or long-running workloads. Native desktop/mobile installers were not validated by this repository's CI.

## Links

- [Compare changes since v2026.09.21](https://github.com/soltros/Supernova/compare/v2026.09.21...f492400551d3d766e2dd6336dfb1578f861cfacc)
- [Operations guide](docs/OPERATIONS.md)
- [Production review](docs/PRODUCTION_REVIEW_2026-11-15.md)
- [Desktop/mobile client project](https://github.com/soltros/supernova-desktop)
