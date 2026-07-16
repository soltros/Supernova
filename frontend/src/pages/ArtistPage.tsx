import React, { useState, useEffect } from 'react';
import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { Play, Download } from 'lucide-react';
import DOMPurify from 'dompurify';
import { apiService } from '../services/api';
import { usePlayer } from '../context/PlayerContext';
import AlbumCard from '../components/AlbumCard';
import HeartButton from '../components/HeartButton';
import type { Artist, Album, Track } from '../types';

const formatTime = (ms: number) => {
  if (!ms) return '--:--';
  const seconds = Math.floor(ms / 1000);
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const ArtistPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const { playContext, currentTrack, isPlaying } = usePlayer();
  
  const [artist, setArtist] = useState<Artist | null>(null);
  const [albums, setAlbums] = useState<Album[]>([]);
  
  const mockArtistAlbum = React.useMemo(() => {
    if (!artist) return null;
    return { id: `artist-${artist.id}`, title: `${artist.name} Top Tracks`, release_year: 0, cover_art_path: '', artist_id: artist.id, artist_name: artist.name } as Album;
  }, [artist]);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);
  const [isDownloadingDiscography, setIsDownloadingDiscography] = useState(false);
  const [downloadProgress, setDownloadProgress] = useState(0);

  const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

  const handleDownloadDiscography = async () => {
    if (albums.length === 0) return;
    setIsDownloadingDiscography(true);
    setDownloadProgress(0);
    const token = localStorage.getItem('sn_token');
    
    for (let i = 0; i < albums.length; i++) {
        setDownloadProgress(i + 1);
        const album = albums[i];
        const a = document.createElement('a');
        a.href = `${API_BASE_URL}/api/download/album/${album.id}?token=${token}`;
        a.download = `${album.title}.zip`;
        a.style.display = 'none';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        
        // Wait 1.5 seconds between triggering downloads
        await new Promise(resolve => setTimeout(resolve, 1500));
    }
    
    setTimeout(() => {
        setIsDownloadingDiscography(false);
        setDownloadProgress(0);
    }, 1000);
  };

  useEffect(() => {
    if (!id) return;
    
    let isMounted = true;
    setLoading(true);
    Promise.all([
      apiService.fetchArtistById(id),
      apiService.fetchAlbums(50, 0, id),
      apiService.fetchTracks(undefined, 50, 0, id)
    ])
      .then(([artistData, albumsData, tracksData]) => {
        if (!isMounted) return;
        setArtist(artistData);
        setAlbums(albumsData || []);
        setTracks(tracksData || []);
        setLoading(false);
      })
      .catch(err => {
        if (!isMounted) return;
        console.error("Failed to load artist data:", err);
        setLoading(false);
      });
      
    return () => { isMounted = false; };
  }, [id]);

  if (loading) {
    return <div className="content-scroll"><div style={{ padding: '32px' }}><p>Loading artist...</p></div></div>;
  }

  if (!artist) {
    return <div className="content-scroll"><div style={{ padding: '32px' }}><p>Artist not found.</p></div></div>;
  }

  return (
    <div className="content-scroll">
      {/* Artist Header */}
      <div className="artist-header">
        {/* Background Blur Image */}
        {artist.image_url && artist.image_url !== 'NOT_FOUND' && (
          <div style={{
            position: 'absolute',
            top: '-50%', left: '-50%', right: '-50%', bottom: '-50%',
            backgroundImage: `url(${artist.image_url})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            filter: 'blur(60px) brightness(0.4)',
            zIndex: 0
          }} />
        )}
        
        <div className="artist-header-content">
          <div className="artist-avatar-container">
             {artist.image_url && artist.image_url !== 'NOT_FOUND' ? (
                <img src={artist.image_url} alt={artist.name} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              ) : (
                <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '64px', fontWeight: 800, color: 'var(--text-muted)' }}>
                  {artist.name.charAt(0)}
                </div>
              )}
          </div>
          <div className="artist-header-text">
            <p className="artist-verified">Verified Artist</p>
            <h1 className="artist-title">{artist.name}</h1>
          </div>
        </div>
      </div>

      <div className="page-container">
        {/* Play Controls & Action Bar */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '24px', marginBottom: '40px' }}>
          <button 
            style={{ 
              width: '56px', height: '56px', 
              borderRadius: '50%', 
              background: 'var(--accent-primary)', 
              color: 'white', 
              display: 'flex', alignItems: 'center', justifyContent: 'center', 
              border: 'none',
              cursor: 'pointer',
              boxShadow: 'var(--accent-glow)',
              transition: 'var(--transition-fast)'
            }}
            onClick={() => {
              if (tracks.length > 0 && mockArtistAlbum) {
                playContext(tracks, 0, mockArtistAlbum);
              }
            }}
            onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.05)'}
            onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
          >
            <Play size={28} fill="currentColor" />
          </button>
          <HeartButton entityType="artist" entityId={artist.id} />
          
          <button 
            style={{ 
              width: '56px', height: '56px', 
              borderRadius: '50%', 
              background: 'rgba(255,255,255,0.1)', 
              color: 'white', 
              display: 'flex', alignItems: 'center', justifyContent: 'center', 
              border: 'none',
              cursor: isDownloadingDiscography ? 'default' : 'pointer',
              transition: 'var(--transition-fast)'
            }}
            onClick={handleDownloadDiscography}
            disabled={isDownloadingDiscography}
            title="Download Discography"
            onMouseEnter={(e) => { if(!isDownloadingDiscography) e.currentTarget.style.background = 'rgba(255,255,255,0.2)'; }}
            onMouseLeave={(e) => { if(!isDownloadingDiscography) e.currentTarget.style.background = 'rgba(255,255,255,0.1)'; }}
          >
            <Download size={24} />
          </button>
          
          {isDownloadingDiscography && (
            <div style={{ color: 'var(--accent-primary)', fontSize: '14px', fontWeight: 600 }}>
              Starting download {downloadProgress} of {albums.length}...
            </div>
          )}
        </div>

        {artist.bio && artist.bio !== 'NOT_FOUND' && (
          <div style={{ marginBottom: '48px', maxWidth: '800px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>About</h2>
            <p style={{ color: 'var(--text-secondary)', lineHeight: 1.6, fontSize: '15px' }} dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(artist.bio) }}></p>
          </div>
        )}

        {/* Popular Tracks */}
        {tracks.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Popular Tracks</h2>
            <div className="track-list">
              <div className="track-list-header">
                <div className="track-number">#</div>
                <div className="track-title-cell">TITLE</div>
                <div className="track-actions"></div>
                <div className="track-duration">TIME</div>
              </div>
              {tracks.slice(0, 5).map((track, index) => {
                const isCurrentlyPlaying = currentTrack?.id === track.id;
                
                return (
                  <div 
                    key={track.id}
                    className={`track-row ${isCurrentlyPlaying ? 'playing' : ''}`}
                    onDoubleClick={() => { if (mockArtistAlbum) playContext(tracks, index, mockArtistAlbum); }}
                  >
                    <div className="track-number">
                      {isCurrentlyPlaying && isPlaying ? (
                        <div className="playing-indicator" style={{ background: 'var(--primary-color)', width: '16px', height: '16px', borderRadius: '50%' }} />
                      ) : (
                        <span>{index + 1}</span>
                      )}
                    </div>
                    <div className="track-title-cell">
                      <div className="track-title-text" style={{ color: isCurrentlyPlaying ? 'var(--primary-color)' : 'var(--text-primary)' }}>
                        {track.title}
                      </div>
                    </div>
                    <div className="track-actions">
                      <button 
                        className="play-btn-small" 
                        onClick={(e) => {
                          e.stopPropagation();
                          if (mockArtistAlbum) playContext(tracks, index, mockArtistAlbum);
                        }}
                      >
                        <Play size={18} fill="currentColor" />
                      </button>
                    </div>
                    <div className="track-duration">{formatTime(track.duration_ms)}</div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Albums */}
        {albums.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Albums</h2>
            <div className="album-grid">
              {albums.map(album => (
                <AlbumCard key={album.id} album={album} />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ArtistPage;
