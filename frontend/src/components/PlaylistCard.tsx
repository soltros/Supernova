import type { FC } from 'react';
import { Link } from 'react-router-dom';
import type { Playlist } from '../types';
import { ListMusic } from 'lucide-react';

interface Props {
  playlist: Playlist;
}

const PlaylistCard: FC<Props> = ({ playlist }) => {
  return (
    <Link to="/playlists" state={{ selectedPlaylistId: playlist.id }} style={{ textDecoration: 'none', color: 'inherit' }}>
      <div className="album-card">
        <div style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden', backgroundColor: 'var(--bg-glass)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <ListMusic size={64} style={{ opacity: 0.2, color: 'var(--text-secondary)' }} />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', marginTop: '12px' }}>
          <h3 style={{ margin: '0 0 4px 0', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap', fontSize: '15px', fontWeight: 600 }}>{playlist.name}</h3>
          <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>Playlist</span>
        </div>
      </div>
    </Link>
  );
};

export default PlaylistCard;
