import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { Link } from 'react-router-dom';
import { apiService } from '../services/api';
import { usePlayer } from '../context/PlayerContext';
import AlbumCard from '../components/AlbumCard';
import type { Album, Track } from '../types';

const formatTime = (ms: number) => {
  if (!ms) return '--:--';
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
  const [releaseRadar, setReleaseRadar] = useState<any[]>([]);
  const [similarArtists, setSimilarArtists] = useState<any[]>([]);

  useEffect(() => {
    let isMounted = true;
    apiService.fetchDashboard()
      .then(data => {
        if (!isMounted) return;
        setRecentlyAdded(data.recently_added_albums || []);
        setRecentlyPlayed(data.recently_played_tracks || []);
        setFavorites(data.favorite_tracks || []);
        setLoading(false);
      })
      .catch(err => {
        if (!isMounted) return;
        console.error("Failed to load dashboard:", err);
        setLoading(false);
      });
      
    apiService.fetchDiscovery()
      .then(data => {
        if (!isMounted) return;
        if (data.release_radar) {
           // filter out duplicates by collectionName
           const unique = data.release_radar.filter((v: any, i: number, a: any[]) => a.findIndex(t => (t.collectionName === v.collectionName)) === i);
           setReleaseRadar(unique);
        }
        if (data.similar_artists) {
           const uniqueSim = data.similar_artists.filter((v: any, i: number, a: any[]) => a.findIndex(t => (t.name === v.name)) === i);
           setSimilarArtists(uniqueSim);
        }
      })
      .catch(console.error);
    return () => { isMounted = false; };
  }, []);

  if (loading) {
    return <div className="content-scroll"><div style={{ padding: '32px' }}><p>Loading Home...</p></div></div>;
  }

  const renderTrackRow = (track: Track, index: number, contextTracks: Track[], contextId: string) => {
    const isCurrentlyPlaying = currentTrack?.id === track.id;
    return (
      <div 
        key={`${contextId}-${track.id}-${index}`}
        className={`track-row ${isCurrentlyPlaying ? 'playing' : ''}`}
        onDoubleClick={() => playContext(contextTracks, index, { id: contextId, title: 'Home Tracks', release_year: 0, cover_art_path: '', artist_id: '', artist_name: '' } as Album)}
      >
        <div className="track-number">
          {isCurrentlyPlaying && isPlaying ? (
            <div className="playing-indicator" style={{ background: 'var(--accent-primary)', width: '16px', height: '16px', borderRadius: '50%' }} />
          ) : (
            <span>{index + 1}</span>
          )}
        </div>
        <div className="track-title-cell" style={{ display: 'flex', flexDirection: 'column' }}>
          <div className="track-title-text" style={{ color: isCurrentlyPlaying ? 'var(--accent-primary)' : 'var(--text-primary)' }}>
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
      <div className="page-container">
        <h1 style={{ fontSize: '48px', fontWeight: 900, marginBottom: '40px', letterSpacing: '-1.5px' }}>Home</h1>

        {releaseRadar.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Release Radar</h2>
            <p style={{ color: 'var(--text-muted)', marginBottom: '24px', marginTop: '-8px' }}>Brand new releases from artists in your library</p>
            <div className="album-grid">
              {releaseRadar.map((release, i) => (
                <a key={i} href={release.collectionViewUrl} target="_blank" rel="noreferrer" style={{ textDecoration: 'none' }}>
                  <div className="album-card" style={{ cursor: 'pointer' }}>
                    <div className="album-cover-container">
                      <img src={release.artworkUrl100.replace('100x100bb', '300x300bb')} alt={release.collectionName} className="album-cover" />
                    </div>
                  <div className="album-info" style={{ display: 'flex', flexDirection: 'column' }}>
                    <h3 style={{ margin: '0 0 4px 0', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>{release.collectionName}</h3>
                    <div style={{ fontSize: '13px', color: 'var(--text-muted)', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>{release.artistName}</div>
                    <div style={{ color: 'var(--accent-primary)', fontSize: '11px', marginTop: '4px' }}>
                       {new Date(release.releaseDate).getFullYear()} • Out Now
                    </div>
                  </div>
                  </div>
                </a>
              ))}
            </div>
          </div>
        )}

        {similarArtists.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Discover</h2>
            <p style={{ color: 'var(--text-muted)', marginBottom: '24px', marginTop: '-8px' }}>Similar artists based on your library (via Last.fm)</p>
            <div className="album-grid">
              {similarArtists.map((artist, i) => {
                const CardContent = (
                  <div className="album-card" style={{ cursor: 'pointer' }}>
                    <div className="album-cover-container" style={{ borderRadius: '50%' }}>
                      {artist.image ? (
                        <img src={artist.image} alt={artist.name} className="album-cover" style={{ borderRadius: '50%' }} />
                      ) : (
                        <div className="album-cover" style={{ borderRadius: '50%', background: 'var(--surface-light)' }} />
                      )}
                    </div>
                    <div className="album-info" style={{ display: 'flex', flexDirection: 'column', marginTop: '12px', textAlign: 'center' }}>
                      <h3 style={{ margin: '0 0 4px 0', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>{artist.name}</h3>
                      <div style={{ fontSize: '13px', color: 'var(--text-muted)', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>Because you listen to {artist.basedOn}</div>
                    </div>
                  </div>
                );

                return artist.id ? (
                  <Link key={i} to={`/artist/${artist.id}`} style={{ textDecoration: 'none', color: 'inherit' }}>
                    {CardContent}
                  </Link>
                ) : (
                  <a key={i} href={`https://www.youtube.com/results?search_query=${encodeURIComponent(artist.name + " music")}`} target="_blank" rel="noreferrer" style={{ textDecoration: 'none', color: 'inherit' }}>
                    {CardContent}
                  </a>
                );
              })}
            </div>
          </div>
        )}

        {recentlyPlayed.length > 0 && (
          <div style={{ marginBottom: '48px' }}>
            <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '16px' }}>Recently Played</h2>
            <div className="track-list">
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
            <div className="track-list">
              {favorites.map((track, index) => renderTrackRow(track, index, favorites, 'home-favorites'))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default HomePage;
