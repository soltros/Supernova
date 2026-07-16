import React, { useState, useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { usePlayer } from '../context/PlayerContext';
import { Minimize2 } from 'lucide-react';
import { apiService } from '../services/api';

const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

interface FullScreenPlayerProps {
  isOpen: boolean;
  onClose: () => void;
}

const FullScreenPlayer: React.FC<FullScreenPlayerProps> = ({ isOpen, onClose }) => {
  const { currentTrack, currentAlbum, audioElement } = usePlayer();
  const [currentTime, setCurrentTime] = useState(0);
  const [syncedLines, setSyncedLines] = useState<{time: number, text: string}[]>([]);
  const [plainLyrics, setPlainLyrics] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!audioElement) return;
    const updateTime = () => setCurrentTime(audioElement.currentTime);
    audioElement.addEventListener('timeupdate', updateTime);
    return () => audioElement.removeEventListener('timeupdate', updateTime);
  }, [audioElement]);

  useEffect(() => {
    if (isOpen) {
      if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen().catch(err => {
          console.error("Error attempting to enable fullscreen:", err);
        });
      }
    } else {
      if (document.fullscreenElement) {
        document.exitFullscreen().catch(err => {
          console.error("Error attempting to exit fullscreen:", err);
        });
      }
    }
  }, [isOpen]);

  useEffect(() => {
    const handleFullscreenChange = () => {
      if (!document.fullscreenElement && isOpen) onClose();
    };
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange);
  }, [isOpen, onClose]);

  // Fetch lyrics
  useEffect(() => {
    if (!isOpen || !currentTrack || !currentAlbum) return;
    setLoading(true);
    setError(null);
    setSyncedLines([]);
    setPlainLyrics(null);
    
    apiService.getLyrics(
      currentTrack.artist_name || '',
      currentTrack.title || '',
      currentAlbum.title || '',
      currentTrack.duration_ms / 1000
    ).then(data => {
      if (data.syncedLyrics) {
        const lines = data.syncedLyrics.split('\n');
        const parsed = lines.map((line: string) => {
          const match = line.match(/\[(\d{2}):(\d{2}\.\d{2})\](.*)/);
          if (match) {
            return { time: parseInt(match[1]) * 60 + parseFloat(match[2]), text: match[3].trim() };
          }
          return null;
        }).filter((l: any) => l !== null) as {time: number, text: string}[];
        setSyncedLines(parsed);
      } else if (data.plainLyrics) {
        setPlainLyrics(data.plainLyrics);
      } else {
        setError("No lyrics found");
      }
    }).catch(() => {
      setError("Lyrics not found for this track.");
    }).finally(() => setLoading(false));
  }, [isOpen, currentTrack, currentAlbum]);

  let activeIndex = -1;
  for (let i = 0; i < syncedLines.length; i++) {
    if (currentTime >= syncedLines[i].time) activeIndex = i;
    else break;
  }

  useEffect(() => {
    if (activeIndex !== -1 && containerRef.current) {
      const activeEl = containerRef.current.querySelector('.lyric-line.active');
      if (activeEl) {
        activeEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    }
  }, [activeIndex]);

  if (!isOpen) return null;

  const coverUrl = currentAlbum 
    ? (currentAlbum.cover_art_url ? currentAlbum.cover_art_url : `${API_BASE_URL}/api/art/album/${currentAlbum.id}`)
    : '';

  return createPortal(
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      backgroundColor: '#0a0a0f', zIndex: 9999,
      display: 'flex', color: '#fff', animation: 'fadeIn 0.3s ease'
    }}>
      {/* Background Blur */}
      {coverUrl && (
        <div style={{
          position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
          backgroundImage: `url(${coverUrl})`, backgroundSize: 'cover', backgroundPosition: 'center',
          filter: 'blur(100px) brightness(0.2)', zIndex: -1
        }} />
      )}

      {/* Top Bar with Minimize */}
      <button 
        onClick={onClose}
        style={{
          position: 'absolute', top: '40px', right: '40px',
          background: 'rgba(255,255,255,0.1)', border: 'none', color: '#fff',
          borderRadius: '50%', padding: '12px', cursor: 'pointer',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          transition: 'background 0.2s ease', zIndex: 10
        }}
        onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.2)'}
        onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
      >
        <Minimize2 size={24} />
      </button>

      {/* Split Content */}
      <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '60px' }}>
        
        {/* Left Column: Art & Info */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', maxWidth: '600px' }}>
          <div style={{
            width: '100%', maxWidth: 'min(500px, 50vh)', aspectRatio: '1/1',
            borderRadius: '16px', overflow: 'hidden',
            boxShadow: '0 20px 60px rgba(0,0,0,0.6)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(255,255,255,0.05)',
            marginBottom: '40px'
          }}>
            {coverUrl ? (
              <img src={coverUrl} alt="Album Art" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
            ) : (
              <span style={{ fontSize: '64px', color: 'rgba(255,255,255,0.2)' }}>♪</span>
            )}
          </div>
          <div style={{ textAlign: 'center', width: '100%' }}>
            <h1 style={{ fontSize: '32px', fontWeight: 800, margin: '0 0 8px 0', textShadow: '0 2px 10px rgba(0,0,0,0.5)' }}>
              {currentTrack?.title || 'No Track Selected'}
            </h1>
            <p style={{ fontSize: '20px', color: 'rgba(255,255,255,0.7)', margin: 0, fontWeight: 500 }}>
              {currentTrack?.artist_name || 'Unknown Artist'}
            </p>
          </div>
        </div>

        {/* Right Column: Lyrics */}
        <div 
          ref={containerRef} 
          className="content-scroll" 
          style={{ flex: 1, height: '100%', overflowY: 'auto', display: 'flex', flexDirection: 'column', padding: '20px 40px', maskImage: 'linear-gradient(to bottom, transparent 0%, black 15%, black 85%, transparent 100%)', WebkitMaskImage: 'linear-gradient(to bottom, transparent 0%, black 15%, black 85%, transparent 100%)' }}
        >
          {loading && <div style={{ margin: 'auto', color: 'rgba(255,255,255,0.5)' }}>Loading lyrics...</div>}
          {error && !loading && <div style={{ margin: 'auto', color: 'rgba(255,255,255,0.5)' }}>{error}</div>}
          
          {syncedLines.length > 0 && !loading && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', padding: '50vh 0', alignItems: 'center' }}>
              {syncedLines.map((line, idx) => {
                const isActive = idx === activeIndex;
                const isPassed = idx < activeIndex;
                return (
                  <div 
                    key={idx} 
                    className={`lyric-line ${isActive ? 'active' : ''}`}
                    style={{
                      fontSize: isActive ? '36px' : '28px',
                      fontWeight: isActive ? 700 : 600,
                      color: isActive ? '#fff' : (isPassed ? 'rgba(255,255,255,0.3)' : 'rgba(255,255,255,0.5)'),
                      textAlign: 'center',
                      transition: 'all 0.3s ease',
                      transform: isActive ? 'scale(1.05)' : 'scale(1)',
                      lineHeight: '1.4'
                    }}
                  >
                    {line.text || '♪'}
                  </div>
                );
              })}
            </div>
          )}

          {plainLyrics && !syncedLines.length && !loading && (
            <div style={{ margin: 'auto', fontSize: '24px', lineHeight: '1.8', color: 'rgba(255,255,255,0.7)', textAlign: 'center', whiteSpace: 'pre-wrap', padding: '50px 0' }}>
              {plainLyrics}
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
};

export default FullScreenPlayer;
