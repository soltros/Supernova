import React, { createContext, useState, useContext, useEffect, useCallback } from 'react';
import type { FC, ReactNode } from 'react';
import { apiService } from '../services/api';

interface HeartsState {
  heartedIds: Set<string>;
  toggleHeart: (entityType: string, entityId: string, metadata?: any) => Promise<void>;
  isHearted: (entityId: string) => boolean;
  refreshHearts: () => Promise<void>;
}

const HeartsContext = createContext<HeartsState | undefined>(undefined);

export const HeartsProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [heartedIds, setHeartedIds] = useState<Set<string>>(new Set());

  const refreshHearts = useCallback(async () => {
    try {
      const hearts = await apiService.fetchHearts();
      const newSet = new Set((hearts || []).map(h => h.entity_id));
      setHeartedIds(newSet);

      const migrateLegacy = async (key: string, entityType: 'radio' | 'podcast', idOf: (value: any) => string) => {
        let cached: any[] = [];
        try {
          const parsed = JSON.parse(localStorage.getItem(key) || '[]');
          if (Array.isArray(parsed)) cached = parsed;
        } catch {
          return;
        }
        if (cached.length === 0) return;
        const migrated = new Set<string>();
        for (const value of cached) {
          const id = idOf(value);
          if (!id || !newSet.has(id)) continue;
          try {
            await apiService.addHeart(entityType, id, value);
            migrated.add(id);
          } catch {
            // Keep failed entries for a later retry.
          }
        }
        if (migrated.size === 0) return;
        const remaining = cached.filter(value => !migrated.has(idOf(value)));
        if (remaining.length === 0) localStorage.removeItem(key);
        else localStorage.setItem(key, JSON.stringify(remaining));
      };

      await migrateLegacy('heartedRadioStations', 'radio', value => String(value?.stationuuid || ''));
      await migrateLegacy('heartedPodcasts', 'podcast', value => String(value?.id ?? ''));
    } catch (e) {
      console.error("Failed to fetch hearts:", e);
    }
  }, []);

  useEffect(() => {
    refreshHearts();
  }, [refreshHearts]);

  const toggleHeart = useCallback(async (entityType: string, entityId: string, metadata?: any) => {
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
        await apiService.addHeart(entityType, entityId, metadata);
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
  }, [heartedIds]);

  const isHearted = useCallback((entityId: string) => heartedIds.has(entityId), [heartedIds]);

  const value = React.useMemo(() => ({
    heartedIds, toggleHeart, isHearted, refreshHearts
  }), [heartedIds, toggleHeart, isHearted, refreshHearts]);

  return (
    <HeartsContext.Provider value={value}>
      {children}
    </HeartsContext.Provider>
  );
};

export const useHearts = () => {
  const context = useContext(HeartsContext);
  if (!context) throw new Error('useHearts must be used within HeartsProvider');
  return context;
};
