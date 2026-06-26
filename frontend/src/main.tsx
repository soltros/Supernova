import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.tsx';
import { PlayerProvider } from './context/PlayerContext';

// Forcefully unregister any old, aggressively cached Service Workers
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.getRegistrations().then(registrations => {
    for (const registration of registrations) {
      registration.unregister();
      console.log('Unregistered old Service Worker');
    }
  });

  // Register the new Service Worker and force it to ignore HTTP caching
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js', { updateViaCache: 'none' }).catch(err => {
      console.log('ServiceWorker registration failed: ', err);
    });
  });
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {/* Wrap the entire app in the Router so navigation works */}
    <BrowserRouter>
      {/* Wrap the app in the global audio state provider */}
      <PlayerProvider>
        <App />
      </PlayerProvider>
    </BrowserRouter>
  </StrictMode>,
);
