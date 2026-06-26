import { Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import TopBar from './components/TopBar';
import PlayerBar from './components/PlayerBar';
import LibraryPage from './pages/LibraryPage';
import AlbumPage from './pages/AlbumPage';
import HeartsPage from './pages/HeartsPage';
import { PlayerProvider } from './context/PlayerContext';
import { HeartsProvider } from './context/HeartsContext';

import './index.css';
import './App.css';

function App() {
  return (
    <HeartsProvider>
      <PlayerProvider>
        <div className="app-container">
          <Sidebar />
          
          <main className="main-content">
            <TopBar />
            
            {/* Client-side routing maps URLs directly to React Components */}
            <Routes>
              <Route path="/" element={<LibraryPage />} />
              <Route path="/album/:id" element={<AlbumPage />} />
              <Route path="/hearts" element={<HeartsPage />} />
              
              {/* Temporary placeholders until we build out the respective pages */}
              <Route path="/artists" element={<div className="content-scroll"><h2 className="section-title">Artists</h2></div>} />
              <Route path="/albums" element={<div className="content-scroll"><h2 className="section-title">Albums</h2></div>} />
              <Route path="/playlists" element={<div className="content-scroll"><h2 className="section-title">Playlists</h2></div>} />
              <Route path="/settings" element={<div className="content-scroll"><h2 className="section-title">Settings</h2></div>} />
            </Routes>
          </main>

          <PlayerBar />
        </div>
      </PlayerProvider>
    </HeartsProvider>
  );
}

export default App;
