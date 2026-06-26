import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { apiService } from '../services/api';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from '../components/HeartButton';
import AddToPlaylistMenu from '../components/AddToPlaylistMenu';
import type { Album, Track } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const formatTime = (ms: number) => {
  const seconds = Math.floor(ms / 1000);
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const AlbumPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const { playContext, currentTrack, isPlaying } = usePlayer();
  
  const [album, setAlbum] = useState<Album | null>(null);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    
    Promise.all([
      apiService.fetchAlbumById(id),
      apiService.fetchTracks(id, 100, 0)
    ])
    .then(([albumData, tracksData]) => {
      setAlbum(albumData);
      setTracks(tracksData || []);
      setLoading(false);
    })
    .catch(err => {
      console.error(err);
      setLoading(false);
    });
  }, [id]);

  if (loading) return <div className="content-scroll">Loading...</div>;
  if (!album) return <div className="content-scroll">Album not found</div>;

  return (
    <div className="content-scroll">
      {/* Hero Section */}
      <div style={{ display: 'flex', gap: '32px', marginBottom: '40px' }}>
        <div style={{ position: 'relative', width: '250px', height: '250px', flexShrink: 0, margin: 0, borderRadius: '8px', overflow: 'hidden', boxShadow: '0 8px 32px rgba(0,0,0,0.5)' }}>
          <img 
            src={`${API_BASE_URL}/api/art/album/${album.id}`}
            alt={album.title}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              if (e.currentTarget.nextElementSibling) {
                (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'flex';
              }
            }}
          />
          <div className="album-art-placeholder glass-panel" style={{ display: 'none', width: '100%', height: '100%', position: 'absolute', top: 0, left: 0, margin: 0, borderRadius: 0 }}>
            <span style={{ fontSize: '72px' }}>{album.title.charAt(0)}</span>
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'flex-end', paddingBottom: '16px' }}>
          <span style={{ textTransform: 'uppercase', fontSize: '12px', fontWeight: 600, letterSpacing: '1px', color: 'var(--text-secondary)' }}>Album</span>
          <h1 style={{ fontSize: '48px', fontWeight: 800, margin: '8px 0', letterSpacing: '-1px' }}>{album.title}</h1>
          <p style={{ color: 'var(--text-secondary)' }}>{album.release_year > 0 ? album.release_year : 'Unknown Year'} • {tracks.length} tracks</p>
          
          <button 
            className="btn-primary" 
            style={{ width: '120px', marginTop: '24px' }}
            onClick={() => tracks.length > 0 && playContext(tracks, 0, album)}
          >
            Play Album
          </button>
        </div>
      </div>

      {/* Tracklist */}
      <div className="tracklist">
        {tracks.map(track => {
          const isThisTrackPlaying = currentTrack?.id === track.id;
          return (
            <div 
              key={track.id} 
              className="track-row"
              style={{
                display: 'grid',
                gridTemplateColumns: '40px 1fr 40px 100px',
                padding: '12px 16px',
                borderRadius: '8px',
                alignItems: 'center',
                cursor: 'pointer',
                background: isThisTrackPlaying ? 'var(--bg-glass)' : 'transparent',
                transition: 'background 0.2s'
              }}
              onClick={() => playContext(tracks, tracks.findIndex(t => t.id === track.id), album)}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-glass-hover)'}
              onMouseLeave={(e) => e.currentTarget.style.background = isThisTrackPlaying ? 'var(--bg-glass)' : 'transparent'}
            >
              <div style={{ color: isThisTrackPlaying ? 'var(--accent-primary)' : 'var(--text-muted)' }}>
                {isThisTrackPlaying && isPlaying ? '▶' : track.track_number}
              </div>
              <div style={{ fontWeight: isThisTrackPlaying ? 600 : 400, color: isThisTrackPlaying ? 'var(--accent-primary)' : 'var(--text-primary)' }}>
                {track.title}
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', alignItems: 'center' }}>
                <HeartButton entityType="track" entityId={track.id} />
                <AddToPlaylistMenu trackId={track.id} />
              </div>
              <div style={{ textAlign: 'right', color: 'var(--text-muted)', fontSize: '13px' }}>
                {formatTime(track.duration_ms)}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default AlbumPage;
