import { useState } from 'react';
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
import { PlayerProvider } from './context/PlayerContext';
import { AuthProvider, useAuth } from './context/AuthContext';

import './index.css';
import './App.css';

function AppContent() {
  const { user } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  if (!user) {
    return <LoginPage />;
  }

  return (
    <HeartsProvider>
      <PlaylistsProvider>
        <PlayerProvider>
          <div className="app-container">
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
        </PlayerProvider>
      </PlaylistsProvider>
    </HeartsProvider>
  );
}

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

export default App;
