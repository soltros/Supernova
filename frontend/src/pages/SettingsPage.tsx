import React, { useState } from 'react';
import { apiService } from '../services/api';

const SettingsPage: React.FC = () => {
  const handleClearCache = async () => {
    if ('serviceWorker' in navigator) {
      const registrations = await navigator.serviceWorker.getRegistrations();
      for (const registration of registrations) {
        await registration.unregister();
      }
    }
    // Clear local storage excluding auth tokens
    const token = localStorage.getItem('sn_token');
    const user = localStorage.getItem('sn_user');
    localStorage.clear();
    if (token) localStorage.setItem('sn_token', token);
    if (user) localStorage.setItem('sn_user', user);
    
    alert('Cache cleared successfully! Reloading...');
    window.location.reload();
  };

  const [isResetting, setIsResetting] = useState(false);
  const handleResetArtists = async () => {
    if (!window.confirm("Are you sure? This will clear all artist bios and images and require re-fetching from Last.fm.")) return;
    
    setIsResetting(true);
    try {
      await apiService.resetArtists();
      alert("Artist data cleared successfully. The background scanner will re-fetch data shortly.");
    } catch (e) {
      alert("Failed to reset artist data.");
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div className="content-scroll">
      <div className="page-container">
        <h1 style={{ fontSize: '32px', fontWeight: 800, marginBottom: '32px', letterSpacing: '-1px' }}>Settings</h1>
        
        <div style={{ display: 'grid', gap: '24px', maxWidth: '600px' }}>
          
          <div style={{ background: 'var(--bg-glass)', padding: '24px', borderRadius: '16px', border: '1px solid var(--border-glass)' }}>
            <h3 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '8px' }}>Application Cache</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '14px', marginBottom: '16px' }}>
              If you are experiencing issues with album art not loading or the UI being stuck on an older version, clear the local cache and service workers.
            </p>
            <button 
              onClick={handleClearCache}
              style={{ background: 'rgba(255, 68, 68, 0.1)', border: '1px solid rgba(255, 68, 68, 0.2)', color: '#ff4444', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, transition: 'var(--transition-fast)' }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 68, 68, 0.2)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255, 68, 68, 0.1)'}
            >
              Clear Cache & Reload
            </button>
          </div>

          <div style={{ background: 'var(--bg-glass)', padding: '24px', borderRadius: '16px', border: '1px solid var(--border-glass)' }}>
            <h3 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '8px' }}>Library Management</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '14px', marginBottom: '16px' }}>
              Manually trigger a full rescan of your media directory. Use this if music you dragged into Supernova isn't showing up.
            </p>
            <button 
              onClick={() => {
                apiService.scanLibrary().then(() => alert("Library scan started in the background. Your music will appear shortly."));
              }}
              style={{ background: 'rgba(139, 92, 246, 0.1)', border: '1px solid rgba(139, 92, 246, 0.2)', color: 'var(--accent-primary)', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, transition: 'var(--transition-fast)' }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(139, 92, 246, 0.2)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(139, 92, 246, 0.1)'}
            >
              Scan Library
            </button>
          </div>

          <div style={{ background: 'var(--bg-glass)', padding: '24px', borderRadius: '16px', border: '1px solid var(--border-glass)' }}>
            <h3 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '8px' }}>Metadata Refresh</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '14px', marginBottom: '16px' }}>
              If artist images or biographies are broken, you can clear the database cache to force a re-fetch from Last.fm.
            </p>
            <button 
              onClick={handleResetArtists}
              disabled={isResetting}
              style={{ background: 'rgba(236, 72, 153, 0.1)', border: '1px solid rgba(236, 72, 153, 0.2)', color: 'var(--accent-secondary)', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, transition: 'var(--transition-fast)', opacity: isResetting ? 0.5 : 1, cursor: isResetting ? 'not-allowed' : 'pointer' }}
              onMouseEnter={(e) => { if (!isResetting) e.currentTarget.style.background = 'rgba(236, 72, 153, 0.2)' }}
              onMouseLeave={(e) => { if (!isResetting) e.currentTarget.style.background = 'rgba(236, 72, 153, 0.1)' }}
            >
              {isResetting ? 'Clearing...' : 'Clear Artist Data'}
            </button>
          </div>

          <div style={{ background: 'var(--bg-glass)', padding: '24px', borderRadius: '16px', border: '1px solid var(--border-glass)' }}>
            <h3 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '8px' }}>About Supernova</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>
              Supernova is a next-generation music player built with a futuristic glassmorphism UI.
              <br /><br />
              Version: 1.0.0 (Phase 9 UI Overhaul)
            </p>
          </div>

        </div>
      </div>
    </div>
  );
};

export default SettingsPage;
