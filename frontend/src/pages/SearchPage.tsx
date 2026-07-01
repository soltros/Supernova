import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
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
  const navigate = useNavigate();

  useEffect(() => {
    if (!query) return;

    const fetchResults = async () => {
      setLoading(true);
      try {
        const results = await apiService.search(query);
        setArtists(results.artists || []);
        setAlbums(results.albums || []);
        setTracks(results.tracks || []);
      } catch (error) {
        console.error('Search failed:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchResults();
  }, [query]);

  if (!query) {
    return (
      <div className="page-container flex-center">
        <h2>Enter a search term to begin</h2>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <h1 className="page-title">Search Results for "{query}"</h1>

      {loading ? (
        <div className="loading-spinner" />
      ) : (
        <div className="dashboard-content">
          {artists.length > 0 && (
            <section className="dashboard-section">
              <h2 className="section-title">Artists</h2>
              <div className="grid">
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
              <div className="grid">
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
                      className={`track-item ${isCurrentTrack ? 'active' : ''}`}
                      onDoubleClick={() => playContext(tracks, index, { id: 'search', title: `Search: ${query}`, release_year: 0, cover_art_path: '', artist_id: '', artist_name: '' } as Album)}
                    >
                      <div className="track-index">
                        {isCurrentTrack && isPlaying ? (
                          <div className="playing-indicator">
                            <span className="bar"></span>
                            <span className="bar"></span>
                            <span className="bar"></span>
                          </div>
                        ) : (
                          index + 1
                        )}
                      </div>
                      
                      <div className="track-info">
                        <div className="track-title">{track.title}</div>
                        <div className="track-artist">{track.artist_name || 'Unknown Artist'} &bull; {track.album_title || 'Unknown Album'}</div>
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
            <div className="flex-center" style={{ marginTop: '4rem' }}>
              <h2>No results found for "{query}"</h2>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default SearchPage;
