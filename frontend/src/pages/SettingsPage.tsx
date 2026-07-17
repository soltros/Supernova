import React, { useState, useEffect } from 'react';
import { CheckCircle, XCircle } from 'lucide-react';
import { apiService } from '../services/api';
import { useToast } from '../context/ToastContext';

const SettingsPage: React.FC = () => {
  const { addToast, confirm } = useToast();
  const [plugins, setPlugins] = useState<any[]>([]);
  const [scanStatus, setScanStatus] = useState<{status: string, files_scanned: number}>({ status: 'idle', files_scanned: 0 });
  const [lastfmSession, setLastfmSession] = useState<string | null>(localStorage.getItem('lastfm_session'));

  useEffect(() => {
    // Handle Last.fm OAuth callback
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    if (token) {
      apiService.exchangeLastFmToken(token)
      .then(data => {
        if (data.session_key) {
          localStorage.setItem('lastfm_session', data.session_key);
          setLastfmSession(data.session_key);
          window.history.replaceState({}, document.title, '/settings');
        }
      })
      .catch(console.error);
    }

    apiService.getPlugins()
      .then(data => setPlugins(data))
      .catch(err => console.error('Failed to load plugins', err));
      
    // Poll scan progress
    const checkProgress = () => {
      apiService.getScanProgress()
        .then(data => setScanStatus(data))
        .catch(() => {});
    };
    checkProgress();
    const interval = setInterval(checkProgress, 2000);
    return () => clearInterval(interval);
  }, []);

  const handleClearCache = async () => {
    if ('serviceWorker' in navigator) {
      const registrations = await navigator.serviceWorker.getRegistrations();
      for (const registration of registrations) {
        await registration.unregister();
      }
    }
    // Selectively remove non-auth cache keys - never clear the entire storage
    // to avoid a race condition that could log the user out mid-clear
    const keysToRemove: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key !== 'sn_token' && key !== 'sn_user' && key !== 'lastfm_session') {
        keysToRemove.push(key);
      }
    }
    keysToRemove.forEach(k => localStorage.removeItem(k));
    
    addToast('Cache cleared successfully! Reloading...', 'success');
    window.location.reload();
  };

  const [isResetting, setIsResetting] = useState(false);
  const handleResetArtists = async () => {
    const isConfirmed = await confirm("Are you sure? This will clear all artist bios and images and require re-fetching from Last.fm.");
    if (!isConfirmed) return;
    
    setIsResetting(true);
    try {
      await apiService.resetArtists();
      addToast("Artist data cleared successfully. The background scanner will re-fetch data shortly.", 'success');
    } catch (e) {
      addToast("Failed to reset artist data.", 'error');
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
            <h3 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '16px' }}>Installed Plugins</h3>
            {plugins.length === 0 ? (
              <p style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>No plugins are currently registered in the backend.</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {plugins.map(p => (
                  <div key={p.id} style={{ display: 'flex', alignItems: 'center', gap: '16px', padding: '12px', background: 'rgba(255,255,255,0.02)', borderRadius: '8px', border: '1px solid var(--border-glass)', opacity: p.enabled ? 1 : 0.6 }}>
                    {p.enabled ? <CheckCircle color="#10b981" size={24} /> : <XCircle color="var(--text-muted)" size={24} />}
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <strong style={{ color: 'var(--text-primary)' }}>{p.name}</strong>
                        {p.enabled && <span style={{ fontSize: '10px', background: 'rgba(16, 185, 129, 0.2)', color: '#10b981', padding: '2px 6px', borderRadius: '4px', fontWeight: 700, textTransform: 'uppercase' }}>Active</span>}
                      </div>
                      <p style={{ color: 'var(--text-secondary)', fontSize: '12px', marginTop: '4px' }}>{p.description}</p>
                      {!p.enabled && <p style={{ color: 'var(--text-muted)', fontSize: '11px', marginTop: '6px' }}>Enable by setting <code style={{ background: 'rgba(0,0,0,0.2)', padding: '2px 4px', borderRadius: '4px' }}>SUPERNOVA_PLUGIN_{p.id.toUpperCase()}=true</code></p>}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
          
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
              Manage your media directory and clean up metadata.
            </p>
            <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flexWrap: 'wrap' }}>
              <button 
                onClick={() => {
                  apiService.scanLibrary().catch(console.error);
                }}
                disabled={scanStatus.status === 'scanning'}
                style={{ background: 'rgba(139, 92, 246, 0.1)', border: '1px solid rgba(139, 92, 246, 0.2)', color: 'var(--accent-primary)', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, transition: 'var(--transition-fast)', opacity: scanStatus.status === 'scanning' ? 0.5 : 1 }}
              >
                {scanStatus.status === 'scanning' ? 'Scanning...' : 'Scan Library'}
              </button>

              {plugins.some(p => p.id === 'autotagger' && p.enabled) && (
                <button 
                  onClick={() => {
                    apiService.runPluginJob('autotagger').catch(console.error);
                    addToast("Auto-tagging job started in the background. Check backend logs for progress.", "success");
                  }}
                  style={{ background: 'rgba(16, 185, 129, 0.1)', border: '1px solid rgba(16, 185, 129, 0.2)', color: '#10b981', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, cursor: 'pointer' }}
                >
                  Run Auto-Tagger
                </button>
              )}

              {plugins.some(p => p.id === 'artistmerger' && p.enabled) && (
                <button 
                  onClick={() => {
                    apiService.runPluginJob('artistmerger').catch(console.error);
                    addToast("Artist merger job started in the background. Check backend logs for progress.", "success");
                  }}
                  style={{ background: 'rgba(245, 158, 11, 0.1)', border: '1px solid rgba(245, 158, 11, 0.2)', color: '#f59e0b', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, cursor: 'pointer' }}
                >
                  Merge Similar Artists
                </button>
              )}

              {plugins.some(p => p.id === 'deduper' && p.enabled) && (
                <button 
                  onClick={() => {
                    apiService.runPluginJob('deduper').catch(console.error);
                    addToast("Hide Duplicates job started in the background. Check backend logs for progress.", "success");
                  }}
                  style={{ background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.2)', color: '#ef4444', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, cursor: 'pointer' }}
                >
                  Hide Duplicates
                </button>
              )}

              {plugins.some(p => p.id === 'albummerger' && p.enabled) && (
                <button 
                  onClick={() => {
                    apiService.runPluginJob('albummerger').catch(console.error);
                    addToast("Album merger job started in the background. Check backend logs for progress.", "success");
                  }}
                  style={{ background: 'rgba(59, 130, 246, 0.1)', border: '1px solid rgba(59, 130, 246, 0.2)', color: '#3b82f6', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, cursor: 'pointer' }}
                >
                  Merge Similar Albums
                </button>
              )}

              {scanStatus.status === 'scanning' && (
                <span style={{ color: 'var(--text-secondary)', fontSize: '14px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <div className="loader" style={{ width: '16px', height: '16px', borderWidth: '2px', borderColor: 'rgba(255,255,255,0.1)', borderTopColor: 'var(--accent-primary)' }}></div>
                  Processed {scanStatus.files_scanned} files...
                </span>
              )}
            </div>
          </div>

          {plugins.some(p => p.id === 'lastfm' && p.enabled) && (
            <div style={{ background: 'var(--bg-glass)', padding: '24px', borderRadius: '16px', border: '1px solid var(--border-glass)' }}>
              <h3 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <div style={{ width: '20px', height: '20px', borderRadius: '4px', background: '#ba0000', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 900, fontSize: '12px' }}>
                  L
                </div>
                Last.fm Scrobbling
              </h3>
              <p style={{ color: 'var(--text-secondary)', fontSize: '14px', marginBottom: '16px' }}>
                Connect your Last.fm account to automatically track your listening history.
              </p>
              {lastfmSession ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', background: 'rgba(16, 185, 129, 0.1)', border: '1px solid rgba(16, 185, 129, 0.2)', padding: '10px 16px', borderRadius: '8px' }}>
                    <CheckCircle color="#10b981" size={18} />
                    <span style={{ color: '#10b981', fontWeight: 600, fontSize: '14px' }}>Connected & Scrobbling</span>
                  </div>
                  <button 
                    onClick={() => {
                      localStorage.removeItem('lastfm_session');
                      setLastfmSession(null);
                    }}
                    style={{ background: 'transparent', border: '1px solid var(--border-glass)', color: 'var(--text-primary)', padding: '10px 16px', borderRadius: '8px', fontWeight: 600, transition: 'var(--transition-fast)' }}
                  >
                    Disconnect
                  </button>
                </div>
              ) : (
                <button 
                  onClick={() => {
                    const cb = window.location.origin + "/settings";
                    apiService.getLastFmAuthUrl(cb)
                      .then(data => {
                        if (data.url) {
                          window.location.href = data.url;
                        }
                      });
                  }}
                  style={{ background: '#ba0000', color: '#fff', border: 'none', padding: '10px 20px', borderRadius: '8px', fontWeight: 600, cursor: 'pointer', transition: 'var(--transition-fast)' }}
                  onMouseEnter={(e) => e.currentTarget.style.background = '#d00000'}
                  onMouseLeave={(e) => e.currentTarget.style.background = '#ba0000'}
                >
                  Connect Last.fm
                </button>
              )}
            </div>
          )}

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
              Version: 2026.07.17
            </p>
          </div>

        </div>
      </div>
    </div>
  );
};

export default SettingsPage;
