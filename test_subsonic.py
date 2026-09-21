#!/usr/bin/env python3
"""Small manual Subsonic/OpenSubsonic smoke client for a running Supernova server.

This script is deliberately not a replacement for the Go protocol regression suite.
It is useful for an operator or reviewer who wants to exercise a deployed instance.
"""

import argparse
import json
import os
import sys
import urllib.parse

import requests


DEFAULT_BASE_URL = os.environ.get("SUPERNOVA_SUBSONIC_URL", "http://localhost:8080/rest")
DEFAULT_TIMEOUT = float(os.environ.get("SUPERNOVA_SUBSONIC_TIMEOUT", "15"))


class SubsonicClient:
    def __init__(self, base_url, username, password, timeout=DEFAULT_TIMEOUT):
        self.base_url = base_url.rstrip("/")
        self.username = username
        self.password = password
        self.timeout = timeout
        self.version = "1.16.1"
        self.client = "supernova_smoke"
        self.format = "json"

    def _auth_params(self):
        return {
            "u": self.username,
            "p": self.password,
            "v": self.version,
            "c": self.client,
            "f": self.format,
        }

    @staticmethod
    def _masked(params):
        masked = dict(params)
        if "p" in masked:
            masked["p"] = "***"
        return masked

    def request(self, endpoint, params=None, method="GET"):
        payload = self._auth_params()
        if params:
            payload.update(params)

        url = f"{self.base_url}/{endpoint}"
        print(f"-> {method} {url} | params: {self._masked(payload)}")

        try:
            if method == "POST":
                response = requests.post(url, data=payload, timeout=self.timeout)
            else:
                response = requests.get(url, params=payload, timeout=self.timeout)
            response.raise_for_status()
            data = response.json()
        except requests.Timeout as exc:
            raise RuntimeError(f"{endpoint} timed out after {self.timeout:g}s") from exc
        except requests.RequestException as exc:
            detail = ""
            if exc.response is not None:
                detail = f": {exc.response.text[:500]}"
            raise RuntimeError(f"{endpoint} request failed{detail}") from exc
        except ValueError as exc:
            raise RuntimeError(f"{endpoint} returned invalid JSON") from exc

        envelope = data.get("subsonic-response") if isinstance(data, dict) else None
        if isinstance(envelope, dict) and envelope.get("status") == "failed":
            error = envelope.get("error") or {}
            raise RuntimeError(
                f"{endpoint} protocol error {error.get('code', '?')}: "
                f"{error.get('message', 'unknown error')}"
            )
        return data

    def ping(self):
        return self.request("ping")

    def get_indexes(self):
        return self.request("getIndexes")

    def get_artists(self):
        return self.request("getArtists")

    def get_artist(self, artist_id):
        return self.request("getArtist", {"id": artist_id})

    def get_music_directory(self, directory_id):
        return self.request("getMusicDirectory", {"id": directory_id})

    def get_album(self, album_id):
        return self.request("getAlbum", {"id": album_id})

    def get_album_list(self, list_type="newest", size=10, offset=0, version2=False):
        endpoint = "getAlbumList2" if version2 else "getAlbumList"
        return self.request(endpoint, {
            "type": list_type,
            "size": str(size),
            "offset": str(offset),
        })

    def search(self, query, artist_count=20, album_count=20, song_count=20):
        return self.request("search3", {
            "query": query,
            "artistCount": str(artist_count),
            "albumCount": str(album_count),
            "songCount": str(song_count),
        })

    def get_playlists(self):
        return self.request("getPlaylists")

    def get_playlist(self, playlist_id):
        return self.request("getPlaylist", {"id": playlist_id})

    def create_playlist(self, name, song_ids=None):
        params = {"name": name}
        if song_ids:
            params["songId"] = song_ids
        return self.request("createPlaylist", params, method="POST")

    def update_playlist(self, playlist_id, name=None, add=None, remove_indexes=None):
        params = {"playlistId": playlist_id}
        if name is not None:
            params["name"] = name
        if add:
            params["songIdToAdd"] = add
        if remove_indexes:
            params["songIndexToRemove"] = [str(i) for i in remove_indexes]
        return self.request("updatePlaylist", params, method="POST")

    def delete_playlist(self, playlist_id):
        return self.request("deletePlaylist", {"id": playlist_id}, method="POST")

    def get_starred(self):
        return self.request("getStarred")

    def star(self, track_ids=None, album_ids=None, artist_ids=None):
        params = {}
        if track_ids:
            params["id"] = track_ids
        if album_ids:
            params["albumId"] = album_ids
        if artist_ids:
            params["artistId"] = artist_ids
        return self.request("star", params, method="POST")

    def unstar(self, track_ids=None, album_ids=None, artist_ids=None):
        params = {}
        if track_ids:
            params["id"] = track_ids
        if album_ids:
            params["albumId"] = album_ids
        if artist_ids:
            params["artistId"] = artist_ids
        return self.request("unstar", params, method="POST")

    def media_url(self, endpoint, item_id, **extra):
        params = self._auth_params()
        params.pop("f", None)
        params["id"] = item_id
        for key, value in extra.items():
            if value is not None:
                params[key] = str(value)
        return f"{self.base_url}/{endpoint}?{urllib.parse.urlencode(params)}"


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--timeout", type=float, default=DEFAULT_TIMEOUT)
    parser.add_argument("username")
    parser.add_argument("password")
    parser.add_argument("endpoint")
    parser.add_argument("args", nargs="*")
    return parser


def require_arg(args, index, label):
    try:
        return args[index]
    except IndexError as exc:
        raise SystemExit(f"missing required argument: {label}") from exc


def main():
    ns = build_parser().parse_args()
    client = SubsonicClient(ns.base_url, ns.username, ns.password, ns.timeout)
    ep, args = ns.endpoint, ns.args

    if ep == "ping":
        result = client.ping()
    elif ep == "getIndexes":
        result = client.get_indexes()
    elif ep == "getArtists":
        result = client.get_artists()
    elif ep == "getArtist":
        result = client.get_artist(require_arg(args, 0, "artist id"))
    elif ep == "getMusicDirectory":
        result = client.get_music_directory(require_arg(args, 0, "directory id"))
    elif ep == "getAlbum":
        result = client.get_album(require_arg(args, 0, "album id"))
    elif ep in ("getAlbumList", "getAlbumList2"):
        list_type = args[0] if args else "newest"
        size = int(args[1]) if len(args) > 1 else 10
        offset = int(args[2]) if len(args) > 2 else 0
        result = client.get_album_list(list_type, size, offset, ep.endswith("2"))
    elif ep == "search3":
        result = client.search(require_arg(args, 0, "query"))
    elif ep == "getPlaylists":
        result = client.get_playlists()
    elif ep == "getPlaylist":
        result = client.get_playlist(require_arg(args, 0, "playlist id"))
    elif ep == "createPlaylist":
        result = client.create_playlist(
            require_arg(args, 0, "playlist name"),
            args[1:] or None,
        )
    elif ep == "updatePlaylist":
        result = client.update_playlist(
            require_arg(args, 0, "playlist id"),
            args[1] if len(args) > 1 else None,
        )
    elif ep == "deletePlaylist":
        result = client.delete_playlist(require_arg(args, 0, "playlist id"))
    elif ep == "getStarred":
        result = client.get_starred()
    elif ep == "star":
        result = client.star(track_ids=args)
    elif ep == "unstar":
        result = client.unstar(track_ids=args)
    elif ep == "getCoverArt":
        print(client.media_url("getCoverArt", require_arg(args, 0, "id")))
        return
    elif ep == "stream":
        item_id = require_arg(args, 0, "id")
        fmt = args[1] if len(args) > 1 else None
        print(client.media_url("stream", item_id, format=fmt))
        return
    else:
        raise SystemExit(f"unknown endpoint: {ep}")

    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    try:
        main()
    except RuntimeError as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(2)
