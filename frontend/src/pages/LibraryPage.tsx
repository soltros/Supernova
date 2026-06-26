import { useState, useEffect } from 'react';
import type { FC } from 'react';
import AlbumCard from '../components/AlbumCard';
import { usePlayer } from '../context/PlayerContext';
import type { Album, Track } from '../types';
import { apiService } from '../services/api';

const LibraryPage: FC = () => {
  const [albums, setAlbums] = useState<Album[]>([]);
  const [recentScrobbles, setRecentScrobbles] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);
  const { playContext } = usePlayer();

  useEffect(() => {
    Promise.all([
      apiService.fetchAlbums(12, 0),
      apiService.getRecentScrobbles().catch(() => []) // Graceful fail if no history
    ])
    .then(([albumsData, scrobblesData]) => {
      setAlbums(albumsData || []);
      setRecentScrobbles(scrobblesData || []);
      setLoading(false);
    })
    .catch(err => {
      console.error("Failed to fetch library:", err);
      setLoading(false);
    });
  }, []);

  return (
    <section className="content-scroll">
      {/* Recently Played History */}
      {recentScrobbles.length > 0 && (
        <div style={{ marginBottom: '40px' }}>
          <h2 className="section-title">Recently Played</h2>
          <div className="tracklist" style={{ display: 'grid', gap: '8px' }}>
            {recentScrobbles.slice(0, 5).map((track, i) => (
              <div 
                key={`${track.id}-${i}`}
                className="track-row glass-panel"
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr auto',
                  padding: '12px 16px',
                  borderRadius: '8px',
                  alignItems: 'center',
                  cursor: 'pointer'
                }}
                // To play a scrobble, we fake an album context since we just have the track
                onClick={() => playContext([track], 0, { id: track.album_id, title: "History", release_year: 0, cover_art_path: "" })}
              >
                <div style={{ fontWeight: 600 }}>{track.title}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      <h2 className="section-title">Recently Added Albums</h2>
      {loading ? (
        <p style={{ color: 'var(--text-muted)' }}>Loading library from server...</p>
      ) : (
        <div className="album-grid">
          {albums.length > 0 ? albums.map(album => (
            <AlbumCard key={album.id} album={album} />
          )) : (
            <p style={{ color: 'var(--text-muted)' }}>Your library is completely empty.</p>
          )}
        </div>
      )}
    </section>
  );
};

export default LibraryPage;
