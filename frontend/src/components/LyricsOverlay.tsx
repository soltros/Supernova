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
      position: 'absolute',
      bottom: '100%',
      right: '24px',
      width: '360px',
      height: '500px',
      marginBottom: '16px',
      background: 'rgba(20, 20, 20, 0.95)',
      border: '1px solid rgba(255, 255, 255, 0.1)',
      borderRadius: '24px',
      boxShadow: '0 20px 40px rgba(0,0,0,0.5)',
      backdropFilter: 'blur(20px)',
      zIndex: 1000,
      display: 'flex',
      flexDirection: 'column',
      padding: '24px',
      color: '#fff',
      animation: 'slideUp 0.3s cubic-bezier(0.16, 1, 0.3, 1)'
    }}>
      <button 
        onClick={onClose}
        style={{ position: 'absolute', top: '16px', right: '16px', background: 'rgba(255,255,255,0.1)', border: 'none', color: '#fff', cursor: 'pointer', borderRadius: '50%', padding: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      >
        <X size={16} />
      </button>

      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <Mic2 size={24} color="var(--accent-primary)" />
        <h2 style={{ margin: 0, fontSize: '20px', fontWeight: 700 }}>Lyrics</h2>
      </div>

      <div ref={containerRef} className="content-scroll" style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        {loading && <div className="loader"></div>}
        
        {error && (
          <div style={{ color: 'var(--text-muted)', fontSize: '14px', textAlign: 'center', marginTop: '40px' }}>
            {error}
          </div>
        )}
        
        {lyricsData && !loading && (
          <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: '16px', paddingBottom: '40px' }}>
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
                      fontSize: isActive ? '20px' : '16px', 
                      fontWeight: isActive ? 800 : 500,
                      color: isActive ? '#fff' : (isPassed ? 'rgba(255,255,255,0.4)' : 'rgba(255,255,255,0.2)'),
                      margin: 0,
                      transition: 'all 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
                      textAlign: 'left',
                      lineHeight: '1.4',
                      transform: isActive ? 'scale(1.02)' : 'scale(1)',
                      transformOrigin: 'left center'
                    }}
                  >
                    {line.text || '♪'}
                  </p>
                );
              })
            ) : (
              // Plain text lyrics fallback
              <div style={{ fontSize: '15px', lineHeight: '1.8', color: 'rgba(255,255,255,0.8)', textAlign: 'left', whiteSpace: 'pre-wrap' }}>
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
