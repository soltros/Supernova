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
                Stream your music anywhere. Exposes a fully-compatible OpenSubsonic REST API, allowing you to use mobile apps like Aonsoku, Symfonium, DSub, and Amperfy.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>Supports modern token auth</span>
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
                <span className="plugin-badge" style={{ background: 'rgba(255,255,255,0.1)', color: '#fff' }}>Coming Soon</span>
              </div>
              <h3 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>MusicBrainz AutoTagger</h3>
              <p style={{ color: 'var(--text-secondary)', lineHeight: '1.6', marginBottom: '20px' }}>
                Automatically scans your files, generates acoustic fingerprints, and fixes messy or missing ID3 tags to keep your library pristine.
              </p>
              <div style={{ fontSize: '0.9rem', color: '#9ca3af', display: 'flex', alignItems: 'center' }}>
                <Zap size={14} style={{ marginRight: '6px', color: '#fbbf24' }} />
                <span>In Development</span>
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
