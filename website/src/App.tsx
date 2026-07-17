function App() {
  return (
    <>
      <div className="bg-gradients">
        <div className="glow-orb primary"></div>
        <div className="glow-orb secondary"></div>
      </div>

      <nav className="navbar">
        <a href="#" className="logo-container">
          <img src="/logo.svg" alt="Supernova Logo" />
          <span className="logo-text">Supernova</span>
        </a>
        <div className="nav-links">
          <a href="#features" className="nav-link">Features</a>
          <a href="https://github.com/soltros/Supernova" target="_blank" rel="noreferrer" className="nav-link">GitHub</a>
          <a href="#download" className="btn btn-primary" style={{ padding: '8px 20px' }}>Download</a>
        </div>
      </nav>

      <div className="container">
        <section className="hero">
          <div className="hero-content">
            <div className="hero-text">
              <h1>The Infinite Local Music Player</h1>
              <p>Experience your local music library in a completely new light. Gorgeous glassmorphism UI, AI-first lyrics fetching, and seamless metadata integration.</p>
              <div className="hero-actions">
                <a href="#download" className="btn btn-primary">Get Started for Free</a>
                <a href="https://github.com/soltros/Supernova" target="_blank" rel="noreferrer" className="btn btn-glass">View Source Code</a>
              </div>
            </div>
            <div className="hero-image-wrapper">
              <img src="/hero-screenshot.png" alt="Supernova UI" className="hero-image" />
            </div>
          </div>
        </section>

        <section id="features" className="features">
          <h2 className="section-title">Beyond a typical player</h2>
          <div className="feature-grid">
            <div className="feature-card">
              <div className="feature-icon">✨</div>
              <h3>Stunning Glassmorphism</h3>
              <p>Built with a breathtaking modern glassmorphism design that reacts and glows with your music. Every interaction feels alive.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon">🎵</div>
              <h3>Automagic Metadata</h3>
              <p>Supernova seamlessly links your local files to Last.fm and iTunes to fetch high-res artwork, artist bios, and similar artists instantly.</p>
            </div>
            <div className="feature-card">
              <div className="feature-icon">📝</div>
              <h3>AI-First Lyrics</h3>
              <p>Don't have lyrics for a track? Supernova's built-in AI will intelligently fetch them for you. Sing along in style.</p>
            </div>
          </div>
        </section>

        <section id="download" className="download-section">
          <div className="download-box">
            <h2>Ready to launch?</h2>
            <p>Supernova runs seamlessly as a Docker container. Connect it to your local music directory and let the magic happen.</p>
            <div style={{ background: 'rgba(0,0,0,0.5)', padding: '24px', borderRadius: '12px', textAlign: 'left', fontFamily: 'monospace', color: '#a5b4fc', border: '1px solid rgba(255,255,255,0.1)' }}>
              docker run -d \<br/>
              &nbsp;&nbsp;-p 8080:8080 -p 5174:80 \<br/>
              &nbsp;&nbsp;-v /path/to/your/music:/root/Music:ro \<br/>
              &nbsp;&nbsp;ghcr.io/soltros/supernova:latest
            </div>
          </div>
        </section>

        <footer>
          <p>© {new Date().getFullYear()} Supernova Open Source Project. Designed and built with ❤️</p>
        </footer>
      </div>
    </>
  );
}

export default App;
