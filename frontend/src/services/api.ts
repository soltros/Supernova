import type { Album, Artist, Track, AuthResponse } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const getHeaders = () => {
  const token = localStorage.getItem('sn_token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { 'Authorization': `Bearer ${token}` } : {})
  };
};

export const apiService = {
  // Auth
  register: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await fetch(`${API_BASE_URL}/api/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    if (!response.ok) throw new Error('Failed to register');
    return response.json();
  },

  login: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    if (!response.ok) throw new Error('Failed to login');
    return response.json();
  },
  fetchAlbums: async (limit: number = 50, offset: number = 0): Promise<Album[]> => {
    const response = await fetch(`${API_BASE_URL}/api/albums?limit=${limit}&offset=${offset}`);
    if (!response.ok) throw new Error('Failed to fetch albums');
    return response.json();
  },

  fetchAlbumById: async (id: string): Promise<Album> => {
    const response = await fetch(`${API_BASE_URL}/api/albums/${id}`);
    if (!response.ok) throw new Error('Failed to fetch album');
    return response.json();
  },

  fetchArtists: async (limit: number = 50, offset: number = 0): Promise<Artist[]> => {
    const response = await fetch(`${API_BASE_URL}/api/artists?limit=${limit}&offset=${offset}`);
    if (!response.ok) throw new Error('Failed to fetch artists');
    return response.json();
  },

  fetchTracks: async (albumId: string, limit: number = 50, offset: number = 0): Promise<Track[]> => {
    const response = await fetch(`${API_BASE_URL}/api/tracks?album_id=${albumId}&limit=${limit}&offset=${offset}`);
    if (!response.ok) throw new Error('Failed to fetch tracks');
    return response.json();
  },

  // Hearts API
  fetchHearts: async (): Promise<{ entity_id: string }[]> => {
    const response = await fetch(`${API_BASE_URL}/api/hearts`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to fetch hearts');
    return response.json();
  },

  addHeart: async (entityType: string, entityId: string): Promise<void> => {
    const response = await fetch(`${API_BASE_URL}/api/hearts`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ entity_type: entityType, entity_id: entityId })
    });
    if (!response.ok) throw new Error('Failed to add heart');
  },

  removeHeart: async (entityType: string, entityId: string): Promise<void> => {
    const response = await fetch(`${API_BASE_URL}/api/hearts?entity_type=${entityType}&entity_id=${entityId}`, {
      method: 'DELETE',
      headers: getHeaders()
    });
    if (!response.ok) throw new Error('Failed to remove heart');
  },
  
  importHearts: async (file: File): Promise<void> => {
    const response = await fetch(`${API_BASE_URL}/api/hearts/import`, {
      method: 'POST',
      headers: getHeaders(),
      body: await file.text()
    });
    if (!response.ok) throw new Error('Failed to import hearts');
  },

  // Scrobbling
  scrobbleTrack: async (trackId: string): Promise<void> => {
    const response = await fetch(`${API_BASE_URL}/api/scrobbles`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ track_id: trackId })
    });
    if (!response.ok) throw new Error('Failed to scrobble track');
  },
  
  getRecentScrobbles: async (): Promise<Track[]> => {
    const response = await fetch(`${API_BASE_URL}/api/scrobbles/recent`, { headers: getHeaders() });
    if (!response.ok) throw new Error('Failed to fetch recent scrobbles');
    return response.json();
  }
};
