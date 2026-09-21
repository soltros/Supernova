# Supernova Desktop

Native Flutter client for the Supernova server.

This replaces the old Electron wrapper. It talks directly to the native Supernova REST API and does not embed the web frontend.

## Current surface

- instance setup
- login and registration
- secure token storage through the Linux secret service
- dashboard, albums, artists and search
- native music playback using scoped media tickets
- playlists and favorites
- podcasts and Podcast Index search
- internet radio and saved stations
- administrator scan/enrichment controls
- plugin status

## Development

Flutter 3.47+ is recommended.

On Debian/Ubuntu:

```bash
sudo apt install clang cmake ninja-build pkg-config libgtk-3-dev libstdc++-12-dev libsecret-1-dev
flutter pub get
flutter run -d linux
```
