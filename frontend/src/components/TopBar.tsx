import type { FC } from 'react';

const TopBar: FC<{ onMenuClick?: () => void }> = ({ onMenuClick }) => {
  return (
    <header className="top-bar">
      {onMenuClick && (
        <button className="mobile-menu-btn" onClick={onMenuClick}>
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg>
        </button>
      )}
      <input 
        type="text" 
        className="search-bar" 
        placeholder="Search artists, albums, or tracks..." 
      />
      <div className="user-profile"></div>
    </header>
  );
};

export default TopBar;
