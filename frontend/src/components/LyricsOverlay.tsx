import React, { useEffect, useState, useRef } from 'react';
import { apiService } from '../services/api';
import { usePlayer } from '../context/PlayerContext';
import { X, Mic2 } from 'lucide-react';

interface LyricsOverlayProps {
  isOpen: boolean;
  onClose: () => void;
}

const LyricsOverlay: React.FC<LyricsOverlayProps> = ({ isOpen, onClose }) => {
  const { currentTrack, currentAlbum, audioElement } = usePlayer();
  const [lyricsData, setLyricsData] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentTime, setCurrentTime] = useState(0);
  
  // Track parsed synced lyrics
  const [syncedLines, setSyncedLines] = useState<{time: number, text: string}[]>([]);

  useEffect(() => {
    if (!isOpen) return;
    if (!currentTrack || !currentAlbum) {
      setError("No track playing");
      return;
    }

    setLoading(true);
    setError(null);
    setLyricsData(null);
    setSyncedLines([]);

    // Ensure required metadata exists before calling the API
    if (!currentTrack.artist_name || !currentTrack.title || !currentAlbum.title || currentTrack.duration_ms == null) {
      setError("Track metadata incomplete");
      setLoading(false);
      return;
    }

    // Fetch lyrics via plugin API
    apiService.getLyrics(
      currentTrack.artist_name,
      currentTrack.title,
      currentAlbum.title,
      currentTrack.duration_ms / 1000
    )
      .then(data => {
        setLyricsData(data);
        if (data.syncedLyrics) {
          parseSyncedLyrics(data.syncedLyrics);
        }
      })
      .catch(err => {
        console.error("Lyrics error:", err);
        setError("Lyrics not found for this track.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [isOpen, currentTrack, currentAlbum]);

  useEffect(() => {
    if (!isOpen || !audioElement) return;

    const handleTimeUpdate = () => {
      setCurrentTime(audioElement.currentTime);
    };

    audioElement.addEventListener('timeupdate', handleTimeUpdate);
    return () => audioElement.removeEventListener('timeupdate', handleTimeUpdate);
  }, [isOpen, audioElement]);

  const parseSyncedLyrics = (lrc: string) => {
    const lines = lrc.split('\n');
    const parsed = lines.map(line => {
      // match [mm:ss.xx]
      const match = line.match(/\[(\d{2}):(\d{2}\.\d{2})\](.*)/);
      if (match) {
        const mins = parseInt(match[1]);
        const secs = parseFloat(match[2]);
        const text = match[3].trim();
        return { time: mins * 60 + secs, text };
      }
      return null;
    }).filter(l => l !== null) as {time: number, text: string}[];
    
    setSyncedLines(parsed);
  };

  // Find active line
  let activeIndex = -1;
  for (let i = 0; i < syncedLines.length; i++) {
    if (currentTime >= syncedLines[i].time) {
      activeIndex = i;
    } else {
      break; // Since it's sorted chronologically
    }
  }

  // Auto-scroll to active line
  const containerRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (activeIndex !== -1 && containerRef.current) {
      const activeEl = containerRef.current.querySelector('.lyric-line.active');
      if (activeEl) {
        activeEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    }
  }, [activeIndex]);

  if (!isOpen) return null;

  return (
    <div style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: '90px', // above player bar
      background: 'rgba(0, 0, 0, 0.85)',
      backdropFilter: 'blur(20px)',
      zIndex: 1000,
      display: 'flex',
      flexDirection: 'column',
      padding: '40px',
      color: '#fff',
      animation: 'fadeIn 0.3s ease'
    }}>
      <button 
        onClick={onClose}
        style={{ position: 'absolute', top: '24px', right: '24px', background: 'transparent', border: 'none', color: '#fff', cursor: 'pointer' }}
      >
        <X size={32} />
      </button>

      <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '32px' }}>
        <Mic2 size={32} color="var(--accent-primary)" />
        <h2 style={{ margin: 0, fontSize: '28px', fontWeight: 800 }}>Lyrics</h2>
      </div>

      <div ref={containerRef} style={{ flex: 1, overflowY: 'auto', paddingRight: '20px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        {loading && <div className="loader"></div>}
        
        {error && (
          <div style={{ color: 'var(--text-muted)', fontSize: '18px', textAlign: 'center', marginTop: '40px' }}>
            {error}
          </div>
        )}
        
        {lyricsData && !loading && (
          <div style={{ width: '100%', maxWidth: '800px', display: 'flex', flexDirection: 'column', gap: '16px', paddingBottom: '100px' }}>
            {syncedLines.length > 0 ? (
              // Synced Lyrics rendering
              syncedLines.map((line, idx) => {
                const isActive = idx === activeIndex;
                const isPassed = idx < activeIndex;
                return (
                  <p 
                    key={idx} 
                    className={`lyric-line ${isActive ? 'active' : ''}`}
                    style={{ 
                      fontSize: isActive ? '36px' : '24px', 
                      fontWeight: isActive ? 800 : 600,
                      color: isActive ? '#fff' : (isPassed ? 'rgba(255,255,255,0.4)' : 'rgba(255,255,255,0.2)'),
                      margin: 0,
                      transition: 'all 0.3s ease',
                      textAlign: 'center',
                      lineHeight: '1.4',
                      transform: isActive ? 'scale(1.05)' : 'scale(1)'
                    }}
                  >
                    {line.text || '♪'}
                  </p>
                );
              })
            ) : (
              // Plain text lyrics fallback
              <div style={{ fontSize: '20px', lineHeight: '1.8', color: 'rgba(255,255,255,0.8)', textAlign: 'center', whiteSpace: 'pre-wrap' }}>
                {lyricsData.plainLyrics}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default LyricsOverlay;
