import type { Album, Artist, Track, AuthResponse, Playlist } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const getHeaders = () => {
  const token = localStorage.getItem('sn_token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { 'Authorization': `Bearer ${token}` } : {})
  };
};

const fetchWithAuth = async (url: string, options: RequestInit = {}) => {
  const response = await fetch(url, options);
  if (response.status === 401) {
    // Phantom User fix: If the backend explicitly rejects our token, wipe state and reload
    localStorage.removeItem('sn_user');
    localStorage.removeItem('sn_token');
    window.location.reload();
  }
  return response;
};

export const apiService = {
  // Auth
  register: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    if (!response.ok) throw new Error('Failed to register');
    return response.json();
  },

  login: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    if (!response.ok) throw new Error('Failed to login');
    return response.json();
  },
  fetchAlbums: async (limit: number = 50, offset: number = 0, artistId?: string): Promise<Album[]> => {
    let url = `${API_BASE_URL}/api/albums?limit=${limit}&offset=${offset}`;
    if (artistId) url += `&artist_id=${artistId}`;
    const response = await fetchWithAuth(url);
    if (!response.ok) throw new Error('Failed to fetch albums');
    return response.json();
  },

  fetchAlbumById: async (id: string): Promise<Album> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/albums/${id}`);
    if (!response.ok) throw new Error('Failed to fetch album');
    return response.json();
  },

  fetchArtists: async (limit: number = 50, offset: number = 0): Promise<Artist[]> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/artists?limit=${limit}&offset=${offset}`);
    if (!response.ok) throw new Error('Failed to fetch artists');
    return response.json();
  },

  fetchArtistById: async (id: string): Promise<Artist> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/artists/${id}`);
    if (!response.ok) throw new Error('Failed to fetch artist');
    return response.json();
  },

  fetchTracks: async (albumId?: string, limit: number = 50, offset: number = 0, artistId?: string): Promise<Track[]> => {
    let url = `${API_BASE_URL}/api/tracks?limit=${limit}&offset=${offset}`;
    if (albumId) url += `&album_id=${albumId}`;
    if (artistId) url += `&artist_id=${artistId}`;
    const response = await fetchWithAuth(url);
    if (!response.ok) throw new Error('Failed to fetch tracks');
    return response.json();
  },

  // Hearts API
  fetchHearts: async (): Promise<{ entity_id: string }[]> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/hearts`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to fetch hearts');
    return response.json();
  },

  addHeart: async (entityType: string, entityId: string): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/hearts`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ entity_type: entityType, entity_id: entityId })
    });
    if (!response.ok) throw new Error('Failed to add heart');
  },

  removeHeart: async (entityType: string, entityId: string): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/hearts?entity_type=${entityType}&entity_id=${entityId}`, {
      method: 'DELETE',
      headers: getHeaders()
    });
    if (!response.ok) throw new Error('Failed to remove heart');
  },
  
  importHearts: async (file: File): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/hearts/import`, {
      method: 'POST',
      headers: getHeaders(),
      body: await file.text()
    });
    if (!response.ok) throw new Error('Failed to import hearts');
  },

  // Scrobbling
  scrobbleTrack: async (trackId: string): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/scrobbles`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ track_id: trackId })
    });
    if (!response.ok) throw new Error('Failed to scrobble track');
  },
  
  getRecentScrobbles: async (): Promise<Track[]> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/scrobbles/recent`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to fetch recent scrobbles');
    return response.json();
  },

  // Playlists API
  fetchPlaylists: async (): Promise<Playlist[]> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to fetch playlists');
    return response.json();
  },

  createPlaylist: async (name: string): Promise<Playlist> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ name })
    });
    if (!response.ok) throw new Error('Failed to create playlist');
    return response.json();
  },

  deletePlaylist: async (id: string): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists/${id}`, {
      method: 'DELETE',
      headers: getHeaders()
    });
    if (!response.ok) throw new Error('Failed to delete playlist');
  },

  fetchPlaylistTracks: async (id: string): Promise<Track[]> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists/${id}/tracks`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to fetch playlist tracks');
    return response.json();
  },

  addTrackToPlaylist: async (playlistId: string, trackId: string): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists/${playlistId}/tracks`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ track_id: trackId })
    });
    if (!response.ok) throw new Error('Failed to add track to playlist');
  },

  removeTrackFromPlaylist: async (playlistId: string, trackId: string): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists/${playlistId}/tracks/${trackId}`, {
      method: 'DELETE',
      headers: getHeaders()
    });
    if (!response.ok) throw new Error('Failed to remove track from playlist');
  },

  exportPlaylists: async (): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists/export`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to export playlists');
    
    // Trigger download
    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'supernova_playlists_backup.json';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },

  importPlaylists: async (file: File): Promise<void> => {
    const response = await fetchWithAuth(`${API_BASE_URL}/api/playlists/import`, {
      method: 'POST',
      headers: getHeaders(),
      body: await file.text()
    });
    if (!response.ok) throw new Error('Failed to import playlists');
  }
};
