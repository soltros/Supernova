import type { FC } from 'react';

const TopBar: FC = () => {
  return (
    <header className="top-bar glass-panel">
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
