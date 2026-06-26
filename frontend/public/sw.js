const CACHE_NAME = 'supernova-pwa-cache-v1';

// We want to cache the application shell so it boots instantly even offline.
const urlsToCache = [
  '/',
  '/index.html',
  '/manifest.json'
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => cache.addAll(urlsToCache))
  );
});

self.addEventListener('fetch', event => {
  // Pass-through API requests so we don't accidentally cache dynamic JSON or music streams
  if (event.request.url.includes('/api/')) {
    return;
  }

  // Stale-While-Revalidate / Dynamic Read-Through caching for Vite assets
  event.respondWith(
    caches.match(event.request).then(cachedResponse => {
      // 1. Immediately return cached asset if it exists (offline-first)
      if (cachedResponse) {
        return cachedResponse;
      }
      
      // 2. Otherwise, fetch from network
      return fetch(event.request).then(networkResponse => {
        // Only cache valid HTTP responses (ignore cross-origin, errors, etc.)
        if (!networkResponse || networkResponse.status !== 200 || networkResponse.type !== 'basic') {
          return networkResponse;
        }
        
        // Clone the response because the stream can only be consumed once!
        const responseToCache = networkResponse.clone();
        caches.open(CACHE_NAME).then(cache => {
          cache.put(event.request, responseToCache);
        });
        
        return networkResponse;
      });
    })
  );
});
