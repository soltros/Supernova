import { createContext, useState, useContext, useEffect, useCallback } from 'react';
import type { FC, ReactNode } from 'react';
import { apiService } from '../services/api';

interface HeartsState {
  heartedIds: Set<string>;
  toggleHeart: (entityType: string, entityId: string) => Promise<void>;
  isHearted: (entityId: string) => boolean;
  refreshHearts: () => Promise<void>;
}

const HeartsContext = createContext<HeartsState | undefined>(undefined);

export const HeartsProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [heartedIds, setHeartedIds] = useState<Set<string>>(new Set());

  const refreshHearts = useCallback(async () => {
    try {
      const hearts = await apiService.fetchHearts();
      const newSet = new Set(hearts.map(h => h.entity_id));
      setHeartedIds(newSet);
    } catch (e) {
      console.error("Failed to fetch hearts:", e);
    }
  }, []);

  useEffect(() => {
    refreshHearts();
  }, [refreshHearts]);

  const toggleHeart = async (entityType: string, entityId: string) => {
    const currentlyHearted = heartedIds.has(entityId);
    
    // Optimistic UI update
    setHeartedIds(prev => {
      const next = new Set(prev);
      if (currentlyHearted) next.delete(entityId);
      else next.add(entityId);
      return next;
    });

    try {
      if (currentlyHearted) {
        await apiService.removeHeart(entityType, entityId);
      } else {
        // The backend securely generates the UUID now
        await apiService.addHeart(entityType, entityId);
      }
    } catch (e) {
      console.error("Failed to toggle heart:", e);
      // Rollback optimistic update on error
      setHeartedIds(prev => {
        const next = new Set(prev);
        if (currentlyHearted) next.add(entityId);
        else next.delete(entityId);
        return next;
      });
    }
  };

  const isHearted = (entityId: string) => heartedIds.has(entityId);

  return (
    <HeartsContext.Provider value={{ heartedIds, toggleHeart, isHearted, refreshHearts }}>
      {children}
    </HeartsContext.Provider>
  );
};

export const useHearts = () => {
  const context = useContext(HeartsContext);
  if (!context) throw new Error('useHearts must be used within HeartsProvider');
  return context;
};
