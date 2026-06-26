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
    <footer className="player-bar glass-panel" style={{ display: 'flex', flexDirection: 'column', padding: 0 }}>
      {/* Interactive Progress Bar */}
      <div 
        className="progress-bar-container" 
        style={{ width: '100%', height: '4px', background: 'var(--border-glass)', cursor: 'pointer' }}
        onClick={handleProgressClick}
      >
        <div 
          className="progress-bar-fill" 
          style={{ width: `${progress}%`, height: '100%', background: 'var(--accent-primary)', transition: 'width 0.1s linear' }}
        ></div>
      </div>

      {/* Main Player UI */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 32px', height: '86px', width: '100%' }}>
        
        {/* Left Side: Now Playing Metadata */}
        <div className="now-playing">
          <div style={{ position: 'relative', width: '50px', height: '50px', borderRadius: '4px', overflow: 'hidden', flexShrink: 0 }}>
            {currentAlbum ? (
              <img 
                src={`${API_BASE_URL}/api/art/album/${currentAlbum.id}`} 
                alt="art"
                style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                onError={(e) => {
                  e.currentTarget.style.display = 'none';
                  if (e.currentTarget.nextElementSibling) {
                    (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'block';
                  }
                }}
              />
            ) : null}
            <div className="track-art-small" style={{ display: currentAlbum ? 'none' : 'block', width: '100%', height: '100%', position: 'absolute', top: 0, left: 0, background: 'var(--bg-glass)' }}></div>
          </div>
          <div className="track-details">
            <h4>{currentTrack ? currentTrack.title : 'No Track Playing'}</h4>
            <p>{currentAlbum ? currentAlbum.title : 'Select a track to begin'}</p>
          </div>
        </div>
        
        {/* Center: Playback Controls & Time */}
        <div className="player-controls" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
          <div style={{ display: 'flex', gap: '20px', alignItems: 'center' }}>
            <button className="control-btn" onClick={playPrev}>⏮</button>
            <button className="control-btn play-btn" onClick={togglePlay}>
              {isPlaying ? '⏸' : '▶'}
            </button>
            <button className="control-btn" onClick={playNext}>⏭</button>
          </div>
          <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '8px', display: 'flex', gap: '10px' }}>
            <span>{formatTime(currentTime)}</span>
            <span>/</span>
            <span>{formatTime(displayDuration)}</span>
          </div>
        </div>
        
        {/* Right Side: Interactive Volume Slider */}
        <div className="volume-controls" style={{ display: 'flex', alignItems: 'center', gap: '12px', width: '250px', justifyContent: 'flex-end' }}>
          <span style={{ cursor: 'pointer' }} onClick={() => changeVolume(volume === 0 ? 1 : 0)}>
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
              accentColor: 'var(--accent-primary)' // Uses standard HTML5 slider but styles it with our brand color
            }}
          />
        </div>
        
      </div>
    </footer>
  );
};

export default PlayerBar;
