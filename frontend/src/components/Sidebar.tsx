import type { FC } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { usePlaylists } from '../context/PlaylistsContext';

const Sidebar: FC<{ isOpen?: boolean; onClose?: () => void }> = ({ isOpen, onClose }) => {
  const { logout } = useAuth();
  const { playlists } = usePlaylists();

  return (
    <>
      {isOpen && <div className="sidebar-overlay" onClick={onClose} />}
      <aside className={`sidebar ${isOpen ? 'open' : ''}`}>
      <div className="logo-container">
        <img src="/logo.svg" alt="Supernova" className="logo-icon" />
        <h1 className="text-gradient">Supernova</h1>
      </div>
      <nav className="nav-menu">
        {/* NavLink automatically injects an 'active' class when the URL matches */}
        <NavLink to="/" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose} end>
          Home
        </NavLink>
        <NavLink to="/hearts" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Hearts
        </NavLink>
        <NavLink to="/artists" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Artists
        </NavLink>
        <NavLink to="/albums" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Albums
        </NavLink>
        <NavLink to="/radio" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Radio
        </NavLink>
        <NavLink to="/podcasts" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Podcasts
        </NavLink>
        <NavLink to="/playlists" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Playlists
        </NavLink>
        {playlists.length > 0 && (
          <div style={{ marginLeft: '16px', display: 'flex', flexDirection: 'column', gap: '4px', borderLeft: '1px solid var(--border-glass)', paddingLeft: '8px' }}>
            {playlists.map(playlist => (
              <Link 
                key={playlist.id}
                to="/playlists" 
                state={{ selectedPlaylistId: playlist.id }}
                className="nav-item"
                style={{ fontSize: '13px', padding: '8px 12px', opacity: 0.8 }}
                onClick={onClose}
              >
                {playlist.name}
              </Link>
            ))}
          </div>
        )}
        <NavLink to="/settings" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>
          Settings
        </NavLink>
      </nav>
      
      <div style={{ marginTop: 'auto' }}>
        <button 
          onClick={logout}
          style={{ width: '100%', padding: '12px', background: 'rgba(255, 68, 68, 0.1)', border: '1px solid rgba(255, 68, 68, 0.2)', color: '#ff4444', borderRadius: '12px', cursor: 'pointer', fontWeight: 600, fontSize: '14px', transition: 'var(--transition-fast)' }}
          onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 68, 68, 0.2)'}
          onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255, 68, 68, 0.1)'}
        >
          Log Out
        </button>
      </div>
    </aside>
    </>
  );
};

export default Sidebar;
