import React, { useState, useEffect } from 'react';
import DOMPurify from 'dompurify';
import { Play, Search, Mic, ChevronLeft, Download, Upload, Plus, Check } from 'lucide-react';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from '../components/HeartButton';
import { apiService } from '../services/api';

const PodcastsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'subscriptions' | 'search'>('subscriptions');
  const [query, setQuery] = useState('');
  const [podcasts, setPodcasts] = useState<any[]>([]);
  const [subscriptions, setSubscriptions] = useState<any[]>([]);
  const [selectedPodcast, setSelectedPodcast] = useState<any | null>(null);
  const [episodes, setEpisodes] = useState<any[]>([]);
  const [progressData, setProgressData] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { internalPlay } = usePlayer();

  const loadSubscriptions = async () => {
    try {
      const subs = await apiService.getPodcastSubscriptions();
      setSubscriptions(subs || []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    if (activeTab === 'subscriptions' && !selectedPodcast) {
      loadSubscriptions();
    }
  }, [activeTab, selectedPodcast]);

  const searchPodcasts = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!query) return;
    
    setLoading(true);
    setError(null);
    setSelectedPodcast(null);
    
    try {
      const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
      const response = await fetch(`${API_BASE_URL}/api/plugins/podcasts/search?q=${encodeURIComponent(query)}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Podcast Index API keys are missing. Please add PODCAST_INDEX_API_KEY and PODCAST_INDEX_API_SECRET to your .env file.');
        }
        throw new Error('Search failed');
      }
      const data = await response.json();
      setPodcasts(data || []);
      setActiveTab('search');
    } catch (err: any) {
      console.error(err);
      setError(err.message || "Failed to search podcasts. Ensure the Podcasts plugin is enabled in Settings.");
    } finally {
      setLoading(false);
    }
  };

  const loadEpisodes = async (podcast: any) => {
    setLoading(true);
    setError(null);
    try {
      const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
      // Support both PodcastIndex format (id) and our subscription format (feed_id)
      const feedId = podcast.feed_id || podcast.id;
      
      const response = await fetch(`${API_BASE_URL}/api/plugins/podcasts/episodes?id=${feedId}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Podcast Index API keys are missing.');
        }
        throw new Error('Failed to fetch episodes');
      }
      const data = await response.json();
      const epData = data || [];
      setEpisodes(epData);
      
      // Also fetch progress for these episodes
      const epIds = epData.map((e: any) => e.id.toString());
      if (epIds.length > 0) {
        const prog = await apiService.getPodcastProgressBatch(epIds);
        setProgressData(prog || {});
      }
      
      setSelectedPodcast(podcast);
    } catch (err: any) {
      console.error(err);
      setError(err.message || "Failed to load episodes.");
    } finally {
      setLoading(false);
    }
  };

  const playEpisode = (episode: any) => {
    const mockTrack = {
      id: `podcast-${episode.id}`,
      title: episode.title,
      artist_name: selectedPodcast.title,
      duration_ms: episode.duration * 1000,
      stream_url: episode.enclosureUrl,
      album_id: 'podcast',
      track_number: 0,
      disc_number: 0,
      format: 'podcast',
      bitrate: 128
    };
    const mockAlbum = {
      id: 'podcast',
      title: 'Podcast Episode',
      cover_art_url: episode.image || selectedPodcast.image || selectedPodcast.image_url || ''
    };
    
    // If we have progress, we'd ideally tell the player to start at that position
    // (This requires PlayerContext modification which we'll handle separately)
    
    if (internalPlay) {
      internalPlay(mockTrack, mockAlbum, {
        podcast_episode_id: episode.id.toString(),
        start_position_ms: progressData[episode.id.toString()]?.position_ms || 0
      });
    }
  };

  const handleSubscribe = async (e: React.MouseEvent, podcast: any) => {
    e.stopPropagation();
    try {
      const feedId = podcast.feed_id || podcast.id?.toString();
      await apiService.subscribeToPodcast(feedId, podcast.url, podcast.title, podcast.image);
      loadSubscriptions();
    } catch (err) {
      console.error(err);
    }
  };

  const handleUnsubscribe = async (e: React.MouseEvent, feedId: string) => {
    e.stopPropagation();
    try {
      await apiService.unsubscribeFromPodcast(feedId);
      loadSubscriptions();
    } catch (err) {
      console.error(err);
    }
  };

  const isSubscribed = (id: string) => {
    const checkId = id?.toString();
    return subscriptions.some(s => s.feed_id === checkId);
  };

  const handleImportOPML = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    try {
      await apiService.importOPML(e.target.files[0]);
      alert("OPML import started in the background. Your subscriptions will appear here shortly.");
    } catch (err) {
      console.error(err);
      alert("Failed to import OPML.");
    }
    e.target.value = '';
  };

  const formatProgress = (durationSec: number, positionMs: number) => {
    if (!positionMs) return '';
    const totalMs = durationSec * 1000;
    if (totalMs <= 0) return '';
    const pct = Math.min(100, Math.floor((positionMs / totalMs) * 100));
    return `${pct}%`;
  };

  return (
    <div className="content-scroll">
      <div className="page-container">
        <div className="page-header" style={{ display: 'flex', alignItems: 'center', gap: '16px', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            {selectedPodcast && (
              <button 
                onClick={() => setSelectedPodcast(null)}
                className="btn"
                style={{ background: 'var(--bg-secondary)', padding: '12px', borderRadius: '50%' }}
              >
                <ChevronLeft size={24} />
              </button>
            )}
            <div>
              <h1 className="page-title">
                <Mic size={32} style={{ marginRight: '16px', color: 'var(--accent-primary)' }} />
                {selectedPodcast ? selectedPodcast.title : "Podcasts"}
              </h1>
              <p className="page-subtitle">
                {selectedPodcast ? "Episodes" : "Manage subscriptions and discover feeds"}
              </p>
            </div>
          </div>
          
          {!selectedPodcast && (
            <div style={{ display: 'flex', gap: '12px' }}>
              <input type="file" id="opml-upload" accept=".opml,.xml" style={{ display: 'none' }} onChange={handleImportOPML} />
              <label htmlFor="opml-upload" style={{ display: 'flex', alignItems: 'center', gap: '8px', background: 'var(--bg-glass)', border: '1px solid var(--border-glass-bright)', cursor: 'pointer', padding: '10px 16px', borderRadius: '12px', color: 'var(--text-primary)', fontWeight: 600, transition: 'var(--transition-fast)' }} onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-glass-hover)'} onMouseLeave={(e) => e.currentTarget.style.background = 'var(--bg-glass)'}>
                <Upload size={18} /> Import OPML
              </label>
              <button onClick={() => apiService.exportOPML()} style={{ display: 'flex', alignItems: 'center', gap: '8px', background: 'var(--accent-gradient)', padding: '10px 16px', borderRadius: '12px', border: 'none', color: 'white', fontWeight: 600, cursor: 'pointer', boxShadow: 'var(--accent-glow)' }}>
                <Download size={18} /> Export OPML
              </button>
            </div>
          )}
        </div>

        {!selectedPodcast && (
          <>
            <div style={{ display: 'flex', gap: '24px', marginBottom: '24px', borderBottom: '1px solid var(--border-glass-bright)' }}>
              <button 
                onClick={() => setActiveTab('subscriptions')}
                style={{ background: 'none', border: 'none', color: activeTab === 'subscriptions' ? 'var(--text-primary)' : 'var(--text-muted)', fontWeight: 600, fontSize: '16px', padding: '0 0 12px 0', borderBottom: activeTab === 'subscriptions' ? '2px solid var(--accent-primary)' : '2px solid transparent', cursor: 'pointer', transition: 'var(--transition-fast)' }}
              >
                Subscriptions
              </button>
              <button 
                onClick={() => setActiveTab('search')}
                style={{ background: 'none', border: 'none', color: activeTab === 'search' ? 'var(--text-primary)' : 'var(--text-muted)', fontWeight: 600, fontSize: '16px', padding: '0 0 12px 0', borderBottom: activeTab === 'search' ? '2px solid var(--accent-primary)' : '2px solid transparent', cursor: 'pointer', transition: 'var(--transition-fast)' }}
              >
                Discover
              </button>
            </div>

            {activeTab === 'search' && (
              <form onSubmit={searchPodcasts} style={{ display: 'flex', gap: '16px', marginBottom: '32px' }}>
                <div style={{ position: 'relative', flex: 1, maxWidth: '500px' }}>
                  <Search size={20} style={{ position: 'absolute', left: '16px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                  <input 
                    type="text" 
                    placeholder="Search PodcastIndex.org..." 
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
            )}
          </>
        )}

        {error && (
          <div style={{ color: '#ff4444', marginBottom: '24px', padding: '16px', background: 'rgba(255,68,68,0.1)', borderRadius: '12px' }}>
            {error}
          </div>
        )}

        {loading && <div className="loading-spinner" />}

        {!selectedPodcast && !loading && activeTab === 'subscriptions' && (
          <div className="album-grid">
            {subscriptions.length === 0 ? (
              <div style={{ gridColumn: '1 / -1', padding: '48px', textAlign: 'center', background: 'var(--bg-glass)', borderRadius: '24px', border: '1px solid var(--border-glass-bright)' }}>
                <p style={{ fontSize: '18px', color: 'var(--text-secondary)' }}>You aren't subscribed to any podcasts yet.</p>
                <button onClick={() => setActiveTab('search')} className="btn btn-primary" style={{ marginTop: '16px' }}>Discover Podcasts</button>
              </div>
            ) : (
              subscriptions.map(podcast => (
                <div key={`sub-${podcast.feed_id}`} className="album-card" onClick={() => loadEpisodes(podcast)} style={{ cursor: 'pointer' }}>
                  <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                    {podcast.image_url ? (
                      <img src={podcast.image_url} alt={podcast.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <Mic size={48} color="var(--text-muted)" />
                    )}
                  </div>
                  <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                    <div style={{ overflow: 'hidden' }}>
                      <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.title}</h3>
                    </div>
                    <div onClick={(e) => handleUnsubscribe(e, podcast.feed_id)} style={{ marginLeft: '8px' }}>
                      <button style={{ background: 'var(--bg-glass)', border: 'none', borderRadius: '50%', padding: '6px', cursor: 'pointer', color: 'var(--accent-primary)' }}>
                        <Check size={16} />
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {!selectedPodcast && !loading && activeTab === 'search' && podcasts.length > 0 && (
          <div className="album-grid">
            {podcasts.map(podcast => (
              <div key={`search-${podcast.id}`} className="album-card" onClick={() => loadEpisodes(podcast)} style={{ cursor: 'pointer' }}>
                <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                  {podcast.image ? (
                    <img src={podcast.image} alt={podcast.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  ) : (
                    <Mic size={48} color="var(--text-muted)" />
                  )}
                  <div className="album-play-overlay">
                    <Play fill="white" size={24} />
                  </div>
                </div>
                <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                  <div style={{ overflow: 'hidden' }}>
                    <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.title}</h3>
                    <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.author || 'Unknown'}</p>
                  </div>
                  <div style={{ marginLeft: '8px', display: 'flex', gap: '4px' }}>
                    <div onClick={(e) => isSubscribed(podcast.id) ? handleUnsubscribe(e, podcast.id?.toString()) : handleSubscribe(e, podcast)}>
                      <button style={{ background: isSubscribed(podcast.id) ? 'var(--bg-glass)' : 'transparent', border: '1px solid var(--border-glass-bright)', borderRadius: '50%', padding: '6px', cursor: 'pointer', color: isSubscribed(podcast.id) ? 'var(--accent-primary)' : 'var(--text-primary)' }}>
                        {isSubscribed(podcast.id) ? <Check size={16} /> : <Plus size={16} />}
                      </button>
                    </div>
                    <div onClick={(e) => e.stopPropagation()}>
                      <HeartButton entityType="podcast" entityId={podcast.id?.toString()} metadata={podcast} />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {selectedPodcast && !loading && (
          <div>
            <div style={{ display: 'flex', gap: '24px', marginBottom: '32px', background: 'var(--bg-glass)', padding: '24px', borderRadius: '16px', border: '1px solid var(--border-glass-bright)' }}>
              <div style={{ width: '150px', height: '150px', borderRadius: '8px', overflow: 'hidden', flexShrink: 0, background: 'var(--bg-secondary)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                {(selectedPodcast.image || selectedPodcast.image_url) ? (
                  <img src={selectedPodcast.image || selectedPodcast.image_url} alt={selectedPodcast.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                ) : (
                  <Mic size={48} color="var(--text-muted)" />
                )}
              </div>
              <div>
                <h2 style={{ fontSize: '24px', margin: '0 0 8px 0' }}>{selectedPodcast.title}</h2>
                {selectedPodcast.author && <p style={{ margin: '0 0 16px 0', color: 'var(--text-muted)' }}>{selectedPodcast.author}</p>}
                
                <div style={{ display: 'flex', gap: '12px' }}>
                  <button 
                    onClick={(e) => {
                      const feedId = selectedPodcast.feed_id || selectedPodcast.id?.toString();
                      isSubscribed(feedId) ? handleUnsubscribe(e, feedId) : handleSubscribe(e, selectedPodcast)
                    }}
                    style={{ background: isSubscribed(selectedPodcast.feed_id || selectedPodcast.id?.toString()) ? 'var(--bg-glass)' : 'var(--accent-gradient)', padding: '10px 20px', borderRadius: '24px', border: 'none', color: 'white', fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '8px' }}
                  >
                    {isSubscribed(selectedPodcast.feed_id || selectedPodcast.id?.toString()) ? <><Check size={18} /> Subscribed</> : <><Plus size={18} /> Subscribe</>}
                  </button>
                  <HeartButton entityType="podcast" entityId={(selectedPodcast.feed_id || selectedPodcast.id)?.toString()} metadata={selectedPodcast} />
                </div>
              </div>
            </div>

            <div className="track-list" style={{ marginTop: '24px' }}>
              {episodes.map((episode) => {
                const epId = episode.id.toString();
                const prog = progressData[epId];
                const pct = formatProgress(episode.duration, prog?.position_ms);
                
                return (
                  <div 
                    key={epId}
                    className="track-row"
                    onDoubleClick={() => playEpisode(episode)}
                    style={{ display: 'flex', alignItems: 'flex-start', padding: '16px', gap: '16px' }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flex: 1, overflow: 'hidden' }}>
                      <div 
                        className="play-btn-overlay"
                        onClick={() => playEpisode(episode)}
                        style={{ cursor: 'pointer', padding: '12px', background: 'var(--bg-glass)', borderRadius: '50%', flexShrink: 0 }}
                      >
                        <Play size={20} fill="var(--text-primary)" color="var(--text-primary)" />
                      </div>
                      
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontWeight: 600, marginBottom: '4px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {episode.title}
                        </div>
                        <div style={{ fontSize: '13px', color: 'var(--text-muted)', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }} dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(episode.description) }} />
                        
                        {prog && prog.position_ms > 0 && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '8px' }}>
                            <div style={{ height: '4px', background: 'var(--bg-secondary)', borderRadius: '2px', flex: 1, maxWidth: '200px', overflow: 'hidden' }}>
                              <div style={{ height: '100%', background: 'var(--accent-primary)', width: pct }} />
                            </div>
                            <span style={{ fontSize: '12px', color: 'var(--accent-primary)', fontWeight: 600 }}>{prog.completed ? 'Played' : pct}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default PodcastsPage;
