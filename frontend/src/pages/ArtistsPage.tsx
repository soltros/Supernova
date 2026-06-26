import React, { useState, useEffect } from 'react';
import type { Artist } from '../types';
import { apiService } from '../services/api';

const ArtistsPage: React.FC = () => {
  const [artists, setArtists] = useState<Artist[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiService.fetchArtists(50, 0)
      .then(data => {
        setArtists(data || []);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch artists:", err);
        setLoading(false);
      });
  }, []);

  return (
    <div className="content-scroll">
      <div style={{ padding: '32px' }}>
        <h1 style={{ fontSize: '32px', fontWeight: 800, marginBottom: '32px', letterSpacing: '-1px' }}>Artists</h1>
        
        {loading ? (
          <p style={{ color: 'var(--text-muted)' }}>Loading artists...</p>
        ) : artists.length > 0 ? (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: '24px' }}>
            {artists.map(artist => (
              <div 
                key={artist.id} 
                style={{ 
                  background: 'var(--bg-glass)', 
                  padding: '24px', 
                  borderRadius: '16px', 
                  border: '1px solid var(--border-glass-bright)', 
                  display: 'flex', 
                  flexDirection: 'column', 
                  alignItems: 'center',
                  transition: 'var(--transition-fast)',
                  cursor: 'pointer'
                }}
                onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-4px)'; e.currentTarget.style.background = 'var(--bg-glass-hover)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.background = 'var(--bg-glass)'; }}
              >
                <div style={{ width: '120px', height: '120px', borderRadius: '50%', background: 'var(--bg-secondary)', marginBottom: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '48px', fontWeight: 800, color: 'var(--text-muted)' }}>
                  {artist.name.charAt(0)}
                </div>
                <h3 style={{ fontSize: '16px', fontWeight: 700, textAlign: 'center' }}>{artist.name}</h3>
              </div>
            ))}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', padding: '64px', color: 'var(--text-secondary)' }}>
            <div style={{ fontSize: '64px', marginBottom: '16px', opacity: 0.2 }}>🎤</div>
            <h3 style={{ fontSize: '24px', fontWeight: 600, color: 'var(--text-primary)' }}>No Artists Found</h3>
            <p>Your library doesn't contain any artists yet.</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ArtistsPage;
