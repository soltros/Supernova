import type { FC } from 'react';
import { useState } from 'react';
import { LogOut, UserRound } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const TopBar: FC<{ onMenuClick?: () => void }> = ({ onMenuClick }) => {
  const [query, setQuery] = useState('');
  const navigate = useNavigate();
  const { user, logout } = useAuth();

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && query.trim()) {
      navigate(`/search?q=${encodeURIComponent(query.trim())}`);
    }
  };

  return (
    <header className="top-bar">
      {onMenuClick && (
        <button type="button" className="mobile-menu-btn" onClick={onMenuClick} aria-label="Open navigation menu">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg>
        </button>
      )}
      <input type="text" className="search-bar" placeholder="Search artists, albums, or tracks..."
        aria-label="Search music" value={query} onChange={(e) => setQuery(e.target.value)} onKeyDown={handleKeyDown} />
      <div className="user-profile">
        <span className="topbar-username" title={user?.username}><UserRound size={16} aria-hidden="true" /> {user?.username}</span>
        <button type="button" className="topbar-logout" onClick={logout} title="Log out and choose another user">
          <LogOut size={16} aria-hidden="true" /><span>Switch user</span>
        </button>
      </div>
    </header>
  );
};
export default TopBar;
