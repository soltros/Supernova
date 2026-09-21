// @vitest-environment jsdom
import React from 'react';
import { act, cleanup, render } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PlayerProvider, usePlayer } from '../src/context/PlayerContext';
import { apiService } from '../src/services/api';
import type { Album, Track } from '../src/types';
vi.mock('../src/services/api', () => ({ apiService: {
  savePodcastProgress: vi.fn().mockResolvedValue(undefined),
  scrobbleTrack: vi.fn().mockResolvedValue(undefined),
  updateNowPlayingToLastFm: vi.fn().mockResolvedValue(undefined),
  scrobbleToLastFm: vi.fn().mockResolvedValue(undefined),
  mediaUrl: vi.fn((scope: string, resource: string) => Promise.resolve(`/api/stream/${resource}?ticket=test-ticket`))
} }));
class FakeAudio extends EventTarget {
  src = ''; volume = 1; currentTime = 0; duration = 120; paused = true; ended = false;
  play = vi.fn(() => { this.paused = false; this.dispatchEvent(new Event('play')); return Promise.resolve(); });
  pause = vi.fn(() => { if (!this.paused) { this.paused = true; this.dispatchEvent(new Event('pause')); } });
}
let audio: FakeAudio;
let player: ReturnType<typeof usePlayer>;
const album: Album = { id: 'album', title: 'Album', release_year: 2026, cover_art_path: '' };
const track: Track = { id: 'song', title: 'Song', album_id: 'album', duration_ms: 120000, track_number: 1, disc_number: 1, bitrate: 192, format: 'mp3' };
function Probe() { player = usePlayer(); return null; }
beforeEach(() => { localStorage.clear(); vi.clearAllMocks(); audio = new FakeAudio(); vi.stubGlobal('Audio', function () { return audio; }); });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
describe('playback regression coverage', () => {
  it('stops and detaches audio on logout/unmount', async () => {
    const view = render(<PlayerProvider><Probe /></PlayerProvider>);
    await act(async () => { player.playContext([track], 0, album); });
    expect(player.isPlaying).toBe(true);
    view.unmount(); expect(audio.paused).toBe(true); expect(audio.src).toBe('');
  });
  it('replaces a stalled play request without waiting and ignores its late rejection', async () => {
    render(<PlayerProvider><Probe /></PlayerProvider>);
    let reject!: (reason: Error) => void;
    audio.play.mockImplementationOnce(() => new Promise((_, fail) => { reject = fail; }));
    act(() => { player.playContext([track], 0, album); });
    await act(async () => { player.playContext([{ ...track, id: 'new' }], 0, album); });
    expect(audio.src).toContain('/api/stream/new'); expect(player.isPlaying).toBe(true);
    await act(async () => { reject(new Error('old source cancelled')); });
    expect(player.currentTrack?.id).toBe('new'); expect(player.isPlaying).toBe(true);
  });
  it('saves podcast identity and resume position without scrobbling the old music queue', async () => {
    localStorage.setItem('lastfm_session', 'test');
    render(<PlayerProvider><Probe /></PlayerProvider>);
    await act(async () => { player.playContext([track], 0, album); });
    vi.clearAllMocks();
    const episode = { ...track, id: 'podcast-ep1', stream_url: 'https://example.test/episode.mp3' };
    await act(async () => { player.internalPlay(episode, album, { start_position_ms: 20000 }); });
    act(() => { audio.dispatchEvent(new Event('loadedmetadata')); });
    expect(audio.currentTime).toBe(20); expect(player.queue.map(t => t.id)).toEqual(['podcast-ep1']);
    act(() => { audio.currentTime = 40; audio.pause(); });
    expect(apiService.savePodcastProgress).toHaveBeenCalledWith('ep1', 40000, false);
    expect(apiService.updateNowPlayingToLastFm).not.toHaveBeenCalled();
    act(() => { audio.ended = true; audio.dispatchEvent(new Event('ended')); });
    expect(apiService.savePodcastProgress).toHaveBeenLastCalledWith('ep1', 0, true);
  });
  it('allows seek to zero and clamps volume and seek inputs', async () => {
    render(<PlayerProvider><Probe /></PlayerProvider>);
    await act(async () => { player.playContext([track], 0, album); });
    act(() => { player.seekTo(50); }); expect(audio.currentTime).toBe(60);
    act(() => { player.seekTo(0); player.changeVolume(2); }); expect(audio.currentTime).toBe(0); expect(audio.volume).toBe(1);
    act(() => { player.seekTo(200); player.changeVolume(-1); }); expect(audio.currentTime).toBe(120); expect(audio.volume).toBe(0);
  });
});
