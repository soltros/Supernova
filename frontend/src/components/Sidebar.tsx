import type { FC } from 'react';
import { NavLink } from 'react-router-dom';

const Sidebar: FC = () => {
  return (
    <aside className="sidebar glass-panel">
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
    </aside>
  );
};

export default Sidebar;
