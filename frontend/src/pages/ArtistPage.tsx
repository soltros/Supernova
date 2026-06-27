import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { Play } from 'lucide-react';
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
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    
    setLoading(true);
    Promise.all([
      apiService.fetchArtistById(id),
      apiService.fetchAlbums(50, 0, id),
      apiService.fetchTracks(undefined, 50, 0, id)
    ])
      .then(([artistData, albumsData, tracksData]) => {
        setArtist(artistData);
        setAlbums(albumsData || []);
        setTracks(tracksData || []);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to load artist data:", err);
        setLoading(false);
      });
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
      <div style={{ 
        position: 'relative',
        height: '300px', 
        display: 'flex', 
        alignItems: 'flex-end', 
        padding: '32px',
        overflow: 'hidden'
      }}>
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
        
        <div style={{ position: 'relative', zIndex: 1, display: 'flex', alignItems: 'flex-end', gap: '32px', width: '100%' }}>
          <div style={{ width: '200px', height: '200px', borderRadius: '50%', background: 'var(--bg-secondary)', overflow: 'hidden', boxShadow: '0 16px 32px rgba(0,0,0,0.5)', flexShrink: 0 }}>
             {artist.image_url && artist.image_url !== 'NOT_FOUND' ? (
                <img src={artist.image_url} alt={artist.name} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              ) : (
                <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '64px', fontWeight: 800, color: 'var(--text-muted)' }}>
                  {artist.name.charAt(0)}
                </div>
              )}
          </div>
          <div style={{ flexGrow: 1 }}>
            <p style={{ fontSize: '14px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '2px', marginBottom: '8px', color: 'var(--text-primary)' }}>Verified Artist</p>
            <h1 style={{ fontSize: '64px', fontWeight: 900, marginBottom: '16px', letterSpacing: '-2px', lineHeight: 1 }}>{artist.name}</h1>
          </div>
        </div>
      </div>

      <div style={{ padding: '32px' }}>
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
              if (tracks.length > 0) {
                playContext(tracks, 0, { id: `artist-${artist.id}`, title: `${artist.name} Top Tracks`, release_year: 0, cover_art_path: '', artist_id: artist.id, artist_name: artist.name } as Album);
              }
            }}
            onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.05)'}
            onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
          >
            <Play size={28} fill="currentColor" />
          </button>
          <HeartButton entityType="artist" entityId={artist.id} />
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
                    onDoubleClick={() => playContext(tracks, index, { id: `artist-${artist.id}`, title: `${artist.name} Top Tracks`, release_year: 0, cover_art_path: '', artist_id: artist.id, artist_name: artist.name } as Album)}
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
                          playContext(tracks, index, { id: `artist-${artist.id}`, title: `${artist.name} Top Tracks`, release_year: 0, cover_art_path: '', artist_id: artist.id, artist_name: artist.name } as Album);
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
