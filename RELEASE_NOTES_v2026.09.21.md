# Supernova v2026.09.21

This one is mostly fixes — a lot of them.

Over the last few days I went through Supernova pretty much end to end: the scanner, database, Subsonic support, auth, playback, desktop app, CLI, PWA, build pipeline, packaging, and the docs. That turned into a much larger cleanup than I originally expected, but it also knocked out a long list of bugs and rough edges that had been hanging around.

There aren't a ton of shiny new features here. The point of this release is that Supernova should behave better, fail more cleanly, and be a lot safer with your library and account data.

## The big stuff

### Library scanning is a lot more reliable

The scanner has had a pretty major rewrite.

Supernova now uses file size, nanosecond timestamps, and a content fingerprint to help identify tracks. In normal move/rename cases, that means it can keep the existing track ID instead of deleting the old track and creating a new one.

That matters because the track ID is what playlists, favorites, scrobbles, and other parts of the database point at.

The watcher also handles removals and renames better, does bounded debouncing instead of spawning a pile of sleeping goroutines, reconciles files that disappeared while the server was offline, and shuts its workers down cleanly.

Untagged music also no longer gets dumped into one giant `Unknown Album`; folder names are used as a better fallback.

### Subsonic support got a serious pass

A lot of the Subsonic layer worked well enough for basic use but had edge cases where it would acknowledge things it hadn't really done, ignore paging arguments, or behave differently from what a client asked for.

That has been cleaned up.

Playlist changes are transactional now, repeated tracks are supported, index-based removals work properly, and create/update responses reflect the state that was actually written.

Search now respects separate artist/album/song limits and offsets, album list endpoints actually page and sort according to the supported list type, and streaming can honor supported format, bitrate, and seek requests.

Raw playback still keeps normal HTTP range support.

I'm still calling this a Subsonic/OpenSubsonic compatibility layer rather than claiming 100% protocol parity. Different clients do different things, and I'd rather be accurate about what Supernova supports.

### Sessions can actually be revoked now

Authentication has been reworked around real persisted sessions.

Logging out now revokes the current session. Changing your password can revoke your other sessions. Login and registration attempts are rate limited.

Media URLs also no longer carry your full bearer token around in the query string. The web player and download links use short-lived tickets scoped to the specific media resource instead.

There's also a new optional-but-recommended setting:

```env
SUBSONIC_CREDENTIAL_KEY=your_own_random_secret
```

This is used for the reversible password copy required by Subsonic token/salt authentication, so changing `JWT_SECRET` doesn't also destroy your Subsonic credentials.

Generate it separately:

```bash
openssl rand -hex 32
```

Existing installations using the old JWT-based encryption path have a migration fallback.

### Safer media access

Media files are now opened through a rooted filesystem handle rather than checking a path first and opening it later.

That closes a class of path/symlink race problems and gives both normal streaming and downloads a single safer way to access the library.

FFmpeg work is also globally bounded now, as are outgoing requests to services like Last.fm, MusicBrainz, Podcast Index, RadioBrowser, and LRCLIB. A busy server should back off instead of spawning unlimited expensive work.

### Enrichment failures don't get stuck forever

Temporary Last.fm/MusicBrainz failures now have persisted retry state with exponential backoff.

A provider outage shouldn't permanently mark an album as missing data, and it also shouldn't cause Supernova to retry the same failed item every few seconds forever.

### Playlist and backup fixes

Playlists now use real entry IDs instead of treating `playlist + track` as the unique key. So yes, you can finally put the same song in a playlist twice without the database fighting you.

Backup/import behavior was tightened up too. Playlist and favorite restores are transactional, use more stable references where possible, and external podcast/radio favorite metadata can be backed up with the account rather than living only in one browser.

### Album downloads no longer fail halfway into a "successful" ZIP

Album ZIP files are staged and finalized before Supernova sends the successful download response.

If something goes wrong while reading a track or building the ZIP, the request fails instead of handing you a truncated archive with a 200 OK attached to it.

### Background jobs now know about each other

Full scans and mutation jobs now run through a shared supervisor instead of each subsystem doing its own thing.

That means fewer overlapping database-changing jobs, proper cancellation, and cleaner shutdown behavior.

### Desktop app replaced with Flutter

The old Electron wrapper has been retired. Supernova now has a native Flutter Linux client that talks directly to the REST API instead of embedding the web frontend.

Navigation is restricted to the configured Supernova server, arbitrary popups and permissions are blocked, IPC is restricted to trusted frames, the renderer stays sandboxed, and Last.fm login happens through a dedicated OAuth window with a validated callback.

### CLI and Python GUI fixes

CLI downloads are now written to a temporary file and only moved into place after the transfer has completed and synced successfully.

The CLI also won't silently overwrite an existing destination, and large downloads no longer inherit the short timeout used for normal API calls.

The Python GUI had several annoying edge cases fixed too: grouped favorites, stale async results, identifying which rows are actually playable tracks, locating the bundled CLI, and process cleanup that could hang.

### Web player and UI fixes

Large libraries should no longer silently stop at arbitrary frontend limits.

Queue state is more reliable with mixed albums and external playback, stale requests are discarded, and a number of actions that previously looked successful even when the server rejected them now show the failure instead.

There was also another pass over keyboard behavior, focus states, labels, reduced motion, and mobile layout issues.

### PWA behavior is less magical and more honest

The service worker now sticks to the app shell and static assets.

Authenticated API calls and music are deliberately network-only. Supernova does not currently provide offline-library downloads through the PWA, and the code/docs now say that plainly.

## About the duplicate/album/artist merge tools

These deserve a special note.

The old automatic dedupe/merge jobs had cases where they could make destructive guesses about genuinely different tracks, releases, or artists.

Rather than trying to patch around that with another heuristic, I've removed the destructive worker code for now.

The maintenance plugins are still there, but they're read-only previews. The actual merge/delete side will stay disabled until there is a proper recovery design: verified backups, explicit mappings, transactional updates, a journal, and a real undo path.

I'd rather ship a disabled tool than one that can quietly eat library history.

## Builds, releases, and CI

The release pipeline has been cleaned up as well.

Docker publication is now gated behind the regression workflow, duplicate/competing publish workflows were removed, component build contexts were fixed, immutable SHA tags are generated, and images include provenance/SBOM data.

The AUR metadata was also corrected to use the project's GPL-3.0-only license and a pinned SHA-256 for the release artifact.

CI now covers the backend, CLI, frontend, website build, Electron security helpers, and the manual Subsonic smoke client.

## Documentation cleanup

A bunch of older documentation had gotten ahead of the code.

Claims about complete OpenSubsonic parity, runtime `.so` plugins, GraphQL, offline music, automatic destructive dedupe, waveforms, and a few other roadmap ideas have been cleaned up.

The README, website, and whitepaper now try to separate what exists today from what is planned or intentionally disabled.

## Upgrading

Back up your database before upgrading. This release contains several database migrations.

If you use Subsonic token/salt authentication, add a separate credential key:

```env
JWT_SECRET=your_existing_jwt_secret
SUBSONIC_CREDENTIAL_KEY=a_different_random_secret
```

Then recreate/update Supernova normally and let the migrations run.

For an important library, I strongly recommend testing the upgrade against a copy of the database first.

## One last thing

This release touched a lot of code.

The automated test suite is considerably better than it was before this work, but there are still things that need real-world testing rather than another unit test: third-party Subsonic clients, actual install/upgrade paths, restores, accessibility on physical devices, and external-service failures.

The goal here wasn't to declare Supernova "finished." It was to get rid of a pile of known bad behavior and give the project a much better base to build on.

Thanks for testing it.
