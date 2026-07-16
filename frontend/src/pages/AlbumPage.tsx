import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Play, Download, PlaySquare, ListPlus } from 'lucide-react';
import { createPortal } from 'react-dom';
import { apiService } from '../services/api';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from '../components/HeartButton';
import AddToPlaylistMenu from '../components/AddToPlaylistMenu';
import type { Album, Track } from '../types';
import DOMPurify from 'dompurify';

const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

const formatTime = (ms: number) => {
  if (!ms) return '--:--';
  const seconds = Math.floor(ms / 1000);
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const AlbumPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const { playContext, currentTrack, isPlaying, insertNext, enqueue } = usePlayer();
  const [contextMenu, setContextMenu] = useState<{ x: number, y: number, track: any } | null>(null);

  useEffect(() => {
    const handleClick = () => setContextMenu(null);
    window.addEventListener('click', handleClick);
    return () => window.removeEventListener('click', handleClick);
  }, []);
  
  const [album, setAlbum] = useState<Album | null>(null);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    
    let isMounted = true;
    Promise.all([
      apiService.fetchAlbumById(id),
      apiService.fetchTracks(id, 100, 0)
    ])
    .then(([albumData, tracksData]) => {
      if (!isMounted) return;
      setAlbum(albumData);
      setTracks(tracksData || []);
      setLoading(false);
    })
    .catch(err => {
      if (!isMounted) return;
      console.error(err);
      setLoading(false);
    });
    
    return () => { isMounted = false; };
  }, [id]);

  if (loading) return <div className="content-scroll">Loading...</div>;
  if (!album) return <div className="content-scroll">Album not found</div>;

  return (
    <div className="content-scroll" style={{ padding: 0 }}>
      {/* Cinematic Hero Section */}
      <div className="album-header">
        {/* Background Blur */}
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundImage: `url(${API_BASE_URL}/api/art/album/${album.id})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          filter: 'blur(100px) saturate(2)',
          opacity: 0.25,
          zIndex: 0,
          maskImage: 'linear-gradient(to bottom, black 40%, transparent 100%)',
          WebkitMaskImage: 'linear-gradient(to bottom, black 40%, transparent 100%)'
        }} />
        
        <div className="album-cover-container">
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
          <div className="album-art-placeholder" style={{ display: 'none', width: '100%', height: '100%', position: 'absolute', top: 0, left: 0, margin: 0, borderRadius: 0, alignItems: 'center', justifyContent: 'center' }}>
            <span style={{ fontSize: '96px', fontWeight: 800 }}>{album.title.charAt(0)}</span>
          </div>
        </div>
        <div className="album-header-text">
          <span style={{ textTransform: 'uppercase', fontSize: '13px', fontWeight: 800, letterSpacing: '2px', color: 'var(--text-secondary)' }}>Album</span>
          <h1 className="album-title">{album.title}</h1>
          <p className="album-meta">
            <span style={{ color: 'var(--text-primary)' }}>{album.release_year > 0 ? album.release_year : 'Unknown Year'}</span>
            <span>•</span>
            <span>{tracks.length} tracks</span>
          </p>
          
          <div style={{ display: 'flex', gap: '16px', marginTop: '32px' }}>
            <button 
              style={{ width: '56px', height: '56px', borderRadius: '50%', background: 'var(--accent-primary)', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: 'var(--accent-glow)', cursor: 'pointer', border: 'none' }}
              onClick={() => playContext(tracks, 0, album)}
              title="Play Album"
            >
              <Play size={24} fill="currentColor" />
            </button>
            <button 
              style={{ width: '56px', height: '56px', borderRadius: '50%', background: 'rgba(255,255,255,0.1)', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', border: 'none', transition: 'background 0.2s' }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.2)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
              title="Download Album"
              onClick={() => {
                const token = localStorage.getItem('sn_token');
                window.location.href = `${API_BASE_URL}/api/download/album/${album.id}?token=${token}`;
              }}
            >
              <Download size={24} />
            </button>
          </div>
        </div>
      </div>
      <div className="page-container">
        {album.bio && album.bio !== 'NOT_FOUND' && (
          <div style={{ marginBottom: '48px', maxWidth: '800px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>About</h2>
            <p style={{ color: 'var(--text-secondary)', lineHeight: 1.6, fontSize: '15px' }} dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(album.bio) }}></p>
          </div>
        )}

        {/* Tracklist */}
        <div className="track-list">
        {tracks.map((track, index) => {
          const isThisTrackPlaying = currentTrack?.id === track.id;
          return (
            <div 
              key={track.id} 
              className={`track-row ${isThisTrackPlaying ? 'playing' : ''}`}
              onClick={() => playContext(tracks, index, album)}
              onContextMenu={(e) => {
                e.preventDefault();
                setContextMenu({ x: e.clientX, y: e.clientY, track });
              }}
            >
              <div className="track-number">
                {isThisTrackPlaying && isPlaying ? (
                  <div className="playing-indicator" style={{ background: 'var(--primary-color)', width: '16px', height: '16px', borderRadius: '50%' }} />
                ) : (
                  <span>{track.track_number}</span>
                )}
              </div>
              <div className="track-title-cell">
                <div className="track-title-text" style={{ color: isThisTrackPlaying ? 'var(--primary-color)' : 'var(--text-primary)' }}>
                  {track.title}
                </div>
                {track.artist_name && track.artist_name !== album.artist_name && (
                  <Link 
                    to={`/artist/${track.artist_id}`}
                    className="track-artist-link"
                    style={{ fontSize: '13px', color: 'var(--text-muted)', textDecoration: 'none', transition: 'color 0.2s', marginTop: '2px' }}
                    onClick={(e) => e.stopPropagation()} // Prevent playing the track when clicking the artist
                    onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'}
                    onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-muted)'}
                  >
                    {track.artist_name}
                  </Link>
                )}
              </div>
              <div className="track-actions">
                <HeartButton entityType="track" entityId={track.id} />
                <AddToPlaylistMenu trackId={track.id} />
              </div>
              <div className="track-duration">
                {formatTime(track.duration_ms)}
              </div>
            </div>
          );
        })}
      </div>
    </div>
    
      {/* Context Menu */}
      {contextMenu && contextMenu.track && createPortal(
        <>
          <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 999 }} onClick={() => setContextMenu(null)} onContextMenu={(e) => { e.preventDefault(); setContextMenu(null); }} />
          <div style={{
            position: 'fixed', top: contextMenu.y, left: contextMenu.x, zIndex: 1000,
            background: 'var(--bg-secondary)', border: '1px solid var(--border-color)', borderRadius: '8px', padding: '8px 0',
            boxShadow: '0 4px 12px rgba(0,0,0,0.5)', minWidth: '160px',
            animation: 'fadeIn 0.1s ease'
          }}>
            <div 
              style={{ padding: '10px 16px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '12px', color: 'var(--text-primary)', fontSize: '14px', transition: 'background 0.2s' }}
              onClick={() => {
                insertNext(contextMenu.track);
                setContextMenu(null);
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <PlaySquare size={16} /> Play Next
            </div>
            
            <div 
              style={{ padding: '10px 16px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '12px', color: 'var(--text-primary)', fontSize: '14px', transition: 'background 0.2s' }}
              onClick={() => {
                enqueue(contextMenu.track);
                setContextMenu(null);
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <ListPlus size={16} /> Add to Queue
            </div>
            
            <div 
              style={{ padding: '10px 16px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '12px', color: 'var(--text-primary)', fontSize: '14px', transition: 'background 0.2s' }}
              onClick={() => {
                const token = localStorage.getItem('sn_token');
                window.location.href = `${API_BASE_URL}/api/download/track/${contextMenu.track.id}?token=${token}`;
                setContextMenu(null);
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <Download size={16} /> Download Track
            </div>
          </div>
        </>,
        document.body
      )}
    </div>
  );
};

export default AlbumPage;
