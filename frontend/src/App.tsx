import { useEffect, useState } from 'react';
import { Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import TopBar from './components/TopBar';
import PlayerBar from './components/PlayerBar';
import HomePage from './pages/HomePage';
import AlbumPage from './pages/AlbumPage';
import HeartsPage from './pages/HeartsPage';
import { PlaylistsPage } from './pages/PlaylistsPage';
import { LoginPage } from './pages/LoginPage';
import ArtistsPage from './pages/ArtistsPage';
import ArtistPage from './pages/ArtistPage';
import AlbumsPage from './pages/AlbumsPage';
import SettingsPage from './pages/SettingsPage';
import RadioPage from './pages/RadioPage';
import PodcastsPage from './pages/PodcastsPage';
import SearchPage from './pages/SearchPage';
import { HeartsProvider } from './context/HeartsContext';
import { PlaylistsProvider } from './context/PlaylistsContext';

import { AuthProvider, useAuth } from './context/AuthContext';
import { PlayerProvider } from './context/PlayerContext';
import { ToastProvider } from './context/ToastContext';

import './index.css';
import './App.css';

function AppContent() {
  const { user } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [updateRegistration, setUpdateRegistration] = useState<ServiceWorkerRegistration | null>(null);

  useEffect(() => {
    const onUpdate = (event: Event) => {
      setUpdateRegistration((event as CustomEvent<ServiceWorkerRegistration>).detail);
    };
    window.addEventListener('supernova:update-ready', onUpdate);
    return () => window.removeEventListener('supernova:update-ready', onUpdate);
  }, []);

  const applyUpdate = () => {
    const waiting = updateRegistration?.waiting;
    if (!waiting) return;
    let reloaded = false;
    navigator.serviceWorker.addEventListener('controllerchange', () => {
      if (reloaded) return;
      reloaded = true;
      window.location.reload();
    }, { once: true });
    waiting.postMessage({ type: 'SKIP_WAITING' });
  };

  if (!user) {
    return <LoginPage />;
  }

  return (
    <PlayerProvider key={user.id}>
    <HeartsProvider>
      <PlaylistsProvider>

          <div className="app-container">
            {updateRegistration && (
              <button
                type="button"
                onClick={applyUpdate}
                aria-label="Reload Supernova to apply the available update"
                style={{ position: 'fixed', top: '12px', right: '12px', zIndex: 20000, padding: '10px 16px', borderRadius: '999px', background: 'var(--accent-primary)', color: 'white', boxShadow: 'var(--accent-glow)' }}
              >
                Update available · Reload
              </button>
            )}
            <Sidebar isOpen={mobileMenuOpen} onClose={() => setMobileMenuOpen(false)} />
            
            <main className="main-content">
              <TopBar onMenuClick={() => setMobileMenuOpen(true)} />
              
              <Routes>
                <Route path="/" element={<HomePage />} />
                <Route path="/album/:id" element={<AlbumPage />} />
                <Route path="/artist/:id" element={<ArtistPage />} />
                <Route path="/hearts" element={<HeartsPage />} />
                
                <Route path="/artists" element={<ArtistsPage />} />
                <Route path="/albums" element={<AlbumsPage />} />
                <Route path="/playlists" element={<PlaylistsPage />} />
                <Route path="/radio" element={<RadioPage />} />
                <Route path="/podcasts" element={<PodcastsPage />} />
                <Route path="/search" element={<SearchPage />} />
                <Route path="/settings" element={<SettingsPage />} />
              </Routes>
            </main>

            <PlayerBar />
          </div>

      </PlaylistsProvider>
    </HeartsProvider>
    </PlayerProvider>
  );
}

function App() {
  return (
    <ToastProvider>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </ToastProvider>
  );
}

export default App;
