import type { FC, MouseEvent, ChangeEvent } from 'react';
import { Link } from 'react-router-dom';
import { Play, Pause, SkipBack, SkipForward, Volume2, VolumeX } from 'lucide-react';
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
      </div>
      
      {/* Center: Playback Controls & Interactive Progress */}
      <div className="player-controls" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%', maxWidth: '400px' }}>
        <div style={{ display: 'flex', gap: '24px', alignItems: 'center', marginBottom: '8px' }}>
          <button className="control-btn" onClick={playPrev}><SkipBack size={20} fill="currentColor" /></button>
          <button className={`control-btn play-btn ${isPlaying ? 'playing' : ''}`} onClick={togglePlay}>
            {isPlaying ? <Pause size={24} fill="currentColor" /> : <Play size={24} fill="currentColor" />}
          </button>
          <button className="control-btn" onClick={playNext}><SkipForward size={20} fill="currentColor" /></button>
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
        <span style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }} onClick={() => changeVolume(volume === 0 ? 1 : 0)}>
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
      
    </footer>
  );
};

export default PlayerBar;
