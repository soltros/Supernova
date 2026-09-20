import { createContext, useState, useContext, useEffect, useRef, useMemo, useCallback } from 'react';
import type { FC, ReactNode } from 'react';
import { apiService } from '../services/api';
import type { Track, Album } from '../types';

interface PlayerState {
  currentTrack: Track | null;
  currentAlbum: Album | null;
  isPlaying: boolean;
  duration: number;
  volume: number;
  queue: Track[];
  queueIndex: number;
  audioElement: HTMLAudioElement | null;
  playContext: (tracks: Track[], startIndex: number, album: Album) => void;
  playNext: () => void;
  playPrev: () => void;
  togglePlay: () => void;
  seekTo: (percent: number) => void;
  changeVolume: (level: number) => void;
  internalPlay: (track: Track, album: Album, options?: { podcast_episode_id?: string, start_position_ms?: number }) => void;
  insertNext: (track: Track) => void;
  enqueue: (track: Track) => void;
}

const PlayerContext = createContext<PlayerState | undefined>(undefined);
const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

export const PlayerProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [currentTrack, setCurrentTrack] = useState<Track | null>(null);
  const [currentAlbum, setCurrentAlbum] = useState<Album | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1.0);

  // Playback Queue State
  const [queue, setQueue] = useState<Track[]>([]);
  const [queueIndex, setQueueIndex] = useState(0);
  const [audioElement, setAudioElement] = useState<HTMLAudioElement | null>(null);

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const activeTrackRef = useRef<Track | null>(null);
  const playbackRequestRef = useRef(0);
  
  // We use Refs for state accessed inside event listeners to avoid stale closures
  const queueRef = useRef<Track[]>([]);
  const queueIndexRef = useRef<number>(0);
  const albumRef = useRef<Album | null>(null);
  const playNextRef = useRef<(() => void) | null>(null);
  const playPrevRef = useRef<(() => void) | null>(null);
  const playPromiseRef = useRef<Promise<void> | null>(null);
  const hasScrobbledRef = useRef<boolean>(false);
  const accumulatedPlayTimeRef = useRef<number>(0);
  const lastTimeRef = useRef<number>(0);

  useEffect(() => {
    const audio = new Audio();
    audio.volume = 1.0;
    audioRef.current = audio;
    setAudioElement(audio);
    const controller = new AbortController();
    const listen = (type: string, callback: EventListener) => audio.addEventListener(type, callback, { signal: controller.signal });
    listen('error', () => setIsPlaying(false));
    listen('loadedmetadata', () => { if (Number.isFinite(audio.duration)) setDuration(audio.duration); });

    listen('timeupdate', () => {
      if (audio.duration && isFinite(audio.duration)) {
        setDuration(audio.duration);
      }
      
      // Phase 6: Internal Scrobbling Engine (Scrub-Proof)
      const activeTrack = activeTrackRef.current;
      if (activeTrack && !activeTrack.stream_url && !hasScrobbledRef.current) {
        // Calculate real time delta
        const diff = audio.currentTime - lastTimeRef.current;
        
        // If diff is between 0 and 1, it's normal playback. If it's huge, it's a seek/scrub.
        if (diff > 0 && diff < 1.0) {
          accumulatedPlayTimeRef.current += diff;
        }
        lastTimeRef.current = audio.currentTime;

        const durationSeconds = activeTrack.duration_ms / 1000;
        // Last.fm Scrobble rules: >= 30 seconds total duration, and scrobble at 50% or 4 mins (240s)
        let threshold = 30; // fallback if duration is unknown
        if (durationSeconds >= 30) {
          threshold = Math.min(240, durationSeconds * 0.5);
        } else if (durationSeconds > 0 && durationSeconds < 30) {
          threshold = Infinity; // Track too short to scrobble
        }
        if (accumulatedPlayTimeRef.current >= threshold) {
          hasScrobbledRef.current = true;
          apiService.scrobbleTrack(activeTrack.id).catch(e => console.error("Scrobble failed:", e));
          
          // Last.fm Scrobbling integration
          const lastfmSession = localStorage.getItem('lastfm_session');
          if (lastfmSession) {
            // Timestamp should be when the track started playing
            const timestamp = Math.floor(Date.now() / 1000) - Math.floor(accumulatedPlayTimeRef.current);
            const artist = activeTrack.artist_name || 'Unknown Artist';
            apiService.scrobbleToLastFm(lastfmSession, artist, activeTrack.title, timestamp)
              .catch(e => console.error("Last.fm scrobble failed:", e));
          }
        }
      }
    });

    listen('ended', () => {
      const activeTrack = activeTrackRef.current;
      if (activeTrack && activeTrack.id.startsWith('podcast-')) {
        const episodeId = activeTrack.id.replace('podcast-', '');
        apiService.savePodcastProgress(episodeId, 0, true).catch(e => console.error("Podcast progress save failed:", e));
      }

      if (playNextRef.current) {
        playNextRef.current();
      }
    });

    listen('play', () => {
      setIsPlaying(true);
      if ('mediaSession' in navigator) {
        navigator.mediaSession.playbackState = 'playing';
      }
      
      // Last.fm Now Playing integration
      const lastfmSession = localStorage.getItem('lastfm_session');
      const activeTrack = activeTrackRef.current;
      if (lastfmSession && activeTrack && !activeTrack.stream_url && accumulatedPlayTimeRef.current < 5) {
        const artist = activeTrack.artist_name || 'Unknown Artist';
        apiService.updateNowPlayingToLastFm(lastfmSession, artist, activeTrack.title)
          .catch(e => console.error("Last.fm now playing failed:", e));
      }
    });
    
    listen('pause', () => {
      setIsPlaying(false);
      if ('mediaSession' in navigator) {
        navigator.mediaSession.playbackState = 'paused';
      }
      
      const activeTrack = activeTrackRef.current;
      if (!audio.ended && activeTrack && activeTrack.id.startsWith('podcast-') && audioRef.current) {
        const episodeId = activeTrack.id.replace('podcast-', '');
        apiService.savePodcastProgress(episodeId, Math.floor(audioRef.current.currentTime * 1000), false).catch(e => console.error("Podcast progress save failed:", e));
      }
    });

    // Wire up global lock-screen media keys to our React refs
    if ('mediaSession' in navigator) {
      navigator.mediaSession.setActionHandler('play', () => audioRef.current?.play());
      navigator.mediaSession.setActionHandler('pause', () => audioRef.current?.pause());
      navigator.mediaSession.setActionHandler('previoustrack', () => playPrevRef.current?.());
      navigator.mediaSession.setActionHandler('nexttrack', () => playNextRef.current?.());
      
      // Support lock-screen scrubbing
      navigator.mediaSession.setActionHandler('seekto', (details) => {
        if (details.seekTime !== undefined && audioRef.current) {
          audioRef.current.currentTime = details.seekTime;
        }
      });
    }

    return () => {
      controller.abort();
      playbackRequestRef.current++;
      audio.pause();
      audioRef.current = null;
      audio.src = '';
      if ('mediaSession' in navigator) {
        navigator.mediaSession.setActionHandler('play', null);
        navigator.mediaSession.setActionHandler('pause', null);
        navigator.mediaSession.setActionHandler('previoustrack', null);
        navigator.mediaSession.setActionHandler('nexttrack', null);
        navigator.mediaSession.setActionHandler('seekto', null);
      }
    };
  }, []);

  const internalPlay = useCallback(async (track: Track, album: Album, options?: { podcast_episode_id?: string, start_position_ms?: number }) => {
    const audio = audioRef.current;
    if (!audio) return;
    const request = ++playbackRequestRef.current;
    // Pause the outgoing source while its identity is still current.
    audio.pause();
    activeTrackRef.current = track;
    if (track.stream_url) {
      queueRef.current = [track]; queueIndexRef.current = 0; albumRef.current = album;
      setQueue([track]); setQueueIndex(0);
    }
    hasScrobbledRef.current = false;
    accumulatedPlayTimeRef.current = 0;
    lastTimeRef.current = 0;
    setCurrentTrack(track); setCurrentAlbum(album);
    setDuration(track.duration_ms / 1000);
    const token = localStorage.getItem('sn_token');
    const tokenQuery = token ? `?token=${encodeURIComponent(token)}` : '';
    audio.src = track.stream_url || `${API_BASE_URL}/api/stream/${track.id}${tokenQuery}`;
    const resumeAt = (options?.start_position_ms || 0) / 1000;
    if (resumeAt > 0) {
      audio.addEventListener('loadedmetadata', () => {
        if (request !== playbackRequestRef.current) return;
        audio.currentTime = resumeAt; lastTimeRef.current = resumeAt;
      }, { once: true });
    }
    try {
      // Replacing src cancels the old play promise; never wait on a stalled source.
      const promise = audio.play(); playPromiseRef.current = promise;
      await promise;
    } catch {
      if (request === playbackRequestRef.current) setIsPlaying(false);
    }
    if (request !== playbackRequestRef.current) return;

    // Update Lock Screen Metadata (Media Session API)
    if ('mediaSession' in navigator) {
      const artUrl = album.cover_art_url 
        ? album.cover_art_url 
        : `${API_BASE_URL || window.location.origin}/api/art/album/${track.album_id || album.id}`;
        
      // Ensure absolute URL (if cover_art_url is a relative path somehow)
      const absoluteArtUrl = artUrl.startsWith('http') ? artUrl : `${window.location.origin}${artUrl.startsWith('/') ? '' : '/'}${artUrl}`;

      navigator.mediaSession.metadata = new MediaMetadata({
        title: track.title,
        artist: track.artist_name || album.title, // Prioritize track artist name
        album: album.title,
        artwork: [
          { src: absoluteArtUrl, sizes: '500x500', type: 'image/jpeg' }
        ]
      });
    }
  }, []);

  const playContext = useCallback(async (tracks: Track[], startIndex: number, album: Album) => {
    if (!Number.isInteger(startIndex) || !tracks[startIndex]) return;
    setQueue(tracks);
    setQueueIndex(startIndex);
    
    queueRef.current = tracks;
    queueIndexRef.current = startIndex;
    albumRef.current = album;
    
    await internalPlay(tracks[startIndex], album);
  }, [internalPlay]);

  const insertNext = useCallback((track: Track) => {
    if (queueRef.current.length === 0) {
      if (currentAlbum) playContext([track], 0, currentAlbum);
      return;
    }
    const newQueue = [...queueRef.current];
    newQueue.splice(queueIndexRef.current + 1, 0, track);
    setQueue(newQueue);
    queueRef.current = newQueue;
  }, [currentAlbum, playContext]);

  const enqueue = useCallback((track: Track) => {
    if (queueRef.current.length === 0) {
      if (currentAlbum) playContext([track], 0, currentAlbum);
      return;
    }
    const newQueue = [...queueRef.current, track];
    setQueue(newQueue);
    queueRef.current = newQueue;
  }, [currentAlbum, playContext]);

  const playNext = useCallback(() => {
    if (queueIndexRef.current < queueRef.current.length - 1) {
      const nextIdx = queueIndexRef.current + 1;
      setQueueIndex(nextIdx);
      queueIndexRef.current = nextIdx;
      if (!albumRef.current) return;
      internalPlay(queueRef.current[nextIdx], albumRef.current);
    } else {
      setIsPlaying(false);
      if (audioRef.current) {
        audioRef.current.pause();
        audioRef.current.currentTime = 0;
      }
    }
  }, [internalPlay]);

  const playPrev = useCallback(() => {
    if (!audioRef.current) return;
    
    if (audioRef.current.currentTime > 3) {
      audioRef.current.currentTime = 0;
      return;
    }

    if (queueIndexRef.current > 0) {
      const prevIdx = queueIndexRef.current - 1;
      setQueueIndex(prevIdx);
      queueIndexRef.current = prevIdx;
      if (!albumRef.current) return;
      internalPlay(queueRef.current[prevIdx], albumRef.current);
    }
  }, [internalPlay]);

  useEffect(() => { playNextRef.current = playNext; playPrevRef.current = playPrev; }, [playNext, playPrev]);

  const togglePlay = useCallback(async () => {
    if (!audioRef.current || !currentTrack) return;
    
    if (!audioRef.current.paused) {
      audioRef.current.pause();
    } else {
      try {
        playPromiseRef.current = audioRef.current.play();
        await playPromiseRef.current;
      } catch {
        console.log("Playback resumed and instantly interrupted.");
      }
    }
  }, [currentTrack]);

  const seekTo = useCallback((percent: number) => {
    if (!audioRef.current) return;
    const activeDuration = duration || (currentTrack ? currentTrack.duration_ms / 1000 : 0);
    if (!Number.isFinite(activeDuration) || activeDuration <= 0 || !Number.isFinite(percent)) return;

    const newTime = (Math.max(0, Math.min(100, percent)) / 100) * activeDuration;
    audioRef.current.currentTime = newTime;
    lastTimeRef.current = newTime; // Prevent scrub spikes
  }, [currentTrack, duration]);

  const changeVolume = useCallback((level: number) => {
    if (!audioRef.current || !Number.isFinite(level)) return;
    const safeLevel = Math.max(0, Math.min(1, level));
    audioRef.current.volume = safeLevel;
    setVolume(safeLevel);
  }, []);

  const value = useMemo(() => ({
    currentTrack, currentAlbum, isPlaying, 
    duration, volume, queue, queueIndex,
    audioElement,
    playContext, playNext, playPrev, togglePlay, seekTo, changeVolume, internalPlay, insertNext, enqueue
  }), [
    currentTrack, currentAlbum, isPlaying, duration, volume, queue, queueIndex,
    audioElement, playContext, playNext, playPrev, togglePlay, seekTo, changeVolume, internalPlay, insertNext, enqueue
  ]);

  return (
    <PlayerContext.Provider value={value}>
      {children}
    </PlayerContext.Provider>
  );
};

export const usePlayer = () => {
  const context = useContext(PlayerContext);
  if (!context) throw new Error('usePlayer must be used within PlayerProvider');
  return context;
};
