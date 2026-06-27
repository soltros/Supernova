import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { Link } from 'react-router-dom';
import { apiService } from '../services/api';
import { usePlayer } from '../context/PlayerContext';
import AlbumCard from '../components/AlbumCard';
import type { Album, Track } from '../types';

const formatTime = (ms: number) => {
  const seconds = Math.floor(ms / 1000);
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const HomePage: FC = () => {
  const { playContext, currentTrack, isPlaying } = usePlayer();
  const [loading, setLoading] = useState(true);
  const [recentlyAdded, setRecentlyAdded] = useState<Album[]>([]);
  const [recentlyPlayed, setRecentlyPlayed] = useState<Track[]>([]);
  const [favorites, setFavorites] = useState<Track[]>([]);

  useEffect(() => {
    apiService.fetchDashboard()
      .then(data => {
        setRecentlyAdded(data.recently_added_albums || []);
        setRecentlyPlayed(data.recently_played_tracks || []);
        setFavorites(data.favorite_tracks || []);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to load dashboard:", err);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return <div className="content-scroll"><div style={{ padding: '32px' }}><p>Loading Home...</p></div></div>;
  }

  const renderTrackRow = (track: Track, index: number, contextTracks: Track[], contextId: string) => {
    const isCurrentlyPlaying = currentTrack?.id === track.id;
    return (
      <div 
        key={track.id + index}
        className="track-row"
        onDoubleClick={() => playContext(contextTracks, index, { id: contextId, title: 'Home Tracks', release_year: 0, cover_art_path: '', artist_id: '', artist_name: '' } as Album)}
      >
        <div className="track-number">
          {isCurrentlyPlaying && isPlaying ? (
            <div className="playing-indicator" style={{ background: 'var(--primary-color)', width: '16px', height: '16px', borderRadius: '50%' }} />
          ) : (
            <span>{index + 1}</span>
          )}
        </div>
        <div className="track-title-cell" style={{ display: 'flex', flexDirection: 'column' }}>
          <div className="track-title-text" style={{ color: isCurrentlyPlaying ? 'var(--primary-color)' : 'var(--text-primary)' }}>
            {track.title}
          </div>
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
        <div className="track-duration">{formatTime(track.duration_ms)}</div>
      </div>
    );
  };

  return (
    <div className="content-scroll">
      <div style={{ padding: '32px' }}>
        <h1 style={{ fontSize: '48px', fontWeight: 900, marginBottom: '40px', letterSpacing: '-1.5px' }}>Home</h1>

        {recentlyPlayed.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Recently Played</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              {recentlyPlayed.map((track, index) => renderTrackRow(track, index, recentlyPlayed, 'home-recent'))}
            </div>
          </div>
        )}

        {recentlyAdded.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Recently Added</h2>
            <div className="album-grid">
              {recentlyAdded.map(album => (
                <AlbumCard key={album.id} album={album} />
              ))}
            </div>
          </div>
        )}

        {favorites.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Your Favorites</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              {favorites.map((track, index) => renderTrackRow(track, index, favorites, 'home-favorites'))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default HomePage;
