import type { FC, MouseEvent } from 'react';
import { Heart } from 'lucide-react';
import { useHearts } from '../context/HeartsContext';

interface Props {
  entityType: 'track' | 'album' | 'artist' | 'playlist' | 'radio' | 'podcast';
  entityId: string;
  size?: number;
}

const HeartButton: FC<Props> = ({ entityType, entityId, size = 18 }) => {
  const { isHearted, toggleHeart } = useHearts();
  const active = isHearted(entityId);

  const handleClick = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation(); // Don't trigger the track's row onClick (which would play the song)
    toggleHeart(entityType, entityId);
  };

  return (
    <button 
      onClick={handleClick}
      style={{
        background: 'none',
        border: 'none',
        cursor: 'pointer',
        fontSize: `${size}px`,
        color: active ? '#ff3b30' : 'var(--text-muted)',
        transition: 'all 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275)',
        transform: active ? 'scale(1.1)' : 'scale(1)',
        padding: '4px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
      }}
      title={active ? "Unheart" : "Heart"}
      onMouseEnter={(e) => {
        e.currentTarget.style.color = '#ff3b30';
        e.currentTarget.style.transform = 'scale(1.2)';
      }}
      onMouseLeave={(e) => {
        if (!active) e.currentTarget.style.color = 'var(--text-muted)';
        e.currentTarget.style.transform = 'scale(1)';
      }}
    >
      <Heart size={size} fill={active ? 'currentColor' : 'none'} strokeWidth={active ? 0 : 2} />
    </button>
  );
};

export default HeartButton;
