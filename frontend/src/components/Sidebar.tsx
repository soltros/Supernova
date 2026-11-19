import type { FC } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { LogOut, UserRound } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { usePlaylists } from '../context/PlaylistsContext';

const Sidebar: FC<{ isOpen?: boolean; onClose?: () => void }> = ({ isOpen, onClose }) => {
  const { user, logout } = useAuth();
  const { playlists } = usePlaylists();
  const switchUser = () => { onClose?.(); logout(); };

  return (
    <>
      {isOpen && <div className="sidebar-overlay" onClick={onClose} />}
      <aside className={`sidebar ${isOpen ? 'open' : ''}`}>
        <div className="logo-container">
          <img src="/logo.svg" alt="Supernova" className="logo-icon" />
          <h1 className="text-gradient">Supernova</h1>
        </div>
        <nav className="nav-menu" aria-label="Main navigation">
          <NavLink to="/" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose} end>Home</NavLink>
          <NavLink to="/hearts" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Hearts</NavLink>
          <NavLink to="/artists" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Artists</NavLink>
          <NavLink to="/albums" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Albums</NavLink>
          <NavLink to="/radio" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Radio</NavLink>
          <NavLink to="/podcasts" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Podcasts</NavLink>
          <NavLink to="/playlists" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Playlists</NavLink>
          {playlists.length > 0 && (
            <div className="sidebar-playlists" aria-label="Your playlists">
              {playlists.map(playlist => (
                <Link key={playlist.id} to="/playlists" state={{ selectedPlaylistId: playlist.id }}
                  className="nav-item sidebar-playlist-link" onClick={onClose} title={playlist.name}>
                  <span className="sidebar-playlist-name">{playlist.name}</span>
                </Link>
              ))}
            </div>
          )}
          <NavLink to="/settings" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} onClick={onClose}>Settings</NavLink>
        </nav>
        <div className="sidebar-account">
          <div className="sidebar-account-name"><UserRound size={17} aria-hidden="true" /><span title={user?.username}>{user?.username}</span></div>
          <button type="button" className="sidebar-account-action" onClick={switchUser}>
            <LogOut size={17} aria-hidden="true" /> Log out / Switch user
          </button>
        </div>
      </aside>
    </>
  );
};

export default Sidebar;
