import type { FC } from 'react';
import { NavLink } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const Sidebar: FC = () => {
  const { logout } = useAuth();

  return (
    <aside className="sidebar glass-panel" style={{ display: 'flex', flexDirection: 'column' }}>
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
      
      <div style={{ marginTop: 'auto', padding: '0 1rem' }}>
        <button 
          onClick={logout}
          style={{ width: '100%', padding: '0.75rem', background: 'transparent', border: '1px solid #404040', color: '#b3b3b3', borderRadius: '4px', cursor: 'pointer' }}
        >
          Log Out
        </button>
      </div>
    </aside>
  );
};

export default Sidebar;
