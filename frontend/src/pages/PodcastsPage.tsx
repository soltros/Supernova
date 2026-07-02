import React, { useState, useEffect } from 'react';
import { Play, Search, Mic, ChevronLeft } from 'lucide-react';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from '../components/HeartButton';

const PodcastsPage: React.FC = () => {
  const [query, setQuery] = useState('');
  const [podcasts, setPodcasts] = useState<any[]>([]);
  const [selectedPodcast, setSelectedPodcast] = useState<any | null>(null);
  const [episodes, setEpisodes] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [recentPodcasts, setRecentPodcasts] = useState<any[]>([]);

  const { internalPlay } = usePlayer() as any;

  useEffect(() => {
    const stored = localStorage.getItem('recentPodcasts');
    if (stored) {
      try {
        setRecentPodcasts(JSON.parse(stored));
      } catch (e) { /* ignore */ }
    }
  }, []);

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
      const response = await fetch(`${API_BASE_URL}/api/plugins/podcasts/episodes?id=${podcast.id}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Podcast Index API keys are missing. Please add PODCAST_INDEX_API_KEY and PODCAST_INDEX_API_SECRET to your .env file.');
        }
        throw new Error('Failed to fetch episodes');
      }
      const data = await response.json();
      setEpisodes(data || []);
      setSelectedPodcast(podcast);

      const newRecents = [podcast, ...recentPodcasts.filter(p => p.id !== podcast.id)].slice(0, 20);
      setRecentPodcasts(newRecents);
      localStorage.setItem('recentPodcasts', JSON.stringify(newRecents));
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
      stream_url: episode.enclosureUrl
    };
    const mockAlbum = {
      id: 'podcast',
      title: 'Podcast Episode',
      cover_art_url: episode.image || selectedPodcast.image || ''
    };
    
    if (internalPlay) {
      internalPlay(mockTrack, mockAlbum);
    }
  };

  return (
    <div className="content-scroll">
      <div className="page-container">
        <div className="page-header" style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
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
              {selectedPodcast ? "Episodes" : "Search and stream from PodcastIndex.org"}
            </p>
          </div>
        </div>

        {!selectedPodcast && (
          <form onSubmit={searchPodcasts} style={{ display: 'flex', gap: '16px', marginBottom: '32px' }}>
            <div style={{ position: 'relative', flex: 1, maxWidth: '500px' }}>
              <Search size={20} style={{ position: 'absolute', left: '16px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
              <input 
                type="text" 
                placeholder="Search podcasts..." 
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

        {error && (
          <div style={{ color: '#ff4444', marginBottom: '24px', padding: '16px', background: 'rgba(255,68,68,0.1)', borderRadius: '12px' }}>
            {error}
          </div>
        )}

        {loading && <div className="loading-spinner" />}

        {!selectedPodcast && !loading && (
          <>
            {podcasts.length === 0 && recentPodcasts.length > 0 && !error && (
              <div style={{ marginBottom: '48px' }}>
                <h2 className="section-title">Recently Viewed Podcasts</h2>
                <div className="album-grid">
                  {recentPodcasts.map(podcast => (
                    <div key={`recent-${podcast.id}`} className="album-card" onClick={() => loadEpisodes(podcast)} style={{ cursor: 'pointer' }}>
                      <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                        {podcast.image ? (
                          <img src={podcast.image} alt={podcast.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                        ) : (
                          <Mic size={48} color="var(--text-muted)" />
                        )}
                      </div>
                      <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                        <div style={{ overflow: 'hidden' }}>
                          <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.title}</h3>
                          <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.author || 'Unknown'}</p>
                        </div>
                        <div onClick={(e) => e.stopPropagation()} style={{ marginLeft: '8px' }}>
                          <HeartButton entityType="podcast" entityId={podcast.id?.toString()} metadata={podcast} />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="album-grid">
              {podcasts.map(podcast => (
                <div key={podcast.id} className="album-card" onClick={() => loadEpisodes(podcast)} style={{ cursor: 'pointer' }}>
                  <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-secondary)' }}>
                    {podcast.image ? (
                      <img src={podcast.image} alt={podcast.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <Mic size={48} color="var(--text-muted)" />
                    )}
                  </div>
                  <div className="album-info" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginTop: '12px' }}>
                    <div style={{ overflow: 'hidden' }}>
                      <h3 style={{ margin: '0 0 4px 0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.title}</h3>
                      <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{podcast.author || 'Unknown'}</p>
                    </div>
                    <div onClick={(e) => e.stopPropagation()} style={{ marginLeft: '8px' }}>
                      <HeartButton entityType="podcast" entityId={podcast.id?.toString()} metadata={podcast} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        {selectedPodcast && !loading && (
          <div className="track-list">
            {episodes.map((episode, index) => (
              <div 
                key={episode.id}
                className="track-row"
                onDoubleClick={() => playEpisode(episode)}
                style={{ display: 'flex', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid var(--border-glass)' }}
              >
                <div className="track-number" style={{ width: '40px', color: 'var(--text-muted)' }}>
                  {index + 1}
                </div>
                <div className="track-title-cell" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                  <div className="track-title-text" style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                    {episode.title}
                  </div>
                  <div style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '4px' }}>
                    {new Date(episode.datePublished * 1000).toLocaleDateString()}
                  </div>
                </div>
                <div style={{ width: '60px', textAlign: 'right', color: 'var(--text-muted)' }}>
                  {Math.floor(episode.duration / 60)}:{String(episode.duration % 60).padStart(2, '0')}
                </div>
                <button 
                  onClick={() => playEpisode(episode)}
                  className="btn"
                  style={{ marginLeft: '16px', background: 'var(--accent-primary)', color: 'white', padding: '8px', borderRadius: '50%' }}
                >
                  <Play size={16} fill="currentColor" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PodcastsPage;
