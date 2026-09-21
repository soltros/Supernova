import { Link } from 'react-router-dom';
import { Heart, Puzzle, Zap, Globe, HardDrive, ListMusic, AudioLines, RadioReceiver, PenTool } from 'lucide-react';

function Plugins() {
  return (
    <>
      <div className="bg-gradients">
        <div className="glow-orb primary"></div>
        <div className="glow-orb secondary" style={{ left: '80%', top: '20%' }}></div>
      </div>

      <nav className="navbar">
        <Link to="/" className="logo-container">
          <img src="/logo.svg" alt="Supernova Logo" />
          <span className="logo-text">Supernova</span>
        </Link>
        <div className="nav-links">
          <Link to="/#features" className="nav-link">Features</Link>
          <Link to="/plugins" className="nav-link" style={{ color: '#fff', textShadow: '0 0 10px rgba(255,255,255,0.5)' }}>Plugins</Link>
          <a href="https://github.com/soltros/Supernova" target="_blank" rel="noreferrer" className="nav-link">GitHub</a>
          <Link to="/#download" className="btn btn-primary" style={{ padding: '8px 20px' }}>Deploy</Link>
        </div>
      </nav>

      <div className="container" style={{ paddingTop: '120px' }}>
        <section className="hero" style={{ minHeight: 'auto', paddingBottom: '40px' }}>
          <div className="hero-content" style={{ flexDirection: 'column', textAlign: 'center' }}>
            <div className="hero-text" style={{ maxWidth: '800px', margin: '0 auto' }}>
              <div style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(236, 72, 153, 0.1)', border: '1px solid rgba(236, 72, 153, 0.2)', padding: '8px 16px', borderRadius: '30px', color: '#ec4899', marginBottom: '24px', fontWeight: 600 }}>
                <Puzzle size={16} style={{ marginRight: '8px' }} />
                <span>Plugin Ecosystem</span>
              </div>
              <h1 style={{ fontSize: '4rem', marginBottom: '24px' }}>Infinitely Extensible</h1>
              <p style={{ fontSize: '1.25rem', color: 'var(--text-secondary)' }}>
                Supernova is built from the ground up to be modular. Whether you want to integrate with external APIs, sync metadata, or stream to mobile clients—there's a plugin for that.
              </p>
            </div>
          </div>
        </section>

        <section className="plugins-section" style={{ paddingTop: '0' }}>
          <div className="plugins-grid">
            
            {/* Last.fm Plugin */}
            <div className="plugin-item" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '16px', padding: '32px', transition: 'all 0.3s ease' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
                <div style={{ background: 'rgba(239, 68, 68, 0.1)', color: '#ef4444', padding: '12px', borderRadius: '12px' }}>
                  <Globe size={28} />
                </div>
                <span className="plugin-badge">Included Core</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>Last.fm Metadata Sync</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Automatically enriches your local library by fetching artist bios, high-resolution imagery, top tags, and discovering similar artists using the Last.fm API.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Requires: LASTFM_API_KEY</span>
              </div>
            </div>

            {/* LRCLIB Plugin */}
            <div className="plugin-item" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '16px', padding: '32px', transition: 'all 0.3s ease' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
                <div style={{ background: 'rgba(59, 130, 246, 0.1)', color: '#3b82f6', padding: '12px', borderRadius: '12px' }}>
                  <ListMusic size={28} />
                </div>
                <span className="plugin-badge">Included Core</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>LRCLIB Synchronized Lyrics</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Transforms your listening experience by automatically downloading and displaying time-synced lyrics (LRC) for tracks in your collection.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Zero Configuration Needed</span>
              </div>
            </div>

            {/* Subsonic API Plugin */}
            <div className="plugin-item" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '16px', padding: '32px', transition: 'all 0.3s ease' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
                <div style={{ background: 'rgba(16, 185, 129, 0.1)', color: '#10b981', padding: '12px', borderRadius: '12px' }}>
                  <HardDrive size={28} />
                </div>
                <span className="plugin-badge">Included Core</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>Subsonic API Layer</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Stream your music through Supernova's implemented Subsonic/OpenSubsonic compatibility subset. Browsing, paged search, playlists, favorites, scrobbling, raw playback, and supported transcoding are available, but this is not a claim of complete protocol conformance.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Compatibility varies by client and endpoint</span>
              </div>
            </div>

            {/* RadioBrowser Plugin */}
            <div className="plugin-item" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '16px', padding: '32px', transition: 'all 0.3s ease' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
                <div style={{ background: 'rgba(245, 158, 11, 0.1)', color: '#f59e0b', padding: '12px', borderRadius: '12px' }}>
                  <RadioReceiver size={28} />
                </div>
                <span className="plugin-badge">Included Core</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>Internet RadioBrowser</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Tap into over 40,000 free internet radio stations globally. Browse by language, country, or genre right from the Supernova interface.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Community Driven</span>
              </div>
            </div>

            {/* Podcast Index Plugin */}
            <div className="plugin-item" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '16px', padding: '32px', transition: 'all 0.3s ease' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
                <div style={{ background: 'rgba(139, 92, 246, 0.1)', color: '#8b5cf6', padding: '12px', borderRadius: '12px' }}>
                  <AudioLines size={28} />
                </div>
                <span className="plugin-badge">Included Core</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>Podcast Index Integration</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Search, subscribe, and listen to millions of podcasts without leaving your music player. Fully integrated with your local library.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Requires: PODCAST_INDEX_API_KEY</span>
              </div>
            </div>

            {/* AutoTagger Plugin */}
            <div className="plugin-item" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '16px', padding: '32px', transition: 'all 0.3s ease' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
                <div style={{ background: 'rgba(236, 72, 153, 0.1)', color: '#ec4899', padding: '12px', borderRadius: '12px' }}>
                  <PenTool size={28} />
                </div>
                <span className="plugin-badge">Included Core</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>MusicBrainz AutoTagger</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Applies database-only metadata cleanup and path-based inference for missing or generic library metadata. It does not rewrite the media files on disk or claim acoustic-fingerprint matching.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Database-only metadata enrichment</span>
              </div>
            </div>

          </div>
        </section>
        <section className="how-to-install" style={{ padding: '40px 20px', maxWidth: '800px', margin: '0 auto 80px', background: 'rgba(255,255,255,0.02)', borderRadius: '16px', border: '1px solid rgba(255,255,255,0.05)' }}>
          <h2 style={{ fontSize: '2rem', marginBottom: '24px', textAlign: 'center', color: '#fff' }}>How Plugins Work</h2>
          <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '32px', textAlign: 'center' }}>
            Supernova currently uses a compile-time Go plugin registry. Official plugins are built into the backend and can be enabled or disabled with environment variables. Arbitrary runtime .so loading is not implemented.
          </p>
          
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            <div style={{ display: 'flex', gap: '20px' }}>
              <div style={{ background: '#ec4899', color: '#fff', width: '32px', height: '32px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', flexShrink: 0 }}>1</div>
              <div>
                <h4 style={{ fontSize: '1.25rem', marginBottom: '8px', color: '#fff' }}>Enable Core Plugins</h4>
                <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '16px' }}>Core plugins are already bundled with Supernova. To enable one, just add its flag to your <code>.env</code> file:</p>
                <div style={{ background: 'rgba(0,0,0,0.5)', padding: '16px', borderRadius: '8px', fontFamily: 'monospace', color: '#a5b4fc', border: '1px solid rgba(255,255,255,0.1)' }}>
                  SUPERNOVA_PLUGIN_LASTFM=true<br/>
                  LASTFM_API_KEY=your_api_key_here
                </div>
              </div>
            </div>
            
            <div style={{ display: 'flex', gap: '20px' }}>
              <div style={{ background: '#ec4899', color: '#fff', width: '32px', height: '32px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', flexShrink: 0 }}>2</div>
              <div>
                <h4 style={{ fontSize: '1.25rem', marginBottom: '8px', color: '#fff' }}>Add a Custom Plugin at Build Time</h4>
                <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '16px' }}>Custom plugins implement the Go <code>plugins.Plugin</code> interface, register themselves with <code>plugins.Register</code>, and are blank-imported by the server so they are compiled into the backend binary.</p>
                <div style={{ background: 'rgba(0,0,0,0.5)', padding: '16px', borderRadius: '8px', fontFamily: 'monospace', color: '#a5b4fc', border: '1px solid rgba(255,255,255,0.1)' }}>
                  backend/internal/plugins/yourplugin/<br/>
                  &nbsp;&nbsp;plugin.go<br/><br/>
                  import _ "github.com/soltros/Supernova/internal/plugins/yourplugin"
                </div>
              </div>
            </div>
            
            <div style={{ display: 'flex', gap: '20px' }}>
              <div style={{ background: '#ec4899', color: '#fff', width: '32px', height: '32px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', flexShrink: 0 }}>3</div>
              <div>
                <h4 style={{ fontSize: '1.25rem', marginBottom: '8px', color: '#fff' }}>Rebuild and Restart</h4>
                <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6' }}>After adding a compile-time plugin, rebuild the backend image or binary and restart Supernova. Built-in plugin enable/disable environment changes only require recreating or restarting the server process.</p>
              </div>
            </div>
          </div>
        </section>

        <footer>
          <p>© {new Date().getFullYear()} Supernova Open Source Project. Designed and built with <Heart size={14} style={{ display: 'inline', verticalAlign: 'middle', margin: '0 4px', color: '#ec4899' }} /></p>
        </footer>
      </div>
    </>
  );
}

export default Plugins;
