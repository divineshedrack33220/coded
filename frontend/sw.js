// sw.js
// This Service Worker disables caching completely

self.addEventListener('install', event => {
  // Activate immediately
  self.skipWaiting();

  // Clear all caches on install
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.map(key => caches.delete(key)))
    )
  );
});

self.addEventListener('activate', event => {
  // Clear all caches again on activate (extra safety)
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.map(key => caches.delete(key)))
    )
  );

  // Take control immediately
  self.clients.claim();
});

// FETCH — always go to network, never cache
self.addEventListener('fetch', event => {
  event.respondWith(
    fetch(event.request, { cache: 'no-store' })
  );
});
