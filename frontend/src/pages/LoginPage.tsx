import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { apiService } from '../services/api';

export const LoginPage: React.FC = () => {
  const { login } = useAuth();
  const [isRegistering, setIsRegistering] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    try {
      if (isRegistering) {
        const data = await apiService.register(username, password);
        login(data);
      } else {
        const data = await apiService.login(username, password);
        login(data);
      }
    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-box">
        <h2 style={{ color: 'white', textAlign: 'center', marginBottom: '32px', fontSize: '32px', fontWeight: 800, letterSpacing: '-1px' }}>
          {isRegistering ? 'Create Account' : 'Welcome Back'}
        </h2>
        
        {error && <div style={{ color: '#ff4444', marginBottom: '24px', textAlign: 'center', background: 'rgba(255, 68, 68, 0.1)', padding: '12px', borderRadius: '8px', border: '1px solid rgba(255, 68, 68, 0.2)' }}>{error}</div>}
        
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          <input
            type="text"
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            style={{ padding: '16px', borderRadius: '12px', border: '1px solid var(--border-glass-bright)', backgroundColor: 'var(--bg-glass)', color: 'white', outline: 'none', fontFamily: 'Outfit', fontSize: '16px' }}
            required
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={{ padding: '16px', borderRadius: '12px', border: '1px solid var(--border-glass-bright)', backgroundColor: 'var(--bg-glass)', color: 'white', outline: 'none', fontFamily: 'Outfit', fontSize: '16px' }}
            required
          />
          <button
            type="submit"
            style={{ padding: '16px', borderRadius: '12px', border: 'none', background: 'var(--accent-gradient)', color: 'white', fontWeight: 700, fontSize: '18px', cursor: 'pointer', boxShadow: 'var(--accent-glow)', marginTop: '8px' }}
          >
            {isRegistering ? 'Sign Up' : 'Log In'}
          </button>
        </form>

        <p style={{ color: 'var(--text-secondary)', textAlign: 'center', marginTop: '32px', fontSize: '15px' }}>
          {isRegistering ? 'Already have an account? ' : "Don't have an account? "}
          <span 
            onClick={() => setIsRegistering(!isRegistering)}
            style={{ color: 'var(--accent-primary)', cursor: 'pointer', fontWeight: 700, textDecoration: 'none' }}
          >
            {isRegistering ? 'Log in' : 'Sign up'}
          </span>
        </p>
      </div>
    </div>
  );
};
