import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import type { User, AuthResponse } from '../types';
import { apiService } from '../services/api';

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (data: AuthResponse) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(() => {
    try {
      const storedUser = localStorage.getItem('sn_user');
      return storedUser ? JSON.parse(storedUser) : null;
    } catch {
      return null;
    }
  });
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('sn_token'));

  const clearLocalSession = useCallback(() => {
    setUser(null);
    setToken(null);
    localStorage.removeItem('sn_user');
    localStorage.removeItem('sn_token');
    localStorage.removeItem('lastfm_session');
  }, []);

  const logout = useCallback(() => {
    apiService.logout().catch(() => {}).finally(clearLocalSession);
  }, [clearLocalSession]);

  useEffect(() => {
    const handleAuthError = () => logout();
    window.addEventListener('auth_error', handleAuthError);
    return () => window.removeEventListener('auth_error', handleAuthError);
  }, [logout]);


  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    const base = import.meta.env.DEV ? (import.meta.env.VITE_API_URL || 'http://localhost:8080') : '';
    fetch(`${base}/api/auth/me`, { headers: { Authorization: `Bearer ${token}` } })
      .then(async response => {
        if (cancelled) return;
        if (response.status === 401) { clearLocalSession(); return; }
        if (!response.ok) return;
        const account = await response.json();
        if (!cancelled) { setUser(account); localStorage.setItem('sn_user', JSON.stringify(account)); }
      }).catch(() => {});
    return () => { cancelled = true; };
  }, [token, clearLocalSession]);

  const login = (data: AuthResponse) => {
    setUser(data.user);
    setToken(data.token);
    localStorage.setItem('sn_user', JSON.stringify(data.user));
    localStorage.setItem('sn_token', data.token);
  };



  return (
    <AuthContext.Provider value={{ user, token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within an AuthProvider');
  return context;
};
