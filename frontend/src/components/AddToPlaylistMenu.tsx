import React, { useState, useEffect, useRef } from 'react';
import { apiService } from '../services/api';
import type { Playlist } from '../types';

interface AddToPlaylistMenuProps {
  trackId: string;
}

const AddToPlaylistMenu: React.FC<AddToPlaylistMenuProps> = ({ trackId }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) {
      apiService.fetchPlaylists().then(data => setPlaylists(data || []));
    }
  }, [isOpen]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  const handleAdd = async (e: React.MouseEvent, playlistId: string) => {
    e.stopPropagation();
    try {
      await apiService.addTrackToPlaylist(playlistId, trackId);
      setIsOpen(false);
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div style={{ position: 'relative' }} ref={menuRef}>
      <button 
        onClick={(e) => { e.stopPropagation(); setIsOpen(!isOpen); }}
        style={{
          background: 'transparent',
          border: 'none',
          color: 'var(--text-muted)',
          cursor: 'pointer',
          fontSize: '18px',
          padding: '0 8px'
        }}
        title="Add to Playlist"
      >
        +
      </button>

      {isOpen && (
        <div style={{
          position: 'absolute',
          right: 0,
          top: '100%',
          backgroundColor: '#282828',
          border: '1px solid #404040',
          borderRadius: '4px',
          boxShadow: '0 4px 12px rgba(0,0,0,0.5)',
          zIndex: 100,
          minWidth: '150px',
          maxHeight: '200px',
          overflowY: 'auto'
        }}>
          {playlists.length === 0 ? (
            <div style={{ padding: '8px 12px', color: 'var(--text-muted)', fontSize: '13px' }}>
              No playlists found
            </div>
          ) : (
            playlists.map(p => (
              <div
                key={p.id}
                onClick={(e) => handleAdd(e, p.id)}
                style={{
                  padding: '8px 12px',
                  cursor: 'pointer',
                  color: 'white',
                  fontSize: '13px',
                  borderBottom: '1px solid #404040'
                }}
                onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#404040'}
                onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
              >
                {p.name}
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
};

export default AddToPlaylistMenu;
