import React, { createContext, useState, useContext, useEffect, useCallback } from 'react';
import type { FC, ReactNode } from 'react';
import { apiService } from '../services/api';
import type { Playlist } from '../types';

interface PlaylistsContextType {
  playlists: Playlist[];
  refreshPlaylists: () => Promise<void>;
  createPlaylist: (name: string) => Promise<Playlist>;
  deletePlaylist: (id: string) => Promise<void>;
}

const PlaylistsContext = createContext<PlaylistsContextType | undefined>(undefined);

export const PlaylistsProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [playlists, setPlaylists] = useState<Playlist[]>([]);

  const refreshPlaylists = useCallback(async () => {
    try {
      const data = await apiService.fetchPlaylists();
      setPlaylists(data || []);
    } catch (err) {
      console.error("Failed to fetch playlists:", err);
    }
  }, []);

  useEffect(() => {
    refreshPlaylists();
  }, [refreshPlaylists]);

  const createPlaylist = useCallback(async (name: string) => {
    const newPlaylist = await apiService.createPlaylist(name);
    await refreshPlaylists();
    return newPlaylist;
  }, [refreshPlaylists]);

  const deletePlaylist = useCallback(async (id: string) => {
    await apiService.deletePlaylist(id);
    await refreshPlaylists();
  }, [refreshPlaylists]);

  const value = React.useMemo(() => ({
    playlists, refreshPlaylists, createPlaylist, deletePlaylist
  }), [playlists, refreshPlaylists, createPlaylist, deletePlaylist]);

  return (
    <PlaylistsContext.Provider value={value}>
      {children}
    </PlaylistsContext.Provider>
  );
};

export const usePlaylists = () => {
  const context = useContext(PlaylistsContext);
  if (context === undefined) {
    throw new Error('usePlaylists must be used within a PlaylistsProvider');
  }
  return context;
};
