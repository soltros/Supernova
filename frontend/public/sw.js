const CACHE_NAME = 'supernova-pwa-cache-v2';

// We want to cache the application shell so it boots instantly even offline.
const urlsToCache = [
  '/',
  '/index.html',
  '/manifest.json'
];

self.addEventListener('install', event => {
  self.skipWaiting(); // Force new service worker to take over immediately
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => cache.addAll(urlsToCache))
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.map(cacheName => {
          if (cacheName !== CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => self.clients.claim()) // Take control of all clients immediately
  );
});

self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);

  // Pass-through API requests and media streams completely
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/stream/')) {
    return;
  }

  // Network-First for HTML (Always fetch fresh index.html, fallback to cache if offline)
  if (event.request.mode === 'navigate' || event.request.destination === 'document') {
    event.respondWith(
      fetch(event.request).then(networkResponse => {
        const responseToCache = networkResponse.clone();
        caches.open(CACHE_NAME).then(cache => cache.put(event.request, responseToCache));
        return networkResponse;
      }).catch(() => {
        return caches.match(event.request).then(cached => {
          if (cached) return cached;
          return caches.match('/index.html'); // Fallback to root shell
        });
      })
    );
    return;
  }

  // Cache-First (Stale-While-Revalidate pattern) for Vite hashed assets
  event.respondWith(
    caches.match(event.request).then(cachedResponse => {
      // Background revalidation
      const fetchPromise = fetch(event.request).then(networkResponse => {
        if (networkResponse && networkResponse.status === 200) {
          const responseToCache = networkResponse.clone();
          caches.open(CACHE_NAME).then(cache => cache.put(event.request, responseToCache));
        }
        return networkResponse;
      }).catch(() => { /* offline silently */ });

      // Return cache immediately if available, otherwise wait for network
      return cachedResponse || fetchPromise;
    })
  );
});
