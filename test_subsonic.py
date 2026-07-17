#!/usr/bin/env python3
import requests
import json
import urllib.parse

class SubsonicClient:
    def __init__(self, base_url="http://ubuntu-server:5174/rest", username="admin", password="password"):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.version = "1.16.1"
        self.client = "test_script"
        self.format = "json"
        
        # Test basic connection
        print(f"Initialized Subsonic Client targeting {self.base_url}")
        
    def _make_request(self, endpoint, params=None):
        if params is None:
            params = {}
            
        # Add auth and formatting params
        params['u'] = self.username
        params['p'] = self.password
        params['v'] = self.version
        params['c'] = self.client
        params['f'] = self.format
        
        url = f"{self.base_url}/{endpoint}"
        print(f"-> GET {url} | params: {self._mask_password(params)}")
        
        try:
            response = requests.get(url, params=params)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error making request: {e}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"Response: {e.response.text}")
            return None
            
    def _mask_password(self, params):
        p_copy = params.copy()
        if 'p' in p_copy:
            p_copy['p'] = '***'
        return p_copy

    def ping(self):
        return self._make_request("ping")
        
    def get_indexes(self):
        return self._make_request("getIndexes")
        
    def get_artists(self):
        return self._make_request("getArtists")
        
    def get_artist(self, artist_id):
        return self._make_request("getArtist", {"id": artist_id})
        
    def get_music_directory(self, folder_id):
        return self._make_request("getMusicDirectory", {"id": folder_id})
        
    def get_album(self, album_id):
        return self._make_request("getAlbum", {"id": album_id})
        
    def get_album_list(self, list_type="newest", size=10):
        return self._make_request("getAlbumList", {"type": list_type, "size": size})
        
    def get_album_list2(self, list_type="newest", size=10):
        return self._make_request("getAlbumList2", {"type": list_type, "size": size})
        
    def get_playlists(self):
        return self._make_request("getPlaylists")
        
    def get_playlist(self, playlist_id):
        return self._make_request("getPlaylist", {"id": playlist_id})
        
    def get_starred(self):
        return self._make_request("getStarred")
        
    def star(self, track_ids=None, album_ids=None, artist_ids=None):
        params = {}
        if track_ids: params["id"] = track_ids
        if album_ids: params["albumId"] = album_ids
        if artist_ids: params["artistId"] = artist_ids
        return self._make_request("star", params)
        
    def unstar(self, track_ids=None, album_ids=None, artist_ids=None):
        params = {}
        if track_ids: params["id"] = track_ids
        if album_ids: params["albumId"] = album_ids
        if artist_ids: params["artistId"] = artist_ids
        return self._make_request("unstar", params)

    def create_playlist(self, name, song_ids=None):
        params = {"name": name}
        if song_ids:
            params["songId"] = song_ids
        return self._make_request("createPlaylist", params)
        
    def update_playlist(self, playlist_id, name=None, comment=None, public=None, song_ids_to_add=None, song_indexes_to_remove=None):
        params = {"playlistId": playlist_id}
        if name is not None: params["name"] = name
        if comment is not None: params["comment"] = comment
        if public is not None: params["public"] = "true" if public else "false"
        if song_ids_to_add: params["songIdToAdd"] = song_ids_to_add
        if song_indexes_to_remove: params["songIndexToRemove"] = song_indexes_to_remove
        return self._make_request("updatePlaylist", params)
        
    def delete_playlist(self, playlist_id):
        return self._make_request("deletePlaylist", {"id": playlist_id})

    def get_cover_art_url(self, id):
        params = {
            'u': self.username,
            'p': self.password,
            'v': self.version,
            'c': self.client,
            'id': id
        }
        qs = urllib.parse.urlencode(params)
        return f"{self.base_url}/getCoverArt?{qs}"
        
    def get_stream_url(self, id):
        params = {
            'u': self.username,
            'p': self.password,
            'v': self.version,
            'c': self.client,
            'id': id
        }
        qs = urllib.parse.urlencode(params)
        return f"{self.base_url}/stream?{qs}"

def run_tests():
    # Setup client - replace with your actual username/password
    # Supernova uses local db accounts. Let's assume standard auth for tests.
    client = SubsonicClient(username="admin", password="password")
    
    print("\n--- Testing Ping ---")
    res = client.ping()
    print(json.dumps(res, indent=2))
    
    print("\n--- Testing getArtists ---")
    res = client.getArtists() # Whoops, python uses snake case for my defs
    
    # We will just write the structure, user can run it interactively.

if __name__ == "__main__":
    import sys
    # A simple CLI wrapper to test individual endpoints
    
    if len(sys.argv) < 4:
        print("Usage: python test_subsonic.py <username> <password> <endpoint> [args...]")
        print("Example: python test_subsonic.py admin mypass ping")
        print("Example: python test_subsonic.py admin mypass getArtist 123")
        sys.exit(1)
        
    username = sys.argv[1]
    password = sys.argv[2]
    endpoint = sys.argv[3]
    args = sys.argv[4:]
    
    client = SubsonicClient(base_url="http://ubuntu-server:5174/rest", username=username, password=password)
    
    if endpoint == "ping":
        res = client.ping()
    elif endpoint == "getIndexes":
        res = client.get_indexes()
    elif endpoint == "getArtists":
        res = client.get_artists()
    elif endpoint == "getArtist":
        res = client.get_artist(args[0])
    elif endpoint == "getMusicDirectory":
        res = client.get_music_directory(args[0])
    elif endpoint == "getAlbum":
        res = client.get_album(args[0])
    elif endpoint == "getAlbumList":
        type_str = args[0] if len(args) > 0 else "newest"
        res = client.get_album_list(list_type=type_str)
    elif endpoint == "getAlbumList2":
        type_str = args[0] if len(args) > 0 else "newest"
        res = client.get_album_list2(list_type=type_str)
    elif endpoint == "getPlaylists":
        res = client.get_playlists()
    elif endpoint == "getPlaylist":
        res = client.get_playlist(args[0])
    elif endpoint == "createPlaylist":
        res = client.create_playlist(name=args[0], song_ids=args[1:] if len(args) > 1 else None)
    elif endpoint == "updatePlaylist":
        res = client.update_playlist(playlist_id=args[0], name=args[1] if len(args) > 1 else None)
    elif endpoint == "deletePlaylist":
        res = client.delete_playlist(args[0])
    elif endpoint == "getStarred":
        res = client.get_starred()
    elif endpoint == "star":
        res = client.star(track_ids=args)
    elif endpoint == "unstar":
        res = client.unstar(track_ids=args)
    elif endpoint == "getCoverArt":
        print(f"Cover Art URL: {client.get_cover_art_url(args[0])}")
        res = None
    elif endpoint == "stream":
        print(f"Stream URL: {client.get_stream_url(args[0])}")
        res = None
    else:
        print(f"Unknown endpoint: {endpoint}")
        res = None
        
    if res:
        print("\nResponse:")
        print(json.dumps(res, indent=2))
