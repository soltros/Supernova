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
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', backgroundColor: '#121212' }}>
      <div style={{ backgroundColor: '#1e1e1e', padding: '2rem', borderRadius: '8px', width: '300px', boxShadow: '0 4px 6px rgba(0,0,0,0.5)' }}>
        <h2 style={{ color: 'white', textAlign: 'center', marginBottom: '1.5rem' }}>
          {isRegistering ? 'Create Account' : 'Welcome Back'}
        </h2>
        
        {error && <div style={{ color: '#ff4444', marginBottom: '1rem', textAlign: 'center' }}>{error}</div>}
        
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          <input
            type="text"
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            style={{ padding: '0.75rem', borderRadius: '4px', border: 'none', backgroundColor: '#2d2d2d', color: 'white' }}
            required
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={{ padding: '0.75rem', borderRadius: '4px', border: 'none', backgroundColor: '#2d2d2d', color: 'white' }}
            required
          />
          <button
            type="submit"
            style={{ padding: '0.75rem', borderRadius: '4px', border: 'none', backgroundColor: '#1db954', color: 'white', fontWeight: 'bold', cursor: 'pointer' }}
          >
            {isRegistering ? 'Sign Up' : 'Log In'}
          </button>
        </form>

        <p style={{ color: '#b3b3b3', textAlign: 'center', marginTop: '1.5rem', fontSize: '0.875rem' }}>
          {isRegistering ? 'Already have an account? ' : "Don't have an account? "}
          <span 
            onClick={() => setIsRegistering(!isRegistering)}
            style={{ color: 'white', cursor: 'pointer', textDecoration: 'underline' }}
          >
            {isRegistering ? 'Log in' : 'Sign up'}
          </span>
        </p>
      </div>
    </div>
  );
};
