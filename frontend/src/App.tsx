import { Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import TopBar from './components/TopBar';
import PlayerBar from './components/PlayerBar';
import LibraryPage from './pages/LibraryPage';
import AlbumPage from './pages/AlbumPage';
import HeartsPage from './pages/HeartsPage';
import { PlaylistsPage } from './pages/PlaylistsPage';
import { LoginPage } from './pages/LoginPage';
import { PlayerProvider } from './context/PlayerContext';
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
            <Route path="/hearts" element={<HeartsPage />} />
            
            <Route path="/artists" element={<div className="content-scroll"><h2 className="section-title">Artists</h2></div>} />
            <Route path="/albums" element={<div className="content-scroll"><h2 className="section-title">Albums</h2></div>} />
            <Route path="/playlists" element={<PlaylistsPage />} />
            <Route path="/settings" element={<div className="content-scroll"><h2 className="section-title">Settings</h2></div>} />
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
