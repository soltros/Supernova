import type { FC } from 'react';
import { Link } from 'react-router-dom';
import type { Album } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface Props {
  album: Album;
}

const AlbumCard: FC<Props> = ({ album }) => {
  return (
    <Link to={`/album/${album.id}`} style={{ textDecoration: 'none', color: 'inherit' }}>
      <div className="album-card">
        <div className="album-art-container" style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '8px', overflow: 'hidden' }}>
          <img 
            src={`${API_BASE_URL}/api/art/album/${album.id}`}
            alt={album.title}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              if (e.currentTarget.nextElementSibling) {
                (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'flex';
              }
            }}
          />
          <div className="album-art-placeholder" style={{ display: 'none', width: '100%', height: '100%', position: 'absolute', top: 0, left: 0 }}>
            <span>{album.title.charAt(0)}</span>
          </div>
        </div>
        <div className="album-info">
          <h3>{album.title}</h3>
        </div>
      </div>
    </Link>
  );
};

export default AlbumCard;
