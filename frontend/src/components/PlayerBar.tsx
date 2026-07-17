import { useEffect, useState, useCallback } from 'react';
import type { FC, MouseEvent, ChangeEvent } from 'react';
import { Link } from 'react-router-dom';
import { Play, Pause, SkipBack, SkipForward, Volume2, VolumeX, ChevronDown, Mic2, Maximize } from 'lucide-react';
import { usePlayer } from '../context/PlayerContext';
import HeartButton from './HeartButton';
import LyricsOverlay from './LyricsOverlay';
import FullScreenPlayer from './FullScreenPlayer';

const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

const formatTime = (seconds: number) => {
  if (!seconds || isNaN(seconds)) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const PlayerBar: FC = () => {
  const { 
    currentTrack, currentAlbum, isPlaying, 
    duration, volume, audioElement,
    togglePlay, seekTo, playNext, playPrev, changeVolume 
  } = usePlayer();

  const [currentTime, setCurrentTime] = useState(0);
  const [progress, setProgress] = useState(0);
  const [isMobileExpanded, setIsMobileExpanded] = useState(false);
  const [showLyrics, setShowLyrics] = useState(false);
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [prevVolume, setPrevVolume] = useState(1);

  useEffect(() => {
    if (!audioElement) return;

    const handleTimeUpdate = () => {
      setCurrentTime(audioElement.currentTime);
      if (audioElement.duration && isFinite(audioElement.duration)) {
        setProgress((audioElement.currentTime / audioElement.duration) * 100);
      } else {
        setProgress(0);
      }
    };

    // Initial setup if we re-render while playing
    handleTimeUpdate();

    audioElement.addEventListener('timeupdate', handleTimeUpdate);
    return () => {
      audioElement.removeEventListener('timeupdate', handleTimeUpdate);
    };
  }, [audioElement]);

  const handleProgressClick = (e: MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const percent = ((e.clientX - rect.left) / rect.width) * 100;
    seekTo(percent);
  };

  const handleVolumeChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    changeVolume(parseFloat(e.target.value));
  }, [changeVolume]);

  // If the browser hasn't calculated duration yet, fallback to our database metadata
  const displayDuration = duration || (currentTrack ? currentTrack.duration_ms / 1000 : 0);

  return (
    <footer className={`player-bar ${isMobileExpanded ? 'expanded' : ''}`}>
      
      {/* Mobile Collapse Button */}
      {isMobileExpanded && (
        <button 
          className="mobile-collapse-btn"
          onClick={(e) => { e.stopPropagation(); setIsMobileExpanded(false); }}
        >
          <ChevronDown size={32} />
        </button>
      )}

      {/* Left Side: Now Playing Metadata */}
      <div 
        className="now-playing" 
        onClick={() => !isMobileExpanded && setIsMobileExpanded(true)}
      >
        <div className="album-art-container" style={{ position: 'relative', width: '60px', height: '60px', flexShrink: 0 }}>
          {currentTrack?.album_id || currentAlbum ? (
            <img 
              src={currentAlbum?.cover_art_url ? currentAlbum.cover_art_url : `${API_BASE_URL}/api/art/album/${currentTrack?.album_id || currentAlbum?.id}`} 
              alt="art"
              className={`track-art-small ${isPlaying ? 'playing' : ''}`}
              onError={(e) => {
                e.currentTarget.style.display = 'none';
                if (e.currentTarget.nextElementSibling) {
                  (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'flex';
                }
              }}
            />
          ) : null}
          <div className={`track-art-small ${isPlaying ? 'playing' : ''}`} style={{ display: (currentTrack?.album_id || currentAlbum) ? 'none' : 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <span style={{ color: 'var(--text-muted)' }}>♪</span>
          </div>
        </div>
        
        <div className="track-details" style={{ display: 'flex', flexDirection: 'column', gap: '2px', overflow: 'hidden', whiteSpace: 'nowrap' }}>
          <h4 style={{ margin: 0, textOverflow: 'ellipsis', overflow: 'hidden' }}>{currentTrack ? currentTrack.title : 'No Track Playing'}</h4>
          <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
            {currentTrack?.artist_id ? (
              <Link to={`/artist/${currentTrack.artist_id}`} style={{ color: 'var(--text-muted)', fontSize: '13px', textDecoration: 'none', transition: 'color 0.2s' }} onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'} onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-muted)'}>
                {currentTrack.artist_name}
              </Link>
            ) : currentAlbum?.artist_id ? (
              <Link to={`/artist/${currentAlbum.artist_id}`} style={{ color: 'var(--text-muted)', fontSize: '13px', textDecoration: 'none', transition: 'color 0.2s' }} onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'} onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-muted)'}>
                {currentAlbum.artist_name}
              </Link>
            ) : null}
            {(currentTrack?.artist_id || currentAlbum?.artist_id) && <span style={{ color: 'var(--text-muted)', fontSize: '10px' }}>•</span>}
            <span style={{ color: 'var(--text-muted)', fontSize: '12px', textOverflow: 'ellipsis', overflow: 'hidden' }}>
              {currentAlbum ? currentAlbum.title : 'Select a track to begin'}
            </span>
          </div>
        </div>
        
        {currentTrack && (
          <div className="heart-btn-container" style={{ marginLeft: '8px' }} onClick={(e) => e.stopPropagation()}>
            <HeartButton entityType="track" entityId={currentTrack.id} />
          </div>
        )}
      </div>

      {/* Mobile Mini Play Button (visible only in compact view) */}
      <div className="mobile-mini-controls" onClick={(e) => e.stopPropagation()}>
        <button 
          className={`control-btn play-btn ${isPlaying ? 'playing' : ''}`} 
          onClick={togglePlay}
          style={{ width: '40px', height: '40px', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', borderRadius: '50%', background: 'transparent', color: 'var(--text-primary)' }}
        >
          {isPlaying ? <Pause size={24} fill="currentColor" /> : <Play size={24} fill="currentColor" />}
        </button>
      </div>
      
      {/* Center: Playback Controls & Progress Bar */}
      <div 
        className="player-controls-container" 
        onClick={(e) => e.stopPropagation()}
      >
        <div className="player-controls">
          <button className="control-btn" onClick={playPrev}><SkipBack size={20} fill="currentColor" /></button>
          <button 
            className={`control-btn play-btn ${isPlaying ? 'playing' : ''}`} 
            onClick={togglePlay}
          >
            {isPlaying ? (
              <Pause size={24} fill="currentColor" />
            ) : (
              <Play size={24} fill="currentColor" style={{ marginLeft: '4px' }} />
            )}
          </button>
          <button className="control-btn" onClick={playNext}><SkipForward size={20} fill="currentColor" /></button>
        </div>
        
        {/* Integrated Progress Bar */}
        <div style={{ width: '100%', maxWidth: '500px', display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontWeight: 500, minWidth: '40px', textAlign: 'right' }}>
            {formatTime(currentTime)}
          </span>
          <div 
            className="progress-bar-container" 
            style={{ flex: 1, height: '6px', background: 'var(--bg-secondary)', borderRadius: '3px', cursor: 'pointer', overflow: 'hidden', position: 'relative' }}
            onClick={handleProgressClick}
          >
            <div 
              className="progress-bar-fill" 
              style={{ width: `${progress}%`, height: '100%', background: 'var(--text-primary)', borderRadius: '3px', transition: 'width 0.1s linear' }}
            ></div>
          </div>
          <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontWeight: 500, minWidth: '40px' }}>
            {formatTime(displayDuration)}
          </span>
        </div>
      </div>
      
      {/* Right Side: Interactive Volume Slider & Extra Controls */}
      <div 
        className="volume-controls" 
        onClick={(e) => e.stopPropagation()}
        style={{ display: 'flex', alignItems: 'center', gap: '16px' }}
      >
        <button 
          className="control-btn" 
          onClick={() => setShowLyrics(!showLyrics)}
          title="Lyrics"
          style={{ padding: '8px', opacity: showLyrics ? 1 : 0.7 }}
        >
          <Mic2 size={20} color={showLyrics ? "var(--accent-primary)" : "currentColor"} />
        </button>

        <button 
          className="control-btn" 
          onClick={() => setIsFullScreen(true)}
          title="Full Screen"
          style={{ padding: '8px' }}
        >
          <Maximize size={18} />
        </button>

        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }} onClick={() => {
            if (volume === 0) {
              changeVolume(prevVolume > 0 ? prevVolume : 1);
            } else {
              setPrevVolume(volume);
              changeVolume(0);
            }
          }}>
            {volume === 0 ? <VolumeX size={20} /> : <Volume2 size={20} />}
          </span>
          <input 
            type="range" 
            min="0" 
            max="1" 
            step="0.01" 
            value={volume} 
            onChange={handleVolumeChange}
            style={{
              width: '100px',
              cursor: 'pointer',
              accentColor: 'var(--accent-primary)'
            }}
          />
        </div>
      </div>
      
      <LyricsOverlay isOpen={showLyrics} onClose={() => setShowLyrics(false)} />
      <FullScreenPlayer isOpen={isFullScreen} onClose={() => setIsFullScreen(false)} />
    </footer>
  );
};

export default PlayerBar;
