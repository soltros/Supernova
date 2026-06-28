import React, { useState, useEffect } from 'react';
import type { Album } from '../types';
import { Disc3 } from 'lucide-react';
import { apiService } from '../services/api';
import AlbumCard from '../components/AlbumCard';

const AlbumsPage: React.FC = () => {
  const [albums, setAlbums] = useState<Album[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiService.fetchAlbums(100, 0)
      .then(data => {
        setAlbums(data || []);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch albums:", err);
        setLoading(false);
      });
  }, []);

  return (
    <div className="content-scroll">
      <div className="page-container">
        <h1 style={{ fontSize: '32px', fontWeight: 800, marginBottom: '32px', letterSpacing: '-1px' }}>All Albums</h1>
        
        {loading ? (
          <p style={{ color: 'var(--text-muted)' }}>Loading albums...</p>
        ) : albums.length > 0 ? (
          <div className="album-grid">
            {albums.map(album => (
              <AlbumCard key={album.id} album={album} />
            ))}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', height: '100%', color: 'var(--text-secondary)' }}>
            <div style={{ marginBottom: '16px', opacity: 0.2 }}>
              <Disc3 size={64} />
            </div>
            <h3 style={{ fontSize: '24px', fontWeight: 600, color: 'var(--text-primary)' }}>No Albums Found</h3>
            <p>Your library doesn't contain any albums yet.</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default AlbumsPage;
