import { useState, useEffect } from 'react';
import type { FC, ChangeEvent } from 'react';
import { Link } from 'react-router-dom';
import { useHearts } from '../context/HeartsContext';
import { usePlayer } from '../context/PlayerContext';
import { apiService } from '../services/api';
import { useToast } from '../context/ToastContext';
import AlbumCard from '../components/AlbumCard';
import ArtistCard from '../components/ArtistCard';
import PlaylistCard from '../components/PlaylistCard';
import HeartButton from '../components/HeartButton';
import type { Album, Track, Artist, Playlist } from '../types';

const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

const formatTime = (ms: number) => {
  if (!ms) return '--:--';
  const seconds = Math.floor(ms / 1000);
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const HeartsPage: FC = () => {
  const { heartedIds, refreshHearts } = useHearts();
  const { playContext, currentTrack, isPlaying } = usePlayer();
  const { addToast } = useToast();
  
  const [loading, setLoading] = useState(true);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [albums, setAlbums] = useState<Album[]>([]);
  const [artists, setArtists] = useState<Artist[]>([]);
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [radioStations, setRadioStations] = useState<any[]>([]);
  const [podcasts, setPodcasts] = useState<any[]>([]);

  useEffect(() => {
    let isMounted = true;
    apiService.fetchHeartDetails()
      .then(data => {
        if (!isMounted) return;
        setTracks(data.tracks || []);
        setAlbums(data.albums || []);
        setArtists(data.artists || []);
        setPlaylists(data.playlists || []);
        setLoading(false);
      })
      .catch(err => {
        if (!isMounted) return;
        console.error("Failed to load heart details:", err);
        setLoading(false);
      });
    return () => { isMounted = false; };
  }, []);

  useEffect(() => {
    try {
      const storedRadio = JSON.parse(localStorage.getItem('heartedRadioStations') || '[]');
      const fallbackRadio = JSON.parse(localStorage.getItem('recentRadioStations') || '[]');
      const mergedRadio = [...storedRadio, ...fallbackRadio];
      const uniqueRadio = mergedRadio.filter((v, i, a) => a.findIndex(t => (t.stationuuid === v.stationuuid)) === i);
      setRadioStations(uniqueRadio.filter((s: any) => heartedIds.has(s.stationuuid)));

      const storedPodcasts = JSON.parse(localStorage.getItem('heartedPodcasts') || '[]');
      const fallbackPodcasts = JSON.parse(localStorage.getItem('recentPodcasts') || '[]');
      const mergedPodcasts = [...storedPodcasts, ...fallbackPodcasts];
      const uniquePodcasts = mergedPodcasts.filter((v, i, a) => a.findIndex(t => (t.id === v.id)) === i);
      setPodcasts(uniquePodcasts.filter((p: any) => heartedIds.has(p.id?.toString())));
    } catch (e) {}
  }, [heartedIds]);

  const handleExport = () => {
    window.open(`${API_BASE_URL}/api/hearts/export`, '_blank');
  };

  const handleImport = async (e: ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    const file = e.target.files[0];
    try {
      await apiService.importHearts(file);
      refreshHearts();
    } catch (err) {
      addToast("Failed to import backup. Ensure it is a valid Supernova JSON backup.", 'error');
    }
    e.target.value = '';
  };

  const renderTrackRow = (track: Track, index: number, contextTracks: Track[], contextId: string) => {
    const isCurrentlyPlaying = currentTrack?.id === track.id;
    return (
      <div 
        key={`${contextId}-${track.id}-${index}`}
        className="track-row"
        onDoubleClick={() => playContext(contextTracks, index, { id: contextId, title: 'Favorite Tracks', release_year: 0, cover_art_path: '', artist_id: '', artist_name: '' } as Album)}
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
            <Link to={`/artist/${track.artist_id}`} className="track-artist-link">
              {track.artist_name || 'Unknown Artist'}
            </Link>
          )}
        </div>
        <div className="track-actions">
          <HeartButton entityType="track" entityId={track.id} />
        </div>
        <div className="track-duration">{formatTime(track.duration_ms)}</div>
      </div>
    );
  };

  if (loading) {
    return (
    <div className="content-scroll">
      <div className="page-container"><p>Loading Hearts...</p></div></div>);
  }

  return (
    <div className="content-scroll">
      <div className="page-container" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px' }}>
        <div>
          <h1 style={{ fontSize: '40px', fontWeight: 800, margin: '0 0 8px 0', letterSpacing: '-1px' }}>Your Hearts</h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: '18px', margin: 0 }}>
            <strong style={{ color: 'var(--text-primary)' }}>{heartedIds.size}</strong> total favorites
          </p>
        </div>
        
        <div style={{ display: 'flex', gap: '12px' }}>
          <input type="file" id="import-upload" accept=".json" style={{ display: 'none' }} onChange={handleImport} />
          <label htmlFor="import-upload" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-glass-bright)', cursor: 'pointer', padding: '12px 24px', borderRadius: '12px', color: 'var(--text-primary)', fontWeight: 600, transition: 'var(--transition-fast)' }} onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-glass-hover)'} onMouseLeave={(e) => e.currentTarget.style.background = 'var(--bg-glass)'}>
            ↓ Import Backup
          </label>
          <button onClick={handleExport} style={{ background: 'var(--accent-gradient)', padding: '12px 24px', borderRadius: '12px', border: 'none', color: 'white', fontWeight: 700, cursor: 'pointer', boxShadow: 'var(--accent-glow)' }}>
            ↑ Export Backup
          </button>
        </div>
      </div>

      {albums.length === 0 && tracks.length === 0 && artists.length === 0 && playlists.length === 0 && radioStations.length === 0 && podcasts.length === 0 && (
        <div style={{ padding: '48px', textAlign: 'center', background: 'var(--bg-glass)', borderRadius: '24px', border: '1px solid var(--border-glass-bright)' }}>
          <p style={{ fontSize: '18px', color: 'var(--text-secondary)' }}>You haven't hearted anything yet!</p>
        </div>
      )}

      {artists.length > 0 && (
        <section style={{ marginBottom: '48px' }}>
          <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            Artists
            <span style={{ fontSize: '14px', padding: '4px 12px', background: 'var(--bg-glass)', borderRadius: '24px', color: 'var(--text-secondary)' }}>{artists.length}</span>
          </h2>
          <div className="album-grid">
            {artists.map(artist => (
              <ArtistCard key={artist.id} artist={artist} />
            ))}
          </div>
        </section>
      )}

      {playlists.length > 0 && (
        <section style={{ marginBottom: '48px' }}>
          <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            Playlists
            <span style={{ fontSize: '14px', padding: '4px 12px', background: 'var(--bg-glass)', borderRadius: '24px', color: 'var(--text-secondary)' }}>{playlists.length}</span>
          </h2>
          <div className="album-grid">
            {playlists.map(playlist => (
              <PlaylistCard key={playlist.id} playlist={playlist} />
            ))}
          </div>
        </section>
      )}

      {radioStations.length > 0 && (
        <section style={{ marginBottom: '48px' }}>
          <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            Radio Stations
            <span style={{ fontSize: '14px', padding: '4px 12px', background: 'var(--bg-glass)', borderRadius: '24px', color: 'var(--text-secondary)' }}>{radioStations.length}</span>
          </h2>
          <div className="album-grid">
            {radioStations.map(station => (
              <Link key={station.stationuuid} to="/radio" state={{ station }} className="album-card" style={{ textDecoration: 'none' }}>
                <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                  {station.favicon ? (
                    <img src={station.favicon} alt={station.name} style={{ width: '70%', height: '70%', objectFit: 'contain' }} />
                  ) : (
                    <span style={{ color: 'var(--text-muted)' }}>Radio</span>
                  )}
                </div>
                <div className="album-info">
                  <h3 className="album-title">{station.name}</h3>
                  <p className="album-artist">{station.country}</p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}

      {podcasts.length > 0 && (
        <section style={{ marginBottom: '48px' }}>
          <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            Podcasts
            <span style={{ fontSize: '14px', padding: '4px 12px', background: 'var(--bg-glass)', borderRadius: '24px', color: 'var(--text-secondary)' }}>{podcasts.length}</span>
          </h2>
          <div className="album-grid">
            {podcasts.map(podcast => (
              <Link key={podcast.id} to="/podcasts" state={{ podcast }} className="album-card" style={{ textDecoration: 'none' }}>
                <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                  {podcast.image ? (
                    <img src={podcast.image} alt={podcast.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  ) : (
                    <span style={{ color: 'var(--text-muted)' }}>Podcast</span>
                  )}
                </div>
                <div className="album-info">
                  <h3 className="album-title">{podcast.title}</h3>
                  <p className="album-artist">{podcast.author || 'Unknown'}</p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}

      {albums.length > 0 && (
        <section style={{ marginBottom: '48px' }}>
          <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            Albums
            <span style={{ fontSize: '14px', padding: '4px 12px', background: 'var(--bg-glass)', borderRadius: '24px', color: 'var(--text-secondary)' }}>{albums.length}</span>
          </h2>
          <div className="album-grid">
            {albums.map(album => (
              <AlbumCard key={album.id} album={album} />
            ))}
          </div>
        </section>
      )}

      {tracks.length > 0 && (
        <section>
          <h2 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            Tracks
            <span style={{ fontSize: '14px', padding: '4px 12px', background: 'var(--bg-glass)', borderRadius: '24px', color: 'var(--text-secondary)' }}>{tracks.length}</span>
          </h2>
          <div className="track-list">
            <div className="track-list-header">
              <div className="track-number">#</div>
              <div className="track-title-cell">TITLE</div>
              <div className="track-actions"></div>
              <div className="track-duration">TIME</div>
            </div>
            {tracks.map((track, i) => renderTrackRow(track, i, tracks, 'favorites'))}
          </div>
        </section>
      )}
    </div>
  );
};

export default HeartsPage;
