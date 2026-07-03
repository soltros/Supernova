import React, { MouseEvent } from 'react';
import { usePlayer } from '../context/PlayerContext';
import { Play, Pause, SkipBack, SkipForward, Minimize2 } from 'lucide-react';
import HeartButton from './HeartButton';

const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

interface FullScreenPlayerProps {
  isOpen: boolean;
  onClose: () => void;
}

const formatTime = (seconds: number) => {
  if (!seconds || isNaN(seconds)) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const FullScreenPlayer: React.FC<FullScreenPlayerProps> = ({ isOpen, onClose }) => {
  const { 
    currentTrack, currentAlbum, isPlaying, 
    currentTime, duration, togglePlayPause, playNext, playPrevious, seekTo
  } = usePlayer();

  if (!isOpen) return null;

  const handleProgressClick = (e: MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const percent = ((e.clientX - rect.left) / rect.width) * 100;
    seekTo(percent);
  };

  const displayDuration = duration || (currentTrack ? currentTrack.duration_ms / 1000 : 0);
  const progress = displayDuration > 0 ? (currentTime / displayDuration) * 100 : 0;
  
  const coverUrl = currentAlbum 
    ? ((currentAlbum as any).cover_art_url ? (currentAlbum as any).cover_art_url : `${API_BASE_URL}/api/art/album/${currentAlbum.id}`)
    : '';

  return (
    <div style={{
      position: 'fixed',
      top: 0, left: 0, right: 0, bottom: 0,
      backgroundColor: '#000',
      zIndex: 9999,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '40px',
      color: '#fff',
      animation: 'fadeIn 0.3s ease'
    }}>
      {/* Background Blur */}
      {coverUrl && (
        <div style={{
          position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
          backgroundImage: `url(${coverUrl})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          filter: 'blur(100px) brightness(0.3)',
          zIndex: -1
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
          transition: 'background 0.2s ease'
        }}
        onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.2)'}
        onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
      >
        <Minimize2 size={24} />
      </button>

      {/* Main Content */}
      <div style={{ width: '100%', maxWidth: '800px', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '40px' }}>
        
        {/* Large Album Art */}
        <div style={{
          width: '100%', maxWidth: '500px', aspectRatio: '1/1',
          borderRadius: '16px', overflow: 'hidden',
          boxShadow: '0 20px 60px rgba(0,0,0,0.6)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(255,255,255,0.05)'
        }}>
          {coverUrl ? (
            <img src={coverUrl} alt="Album Art" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
          ) : (
            <span style={{ fontSize: '64px', color: 'rgba(255,255,255,0.2)' }}>♪</span>
          )}
        </div>

        {/* Track Info */}
        <div style={{ textAlign: 'center', width: '100%' }}>
          <h1 style={{ fontSize: '48px', fontWeight: 800, margin: '0 0 12px 0', textShadow: '0 2px 10px rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '16px' }}>
            {currentTrack?.title || 'No Track Selected'}
            {currentTrack && <HeartButton entityType="track" entityId={currentTrack.id} />}
          </h1>
          <p style={{ fontSize: '24px', color: 'rgba(255,255,255,0.7)', margin: 0, fontWeight: 500 }}>
            {currentTrack?.artist_name || 'Unknown Artist'} • {currentAlbum?.title || 'Unknown Album'}
          </p>
        </div>

        {/* Controls */}
        <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: '24px' }}>
          
          {/* Progress Bar */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span style={{ fontSize: '14px', color: 'rgba(255,255,255,0.7)', minWidth: '40px', textAlign: 'right' }}>
              {formatTime(currentTime)}
            </span>
            <div 
              onClick={handleProgressClick}
              style={{ flex: 1, height: '8px', background: 'rgba(255,255,255,0.2)', borderRadius: '4px', cursor: 'pointer', overflow: 'hidden', position: 'relative' }}
            >
              <div style={{ width: `${progress}%`, height: '100%', background: '#fff', borderRadius: '4px', transition: 'width 0.1s linear' }} />
            </div>
            <span style={{ fontSize: '14px', color: 'rgba(255,255,255,0.7)', minWidth: '40px' }}>
              {formatTime(displayDuration)}
            </span>
          </div>

          {/* Transport Controls */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '32px' }}>
            <button onClick={playPrevious} style={{ background: 'none', border: 'none', color: '#fff', cursor: 'pointer' }}>
              <SkipBack size={36} fill="currentColor" />
            </button>
            <button 
              onClick={togglePlayPause} 
              style={{ 
                background: '#fff', color: '#000', border: 'none', borderRadius: '50%',
                width: '80px', height: '80px', display: 'flex', alignItems: 'center', justifyContent: 'center',
                cursor: 'pointer', boxShadow: '0 8px 24px rgba(0,0,0,0.3)'
              }}
            >
              {isPlaying ? <Pause size={40} fill="currentColor" /> : <Play size={40} fill="currentColor" style={{ marginLeft: '4px' }} />}
            </button>
            <button onClick={playNext} style={{ background: 'none', border: 'none', color: '#fff', cursor: 'pointer' }}>
              <SkipForward size={36} fill="currentColor" />
            </button>
          </div>
        </div>

      </div>
    </div>
  );
};

export default FullScreenPlayer;
