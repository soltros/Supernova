import React, { useState, useEffect } from 'react';
import { Play, Search, Radio, Plus, Check } from 'lucide-react';
import { usePlayer } from '../context/PlayerContext';
import { apiService } from '../services/api';
import HeartButton from '../components/HeartButton';

const RadioPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'subscriptions'|'discover'>('subscriptions');
  const [query, setQuery] = useState('');
  const [country, setCountry] = useState('');
  const [stations, setStations] = useState<any[]>([]);
  const [subscriptions, setSubscriptions] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  
  const { internalPlay } = usePlayer() as any;

  useEffect(() => {
    fetchSubscriptions();
  }, []);

  const fetchSubscriptions = async () => {
    try {
      const subs = await apiService.getRadioSubscriptions();
      setSubscriptions(subs || []);
    } catch (err) {
      console.error(err);
    }
  };

  const handleSubscribe = async (e: React.MouseEvent, station: any) => {
    e.stopPropagation();
    try {
      await apiService.subscribeToRadio(station.stationuuid, station.url_resolved || station.url, station.name, station.favicon);
      await fetchSubscriptions();
    } catch (err) {
      console.error('Failed to subscribe:', err);
    }
  };

  const handleUnsubscribe = async (e: React.MouseEvent, stationId: string) => {
    e.stopPropagation();
    try {
      await apiService.unsubscribeFromRadio(stationId);
      await fetchSubscriptions();
    } catch (err) {
      console.error('Failed to unsubscribe:', err);
    }
  };

  const searchStations = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query && !country) return;
    setLoading(true);
    setError(null);
    setOffset(0);
    try {
      const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
      const response = await fetch(`${API_BASE_URL}/api/plugins/radio/search?q=${encodeURIComponent(query)}&country=${encodeURIComponent(country)}&limit=50&offset=0`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('sn_token')}`
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
    if ((!query && !country) || loading) return;
    setLoading(true);
    const newOffset = offset + 50;
    try {
      const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
      const response = await fetch(`${API_BASE_URL}/api/plugins/radio/search?q=${encodeURIComponent(query)}&country=${encodeURIComponent(country)}&limit=50&offset=${newOffset}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('sn_token')}`
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

  const playStation = (station: any, isSub: boolean = false) => {
    // Determine IDs and fields depending on if it's from search or subscription
    const uuid = isSub ? station.station_id : station.stationuuid;
    const name = station.name;
    const streamUrl = isSub ? station.url : (station.url_resolved || station.url);
    const favicon = station.favicon;
    const locationStr = isSub ? "Subscribed Radio" : (station.country ? `${station.country} Radio` : "Internet Radio");
    const tags = isSub ? "Radio" : (station.tags ? station.tags.split(',')[0] : 'Radio');

    const mockTrack = {
      id: `radio-${uuid}`,
      title: name,
      artist_name: locationStr,
      duration_ms: 0,
      stream_url: streamUrl
    };
    const mockAlbum = {
      id: 'radio',
      title: tags,
      cover_art_url: favicon || ''
    };
    
    if (internalPlay) {
      internalPlay(mockTrack, mockAlbum);
    }
  };

  return (
    <div className="content-scroll">
      <div className="page-container">
        <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <h1 className="page-title">
              <Radio size={32} style={{ marginRight: '16px', color: 'var(--accent-primary)' }} />
              Internet Radio
            </h1>
            <p className="page-subtitle">Stream and subscribe to thousands of global radio stations.</p>
          </div>
        </div>

        <div className="tabs" style={{ display: 'flex', gap: '24px', borderBottom: '1px solid var(--border-subtle)', marginBottom: '32px' }}>
          <button 
            className={`tab-btn ${activeTab === 'subscriptions' ? 'active' : ''}`}
            onClick={() => setActiveTab('subscriptions')}
            style={{
              background: 'none',
              border: 'none',
              padding: '12px 0',
              color: activeTab === 'subscriptions' ? 'var(--accent-primary)' : 'var(--text-muted)',
              borderBottom: activeTab === 'subscriptions' ? '2px solid var(--accent-primary)' : '2px solid transparent',
              cursor: 'pointer',
              fontWeight: 600,
              fontSize: '16px'
            }}
          >
            Subscriptions
          </button>
          <button 
            className={`tab-btn ${activeTab === 'discover' ? 'active' : ''}`}
            onClick={() => setActiveTab('discover')}
            style={{
              background: 'none',
              border: 'none',
              padding: '12px 0',
              color: activeTab === 'discover' ? 'var(--accent-primary)' : 'var(--text-muted)',
              borderBottom: activeTab === 'discover' ? '2px solid var(--accent-primary)' : '2px solid transparent',
              cursor: 'pointer',
              fontWeight: 600,
              fontSize: '16px'
            }}
          >
            Discover
          </button>
        </div>

        {activeTab === 'discover' && (
          <>
            <form onSubmit={searchStations} style={{ display: 'flex', gap: '16px', marginBottom: '32px' }}>
              <div style={{ position: 'relative', flex: 1, maxWidth: '500px' }}>
                <Search size={20} style={{ position: 'absolute', left: '16px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                <input 
                  type="text" 
                  placeholder="Search stations or genres..." 
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
              <div style={{ position: 'relative', flex: 0.5, maxWidth: '250px' }}>
                <input 
                  type="text" 
                  placeholder="Country..." 
                  value={country}
                  onChange={(e) => setCountry(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '16px 24px 16px 24px',
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

            <div className="album-grid">
              {stations.map(station => {
                const isSubscribed = subscriptions.some(s => s.station_id === station.stationuuid);

                return (
                  <div key={station.stationuuid} className="album-card" onClick={() => playStation(station, false)} style={{ cursor: 'pointer' }}>
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
                      <div style={{ overflow: 'hidden', flex: 1 }}>
                        <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{station.name}</h3>
                        <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {station.country || 'Unknown Location'} • {station.tags ? station.tags.split(',')[0] : 'Radio'}
                        </p>
                      </div>
                      <div style={{ display: 'flex', gap: '8px', alignItems: 'center', marginLeft: '8px' }}>
                        <button 
                          className="btn-icon" 
                          onClick={(e) => isSubscribed ? handleUnsubscribe(e, station.stationuuid) : handleSubscribe(e, station)}
                          style={{ 
                            color: isSubscribed ? '#00ff88' : 'var(--text-muted)',
                            background: isSubscribed ? 'rgba(0,255,136,0.1)' : 'transparent',
                            borderRadius: '50%',
                            padding: '6px'
                          }}
                          title={isSubscribed ? "Unsubscribe" : "Subscribe"}
                        >
                          {isSubscribed ? <Check size={18} /> : <Plus size={18} />}
                        </button>
                        <div onClick={(e) => e.stopPropagation()}>
                          <HeartButton entityType="radio" entityId={station.stationuuid} metadata={station} />
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
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
          </>
        )}

        {activeTab === 'subscriptions' && (
          <>
            {subscriptions.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '64px 24px', background: 'var(--bg-glass)', borderRadius: '16px', border: '1px dashed var(--border-glass)' }}>
                <Radio size={48} style={{ color: 'var(--text-muted)', marginBottom: '16px' }} />
                <h2 style={{ margin: '0 0 8px 0', fontSize: '24px' }}>No Radio Subscriptions</h2>
                <p style={{ margin: 0, color: 'var(--text-muted)', fontSize: '16px' }}>
                  Head over to the Discover tab to find and subscribe to global internet radio stations.
                </p>
                <button 
                  className="btn btn-primary"
                  onClick={() => setActiveTab('discover')}
                  style={{ marginTop: '24px' }}
                >
                  Discover Stations
                </button>
              </div>
            ) : (
              <div className="album-grid">
                {subscriptions.map(sub => (
                  <div key={sub.station_id} className="album-card" onClick={() => playStation(sub, true)} style={{ cursor: 'pointer' }}>
                    <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                      {sub.favicon ? (
                        <img 
                          src={sub.favicon} 
                          alt={sub.name} 
                          style={{ objectFit: 'contain', width: '70%', height: '70%' }}
                          onError={(e) => {
                            e.currentTarget.style.display = 'none';
                            if (e.currentTarget.nextElementSibling) {
                              (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'block';
                            }
                          }}
                        />
                      ) : null}
                      <div style={{ display: sub.favicon ? 'none' : 'block' }}>
                        <Radio size={48} color="var(--text-muted)" />
                      </div>
                      <div className="play-overlay">
                        <button className="play-btn">
                          <Play size={24} fill="currentColor" />
                        </button>
                      </div>
                    </div>
                    <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                      <div style={{ overflow: 'hidden', flex: 1 }}>
                        <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{sub.name}</h3>
                        <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          Subscribed Station
                        </p>
                      </div>
                      <div style={{ display: 'flex', gap: '8px', alignItems: 'center', marginLeft: '8px' }}>
                        <button 
                          className="btn-icon" 
                          onClick={(e) => handleUnsubscribe(e, sub.station_id)}
                          style={{ 
                            color: '#00ff88',
                            background: 'rgba(0,255,136,0.1)',
                            borderRadius: '50%',
                            padding: '6px'
                          }}
                          title="Unsubscribe"
                        >
                          <Check size={18} />
                        </button>
                        <div onClick={(e) => e.stopPropagation()}>
                          <HeartButton entityType="radio" entityId={sub.station_id} />
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default RadioPage;
