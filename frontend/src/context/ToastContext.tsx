import React, { createContext, useContext, useState } from 'react';
import type { ReactNode } from 'react';
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react';

interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

interface ConfirmDialog {
  message: string;
  resolve: (value: boolean) => void;
}

interface ToastContextType {
  addToast: (message: string, type?: 'success' | 'error' | 'info') => void;
  confirm: (message: string) => Promise<boolean>;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) throw new Error('useToast must be used within ToastProvider');
  return context;
};

export const ToastProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialog | null>(null);

  const addToast = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
    const id = Date.now() + Math.random();
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
    }, 5000);
  };

  const confirm = (message: string): Promise<boolean> => {
    return new Promise((resolve) => {
      setConfirmDialog({ message, resolve });
    });
  };

  const handleConfirm = (result: boolean) => {
    if (confirmDialog) {
      confirmDialog.resolve(result);
      setConfirmDialog(null);
    }
  };

  return (
    <ToastContext.Provider value={{ addToast, confirm }}>
      {children}
      
      {/* Toast Container */}
      <div style={{ position: 'fixed', bottom: '24px', right: '24px', zIndex: 10000, display: 'flex', flexDirection: 'column', gap: '12px' }}>
        {toasts.map(toast => (
          <div key={toast.id} style={{
            background: 'var(--bg-secondary)',
            border: `1px solid ${toast.type === 'error' ? 'rgba(239,68,68,0.5)' : toast.type === 'success' ? 'rgba(34,197,94,0.5)' : 'var(--border-color)'}`,
            padding: '16px 20px',
            borderRadius: '12px',
            boxShadow: '0 8px 30px rgba(0,0,0,0.4)',
            display: 'flex', alignItems: 'center', gap: '12px',
            color: 'var(--text-primary)', minWidth: '300px', maxWidth: '400px',
            animation: 'slideInRight 0.3s cubic-bezier(0.16, 1, 0.3, 1)'
          }}>
            {toast.type === 'success' && <CheckCircle color="#22c55e" size={20} />}
            {toast.type === 'error' && <AlertCircle color="#ef4444" size={20} />}
            {toast.type === 'info' && <Info color="var(--accent-primary)" size={20} />}
            <span style={{ flex: 1, fontSize: '14px', lineHeight: '1.4' }}>{toast.message}</span>
            <button onClick={() => setToasts(prev => prev.filter(t => t.id !== toast.id))} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer' }}>
              <X size={16} />
            </button>
          </div>
        ))}
      </div>

      {/* Confirm Modal */}
      {confirmDialog && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(10px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          zIndex: 10001, animation: 'fadeIn 0.2s ease'
        }}>
          <div style={{
            background: 'var(--bg-secondary)', border: '1px solid var(--border-color)',
            padding: '32px', borderRadius: '16px', boxShadow: '0 20px 40px rgba(0,0,0,0.5)',
            maxWidth: '400px', width: '100%', textAlign: 'center',
            animation: 'scaleUp 0.2s cubic-bezier(0.16, 1, 0.3, 1)'
          }}>
            <h3 style={{ margin: '0 0 16px 0', fontSize: '20px' }}>Confirm</h3>
            <p style={{ margin: '0 0 32px 0', color: 'var(--text-secondary)', fontSize: '15px', lineHeight: '1.5' }}>{confirmDialog.message}</p>
            <div style={{ display: 'flex', gap: '16px', justifyContent: 'center' }}>
              <button 
                onClick={() => handleConfirm(false)}
                style={{ padding: '10px 24px', background: 'transparent', border: '1px solid var(--border-color)', color: 'var(--text-primary)', borderRadius: '8px', cursor: 'pointer', fontSize: '14px', fontWeight: 600 }}
              >
                Cancel
              </button>
              <button 
                onClick={() => handleConfirm(true)}
                style={{ padding: '10px 24px', background: 'var(--accent-primary)', border: 'none', color: 'white', borderRadius: '8px', cursor: 'pointer', fontSize: '14px', fontWeight: 600, boxShadow: 'var(--accent-glow)' }}
              >
                Confirm
              </button>
            </div>
          </div>
        </div>
      )}
    </ToastContext.Provider>
  );
};
