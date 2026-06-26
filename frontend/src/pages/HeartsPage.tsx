import type { FC, ChangeEvent } from 'react';
import { useHearts } from '../context/HeartsContext';
import { apiService } from '../services/api';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const HeartsPage: FC = () => {
  const { heartedIds, refreshHearts } = useHearts();

  const handleExport = () => {
    // Native browser download directly from the backend
    window.open(`${API_BASE_URL}/api/hearts/export`, '_blank');
  };

  const handleImport = async (e: ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    const file = e.target.files[0];
    
    try {
      await apiService.importHearts(file);
      alert("Successfully imported backup!");
      refreshHearts(); // Sync the UI with the new database hearts
    } catch (err) {
      console.error(err);
      alert("Failed to import backup. Ensure it is a valid Supernova JSON backup.");
    }
    
    // Clear input so we can upload the same file again if needed
    e.target.value = '';
  };

  return (
    <div className="content-scroll">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <h1 style={{ fontSize: '32px', fontWeight: 800 }}>Your Hearts</h1>
        
        <div style={{ display: 'flex', gap: '12px' }}>
          {/* Hidden File Input for Import */}
          <input 
            type="file" 
            id="import-upload" 
            accept=".json" 
            style={{ display: 'none' }} 
            onChange={handleImport}
          />
          <label htmlFor="import-upload" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-glass-bright)', cursor: 'pointer', padding: '12px 24px', borderRadius: '12px', color: 'var(--text-primary)', fontWeight: 600, transition: 'var(--transition-fast)' }} onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-glass-hover)'} onMouseLeave={(e) => e.currentTarget.style.background = 'var(--bg-glass)'}>
            ↓ Import Backup
          </label>
          
          <button onClick={handleExport} style={{ background: 'var(--accent-gradient)', padding: '12px 24px', borderRadius: '12px', border: 'none', color: 'white', fontWeight: 700, cursor: 'pointer', boxShadow: 'var(--accent-glow)' }}>
            ↑ Export Backup
          </button>
        </div>
      </div>

      <div style={{ background: 'var(--bg-glass)', padding: '32px', borderRadius: '24px', border: '1px solid var(--border-glass-bright)', boxShadow: 'var(--shadow-drop)' }}>
        <p style={{ color: 'var(--text-secondary)', fontSize: '18px' }}>
          You have <strong style={{ color: 'var(--text-primary)', fontSize: '24px', margin: '0 4px' }}>{heartedIds.size}</strong> total hearts.
        </p>
        <p style={{ color: 'var(--text-muted)', fontSize: '14px', marginTop: '16px' }}>
          * In a future update, this page will beautifully list all your hearted tracks and albums. For now, you can freely backup and restore your collection!
        </p>
      </div>
    </div>
  );
};

export default HeartsPage;
