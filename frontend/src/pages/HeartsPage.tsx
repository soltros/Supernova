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
          <label htmlFor="import-upload" className="btn-primary" style={{ background: 'var(--bg-glass-hover)', cursor: 'pointer', padding: '8px 16px', borderRadius: '24px', color: 'var(--text-primary)' }}>
            ↓ Import Backup
          </label>
          
          <button onClick={handleExport} className="btn-primary" style={{ padding: '8px 16px' }}>
            ↑ Export Backup
          </button>
        </div>
      </div>

      <div className="glass-panel" style={{ padding: '24px' }}>
        <p style={{ color: 'var(--text-secondary)' }}>
          You have <strong style={{ color: 'var(--accent-primary)' }}>{heartedIds.size}</strong> total hearts.
        </p>
        <p style={{ color: 'var(--text-muted)', fontSize: '13px', marginTop: '12px' }}>
          * In a future update, this page will beautifully list all your hearted tracks and albums. For now, you can freely backup and restore your collection!
        </p>
      </div>
    </div>
  );
};

export default HeartsPage;
