import heroScreenshot from '../assets/hero-screenshot.png';
import { Sparkles, Music, Mic2, Radio, Wand2, Rocket, Heart } from 'lucide-react';
import { Link } from 'react-router-dom';

function Landing() {
  return (
    <>
      <div className="bg-gradients">
        <div className="glow-orb primary"></div>
        <div className="glow-orb secondary"></div>
      </div>

      <nav className="navbar">
        <a href="#" className="logo-container">
          <img src="./logo.svg" alt="Supernova Logo" />
          <span className="logo-text">Supernova</span>
        </a>
        <div className="nav-links">
          <Link to="/#features" className="nav-link">Features</Link>
          <Link to="/plugins" className="nav-link">Plugins</Link>
          <a href="https://github.com/soltros/Supernova" target="_blank" rel="noreferrer" className="nav-link">GitHub</a>
          <Link to="/#download" className="btn btn-primary" style={{ padding: '8px 20px' }}>Deploy</Link>
        </div>
      </nav>

      <div className="container">
        <section className="hero">
          <div className="hero-content">
            <div className="hero-text">
              <h1>The Infinite Local Music Player</h1>
              <p>Experience your local music library in a completely new light. Gorgeous glassmorphism UI, a massive plugin ecosystem, and seamless Docker Compose deployment.</p>
              <div className="hero-actions">
                <a href="#download" className="btn btn-primary">Deploy with Docker</a>
                <a href="https://github.com/soltros/Supernova" target="_blank" rel="noreferrer" className="btn btn-glass">View Source Code</a>
              </div>
            </div>
            <div className="hero-image-wrapper">
              <img src={heroScreenshot} alt="Supernova UI" className="hero-image" />
            </div>
          </div>
        </section>

        <section id="features" className="features">
          <h2 className="section-title">Beyond a typical player</h2>
          <div className="feature-grid">
            <div className="feature-card">
              <div className="feature-icon"><Sparkles size={24} /></div>
              <h3>Stunning Glassmorphism</h3>
              <p>Built with a breathtaking modern glassmorphism design that reacts and glows with your music. Every interaction feels alive.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon"><Music size={24} /></div>
              <h3>Automagic Metadata</h3>
              <p>Supernova seamlessly links your local files to Last.fm and iTunes to fetch high-res artwork, artist bios, and similar artists instantly.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon"><Mic2 size={24} /></div>
              <h3>Synchronized Lyrics</h3>
              <p>Sing along in style. Supernova automatically fetches and synchronizes lyrics for your entire library using LRCLIB.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon"><Radio size={24} /></div>
              <h3>Internet Radio & Podcasts</h3>
              <p>Tune into thousands of global radio stations via RadioBrowser or subscribe to your favorite shows via the Podcast Index.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon"><Wand2 size={24} /></div>
              <h3>Smart Library Management</h3>
              <p>Clean up your library automatically. Built-in deduplication, auto-tagging, and artist merging keep your collection perfectly organized.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon"><Rocket size={24} /></div>
              <h3>Subsonic API Support</h3>
              <p>Stream your library anywhere. Supernova includes full Subsonic API compatibility so you can use third-party mobile apps.</p>
            </div>
          </div>
        </section>



        <section id="download" className="download-section">
          <div className="download-box">
            <h2>Ready to launch?</h2>
            <p>Supernova is deployed instantly using a single `docker-compose.yml` file. Mount your music directory, map your database volume, and let the magic happen.</p>
            <div style={{ background: 'rgba(0,0,0,0.5)', padding: '24px', borderRadius: '12px', textAlign: 'left', fontFamily: 'monospace', color: '#a5b4fc', border: '1px solid rgba(255,255,255,0.1)', overflowX: 'auto', whiteSpace: 'pre' }}>
{`services:
  backend:
    image: ghcr.io/soltros/supernova-backend:main
    container_name: supernova-backend
    ports:
      - "8080:8080"
    volumes:
      - supernova-db:/root/.supernova/db
      - supernova-cache:/root/.supernova/art_cache
      - ./music:/root/Music:ro
    env_file:
      - .env
    restart: unless-stopped

  frontend:
    image: ghcr.io/soltros/supernova-frontend:main
    container_name: supernova-frontend
    ports:
      - "5174:80"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  supernova-db:
  supernova-cache:`}
            </div>
            
            <p style={{ marginTop: '24px', fontSize: '14px' }}>And save this configuration as <code>.env</code> in the same folder:</p>
            <div style={{ background: 'rgba(0,0,0,0.5)', padding: '24px', borderRadius: '12px', textAlign: 'left', fontFamily: 'monospace', color: '#a5b4fc', border: '1px solid rgba(255,255,255,0.1)', overflowX: 'auto', whiteSpace: 'pre' }}>
{`# ---- Core Configuration ----
MEDIA_PATH=/root/Music
DB_PATH=/root/supernova.db
ART_CACHE_PATH=/root/art_cache
PORT=8080
CORS_ALLOWED_ORIGIN=*
JWT_SECRET=change_me_to_a_random_string

# ---- Plugins ----
SUPERNOVA_PLUGIN_LASTFM=true
SUPERNOVA_PLUGIN_LRCLIB=true
SUPERNOVA_PLUGIN_SUBSONIC=true
SUPERNOVA_PLUGIN_RADIOBROWSER=true
SUPERNOVA_PLUGIN_PODCASTS=true

# ---- API Keys ----
LASTFM_API_KEY=your_key_here
LASTFM_API_SECRET=your_secret_here
PODCAST_INDEX_API_KEY=your_key_here
PODCAST_INDEX_API_SECRET=your_secret_here

# ---- Advanced Settings ----
# Set to true to calculate exact audio durations (slower initial library scan)
SUPERNOVA_ENABLE_ESTIMATES=false`}
            </div>

            <p style={{ marginTop: '24px', fontSize: '14px' }}>Then, just run <code>docker-compose up -d</code> to deploy the stack!</p>
          </div>
        </section>

        <footer>
          <p>© {new Date().getFullYear()} Supernova Open Source Project. Designed and built with <Heart size={14} style={{ display: 'inline', verticalAlign: 'middle', margin: '0 4px', color: '#ec4899' }} /></p>
        </footer>
      </div>
    </>
  );
}

export default Landing;
