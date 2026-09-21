import type { FC } from 'react';
import { Link } from 'react-router-dom';
import type { Album } from '../types';

const API_BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';

interface Props { album: Album; }

const AlbumCard: FC<Props> = ({ album }) => (
  <article className="album-card">
    <Link to={`/album/${album.id}`} style={{ textDecoration: 'none', color: 'inherit' }} aria-label={`Open album ${album.title}`}>
      <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden' }}>
        <img
          src={`${API_BASE_URL}/api/art/album/${album.id}`}
          alt=""
          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            if (e.currentTarget.nextElementSibling) (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'flex';
          }}
        />
        <div className="album-art-placeholder" aria-hidden="true" style={{ display: 'none', width: '100%', height: '100%', position: 'absolute', top: 0, left: 0 }}>
          <span>{album.title.charAt(0)}</span>
        </div>
      </div>
      <h3 style={{ margin: '8px 0 4px 0', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>{album.title}</h3>
    </Link>
    {album.artist_id && (
      <Link
        to={`/artist/${album.artist_id}`}
        style={{ fontSize: '13px', color: 'var(--text-muted)', textDecoration: 'none', transition: 'color 0.2s', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}
      >
        {album.artist_name}
      </Link>
    )}
  </article>
);
export default AlbumCard;
