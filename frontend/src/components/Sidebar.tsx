import type { FC } from 'react';
import { NavLink } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const Sidebar: FC = () => {
  const { logout } = useAuth();

  return (
    <aside className="sidebar">
      <div className="logo-container">
        <div className="logo-icon"></div>
        <h1 className="text-gradient">Supernova</h1>
      </div>
      <nav className="nav-menu">
        {/* NavLink automatically injects an 'active' class when the URL matches */}
        <NavLink to="/" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} end>
          Library
        </NavLink>
        <NavLink to="/hearts" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
          Hearts
        </NavLink>
        <NavLink to="/artists" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
          Artists
        </NavLink>
        <NavLink to="/albums" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
          Albums
        </NavLink>
        <NavLink to="/playlists" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
          Playlists
        </NavLink>
        <NavLink to="/settings" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
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
  );
};

export default Sidebar;
