import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { apiService } from '../services/api';
import type { Artist, Album, Track } from '../types';
import ArtistCard from '../components/ArtistCard';
import AlbumCard from '../components/AlbumCard';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from '../components/HeartButton';

const SearchPage: FC = () => {
  const [searchParams] = useSearchParams();
  const query = searchParams.get('q') || '';
  
  const [artists, setArtists] = useState<Artist[]>([]);
  const [albums, setAlbums] = useState<Album[]>([]);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(false);

  const { playContext, currentTrack, isPlaying } = usePlayer();

  useEffect(() => {
    if (!query) return;
    
    let isMounted = true;

    const fetchResults = async () => {
      setLoading(true);
      setArtists([]);
      setAlbums([]);
      setTracks([]);
      try {
        const results = await apiService.search(query);
        if (!isMounted) return;
        setArtists(results.artists || []);
        setAlbums(results.albums || []);
        setTracks(results.tracks || []);
      } catch (error) {
        if (!isMounted) return;
        console.error('Search failed:', error);
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    fetchResults();
    
    return () => { isMounted = false; };
  }, [query]);

  if (!query) {
    return (
      <div className="content-scroll">
        <div className="page-container flex-center">
          <h2>Enter a search term to begin</h2>
        </div>
      </div>
    );
  }

  return (
    <div className="content-scroll">
      <div className="page-container fade-in">
        <h1 className="page-title">Search Results for "{query}"</h1>

        {loading ? (
          <div className="loading-spinner" />
        ) : (
          <div className="dashboard-content">
            {artists.length > 0 && (
              <section className="dashboard-section">
                <h2 className="section-title">Artists</h2>
                <div className="album-grid">
                  {artists.map((artist) => (
                    <ArtistCard 
                      key={artist.id} 
                      artist={artist} 
                    />
                  ))}
                </div>
              </section>
            )}

            {albums.length > 0 && (
              <section className="dashboard-section">
                <h2 className="section-title">Albums</h2>
                <div className="album-grid">
                  {albums.map((album) => (
                    <AlbumCard 
                      key={album.id} 
                      album={album} 
                    />
                  ))}
                </div>
              </section>
            )}

            {tracks.length > 0 && (
              <section className="dashboard-section">
                <h2 className="section-title">Tracks</h2>
                <div className="track-list">
                  {tracks.map((track, index) => {
                    const isCurrentTrack = currentTrack?.id === track.id;
                    
                    return (
                      <div 
                        key={track.id}
                        className={`track-row ${isCurrentTrack ? 'playing' : ''}`}
                        onClick={() => playContext(tracks, index, { id: 'search', title: `Search: ${query}`, release_year: 0, cover_art_path: '', artist_id: '', artist_name: '' } as Album)}
                      >
                        <div className="track-number">
                          {isCurrentTrack && isPlaying ? (
                            <div className="playing-indicator" style={{ background: 'var(--accent-primary)', width: '16px', height: '16px', borderRadius: '50%' }} />
                          ) : (
                            <span>{index + 1}</span>
                          )}
                        </div>
                        
                        <div className="track-title-cell">
                          <div className="track-title-text" style={{ color: isCurrentTrack ? 'var(--accent-primary)' : 'var(--text-primary)' }}>{track.title}</div>
                          <div style={{ fontSize: '13px', color: 'var(--text-secondary)', marginTop: '4px' }}>{track.artist_name} • {track.album_title}</div>
                        </div>

                        <div className="track-actions">
                          <HeartButton entityType="track" entityId={track.id} />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </section>
            )}

            {artists.length === 0 && albums.length === 0 && tracks.length === 0 && (
              <div className="empty-state">
                <p>No results found for "{query}"</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default SearchPage;
