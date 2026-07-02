import React, { useState, useEffect } from 'react';
import { Play, Search, Radio } from 'lucide-react';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from '../components/HeartButton';

const RadioPage: React.FC = () => {
  const [query, setQuery] = useState('');
  const [stations, setStations] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [recentStations, setRecentStations] = useState<any[]>([]);

  useEffect(() => {
    const stored = localStorage.getItem('recentRadioStations');
    if (stored) {
      try {
        setRecentStations(JSON.parse(stored));
      } catch (e) { /* ignore */ }
    }
  }, []);

  const { internalPlay } = usePlayer() as any;

  const searchStations = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query) return;
    setLoading(true);
    setError(null);
    setOffset(0);
    try {
      const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
      const response = await fetch(`${API_BASE_URL}/api/plugins/radio/search?q=${encodeURIComponent(query)}&limit=50&offset=0`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      if (!response.ok) throw new Error('Search failed');
      const data = await response.json();
      setStations(data || []);
      setHasMore((data || []).length === 50);
    } catch (err) {
      console.error(err);
      setError("Failed to search radio stations. Ensure the Radio plugin is enabled in Settings.");
    } finally {
      setLoading(false);
    }
  };

  const loadMore = async () => {
    if (!query || loading) return;
    setLoading(true);
    const newOffset = offset + 50;
    try {
      const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
      const response = await fetch(`${API_BASE_URL}/api/plugins/radio/search?q=${encodeURIComponent(query)}&limit=50&offset=${newOffset}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      if (!response.ok) throw new Error('Search failed');
      const data = await response.json();
      setStations([...stations, ...(data || [])]);
      setOffset(newOffset);
      setHasMore((data || []).length === 50);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const playStation = (station: any) => {
    // Mock a track object so internalPlay can play it
    const mockTrack = {
      id: `radio-${station.stationuuid}`,
      title: station.name,
      artist_name: station.country ? `${station.country} Radio` : "Internet Radio",
      duration_ms: 0,
      stream_url: station.url_resolved || station.url
    };
    const mockAlbum = {
      id: 'radio',
      title: station.tags ? station.tags.split(',')[0] : 'Radio Station',
      cover_art_url: station.favicon || ''
    };
    
    // Call internalPlay directly with our mocked track if exposed, 
    // otherwise we need to modify PlayerContext to handle radio playing.
    if (internalPlay) {
      internalPlay(mockTrack, mockAlbum);
    }

    const newRecents = [station, ...recentStations.filter(s => s.stationuuid !== station.stationuuid)].slice(0, 20);
    setRecentStations(newRecents);
    localStorage.setItem('recentRadioStations', JSON.stringify(newRecents));
  };

  return (
    <div className="content-scroll">
      <div className="page-container">
        <div className="page-header">
          <h1 className="page-title">
            <Radio size={32} style={{ marginRight: '16px', color: 'var(--accent-primary)' }} />
            Internet Radio
          </h1>
          <p className="page-subtitle">Search and stream thousands of global internet radio stations.</p>
        </div>

        <form onSubmit={searchStations} style={{ display: 'flex', gap: '16px', marginBottom: '32px' }}>
          <div style={{ position: 'relative', flex: 1, maxWidth: '500px' }}>
            <Search size={20} style={{ position: 'absolute', left: '16px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
            <input 
              type="text" 
              placeholder="Search stations, genres, or countries..." 
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              style={{
                width: '100%',
                padding: '16px 24px 16px 48px',
                borderRadius: '24px',
                border: '1px solid var(--border-glass)',
                background: 'var(--bg-glass)',
                color: 'var(--text-primary)',
                fontSize: '16px',
                outline: 'none'
              }}
            />
          </div>
          <button 
            type="submit" 
            className="btn btn-primary"
            style={{ padding: '0 32px', borderRadius: '24px' }}
          >
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>

        {error && (
          <div style={{ color: '#ff4444', marginBottom: '24px', padding: '16px', background: 'rgba(255,68,68,0.1)', borderRadius: '12px' }}>
            {error}
          </div>
        )}

        {stations.length === 0 && recentStations.length > 0 && !loading && !error && (
          <div style={{ marginBottom: '48px' }}>
            <h2 className="section-title">Recently Played Stations</h2>
                <div className="album-grid">
                  {recentStations.map(station => (
                    <div key={`recent-${station.stationuuid}`} className="album-card" onClick={() => playStation(station)} style={{ cursor: 'pointer' }}>
                      <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                        {station.favicon ? (
                          <img 
                            src={station.favicon} 
                            alt={station.name} 
                            style={{ objectFit: 'contain', width: '70%', height: '70%' }}
                            onError={(e) => {
                              e.currentTarget.style.display = 'none';
                              if (e.currentTarget.nextElementSibling) {
                                (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'block';
                              }
                            }}
                          />
                        ) : null}
                        <div style={{ display: station.favicon ? 'none' : 'block' }}>
                          <Radio size={48} color="var(--text-muted)" />
                        </div>
                        <div className="play-overlay">
                          <button className="play-btn">
                            <Play size={24} fill="currentColor" />
                          </button>
                        </div>
                      </div>
                      <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                        <div style={{ overflow: 'hidden' }}>
                          <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{station.name}</h3>
                          <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                            {station.country || 'Unknown Location'} • {station.tags ? station.tags.split(',')[0] : 'Radio'}
                          </p>
                        </div>
                        <div onClick={(e) => e.stopPropagation()} style={{ marginLeft: '8px' }}>
                          <HeartButton entityType="radio" entityId={station.stationuuid} />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
          </div>
        )}

        <div className="album-grid">
          {stations.map(station => (
            <div key={station.stationuuid} className="album-card" onClick={() => playStation(station)} style={{ cursor: 'pointer' }}>
              <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                {station.favicon ? (
                  <img 
                    src={station.favicon} 
                    alt={station.name} 
                    style={{ objectFit: 'contain', width: '70%', height: '70%' }}
                    onError={(e) => {
                      e.currentTarget.style.display = 'none';
                      if (e.currentTarget.nextElementSibling) {
                        (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'block';
                      }
                    }}
                  />
                ) : null}
                <div style={{ display: station.favicon ? 'none' : 'block' }}>
                  <Radio size={48} color="var(--text-muted)" />
                </div>
                <div className="play-overlay">
                  <button className="play-btn">
                    <Play size={24} fill="currentColor" />
                  </button>
                </div>
              </div>
              <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                <div style={{ overflow: 'hidden' }}>
                  <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{station.name}</h3>
                  <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {station.country || 'Unknown Location'} • {station.tags ? station.tags.split(',')[0] : 'Radio'}
                  </p>
                </div>
                <div onClick={(e) => e.stopPropagation()} style={{ marginLeft: '8px' }}>
                  <HeartButton entityType="radio" entityId={station.stationuuid} />
                </div>
              </div>
            </div>
          ))}
        </div>

        {hasMore && (
          <div style={{ display: 'flex', justifyContent: 'center', marginTop: '32px' }}>
            <button 
              className="btn btn-secondary" 
              onClick={loadMore}
              disabled={loading}
              style={{ padding: '12px 32px', borderRadius: '24px' }}
            >
              {loading ? 'Loading...' : 'Load More'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default RadioPage;
