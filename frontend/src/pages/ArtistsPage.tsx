import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Mic2, Hash } from 'lucide-react';
import type { Artist } from '../types';
import { apiService } from '../services/api';

const LETTERS = ['#', ...Array.from({ length: 26 }, (_, i) => String.fromCharCode(65 + i))];

const ArtistsPage: React.FC = () => {
  const [artists, setArtists] = useState<Artist[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeLetter, setActiveLetter] = useState<string>('A');

  useEffect(() => {
    setLoading(true);
    apiService.fetchArtists(500, 0, activeLetter)
      .then(data => {
        setArtists(data || []);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch artists:", err);
        setLoading(false);
      });
  }, [activeLetter]);

  return (
    <div className="content-scroll">
      <div className="page-container">
        <h1 style={{ fontSize: '32px', fontWeight: 800, marginBottom: '24px', letterSpacing: '-1px' }}>Artists</h1>
        
        {/* Alphabetical Pagination Bar */}
        <div style={{ 
          display: 'flex', 
          flexWrap: 'wrap', 
          gap: '8px', 
          marginBottom: '32px', 
          background: 'var(--bg-glass)', 
          padding: '16px', 
          borderRadius: '16px', 
          border: '1px solid var(--border-glass-bright)' 
        }}>
          {LETTERS.map(letter => (
            <button
              key={letter}
              onClick={() => setActiveLetter(letter)}
              style={{
                width: '36px',
                height: '36px',
                borderRadius: '8px',
                border: 'none',
                background: activeLetter === letter ? 'var(--accent-primary)' : 'rgba(255,255,255,0.05)',
                color: activeLetter === letter ? '#fff' : 'var(--text-secondary)',
                fontWeight: 700,
                fontSize: '14px',
                cursor: 'pointer',
                transition: 'var(--transition-fast)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: activeLetter === letter ? '0 4px 12px rgba(139, 92, 246, 0.4)' : 'none'
              }}
              onMouseEnter={(e) => {
                if (activeLetter !== letter) {
                  e.currentTarget.style.background = 'rgba(255,255,255,0.1)';
                  e.currentTarget.style.color = 'var(--text-primary)';
                }
              }}
              onMouseLeave={(e) => {
                if (activeLetter !== letter) {
                  e.currentTarget.style.background = 'rgba(255,255,255,0.05)';
                  e.currentTarget.style.color = 'var(--text-secondary)';
                }
              }}
            >
              {letter}
            </button>
          ))}
        </div>

        {loading ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: '64px' }}>
            <div className="loader" style={{ width: '32px', height: '32px', borderWidth: '3px' }}></div>
          </div>
        ) : artists.length > 0 ? (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: '24px' }}>
            {artists.map(artist => (
              <Link 
                to={`/artist/${artist.id}`}
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
                  cursor: 'pointer',
                  textDecoration: 'none',
                  color: 'var(--text-primary)'
                }}
                onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-4px)'; e.currentTarget.style.background = 'var(--bg-glass-hover)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.background = 'var(--bg-glass)'; }}
              >
                <div style={{ width: '120px', height: '120px', borderRadius: '50%', background: 'var(--bg-secondary)', marginBottom: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '48px', fontWeight: 800, color: 'var(--text-muted)', overflow: 'hidden', boxShadow: '0 8px 16px rgba(0,0,0,0.3)' }}>
                  {artist.image_url && artist.image_url !== 'NOT_FOUND' ? (
                    <img src={artist.image_url} alt={artist.name} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  ) : (
                    artist.name.charAt(0)
                  )}
                </div>
                <h3 style={{ fontSize: '16px', fontWeight: 700, textAlign: 'center' }}>{artist.name}</h3>
              </Link>
            ))}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', height: '100%', color: 'var(--text-secondary)', padding: '64px' }}>
            <div style={{ marginBottom: '16px', opacity: 0.2 }}>
              <Mic2 size={64} />
            </div>
            <h3 style={{ fontSize: '24px', fontWeight: 600, color: 'var(--text-primary)' }}>No Artists Found</h3>
            <p>We couldn't find any artists starting with '{activeLetter}'.</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ArtistsPage;
