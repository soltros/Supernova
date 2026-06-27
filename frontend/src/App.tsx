import { Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import TopBar from './components/TopBar';
import PlayerBar from './components/PlayerBar';
import LibraryPage from './pages/LibraryPage';
import AlbumPage from './pages/AlbumPage';
import HeartsPage from './pages/HeartsPage';
import { PlaylistsPage } from './pages/PlaylistsPage';
import { LoginPage } from './pages/LoginPage';
import ArtistsPage from './pages/ArtistsPage';
import ArtistPage from './pages/ArtistPage';
import AlbumsPage from './pages/AlbumsPage';
import SettingsPage from './pages/SettingsPage';
import { HeartsProvider } from './context/HeartsContext';
import { AuthProvider, useAuth } from './context/AuthContext';

import './index.css';
import './App.css';

function AppContent() {
  const { user } = useAuth();

  if (!user) {
    return <LoginPage />;
  }

  return (
    <HeartsProvider>
      <div className="app-container">
        <Sidebar />
        
        <main className="main-content">
          <TopBar />
          
          <Routes>
            <Route path="/" element={<LibraryPage />} />
            <Route path="/album/:id" element={<AlbumPage />} />
            <Route path="/artist/:id" element={<ArtistPage />} />
            <Route path="/hearts" element={<HeartsPage />} />
            
            <Route path="/artists" element={<ArtistsPage />} />
            <Route path="/albums" element={<AlbumsPage />} />
            <Route path="/playlists" element={<PlaylistsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </main>

        <PlayerBar />
      </div>
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
