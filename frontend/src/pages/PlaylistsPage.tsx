import React, { useEffect, useState, useRef } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ListMusic, Trash2, Play } from 'lucide-react';
import { apiService } from '../services/api';
import type { Playlist, Track, Album } from '../types';
import { usePlayer } from '../context/PlayerContext';
import { usePlaylists } from '../context/PlaylistsContext';
import HeartButton from '../components/HeartButton';

export const PlaylistsPage: React.FC = () => {
  const { playlists, createPlaylist, deletePlaylist, refreshPlaylists } = usePlaylists();
  const [selectedPlaylist, setSelectedPlaylist] = useState<Playlist | null>(null);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [newPlaylistName, setNewPlaylistName] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const loading = false;
  const { playContext } = usePlayer();
  const location = useLocation();

  useEffect(() => {
    const state = location.state as { selectedPlaylistId?: string };
    if (state?.selectedPlaylistId) {
      const playlist = playlists.find(p => p.id === state.selectedPlaylistId);
      if (playlist) {
        handleSelectPlaylist(playlist);
      }
    }
  }, [location.state, playlists]);

  const handleCreatePlaylist = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPlaylistName.trim()) return;
    try {
      await createPlaylist(newPlaylistName.trim());
      setNewPlaylistName('');
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
      await deletePlaylist(id);
      if (selectedPlaylist?.id === id) {
        setSelectedPlaylist(null);
        setTracks([]);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleRemoveTrack = async (trackId: string) => {
    if (!selectedPlaylist) return;
    try {
      await apiService.removeTrackFromPlaylist(selectedPlaylist.id, trackId);
      setTracks(prevTracks => prevTracks.filter(t => t.id !== trackId));
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
      await refreshPlaylists();
      alert('Playlists imported successfully!');
    } catch (err) {
      console.error(err);
      alert('Failed to import playlists.');
    }
  };

  if (loading) {
    return <div className="content-scroll"><h2 className="section-title">Loading...</h2></div>;
  }

  return (
    <div className="content-scroll" style={{ display: 'flex', gap: '32px', padding: '32px' }}>
      <div style={{ flex: '1', minWidth: '320px', maxWidth: '400px' }}>
        <h2 className="section-title" style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          Your Playlists
          <div style={{ display: 'flex', gap: '12px' }}>
            <button onClick={handleExport} style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-glass-bright)', color: 'var(--text-primary)', padding: '6px 12px', borderRadius: '8px', cursor: 'pointer', fontSize: '13px', fontWeight: 600 }}>
              Export
            </button>
            <button onClick={() => fileInputRef.current?.click()} style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-glass-bright)', color: 'var(--text-primary)', padding: '6px 12px', borderRadius: '8px', cursor: 'pointer', fontSize: '13px', fontWeight: 600 }}>
              Import
            </button>
            <input type="file" ref={fileInputRef} onChange={handleImport} style={{ display: 'none' }} accept=".json" />
          </div>
        </h2>
        
        <form onSubmit={handleCreatePlaylist} style={{ display: 'flex', gap: '12px', marginBottom: '32px' }}>
          <input 
            type="text" 
            placeholder="New Playlist Name" 
            value={newPlaylistName}
            onChange={(e) => setNewPlaylistName(e.target.value)}
            style={{ flex: '1', padding: '12px 16px', borderRadius: '12px', border: '1px solid var(--border-glass-bright)', backgroundColor: 'var(--bg-glass)', color: 'white', outline: 'none', fontFamily: 'Outfit' }}
          />
          <button type="submit" style={{ padding: '0 24px', borderRadius: '12px', border: 'none', background: 'var(--accent-gradient)', color: 'white', fontWeight: 'bold', cursor: 'pointer', boxShadow: 'var(--accent-glow)' }}>
            Create
          </button>
        </form>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {playlists.map(p => {
            const isActive = selectedPlaylist?.id === p.id;
            return (
              <div 
                key={p.id} 
                style={{ 
                  display: 'flex', justifyContent: 'space-between', alignItems: 'center', 
                  padding: '16px', borderRadius: '12px', cursor: 'pointer',
                  background: isActive ? 'var(--accent-primary)' : 'var(--bg-glass)',
                  border: '1px solid',
                  borderColor: isActive ? 'transparent' : 'var(--border-glass)',
                  boxShadow: isActive ? 'var(--accent-glow)' : 'none',
                  transition: 'var(--transition-fast)'
                }}
                onClick={() => handleSelectPlaylist(p)}
                onMouseEnter={(e) => { if (!isActive) e.currentTarget.style.background = 'var(--bg-glass-hover)' }}
                onMouseLeave={(e) => { if (!isActive) e.currentTarget.style.background = 'var(--bg-glass)' }}
              >
                <span style={{ fontWeight: 600, color: 'white' }}>{p.name}</span>
                <button 
                  onClick={(e) => { e.stopPropagation(); handleDeletePlaylist(p.id); }}
                  style={{ background: 'rgba(255, 68, 68, 0.1)', border: '1px solid rgba(255, 68, 68, 0.2)', color: '#ff4444', cursor: 'pointer', padding: '4px 12px', borderRadius: '6px', fontSize: '12px', fontWeight: 600 }}
                >
                  Delete
                </button>
              </div>
            );
          })}
          {playlists.length === 0 && <p style={{ color: 'var(--text-secondary)' }}>No playlists found.</p>}
        </div>
      </div>

      <div style={{ flex: '2', background: 'var(--bg-glass)', borderRadius: '24px', border: '1px solid var(--border-glass)', padding: '32px', boxShadow: 'var(--shadow-drop)' }}>
        {selectedPlaylist ? (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: '32px' }}>
              <div>
                <span style={{ textTransform: 'uppercase', fontSize: '12px', fontWeight: 800, letterSpacing: '1px', color: 'var(--text-secondary)' }}>Playlist</span>
                <h2 style={{ fontSize: '42px', fontWeight: 900, margin: '8px 0 0 0', letterSpacing: '-1px' }}>{selectedPlaylist.name}</h2>
              </div>
              <div style={{ display: 'flex', gap: '16px', alignItems: 'center' }}>
                <HeartButton entityType="playlist" entityId={selectedPlaylist.id} />
                <button 
                  style={{ width: '56px', height: '56px', borderRadius: '50%', background: 'var(--accent-primary)', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: 'var(--accent-glow)', cursor: 'pointer', border: 'none' }}
                  onClick={() => tracks.length > 0 && playContext(tracks, 0, { id: selectedPlaylist.id, title: selectedPlaylist.name, artist_id: '', release_year: 0, cover_art_path: '' } as Album)}
                >
                  <Play size={24} fill="currentColor" />
                </button>
              </div>
            </div>
            
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              {tracks.map((track, index) => (
                <div 
                  key={track.id + index} 
                  className="track-row"
                  style={{ 
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center', 
                    padding: '12px 16px', backgroundColor: 'transparent', borderRadius: '8px',
                    transition: 'background 0.2s'
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'var(--border-glass-bright)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                >
                  <div style={{ display: 'flex', gap: '16px', alignItems: 'center' }}>
                    <span style={{ color: 'var(--text-secondary)', width: '24px', fontWeight: 600 }}>{index + 1}</span>
                    <div style={{ display: 'flex', flexDirection: 'column' }}>
                      <span 
                        style={{ color: 'white', fontWeight: 600, cursor: 'pointer' }}
                        onClick={() => playContext(tracks, index, { id: selectedPlaylist.id, title: selectedPlaylist.name, artist_id: '', release_year: 0, cover_art_path: '' } as Album)}
                      >
                        {track.title}
                      </span>
                      {track.artist_id && (
                        <Link 
                          to={`/artist/${track.artist_id}`}
                          style={{ fontSize: '12px', color: 'var(--text-muted)', textDecoration: 'none', transition: 'color 0.2s' }}
                          onClick={(e) => e.stopPropagation()}
                          onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'}
                          onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-muted)'}
                        >
                          {track.artist_name}
                        </Link>
                      )}
                    </div>
                  </div>
                  <button 
                    onClick={() => handleRemoveTrack(track.id)}
                    style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', fontWeight: 600, fontSize: '13px' }}
                    onMouseEnter={(e) => e.currentTarget.style.color = '#ff4444'}
                    onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-secondary)'}
                    title="Remove from playlist"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              ))}
              {tracks.length === 0 && <p style={{ color: 'var(--text-secondary)', textAlign: 'center', marginTop: '40px' }}>This playlist is empty.</p>}
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', height: '100%', color: 'var(--text-secondary)' }}>
            <div style={{ marginBottom: '16px', opacity: 0.2 }}>
              <ListMusic size={64} />
            </div>
            <h3 style={{ fontSize: '24px', fontWeight: 600, color: 'var(--text-primary)' }}>Select a playlist</h3>
            <p>Choose a playlist from the sidebar to view its tracks</p>
          </div>
        )}
      </div>
    </div>
  );
};
