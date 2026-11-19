import {
  AudioLines, ChevronRight, Globe, Heart, Home, Library, ListMusic,
  Monitor, Music, Play, Radio, Search, Smartphone,
} from 'lucide-react';
import { Link } from 'react-router-dom';

const clientRepo = 'https://github.com/soltros/supernova-desktop';
const previewGuide = `${clientRepo}/blob/mobile/capacitor/PREVIEW.md`;
const platforms = [
  { name: 'iPhone and iPad', package: 'IPA', instructions: 'Sign and install the unsigned preview with SideStore. After initial setup, install updates from Files over Wi-Fi.' },
  { name: 'Android', package: 'APK', instructions: 'Open the APK on your phone. Allow installation from the browser or file manager if prompted.' },
];

function PhonePreview() {
  return (
    <figure className="client-phone-preview">
      <div className="client-phone" aria-hidden="true">
        <div className="client-phone-title">Library</div>
        <div className="client-phone-list">
          {[
            { text: 'Albums', Icon: Library },
            { text: 'Artists', Icon: Music },
            { text: 'Playlists', Icon: ListMusic },
            { text: 'Favorites', Icon: Heart },
            { text: 'Radio', Icon: Radio },
          ].map(({ text, Icon }) => (
            <div key={text}><Icon size={17} /><span>{text}</span><ChevronRight size={14} /></div>
          ))}
        </div>
        <div className="client-phone-player">
          <span className="client-phone-art"><AudioLines size={18} /></span>
          <span><strong>After Hours</strong><small>Now Playing</small></span>
          <Play size={19} fill="currentColor" />
        </div>
        <div className="client-phone-tabs">
          <span><Home size={18} />Home</span>
          <span className="current"><Library size={18} />Library</span>
          <span><Search size={18} />Search</span>
        </div>
      </div>
      <figcaption>Illustration of the mobile Library and mini-player</figcaption>
    </figure>
  );
}

export default function Clients() {
  return (
    <section id="clients" className="clients-section" aria-labelledby="clients-heading">
      <header className="clients-heading">
        <h2 id="clients-heading" className="section-title">Choose your Supernova client</h2>
        <p>Stream from the same self-hosted library on the web, Linux, iOS, or Android. Sign in to your server to pick up your favorites, playlists, and listening history.</p>
      </header>

      <div className="client-grid">
        <article className="feature-card">
          <div className="feature-icon"><Globe size={24} /></div>
          <h3>Web player</h3>
          <p>Open your server in a browser or install the PWA from a supported browser. Browse and search your library, queue music, make playlists, and explore favorites, listening history, and recommendations.</p>
        </article>
        <article className="feature-card">
          <div className="feature-icon"><Monitor size={24} /></div>
          <h3>Linux desktop</h3>
          <p>Use Supernova in a dedicated desktop app with a library sidebar, keyboard shortcuts, media controls, a playback queue, and track and album downloads. Linux x86-64 previews are packaged as DEB, RPM, and AppImage.</p>
          <a className="client-link" href={clientRepo}>Desktop and mobile source <ChevronRight size={17} /></a>
        </article>
      </div>

      <article id="mobile" className="mobile-showcase">
        <div className="mobile-showcase-copy">
          <div className="feature-icon"><Smartphone size={24} /></div>
          <h3>Supernova on iOS and Android</h3>
          <p>Swipe through album shelves, switch between Home, Library, and Search, and tap the mini-player to open a full-screen Now Playing view. Browse collections and keep the controls within reach.</p>
          <ul className="client-platforms">
            {platforms.map(({ name, package: format, instructions }) => (
              <li key={name}>
                <strong>{name}</strong><span>{format}</span>
                <p>{instructions}</p>
              </li>
            ))}
          </ul>
          <p className="client-note">Connect using a trusted HTTPS server address. Preview packages and installation steps are available from the client project.</p>
          <div className="client-actions">
            <a className="btn btn-primary" href={`${clientRepo}/actions?query=branch%3Amobile%2Fcapacitor`}>Preview builds</a>
            <a className="btn btn-glass" href={previewGuide}>Installation guide</a>
          </div>
        </div>
        <PhonePreview />
      </article>

      <div className="client-plugin-note">
        <Radio size={23} aria-hidden="true" />
        <div>
          <h3>More with server plugins</h3>
          <p>Discover and save stations with Radio Browser, subscribe to podcasts and resume episodes, follow LRCLIB lyrics, and connect Last.fm for scrobbling. Desktop and mobile read the server's plugin list. In either app, open <strong>Settings → Server</strong> to see which plugins are enabled and manage backups or administrator tools.</p>
        </div>
        <Link className="client-link" to="/plugins">Explore plugins <ChevronRight size={17} /></Link>
      </div>
    </section>
  );
}
