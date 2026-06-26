import type { FC, MouseEvent, ChangeEvent } from 'react';
import { usePlayer } from '../context/PlayerContext';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const formatTime = (seconds: number) => {
  if (!seconds || isNaN(seconds)) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
};

const PlayerBar: FC = () => {
  const { 
    currentTrack, currentAlbum, isPlaying, 
    progress, currentTime, duration, volume, 
    togglePlay, seekTo, playNext, playPrev, changeVolume 
  } = usePlayer();

  const handleProgressClick = (e: MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const percent = ((e.clientX - rect.left) / rect.width) * 100;
    seekTo(percent);
  };

  const handleVolumeChange = (e: ChangeEvent<HTMLInputElement>) => {
    changeVolume(parseFloat(e.target.value));
  };

  // If the browser hasn't calculated duration yet, fallback to our database metadata
  const displayDuration = duration || (currentTrack ? currentTrack.duration_ms / 1000 : 0);

  return (
    <footer className="player-bar">
      
      {/* Left Side: Now Playing Metadata */}
      <div className="now-playing">
        <div style={{ position: 'relative', width: '60px', height: '60px', flexShrink: 0 }}>
          {currentAlbum ? (
            <img 
              src={`${API_BASE_URL}/api/art/album/${currentAlbum.id}`} 
              alt="art"
              className={`track-art-small ${isPlaying ? 'playing' : ''}`}
              onError={(e) => {
                e.currentTarget.style.display = 'none';
                if (e.currentTarget.nextElementSibling) {
                  (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'block';
                }
              }}
            />
          ) : (
            <div className={`track-art-small ${isPlaying ? 'playing' : ''}`}></div>
          )}
        </div>
        <div className="track-details">
          <h4>{currentTrack ? currentTrack.title : 'No Track Playing'}</h4>
          <p>{currentAlbum ? currentAlbum.title : 'Select a track to begin'}</p>
        </div>
      </div>
      
      {/* Center: Playback Controls & Interactive Progress */}
      <div className="player-controls" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%', maxWidth: '400px' }}>
        <div style={{ display: 'flex', gap: '24px', alignItems: 'center' }}>
          <button className="control-btn" onClick={playPrev}>⏮</button>
          <button className={`control-btn play-btn ${isPlaying ? 'playing' : ''}`} onClick={togglePlay}>
            {isPlaying ? '⏸' : '▶'}
          </button>
          <button className="control-btn" onClick={playNext}>⏭</button>
        </div>
        
        {/* Floating Progress Bar underneath controls */}
        <div style={{ display: 'flex', alignItems: 'center', width: '100%', gap: '12px', marginTop: '4px' }}>
          <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontWeight: 600 }}>
            {formatTime(currentTime)}
          </span>
          <div 
            className="progress-bar-container" 
            style={{ flex: 1, height: '4px', background: 'rgba(255,255,255,0.1)', borderRadius: '2px', cursor: 'pointer', overflow: 'hidden' }}
            onClick={handleProgressClick}
          >
            <div 
              className="progress-bar-fill" 
              style={{ width: `${progress}%`, height: '100%', background: 'var(--accent-gradient)', borderRadius: '2px', transition: 'width 0.1s linear' }}
            ></div>
          </div>
          <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontWeight: 600 }}>
            {formatTime(displayDuration)}
          </span>
        </div>
      </div>
      
      {/* Right Side: Interactive Volume Slider */}
      <div className="volume-controls">
        <span style={{ cursor: 'pointer', fontSize: '18px' }} onClick={() => changeVolume(volume === 0 ? 1 : 0)}>
          {volume === 0 ? '🔇' : '🔊'}
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
      
    </footer>
  );
};

export default PlayerBar;
