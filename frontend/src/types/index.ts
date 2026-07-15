export interface Album {
  id: string;
  title: string;
  release_year: number;
  cover_art_path: string;
  artist_id?: string;
  artist_name?: string;
  cover_art_url?: string;
  bio?: string;
}

export interface Artist {
  id: string;
  name: string;
  musicbrainz_id: string;
  image_url: string;
  bio: string;
}

export interface Track {
  id: string;
  album_id: string;
  title: string;
  track_number: number;
  disc_number: number;
  duration_ms: number;
  format: string;
  bitrate: number;
  artist_id?: string;
  artist_name?: string;
  album_title?: string;
  stream_url?: string;
}

export interface User {
  id: string;
  username: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface Playlist {
  id: string;
  user_id: string;
  name: string;
  created_at: string;
}

export interface PlaylistBackup {
  name: string;
  created_at: string;
  tracks: string[];
}
