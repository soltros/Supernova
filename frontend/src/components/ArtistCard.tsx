import type { FC } from 'react';
import { Link } from 'react-router-dom';
import type { Artist } from '../types';

interface Props {
  artist: Artist;
}

const ArtistCard: FC<Props> = ({ artist }) => {
  return (
    <Link to={`/artist/${artist.id}`} style={{ textDecoration: 'none', color: 'inherit' }}>
      <div className="album-card" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
        <div style={{ position: 'relative', width: '100%', aspectRatio: '1/1', borderRadius: '50%', overflow: 'hidden', marginBottom: '16px', backgroundColor: 'var(--bg-glass)' }}>
          {artist.image_url && artist.image_url !== 'NOT_FOUND' ? (
            <img 
              src={artist.image_url} 
              alt={artist.name} 
              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
              onError={(e) => {
                e.currentTarget.style.display = 'none';
                if (e.currentTarget.nextElementSibling) {
                  (e.currentTarget.nextElementSibling as HTMLElement).style.display = 'flex';
                }
              }}
            />
          ) : (
            <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '48px', fontWeight: 800, color: 'var(--text-secondary)' }}>
              {artist.name.charAt(0)}
            </div>
          )}
          <div style={{ display: 'none', width: '100%', height: '100%', position: 'absolute', top: 0, left: 0, alignItems: 'center', justifyContent: 'center', fontSize: '48px', fontWeight: 800, color: 'var(--text-secondary)' }}>
            <span>{artist.name.charAt(0)}</span>
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
          <h3 style={{ margin: '0 0 4px 0', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap', fontSize: '18px', fontWeight: 700 }}>{artist.name}</h3>
          <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>Artist</span>
        </div>
      </div>
    </Link>
  );
};

export default ArtistCard;
