import React, { useEffect, useState, useRef } from 'react';
import { apiService } from '../services/api';
import type { Playlist, Track, Album } from '../types';
import { usePlayer } from '../context/PlayerContext';

export const PlaylistsPage: React.FC = () => {
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [selectedPlaylist, setSelectedPlaylist] = useState<Playlist | null>(null);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [newPlaylistName, setNewPlaylistName] = useState('');
  const [loading, setLoading] = useState(true);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { playContext } = usePlayer();

  const loadPlaylists = async () => {
    try {
      const data = await apiService.fetchPlaylists();
      setPlaylists(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPlaylists();
  }, []);

  const handleCreatePlaylist = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPlaylistName.trim()) return;
    try {
      await apiService.createPlaylist(newPlaylistName.trim());
      setNewPlaylistName('');
      loadPlaylists();
    } catch (err) {
      console.error(err);
    }
  };

  const handleSelectPlaylist = async (p: Playlist) => {
    setSelectedPlaylist(p);
    try {
      const t = await apiService.fetchPlaylistTracks(p.id);
      setTracks(t || []);
    } catch (err) {
      console.error(err);
    }
  };

  const handleDeletePlaylist = async (id: string) => {
    try {
      await apiService.deletePlaylist(id);
      if (selectedPlaylist?.id === id) {
        setSelectedPlaylist(null);
        setTracks([]);
      }
      loadPlaylists();
    } catch (err) {
      console.error(err);
    }
  };

  const handleRemoveTrack = async (trackId: string) => {
    if (!selectedPlaylist) return;
    try {
      await apiService.removeTrackFromPlaylist(selectedPlaylist.id, trackId);
      setTracks(tracks.filter(t => t.id !== trackId));
    } catch (err) {
      console.error(err);
    }
  };

  const handleExport = async () => {
    try {
      await apiService.exportPlaylists();
    } catch (err) {
      console.error(err);
    }
  };

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await apiService.importPlaylists(file);
      loadPlaylists();
    } catch (err) {
      console.error(err);
    }
  };

  if (loading) {
    return <div className="content-scroll"><h2 className="section-title">Loading...</h2></div>;
  }

  return (
    <div className="content-scroll" style={{ display: 'flex', gap: '2rem', padding: '2rem' }}>
      <div style={{ flex: '1', minWidth: '300px' }}>
        <h2 className="section-title" style={{ marginBottom: '1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          Your Playlists
          <div>
            <button onClick={handleExport} style={{ background: 'transparent', border: '1px solid #1db954', color: '#1db954', padding: '0.5rem 1rem', borderRadius: '4px', cursor: 'pointer', marginRight: '0.5rem' }}>
              Export
            </button>
            <button onClick={() => fileInputRef.current?.click()} style={{ background: 'transparent', border: '1px solid #1db954', color: '#1db954', padding: '0.5rem 1rem', borderRadius: '4px', cursor: 'pointer' }}>
              Import
            </button>
            <input type="file" ref={fileInputRef} onChange={handleImport} style={{ display: 'none' }} accept=".json" />
          </div>
        </h2>
        
        <form onSubmit={handleCreatePlaylist} style={{ display: 'flex', gap: '0.5rem', marginBottom: '1.5rem' }}>
          <input 
            type="text" 
            placeholder="New Playlist Name" 
            value={newPlaylistName}
            onChange={(e) => setNewPlaylistName(e.target.value)}
            style={{ flex: '1', padding: '0.75rem', borderRadius: '4px', border: 'none', backgroundColor: '#2d2d2d', color: 'white' }}
          />
          <button type="submit" style={{ padding: '0.75rem 1rem', borderRadius: '4px', border: 'none', backgroundColor: '#1db954', color: 'white', fontWeight: 'bold', cursor: 'pointer' }}>
            Create
          </button>
        </form>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          {playlists.map(p => (
            <div 
              key={p.id} 
              style={{ 
                display: 'flex', justifyContent: 'space-between', alignItems: 'center', 
                padding: '1rem', borderRadius: '4px', cursor: 'pointer',
                backgroundColor: selectedPlaylist?.id === p.id ? '#282828' : '#181818'
              }}
              onClick={() => handleSelectPlaylist(p)}
            >
              <span style={{ fontWeight: 'bold', color: 'white' }}>{p.name}</span>
              <button 
                onClick={(e) => { e.stopPropagation(); handleDeletePlaylist(p.id); }}
                style={{ background: 'transparent', border: 'none', color: '#ff4444', cursor: 'pointer' }}
              >
                Delete
              </button>
            </div>
          ))}
          {playlists.length === 0 && <p style={{ color: '#b3b3b3' }}>No playlists found.</p>}
        </div>
      </div>

      <div style={{ flex: '2' }}>
        {selectedPlaylist ? (
          <div>
            <h2 className="section-title" style={{ marginBottom: '1rem' }}>{selectedPlaylist.name}</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {tracks.map((track, index) => (
                <div key={track.id + index} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0.75rem', backgroundColor: '#181818', borderRadius: '4px' }}>
                  <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
                    <span style={{ color: '#b3b3b3', width: '24px' }}>{index + 1}</span>
                    <span 
                      style={{ color: 'white', fontWeight: 'bold', cursor: 'pointer' }}
                      onClick={() => playContext(tracks, index, { id: selectedPlaylist.id, title: selectedPlaylist.name, artist_id: '', release_year: 0, cover_art_path: '' } as Album)}
                    >
                      {track.title}
                    </span>
                  </div>
                  <button 
                    onClick={() => handleRemoveTrack(track.id)}
                    style={{ background: 'transparent', border: 'none', color: '#ff4444', cursor: 'pointer' }}
                  >
                    Remove
                  </button>
                </div>
              ))}
              {tracks.length === 0 && <p style={{ color: '#b3b3b3' }}>This playlist is empty.</p>}
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', color: '#b3b3b3' }}>
            Select a playlist to view tracks
          </div>
        )}
      </div>
    </div>
  );
};
